package switcher

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	"NanoKVM-Server/proto"
	"NanoKVM-Server/service/hid"
)

// Playback is serialised: two overlapping hotkey sequences would interleave
// their reports and send neither.
var playMu sync.Mutex

var ErrNoTarget = errors.New("target not found")

// Press replays a target's hotkey from the server.
//
// The browser plays targets itself over the existing WebSocket, which keeps the
// UI responsive. This path exists so the same targets can be triggered without
// a browser at all, which is what Home Assistant and any other automation
// needs.
func Press(id string) error {
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("load switcher config: %w", err)
	}

	var target *proto.SwitcherTarget
	for i := range cfg.Targets {
		if cfg.Targets[i].Id == id || cfg.Targets[i].Name == id {
			target = &cfg.Targets[i]
			break
		}
	}
	if target == nil {
		return ErrNoTarget
	}

	return play(target, cfg.StepDelayMs)
}

func play(target *proto.SwitcherTarget, stepDelayMs int) error {
	playMu.Lock()
	defer playMu.Unlock()

	h := hid.GetHid()

	for i, step := range target.Steps {
		report := make([]byte, 8)
		slot := 2

		for _, key := range step.Keys {
			report[0] |= key.Modifier
			// A modifier-only key occupies no keycode slot. The report carries
			// at most six keycodes; validation caps a step well below that.
			if key.Keycode != 0 && slot < 8 {
				report[slot] = key.Keycode
				slot++
			}
		}

		if err := h.WriteKeyboardReport(report); err != nil {
			return fmt.Errorf("write key press: %w", err)
		}

		// Release everything before the next step, so a repeated key registers
		// as a second press rather than being held down.
		if err := h.WriteKeyboardReport(make([]byte, 8)); err != nil {
			return fmt.Errorf("write key release: %w", err)
		}

		if i < len(target.Steps)-1 && stepDelayMs > 0 {
			time.Sleep(time.Duration(stepDelayMs) * time.Millisecond)
		}
	}

	log.Debugf("switcher: replayed target %s", target.Name)
	return nil
}

func (s *Service) PressTarget(c *gin.Context) {
	var req proto.PressSwitcherReq
	var rsp proto.Response

	if err := proto.ParseFormRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}

	if err := Press(req.Id); err != nil {
		if errors.Is(err, ErrNoTarget) {
			rsp.ErrRsp(c, -2, "target not found")
			return
		}
		log.Errorf("failed to press switcher target: %s", err)
		rsp.ErrRsp(c, -3, "failed to send hotkey")
		return
	}

	rsp.OkRsp(c)
}

// Targets exposes the configured targets to other services, so the Home
// Assistant bridge can mirror them as buttons without reaching into the
// config file itself.
func Targets() ([]proto.SwitcherTarget, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}
	return cfg.Targets, nil
}
