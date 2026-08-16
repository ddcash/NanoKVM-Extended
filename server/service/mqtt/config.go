package mqtt

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

const (
	ConfigFile     = "/etc/kvm/mqtt.json"
	maxFieldLength = 512
	maxCommands    = 64
)

var (
	configMu   sync.Mutex
	configPath = ConfigFile
)

type Config struct {
	Enabled  bool                `json:"enabled"`
	Broker   string              `json:"broker"`
	Port     int                 `json:"port"`
	Tls      bool                `json:"tls"`
	Username string              `json:"username"`
	Password string              `json:"password"`
	Topic    string              `json:"topic"`
	Commands []proto.MqttCommand `json:"commands"`
}

func defaultConfig() Config {
	return Config{Port: 1883}
}

func (s *Service) GetConfig(c *gin.Context) {
	var rsp proto.Response

	cfg, err := loadConfig()
	if err != nil {
		log.Errorf("failed to load mqtt config: %s", err)
		rsp.ErrRsp(c, -1, "failed to load mqtt config")
		return
	}

	rsp.OkRspWithData(c, &proto.GetMqttConfigRsp{
		Enabled:     cfg.Enabled,
		Broker:      cfg.Broker,
		Port:        cfg.Port,
		Tls:         cfg.Tls,
		Username:    cfg.Username,
		Topic:       cfg.Topic,
		Commands:    cfg.Commands,
		HasPassword: cfg.Password != "",
	})
}

func (s *Service) SetConfig(c *gin.Context) {
	var req proto.SetMqttConfigReq
	var rsp proto.Response

	// Bound directly rather than through proto.ParseFormRequest, which logs the
	// whole request body in debug mode. This one carries the broker password.
	if err := c.ShouldBind(&req); err != nil || req.Enabled == nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}

	current, err := loadConfig()
	if err != nil {
		log.Errorf("failed to load mqtt config: %s", err)
		rsp.ErrRsp(c, -1, "failed to load mqtt config")
		return
	}

	cfg := Config{
		Enabled:  *req.Enabled,
		Broker:   strings.TrimSpace(req.Broker),
		Port:     req.Port,
		Username: req.Username,
		Topic:    strings.TrimSpace(req.Topic),
		Commands: req.Commands,
		Tls:      current.Tls,
		Password: current.Password,
	}
	if req.Tls != nil {
		cfg.Tls = *req.Tls
	}
	// A nil password means "leave what is stored"; the UI never receives the
	// stored value back, so it cannot echo it here.
	if req.Password != nil {
		cfg.Password = *req.Password
	}

	if err := validateConfig(&cfg); err != nil {
		rsp.ErrRsp(c, -2, err.Error())
		return
	}

	if err := saveConfigToPath(configPath, cfg); err != nil {
		log.Errorf("failed to save mqtt config: %s", err)
		rsp.ErrRsp(c, -3, "failed to save mqtt config")
		return
	}

	rsp.OkRsp(c)
}

func validateConfig(cfg *Config) error {
	if cfg.Port == 0 {
		cfg.Port = 1883
	}
	if cfg.Port < 1 || cfg.Port > 65535 {
		return errors.New("port must be between 1 and 65535")
	}
	if cfg.Enabled && cfg.Broker == "" {
		return errors.New("broker is required")
	}
	if len(cfg.Commands) > maxCommands {
		return fmt.Errorf("at most %d commands are supported", maxCommands)
	}

	for _, field := range []string{cfg.Broker, cfg.Username, cfg.Password, cfg.Topic} {
		if len(field) > maxFieldLength {
			return errors.New("configuration field is too long")
		}
	}

	seen := make(map[string]struct{}, len(cfg.Commands))
	for i := range cfg.Commands {
		cmd := &cfg.Commands[i]
		cmd.Name = strings.TrimSpace(cmd.Name)
		cmd.Topic = strings.TrimSpace(cmd.Topic)

		if cmd.Name == "" {
			return errors.New("command name is required")
		}
		if len(cmd.Name) > maxFieldLength || len(cmd.Topic) > maxFieldLength ||
			len(cmd.Payload) > maxFieldLength {
			return errors.New("command field is too long")
		}
		// Wildcards are subscribe-side syntax and are not publishable.
		if strings.ContainsAny(cmd.Topic, "+#") {
			return fmt.Errorf("command %q: topic must not contain wildcards", cmd.Name)
		}
		if cmd.Topic == "" && cfg.Topic == "" {
			return fmt.Errorf("command %q: no topic, and no default topic is set", cmd.Name)
		}
		if _, ok := seen[cmd.Name]; ok {
			return fmt.Errorf("duplicate command name %q", cmd.Name)
		}
		seen[cmd.Name] = struct{}{}
	}

	if strings.ContainsAny(cfg.Topic, "+#") {
		return errors.New("default topic must not contain wildcards")
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
		return Config{}, fmt.Errorf("parse mqtt config: %w", err)
	}

	return cfg, nil
}

func saveConfigToPath(path string, cfg Config) error {
	configMu.Lock()
	defer configMu.Unlock()

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create mqtt config directory: %w", err)
	}

	data, err := json.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("marshal mqtt config: %w", err)
	}

	// Written via a temporary file so a crash cannot leave a half-written
	// config, and mode 0600 because it holds the broker password.
	tmp, err := os.CreateTemp(dir, ".mqtt.json.*")
	if err != nil {
		return fmt.Errorf("create temporary mqtt config: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("set mqtt config permissions: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write mqtt config: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync mqtt config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close mqtt config: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace mqtt config: %w", err)
	}

	if directory, err := os.Open(dir); err == nil {
		_ = directory.Sync()
		_ = directory.Close()
	}

	return nil
}
