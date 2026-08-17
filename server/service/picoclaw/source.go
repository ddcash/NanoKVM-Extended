package picoclaw

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	"NanoKVM-Server/proto"
)

// Where the PicoClaw archive is fetched from.
//
// The stock URLs point at cdn.sipeed.com, which resolves to a CDN that is not
// reachable from every network — connections time out at the TCP level rather
// than failing fast, so an install would sit there and then quietly give up.
// Being able to point this at a mirror turns that from fatal into fixable.
const SourceConfigFile = "/etc/kvm/picoclaw-source.json"

var sourceMu sync.Mutex

type sourceConfig struct {
	DownloadURL string `json:"downloadUrl"`
	ChecksumURL string `json:"checksumUrl"`
}

// downloadURL and checksumURL are read wherever the constants used to be, so
// an override applies without the install path knowing about any of this.
func downloadURL() string {
	if cfg := readSource(); cfg.DownloadURL != "" {
		return cfg.DownloadURL
	}
	return picoclawDownloadURL
}

func checksumURL() string {
	if cfg := readSource(); cfg.ChecksumURL != "" {
		return cfg.ChecksumURL
	}
	return picoclawChecksumURL
}

func (s *Service) GetSource(c *gin.Context) {
	var rsp proto.Response

	cfg := readSource()

	rsp.OkRspWithData(c, &proto.GetPicoclawSourceRsp{
		DownloadUrl: cfg.DownloadURL,
		ChecksumUrl: cfg.ChecksumURL,
		// So the UI can show what it falls back to without duplicating it.
		DefaultDownloadUrl: picoclawDownloadURL,
		DefaultChecksumUrl: picoclawChecksumURL,
	})
}

func (s *Service) SetSource(c *gin.Context) {
	var req proto.SetPicoclawSourceReq
	var rsp proto.Response

	if err := proto.ParseFormRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}

	cfg := sourceConfig{
		DownloadURL: strings.TrimSpace(req.DownloadUrl),
		ChecksumURL: strings.TrimSpace(req.ChecksumUrl),
	}

	// Both empty restores the defaults, which is how the override is cleared.
	for _, candidate := range []string{cfg.DownloadURL, cfg.ChecksumURL} {
		if candidate == "" {
			continue
		}
		if err := validateURL(candidate); err != nil {
			rsp.ErrRsp(c, -2, err.Error())
			return
		}
	}

	// One without the other would mix a mirror's archive with the stock
	// checksum, which fails verification in a way that looks like corruption.
	if (cfg.DownloadURL == "") != (cfg.ChecksumURL == "") {
		rsp.ErrRsp(c, -3, "set both URLs, or neither to use the defaults")
		return
	}

	if err := writeSource(cfg); err != nil {
		log.Errorf("failed to save picoclaw source: %s", err)
		rsp.ErrRsp(c, -4, "failed to save")
		return
	}

	rsp.OkRsp(c)
}

func validateURL(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("%q is not a valid URL", raw)
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return errors.New("URL must start with http:// or https://")
	}
	if parsed.Host == "" {
		return errors.New("URL is missing a host")
	}
	return nil
}

func readSource() sourceConfig {
	sourceMu.Lock()
	defer sourceMu.Unlock()

	var cfg sourceConfig

	data, err := os.ReadFile(SourceConfigFile)
	if err != nil {
		return cfg
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Errorf("failed to parse %s: %s", SourceConfigFile, err)
		return sourceConfig{}
	}

	return cfg
}

func writeSource(cfg sourceConfig) error {
	sourceMu.Lock()
	defer sourceMu.Unlock()

	dir := filepath.Dir(SourceConfigFile)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	data, err := json.Marshal(&cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(SourceConfigFile, data, 0o644)
}
