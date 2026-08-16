package switcher

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	"NanoKVM-Server/proto"
)

// Targets are stored on the device rather than in the browser so the same
// buttons appear from any machine that connects to this KVM.
const (
	ConfigFile = "/etc/kvm/switcher.json"

	defaultStepDelayMs = 120
	maxStepDelayMs     = 5000
	maxTargets         = 32
	maxSteps           = 8
	maxKeysPerStep     = 4
	maxNameLength      = 64
)

var (
	configMu   sync.Mutex
	configPath = ConfigFile
)

type Config struct {
	Targets     []proto.SwitcherTarget `json:"targets"`
	StepDelayMs int                    `json:"stepDelayMs"`
}

func defaultConfig() Config {
	// Never nil: a nil slice marshals to null, which the UI would map over.
	return Config{
		Targets:     []proto.SwitcherTarget{},
		StepDelayMs: defaultStepDelayMs,
	}
}

func (s *Service) GetSwitcher(c *gin.Context) {
	var rsp proto.Response

	cfg, err := loadConfig()
	if err != nil {
		log.Errorf("failed to load switcher config: %s", err)
		rsp.ErrRsp(c, -1, "failed to load switcher config")
		return
	}

	rsp.OkRspWithData(c, &proto.GetSwitcherRsp{
		Targets:     cfg.Targets,
		StepDelayMs: cfg.StepDelayMs,
	})
}

func (s *Service) SetSwitcher(c *gin.Context) {
	var req proto.SetSwitcherReq
	var rsp proto.Response

	if err := proto.ParseFormRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}

	cfg := Config{Targets: req.Targets, StepDelayMs: req.StepDelayMs}
	if err := validate(&cfg); err != nil {
		rsp.ErrRsp(c, -2, err.Error())
		return
	}

	if err := saveConfigToPath(configPath, cfg); err != nil {
		log.Errorf("failed to save switcher config: %s", err)
		rsp.ErrRsp(c, -3, "failed to save switcher config")
		return
	}

	rsp.OkRsp(c)
}

func validate(cfg *Config) error {
	if cfg.Targets == nil {
		cfg.Targets = []proto.SwitcherTarget{}
	}
	if cfg.StepDelayMs == 0 {
		cfg.StepDelayMs = defaultStepDelayMs
	}
	if cfg.StepDelayMs < 0 || cfg.StepDelayMs > maxStepDelayMs {
		return fmt.Errorf("step delay must be between 0 and %d ms", maxStepDelayMs)
	}
	if len(cfg.Targets) > maxTargets {
		return fmt.Errorf("at most %d targets are supported", maxTargets)
	}

	seen := make(map[string]struct{}, len(cfg.Targets))
	for i := range cfg.Targets {
		target := &cfg.Targets[i]
		target.Name = strings.TrimSpace(target.Name)

		if target.Name == "" {
			return errors.New("every target needs a name")
		}
		if len(target.Name) > maxNameLength {
			return fmt.Errorf("name %q is too long", target.Name)
		}
		if _, ok := seen[target.Name]; ok {
			return fmt.Errorf("duplicate target name %q", target.Name)
		}
		seen[target.Name] = struct{}{}

		if len(target.Steps) == 0 {
			return fmt.Errorf("target %q has no keys assigned", target.Name)
		}
		if len(target.Steps) > maxSteps {
			return fmt.Errorf("target %q has more than %d steps", target.Name, maxSteps)
		}
		for _, step := range target.Steps {
			if len(step.Keys) == 0 {
				return fmt.Errorf("target %q has an empty step", target.Name)
			}
			// The HID report carries at most six non-modifier keycodes.
			if len(step.Keys) > maxKeysPerStep {
				return fmt.Errorf("target %q presses too many keys at once", target.Name)
			}
		}
	}

	return nil
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
		return Config{}, fmt.Errorf("parse switcher config: %w", err)
	}
	if cfg.Targets == nil {
		cfg.Targets = []proto.SwitcherTarget{}
	}
	if cfg.StepDelayMs <= 0 {
		cfg.StepDelayMs = defaultStepDelayMs
	}

	return cfg, nil
}

func saveConfigToPath(path string, cfg Config) error {
	configMu.Lock()
	defer configMu.Unlock()

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create switcher config directory: %w", err)
	}

	data, err := json.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("marshal switcher config: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".switcher.json.*")
	if err != nil {
		return fmt.Errorf("create temporary switcher config: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	if err := tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("set switcher config permissions: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write switcher config: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync switcher config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close switcher config: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace switcher config: %w", err)
	}

	if handle, err := os.Open(dir); err == nil {
		_ = handle.Sync()
		_ = handle.Close()
	}

	return nil
}
