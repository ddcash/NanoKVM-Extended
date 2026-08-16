package vm

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	"NanoKVM-Server/proto"
)

// Which rows the OLED draws. Read by the kvm_system daemon, which re-reads the
// file when its timestamp changes, so a save takes effect without a restart.
//
// Plain key=value rather than JSON because the daemon parses it in C with no
// JSON library. It lives outside /kvmapp so the choice survives an application
// update.
const (
	OLEDDisplayFile = "/etc/kvm/oled.conf"
	oledTitleMax    = 20
)

var oledDisplayMu sync.Mutex

func (s *Service) GetOLEDDisplay(c *gin.Context) {
	var rsp proto.Response

	// A missing file means everything is shown, which is the stock behaviour.
	data := &proto.GetOLEDDisplayRsp{
		ShowIp:      true,
		ShowRes:     true,
		ShowType:    true,
		ShowStream:  true,
		ShowQuality: true,
	}

	file, err := os.Open(OLEDDisplayFile)
	if err != nil {
		rsp.OkRspWithData(c, data)
		return
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		// Anything but an explicit 0 counts as on, matching the daemon.
		on := value != "0"

		switch key {
		case "show_ip":
			data.ShowIp = on
		case "show_res":
			data.ShowRes = on
		case "show_type":
			data.ShowType = on
		case "show_stream":
			data.ShowStream = on
		case "show_quality":
			data.ShowQuality = on
		case "title":
			data.Title = value
		}
	}

	rsp.OkRspWithData(c, data)
}

func (s *Service) SetOLEDDisplay(c *gin.Context) {
	var req proto.SetOLEDDisplayReq
	var rsp proto.Response

	if err := proto.ParseFormRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}

	title := strings.TrimSpace(req.Title)
	if len(title) > oledTitleMax {
		rsp.ErrRsp(c, -2, fmt.Sprintf("title must be %d characters or fewer", oledTitleMax))
		return
	}
	// The parser splits on the first =, and a newline would forge a second
	// setting, so neither belongs in a value.
	if strings.ContainsAny(title, "=\n\r") {
		rsp.ErrRsp(c, -3, "title must not contain = or line breaks")
		return
	}

	if err := writeOLEDDisplay(&req, title); err != nil {
		log.Errorf("failed to write oled config: %s", err)
		rsp.ErrRsp(c, -4, "failed to save configuration")
		return
	}

	rsp.OkRsp(c)
}

func writeOLEDDisplay(req *proto.SetOLEDDisplayReq, title string) error {
	oledDisplayMu.Lock()
	defer oledDisplayMu.Unlock()

	dir := filepath.Dir(OLEDDisplayFile)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	var builder strings.Builder
	builder.WriteString("# Written by NanoKVM. Rows shown on the OLED display.\n")
	for _, setting := range []struct {
		key string
		on  bool
	}{
		{"show_ip", req.ShowIp},
		{"show_res", req.ShowRes},
		{"show_type", req.ShowType},
		{"show_stream", req.ShowStream},
		{"show_quality", req.ShowQuality},
	} {
		value := "0"
		if setting.on {
			value = "1"
		}
		builder.WriteString(fmt.Sprintf("%s=%s\n", setting.key, value))
	}
	builder.WriteString(fmt.Sprintf("title=%s\n", title))

	// Replaced atomically: the daemon watches the timestamp and could otherwise
	// read a half-written file.
	tmp, err := os.CreateTemp(dir, ".oled.conf.*")
	if err != nil {
		return fmt.Errorf("create temporary file: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	if err := tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("set permissions: %w", err)
	}
	if _, err := tmp.WriteString(builder.String()); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close: %w", err)
	}
	if err := os.Rename(tmpPath, OLEDDisplayFile); err != nil {
		return fmt.Errorf("replace: %w", err)
	}

	if handle, err := os.Open(dir); err == nil {
		_ = handle.Sync()
		_ = handle.Close()
	}

	return nil
}
