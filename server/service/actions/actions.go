package actions

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	"NanoKVM-Server/proto"
)

// User-defined actions: drive GPIO or PWM, run commands, or any sequence of
// those, triggered from the menu bar or the NanoKVM's own button.
//
// The SoC exposes five gpiochips covering 352-511 and four PWM chips, of which
// the firmware itself uses a handful. The rest are free, which is what makes
// relays, LEDs and anything else wired to the header usable without a shell.
const (
	ConfigFile = "/etc/kvm/actions.json"

	gpioExportPath = "/sys/class/gpio/export"
	gpioBase       = "/sys/class/gpio"

	// The lowest and highest gpio numbers this SoC's chips provide.
	minGPIO = 352
	maxGPIO = 511

	maxActions      = 32
	maxPulseMs      = 10000
	maxNameLength   = 48
	commandTimeout  = 30 * time.Second
	defaultPulseDur = 500
)

// Pins the firmware drives itself. Using them is allowed — repurposing the ATX
// power and reset lines is a reasonable thing to want — but the UI names them
// so it is a choice rather than a surprise.
var reservedGPIO = map[int]string{
	503: "target power",
	504: "power LED",
	505: "target reset",
	507: "target reset",
}

var (
	configMu   sync.Mutex
	configPath = ConfigFile
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func defaultConfig() Config {
	return Config{
		Actions: []Action{},
		Buttons: ButtonMap{KeepDefaults: true},
	}
}

func (s *Service) GetActions(c *gin.Context) {
	var rsp proto.Response

	cfg, err := loadConfig()
	if err != nil {
		log.Errorf("failed to load actions: %s", err)
		rsp.ErrRsp(c, -1, "failed to load actions")
		return
	}

	rsp.OkRspWithData(c, toProto(cfg))
}

func (s *Service) SetActions(c *gin.Context) {
	var req proto.SetActionsReq
	var rsp proto.Response

	if err := proto.ParseFormRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}

	cfg := fromProto(req)
	if err := validate(&cfg); err != nil {
		rsp.ErrRsp(c, -2, err.Error())
		return
	}

	if err := saveConfig(cfg); err != nil {
		log.Errorf("failed to save actions: %s", err)
		rsp.ErrRsp(c, -3, "failed to save actions")
		return
	}

	rsp.OkRspWithData(c, toProto(cfg))
}

func (s *Service) RunAction(c *gin.Context) {
	var req proto.RunActionReq
	var rsp proto.Response

	if err := proto.ParseFormRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}

	action, err := findAction(req.Id)
	if err != nil {
		rsp.ErrRsp(c, -2, "action not found")
		return
	}

	if err := Run(action); err != nil {
		log.Errorf("action %q failed: %s", action.Name, err)
		// The reason is passed through: "step 2 (gpio): export gpio 460" says
		// far more than "action failed".
		rsp.ErrRsp(c, -3, err.Error())
		return
	}

	log.Infof("ran action %q", action.Name)
	rsp.OkRsp(c)
}

// GetGPIOState reports a pin's current value, so a toggle can show its state.
func (s *Service) GetGPIOState(c *gin.Context) {
	var rsp proto.Response

	cfg, err := loadConfig()
	if err != nil {
		rsp.ErrRsp(c, -1, "failed to load actions")
		return
	}

	states := make([]proto.GPIOState, 0)
	seen := make(map[int]struct{})

	for _, action := range cfg.Actions {
		for _, step := range action.Steps {
			if step.Type != "gpio" || step.GPIO == nil {
				continue
			}
			if _, done := seen[step.GPIO.Pin]; done {
				continue
			}
			seen[step.GPIO.Pin] = struct{}{}

			state := proto.GPIOState{Pin: step.GPIO.Pin}
			// A pin that has never been driven is not exported, and reading it
			// would export it as a side effect of merely looking.
			if value, err := ReadGPIO(step.GPIO.Pin); err == nil {
				state.Value = value
				state.Known = true
			}
			states = append(states, state)
		}
	}

	rsp.OkRspWithData(c, &proto.GetGPIOStateRsp{States: states})
}

// RunByID is what the button watcher calls, since a binding stores only an id.
func RunByID(id string) error {
	if id == "" {
		return nil
	}

	action, err := findAction(id)
	if err != nil {
		return err
	}

	return Run(action)
}

func findAction(id string) (*Action, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}

	for i := range cfg.Actions {
		if cfg.Actions[i].Id == id || cfg.Actions[i].Name == id {
			return &cfg.Actions[i], nil
		}
	}

	return nil, errors.New("action not found")
}

