package network

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	"NanoKVM-Server/proto"
)

// Time synchronisation and STUN, both of which reach out to third parties.
//
// ntpd ships in the base image pointed at pool.ntp.org, so a device talks to a
// public pool whether or not anyone wanted it to. The servers are now
// configurable and the whole thing can be switched off. STUN is likewise
// contacted on every WebRTC session; it defaults to off here and this is where
// a server is set for anyone who needs NAT traversal.
const (
	NTPConfigFile  = "/etc/kvm/ntp.json"
	StunConfigFile = "/etc/kvm/stun"

	ntpConfFile   = "/etc/ntp.conf"
	ntpInitScript = "/etc/init.d/S49ntp"

	maxTimeServers = 8
	maxHostLength  = 253
)

var timeSyncMu sync.Mutex

type ntpConfig struct {
	Enabled bool     `json:"enabled"`
	Servers []string `json:"servers"`
}

func (s *Service) GetTimeSync(c *gin.Context) {
	var rsp proto.Response

	cfg := readNTPConfig()

	rsp.OkRspWithData(c, &proto.GetTimeSyncRsp{
		NtpEnabled: cfg.Enabled,
		NtpServers: cfg.Servers,
		Stun:       readStunServer(),
	})
}

func (s *Service) SetTimeSync(c *gin.Context) {
	var req proto.SetTimeSyncReq
	var rsp proto.Response

	if err := proto.ParseFormRequest(c, &req); err != nil || req.NtpEnabled == nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}

	servers, err := normalizeHosts(req.NtpServers)
	if err != nil {
		rsp.ErrRsp(c, -2, err.Error())
		return
	}
	if *req.NtpEnabled && len(servers) == 0 {
		rsp.ErrRsp(c, -3, "at least one NTP server is required when time sync is enabled")
		return
	}

	stun := strings.TrimSpace(req.Stun)
	// "disable" is the sentinel the WebRTC code already understands, so an
	// empty field means the same thing rather than being a third state.
	if stun == "" {
		stun = "disable"
	}
	if stun != "disable" {
		if err := validateHostPort(stun); err != nil {
			rsp.ErrRsp(c, -4, err.Error())
			return
		}
	}

	if err := applyTimeSync(ntpConfig{Enabled: *req.NtpEnabled, Servers: servers}, stun); err != nil {
		log.Errorf("failed to apply time settings: %s", err)
		rsp.ErrRsp(c, -5, "failed to apply settings")
		return
	}

	rsp.OkRsp(c)
}

func applyTimeSync(cfg ntpConfig, stun string) error {
	timeSyncMu.Lock()
	defer timeSyncMu.Unlock()

	if err := writeJSONFile(NTPConfigFile, cfg, 0o644); err != nil {
		return err
	}
	if err := writeFileAtomic(StunConfigFile, []byte(stun+"\n"), 0o644); err != nil {
		return err
	}

	return applyNTP(cfg)
}

// applyNTP rewrites only the server lines of ntp.conf. The restrict lines the
// base image ships are access control and are left alone.
func applyNTP(cfg ntpConfig) error {
	if !cfg.Enabled {
		runInit("stop")
		return nil
	}

	existing, err := os.ReadFile(ntpConfFile)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("read %s: %w", ntpConfFile, err)
	}

	var builder strings.Builder
	for _, server := range cfg.Servers {
		builder.WriteString(fmt.Sprintf("server %s iburst\n", server))
	}

	for _, line := range strings.Split(string(existing), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "server ") || strings.HasPrefix(trimmed, "pool ") {
			continue
		}
		builder.WriteString(line)
		builder.WriteString("\n")
	}

	if err := writeFileAtomic(ntpConfFile, []byte(builder.String()), 0o644); err != nil {
		return err
	}

	runInit("restart")
	return nil
}

// ApplyStoredTimeSync reasserts the stored settings at startup, since ntp.conf
// lives on the root filesystem and an image update can replace it.
func ApplyStoredTimeSync() {
	if _, err := os.Stat(NTPConfigFile); err != nil {
		return
	}

	timeSyncMu.Lock()
	defer timeSyncMu.Unlock()

	if err := applyNTP(readNTPConfig()); err != nil {
		log.Errorf("failed to restore ntp settings: %s", err)
	}
}

// StunServer is read by the WebRTC code. The stored value wins over the YAML
// default so it can be changed without editing a config file by hand.
func StunServer(fallback string) string {
	if stored := readStunServer(); stored != "" {
		return stored
	}
	return fallback
}

func readStunServer() string {
	data, err := os.ReadFile(StunConfigFile)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func readNTPConfig() ntpConfig {
	// Absent means the base image's own configuration is in force, which is
	// enabled and pointed at the public pool.
	cfg := ntpConfig{Enabled: true}

	data, err := os.ReadFile(NTPConfigFile)
	if err != nil {
		cfg.Servers = readNTPServersFromConf()
		return cfg
	}

	if err := parseJSON(data, &cfg); err != nil {
		log.Errorf("failed to parse %s: %s", NTPConfigFile, err)
		cfg.Servers = readNTPServersFromConf()
	}
	if cfg.Servers == nil {
		cfg.Servers = []string{}
	}

	return cfg
}

func readNTPServersFromConf() []string {
	servers := make([]string, 0, 4)

	data, err := os.ReadFile(ntpConfFile)
	if err != nil {
		return servers
	}

	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || (fields[0] != "server" && fields[0] != "pool") {
			continue
		}
		servers = append(servers, fields[1])
	}

	return servers
}

func runInit(action string) {
	if _, err := os.Stat(ntpInitScript); err != nil {
		log.Debugf("%s is absent, skipping %s", ntpInitScript, action)
		return
	}
	if err := exec.Command("sh", "-c", fmt.Sprintf("%s %s", ntpInitScript, action)).Run(); err != nil {
		log.Errorf("failed to %s ntpd: %s", action, err)
	}
}

func normalizeHosts(hosts []string) ([]string, error) {
	normalized := make([]string, 0, len(hosts))
	seen := make(map[string]struct{}, len(hosts))

	for _, host := range hosts {
		host = strings.TrimSpace(host)
		if host == "" {
			continue
		}
		if len(host) > maxHostLength {
			return nil, fmt.Errorf("server name is too long")
		}
		// A space would let one entry become two directives in ntp.conf.
		if strings.ContainsAny(host, " \t\n\r#") {
			return nil, fmt.Errorf("invalid server %q", host)
		}
		if _, duplicate := seen[host]; duplicate {
			continue
		}
		seen[host] = struct{}{}
		normalized = append(normalized, host)
	}

	if len(normalized) > maxTimeServers {
		return nil, fmt.Errorf("at most %d servers are supported", maxTimeServers)
	}

	return normalized, nil
}

func validateHostPort(value string) error {
	host, port, err := net.SplitHostPort(value)
	if err != nil {
		return fmt.Errorf("STUN server must be host:port")
	}
	if host == "" || port == "" {
		return fmt.Errorf("STUN server must be host:port")
	}
	return nil
}

func parseJSON(data []byte, target any) error {
	return json.Unmarshal(data, target)
}

func writeJSONFile(path string, value any, mode os.FileMode) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal %s: %w", path, err)
	}
	return writeFileAtomic(path, data, mode)
}

func writeFileAtomic(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}

	tmp, err := os.CreateTemp(dir, ".tmp.*")
	if err != nil {
		return fmt.Errorf("create temporary file: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	if err := tmp.Chmod(mode); err != nil {
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

	return os.Rename(tmpPath, path)
}
