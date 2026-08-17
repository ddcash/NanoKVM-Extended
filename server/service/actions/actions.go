package actions

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	"NanoKVM-Server/proto"
)

// User-defined actions: drive a GPIO pin, or run a command.
//
// The SoC exposes five gpiochips covering 352-511, of which the firmware itself
// only uses a handful. The rest are free, which is what makes relays, LEDs and
// anything else wired to the header usable without changing firmware.
const (
	ConfigFile = "/etc/kvm/actions.json"

	gpioExportPath   = "/sys/class/gpio/export"
	gpioUnexportPath = "/sys/class/gpio/unexport"
	gpioBase         = "/sys/class/gpio"

	// The lowest and highest gpio numbers any of this SoC's chips provide.
	// Outside this a write would either fail or hit something unrelated.
	minGPIO = 352
	maxGPIO = 511

	maxActions      = 32
	maxPulseMs      = 10000
	maxNameLength   = 48
	commandTimeout  = 30 * time.Second
	defaultPulseDur = 500
)

// Pins the firmware drives itself. Using them is allowed — repurposing the ATX
// power and reset lines is a reasonable thing to want — but the UI says what
// they are so it is a choice rather than a surprise.
var reservedGPIO = map[int]string{
	503: "target power",
	505: "target reset",
	507: "target reset",
	504: "power LED",
}

type GPIOSpec struct {
	Pin int `json:"pin"`
	// "high", "low" or "pulse".
	Mode string `json:"mode"`
	// Pulse length in milliseconds.
	DurationMs int `json:"durationMs"`
	// Wiring where the pin is pulled low to activate, which is how most relay
	// boards are built.
	ActiveLow bool `json:"activeLow"`
}

type Action struct {
	Id   string `json:"id"`
	Name string `json:"name"`
	// "gpio" or "command".
	Type    string    `json:"type"`
	GPIO    *GPIOSpec `json:"gpio,omitempty"`
	Command string    `json:"command,omitempty"`
	// Shown in the menu bar. An action can exist for the physical button alone.
	ShowInMenu bool `json:"showInMenu"`
}

// ButtonMap binds the NanoKVM's own button to actions. There is one button,
// read as gpio-keys on /dev/input/event0, distinguished by how long it is held.
type ButtonMap struct {
	ShortPress string `json:"shortPress"`
	LongPress  string `json:"longPress"`
	// The daemon's own handling (OLED pages, Wi-Fi config, password reset) keeps
	// working alongside a custom action unless this is cleared. Suppressing it
	// entirely would mean changing the C++ daemon, and losing the password reset
	// is not a good trade.
	KeepDefaults bool `json:"keepDefaults"`
}

type Config struct {
	Actions []Action  `json:"actions"`
	Buttons ButtonMap `json:"buttons"`
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
		rsp.ErrRsp(c, -3, fmt.Sprintf("action failed: %s", err))
		return
	}

	log.Infof("ran action %q", action.Name)
	rsp.OkRsp(c)
}

// Run performs an action. Exported so the button watcher can use it.
func Run(action *Action) error {
	switch action.Type {
	case "gpio":
		if action.GPIO == nil {
			return errors.New("no gpio configured")
		}
		return driveGPIO(*action.GPIO)
	case "command":
		return runCommand(action.Command)
	default:
		return fmt.Errorf("unknown action type %q", action.Type)
	}
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

// driveGPIO exports the pin if needed, sets it as an output and writes.
//
// The pin is deliberately left exported afterwards: unexporting resets the
// line, which would drop a relay that was just switched on.
func driveGPIO(spec GPIOSpec) error {
	if spec.Pin < minGPIO || spec.Pin > maxGPIO {
		return fmt.Errorf("gpio %d is outside the range this device provides", spec.Pin)
	}

	pinDir := filepath.Join(gpioBase, fmt.Sprintf("gpio%d", spec.Pin))
	if _, err := os.Stat(pinDir); err != nil {
		if err := os.WriteFile(gpioExportPath, []byte(strconv.Itoa(spec.Pin)), 0o200); err != nil {
			return fmt.Errorf("export gpio %d: %w", spec.Pin, err)
		}
		// udev creates the attributes asynchronously.
		for attempt := 0; attempt < 20; attempt++ {
			if _, err := os.Stat(filepath.Join(pinDir, "value")); err == nil {
				break
			}
			time.Sleep(25 * time.Millisecond)
		}
	}

	if err := os.WriteFile(filepath.Join(pinDir, "direction"), []byte("out"), 0o644); err != nil {
		return fmt.Errorf("set gpio %d as output: %w", spec.Pin, err)
	}

	active, idle := "1", "0"
	if spec.ActiveLow {
		active, idle = "0", "1"
	}

	valuePath := filepath.Join(pinDir, "value")

	switch spec.Mode {
	case "high":
		return os.WriteFile(valuePath, []byte(active), 0o644)
	case "low":
		return os.WriteFile(valuePath, []byte(idle), 0o644)
	case "pulse":
		duration := spec.DurationMs
		if duration <= 0 {
			duration = defaultPulseDur
		}
		if err := os.WriteFile(valuePath, []byte(active), 0o644); err != nil {
			return err
		}
		time.Sleep(time.Duration(duration) * time.Millisecond)
		return os.WriteFile(valuePath, []byte(idle), 0o644)
	default:
		return fmt.Errorf("unknown gpio mode %q", spec.Mode)
	}
}

func runCommand(command string) error {
	command = strings.TrimSpace(command)
	if command == "" {
		return errors.New("no command configured")
	}

	// Bounded, so a command that never returns cannot pin a core on a device
	// that only has one.
	cmd := exec.Command("sh", "-c", command)
	if err := cmd.Start(); err != nil {
		return err
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case err := <-done:
		return err
	case <-time.After(commandTimeout):
		_ = cmd.Process.Kill()
		return fmt.Errorf("command timed out after %s", commandTimeout)
	}
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

		if action.Id == "" {
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

		switch action.Type {
		case "gpio":
			if action.GPIO == nil {
				return fmt.Errorf("action %q has no gpio settings", action.Name)
			}
			if action.GPIO.Pin < minGPIO || action.GPIO.Pin > maxGPIO {
				return fmt.Errorf("action %q: gpio must be between %d and %d", action.Name, minGPIO, maxGPIO)
			}
			switch action.GPIO.Mode {
			case "high", "low", "pulse":
			default:
				return fmt.Errorf("action %q: mode must be high, low or pulse", action.Name)
			}
			if action.GPIO.DurationMs < 0 || action.GPIO.DurationMs > maxPulseMs {
				return fmt.Errorf("action %q: pulse must be between 0 and %d ms", action.Name, maxPulseMs)
			}
		case "command":
			if strings.TrimSpace(action.Command) == "" {
				return fmt.Errorf("action %q has no command", action.Name)
			}
		default:
			return fmt.Errorf("action %q: type must be gpio or command", action.Name)
		}
	}

	// A binding pointing at a deleted action would fail silently on a press.
	for _, binding := range []string{cfg.Buttons.ShortPress, cfg.Buttons.LongPress} {
		if binding == "" {
			continue
		}
		if _, ok := ids[binding]; !ok {
			return errors.New("a button is bound to an action that no longer exists")
		}
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