func validate(cfg *Config) error {
	if cfg.Actions == nil {
		cfg.Actions = []Action{}
	}
	if len(cfg.Actions) > maxActions {
		return fmt.Errorf("at most %d actions are supported", maxActions)
	}

	names := make(map[string]struct{}, len(cfg.Actions))
	ids := make(map[string]struct{}, len(cfg.Actions))

	for i := range cfg.Actions {
		action := &cfg.Actions[i]
		action.Name = strings.TrimSpace(action.Name)

		if action.Id == "" || strings.HasPrefix(action.Id, "new-") {
			action.Id = uuid.NewString()
		}
		if action.Name == "" {
			return errors.New("every action needs a name")
		}
		if len(action.Name) > maxNameLength {
			return fmt.Errorf("name %q is too long", action.Name)
		}
		if _, duplicate := names[action.Name]; duplicate {
			return fmt.Errorf("duplicate action name %q", action.Name)
		}
		names[action.Name] = struct{}{}
		ids[action.Id] = struct{}{}

		if len(action.Steps) == 0 {
			return fmt.Errorf("action %q has no steps", action.Name)
		}
		if len(action.Steps) > maxStepsPerAction {
			return fmt.Errorf("action %q has more than %d steps", action.Name, maxStepsPerAction)
		}

		for j := range action.Steps {
			if err := validateStep(action.Name, j, &action.Steps[j]); err != nil {
				return err
			}
		}
	}

	// A binding pointing at a deleted action would fail silently on a press.
	for _, binding := range []string{
		cfg.Buttons.ShortPress,
		cfg.Buttons.DoublePress,
		cfg.Buttons.LongPress,
		cfg.Buttons.VeryLongPress,
	} {
		if binding == "" {
			continue
		}
		if _, ok := ids[binding]; !ok {
			return errors.New("a button is bound to an action that no longer exists")
		}
	}

	return nil
}

func validateStep(actionName string, index int, step *Step) error {
	where := fmt.Sprintf("action %q step %d", actionName, index+1)

	switch step.Type {
	case "gpio":
		if step.GPIO == nil {
			return fmt.Errorf("%s: no gpio settings", where)
		}
		if step.GPIO.Pin < minGPIO || step.GPIO.Pin > maxGPIO {
			return fmt.Errorf("%s: gpio must be between %d and %d", where, minGPIO, maxGPIO)
		}
		switch step.GPIO.Mode {
		case "high", "low", "pulse", "toggle":
		default:
			return fmt.Errorf("%s: mode must be high, low, pulse or toggle", where)
		}
		if step.GPIO.DurationMs < 0 || step.GPIO.DurationMs > maxPulseMs {
			return fmt.Errorf("%s: pulse must be between 0 and %d ms", where, maxPulseMs)
		}
	case "pwm":
		if step.PWM == nil {
			return fmt.Errorf("%s: no pwm settings", where)
		}
		if step.PWM.Chip < 0 || step.PWM.Channel < 0 {
			return fmt.Errorf("%s: pwm chip and channel must not be negative", where)
		}
		if step.PWM.DutyPercent < 0 || step.PWM.DutyPercent > 100 {
			return fmt.Errorf("%s: duty must be between 0 and 100", where)
		}
		if step.PWM.PeriodNs < 0 {
			return fmt.Errorf("%s: period must not be negative", where)
		}
	case "command":
		if step.Command == nil || strings.TrimSpace(step.Command.Command) == "" {
			return fmt.Errorf("%s: no command", where)
		}
		if step.Command.TimeoutSec < 0 || step.Command.TimeoutSec > maxCommandTimeout {
			return fmt.Errorf("%s: timeout must be between 0 and %d seconds", where, maxCommandTimeout)
		}
	case "delay":
		if step.DelayMs <= 0 || step.DelayMs > maxDelayMs {
			return fmt.Errorf("%s: delay must be between 1 and %d ms", where, maxDelayMs)
		}
	default:
		return fmt.Errorf("%s: type must be gpio, pwm, command or delay", where)
	}

	return nil
}

// ReservedGPIO reports what the firmware uses a pin for, so the UI can warn
// rather than refuse.
func ReservedGPIO(pin int) string {
	return reservedGPIO[pin]
}

func loadConfig() (Config, error) {
	configMu.Lock()
	defer configMu.Unlock()

	content, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return defaultConfig(), nil
		}
		return Config{}, err
	}

	cfg := defaultConfig()
	if err := json.Unmarshal(content, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse actions: %w", err)
	}
	if cfg.Actions == nil {
		cfg.Actions = []Action{}
	}

	// Actions saved before steps existed are converted on read, so upgrading
	// does not silently drop what someone had configured.
	for i := range cfg.Actions {
		cfg.Actions[i].migrate()
	}

	return cfg, nil
}

func saveConfig(cfg Config) error {
	configMu.Lock()
	defer configMu.Unlock()

	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}

	data, err := json.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("marshal actions: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".actions.json.*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	// 0600: an action can hold a command line worth keeping to root.
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	return os.Rename(tmpPath, configPath)
}
