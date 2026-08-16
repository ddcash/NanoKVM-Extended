package tailscale

import (
	"net/http"
	"os/exec"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	"NanoKVM-Server/proto"
)

// The install always fetches tailscale_latest_riscv64.tgz, so updating is a
// reinstall. What was missing was any way to see which version is installed or
// whether a newer one exists, which is what makes updating a decision rather
// than a guess.

// The redirect from the "latest" URL lands on a versioned filename, e.g.
// tailscale_1.86.2_riscv64.tgz, which is where the available version comes from.
var releaseVersion = regexp.MustCompile(`tailscale_([0-9]+\.[0-9]+\.[0-9]+)_`)

func (s *Service) GetVersion(c *gin.Context) {
	var rsp proto.Response

	data := &proto.GetTailscaleVersionRsp{
		Installed: installedVersion(),
		Latest:    latestVersion(),
	}

	// Only claim an update exists when both versions are known and differ;
	// a failed lookup must not nag about an update that may not exist.
	data.UpdateAvailable = data.Installed != "" &&
		data.Latest != "" &&
		data.Installed != data.Latest

	rsp.OkRspWithData(c, data)
}

// Update reinstalls over the existing binaries. State lives in
// /var/lib/tailscale rather than beside them, so the node stays logged in.
func (s *Service) Update(c *gin.Context) {
	var rsp proto.Response

	cli := NewCli()

	// The daemon holds the binary open; replacing it underneath a running
	// process is what produces a half-updated install.
	if err := cli.Stop(); err != nil {
		log.Warnf("failed to stop tailscale before update: %s", err)
	}

	if err := install(); err != nil {
		log.Errorf("failed to update tailscale: %s", err)
		// Bring it back up on the old binaries rather than leaving it stopped.
		_ = cli.Start()
		rsp.ErrRsp(c, -1, "failed to update tailscale")
		return
	}

	if err := cli.Start(); err != nil {
		log.Errorf("failed to start tailscale after update: %s", err)
		rsp.ErrRsp(c, -2, "updated, but failed to restart tailscale")
		return
	}

	rsp.OkRspWithData(c, &proto.GetTailscaleVersionRsp{
		Installed: installedVersion(),
		Latest:    latestVersion(),
	})
}

func installedVersion() string {
	output, err := exec.Command(TailscalePath, "version").Output()
	if err != nil {
		return ""
	}

	// The first line is the version; the rest is build detail.
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 {
		return ""
	}

	return strings.TrimSpace(lines[0])
}

func latestVersion() string {
	// HEAD is enough: only the redirect target is needed, not the archive.
	resp, err := http.Head(OriginalURL)
	if err != nil {
		log.Debugf("failed to query latest tailscale version: %s", err)
		return ""
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	match := releaseVersion.FindStringSubmatch(resp.Request.URL.String())
	if len(match) < 2 {
		return ""
	}

	return match[1]
}
