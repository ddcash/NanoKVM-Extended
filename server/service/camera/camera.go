package camera

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	"NanoKVM-Server/proto"
	"NanoKVM-Server/service/stream/mjpeg"
)

// Token-gated read-only access to the video stream, so something that cannot
// hold a browser session - go2rtc, in particular - can pull it.
//
// This is off until a token is generated, and revoking the token turns it off
// again, which is the whole on/off control: no token, no access. Nothing here
// can send input to the target; it only reads frames.
const (
	ConfigFile = "/etc/kvm/camera.json"
	tokenBytes = 24
)

var (
	configMu   sync.Mutex
	configPath = ConfigFile
)

type Config struct {
	Enabled bool   `json:"enabled"`
	Token   string `json:"token"`
}

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) GetConfig(c *gin.Context) {
	var rsp proto.Response

	cfg, err := loadConfig()
	if err != nil {
		log.Errorf("failed to load camera config: %s", err)
		rsp.ErrRsp(c, -1, "failed to load camera config")
		return
	}

	rsp.OkRspWithData(c, &proto.GetCameraRsp{
		Enabled: cfg.Enabled && cfg.Token != "",
		Token:   cfg.Token,
	})
}

// SetConfig enables or disables access. Enabling mints a fresh token, so
// turning the feature off and on again invalidates any URL handed out before.
func (s *Service) SetConfig(c *gin.Context) {
	var req proto.SetCameraReq
	var rsp proto.Response

	if err := proto.ParseFormRequest(c, &req); err != nil || req.Enabled == nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}

	cfg := Config{}
	if *req.Enabled {
		token, err := generateToken()
		if err != nil {
			log.Errorf("failed to generate camera token: %s", err)
			rsp.ErrRsp(c, -2, "failed to generate token")
			return
		}
		cfg.Enabled = true
		cfg.Token = token
	}

	if err := saveConfig(cfg); err != nil {
		log.Errorf("failed to save camera config: %s", err)
		rsp.ErrRsp(c, -3, "failed to save camera config")
		return
	}

	applyFrameCache(cfg.Enabled)

	rsp.OkRspWithData(c, &proto.GetCameraRsp{
		Enabled: cfg.Enabled,
		Token:   cfg.Token,
	})
}

// ApplyStored restores the frame cache at startup so a snapshot works without
// someone having opened the web UI first.
func ApplyStored() {
	cfg, err := loadConfig()
	if err != nil {
		return
	}
	if cfg.Enabled && cfg.Token != "" {
		applyFrameCache(true)
		log.Info("camera access is enabled")
	}
}

// The snapshot endpoint reads the last frame the streamer saw, which is only
// retained while this cache is on.
func applyFrameCache(enabled bool) {
	if enabled {
		mjpeg.EnableLatestFrameCache()
		return
	}
	mjpeg.DisableLatestFrameCache()
}

// Stream serves the MJPEG stream to a caller holding the token. go2rtc ingests
// multipart/x-mixed-replace directly, so it can re-serve this as WebRTC without
// this device doing any extra encoding.
func (s *Service) Stream(c *gin.Context) {
	if !authorize(c) {
		return
	}

	mjpeg.Connect(c)
}

// Snapshot returns a single frame, for a still_image_url or a poll-based
// camera. Cheaper than holding the stream open.
func (s *Service) Snapshot(c *gin.Context) {
	if !authorize(c) {
		return
	}

	frame, ok := mjpeg.GetLatestFrame()
	if !ok || len(frame.Data) == 0 {
		c.Status(http.StatusServiceUnavailable)
		return
	}

	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, "image/jpeg", frame.Data)
}

// authorize checks the token from either the query string or a header. The
// query string matters because go2rtc and Home Assistant both take a plain URL.
func authorize(c *gin.Context) bool {
	cfg, err := loadConfig()
	if err != nil || !cfg.Enabled || cfg.Token == "" {
		c.Status(http.StatusNotFound)
		return false
	}

	presented := c.Query("token")
	if presented == "" {
		presented = c.GetHeader("X-Camera-Token")
	}

	if subtle.ConstantTimeCompare([]byte(presented), []byte(cfg.Token)) != 1 {
		c.Status(http.StatusUnauthorized)
		return false
	}

	return true
}

func generateToken() (string, error) {
	buf := make([]byte, tokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("read random: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

func loadConfig() (Config, error) {
	configMu.Lock()
	defer configMu.Unlock()

	content, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Config{}, nil
		}
		return Config{}, err
	}

	var cfg Config
	if err := json.Unmarshal(content, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse camera config: %w", err)
	}

	return cfg, nil
}

func saveConfig(cfg Config) error {
	configMu.Lock()
	defer configMu.Unlock()

	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create camera config directory: %w", err)
	}

	data, err := json.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("marshal camera config: %w", err)
	}

	// 0600: the token grants access to whatever is on the target's screen.
	tmp, err := os.CreateTemp(dir, ".camera.json.*")
	if err != nil {
		return fmt.Errorf("create temporary camera config: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("set camera config permissions: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write camera config: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync camera config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close camera config: %w", err)
	}
	if err := os.Rename(tmpPath, configPath); err != nil {
		return fmt.Errorf("replace camera config: %w", err)
	}

	if handle, err := os.Open(dir); err == nil {
		_ = handle.Sync()
		_ = handle.Close()
	}

	return nil
}
