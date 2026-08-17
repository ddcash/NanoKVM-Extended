package actions

import (
	"encoding/binary"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
)

// The NanoKVM's own button, read from the same evdev node the C++ daemon uses.
//
// evdev delivers each event to every reader, so watching it here does not take
// the button away from the daemon: its OLED paging, Wi-Fi setup and password
// reset keep working, and a custom action runs alongside. Suppressing the
// built-in behaviour would mean changing the daemon, and losing the hold-to-
// reset-password recovery path is not a good trade for a remapped button.
const (
	inputDevice = "/dev/input/event0"

	// These match the C++ daemon's own thresholds (KEY_LONG_PRESS and
	// KEY_LONGLONG_PRESS), so a press means the same thing to a user whichever
	// handler reacts to it.
	longPressMs     = 1500
	veryLongPressMs = 9000

	// Two presses closer together than this count as a double press. Long
	// enough to be comfortable, short enough not to delay every single press
	// noticeably.
	doublePressWindowMs = 450

	// Below this a press is noise rather than intent.
	minPressMs = 50

	evKey = 0x01
)

// inputEvent mirrors the kernel's struct input_event. The timeval is two longs,
// which are 64-bit on this platform, so the header is 16 bytes.
type inputEvent struct {
	Seconds      int64
	Microseconds int64
	Type         uint16
	Code         uint16
	Value        int32
}

const inputEventSize = 24

// WatchButton follows the button and runs whatever is bound to it. It returns
// only if the device cannot be opened, which is the case on hardware without
// gpio-keys; the feature is simply unavailable there.
func WatchButton() {
	file, err := os.Open(inputDevice)
	if err != nil {
		log.Debugf("button watcher disabled, cannot open %s: %s", inputDevice, err)
		return
	}
	defer func() { _ = file.Close() }()

	log.Debugf("watching %s for button presses", inputDevice)

	buf := make([]byte, inputEventSize)
	var pressedAt time.Time
	// A short press is held back briefly in case a second one follows, which is
	// the only way to tell a single press from the first half of a double.
	var pendingShort *time.Timer

	for {
		if _, err := file.Read(buf); err != nil {
			log.Errorf("button watcher stopped: %s", err)
			return
		}

		event := inputEvent{
			Type:  binary.LittleEndian.Uint16(buf[16:18]),
			Code:  binary.LittleEndian.Uint16(buf[18:20]),
			Value: int32(binary.LittleEndian.Uint32(buf[20:24])),
		}

		if event.Type != evKey {
			continue
		}

		switch event.Value {
		case 1:
			pressedAt = time.Now()
		case 0:
			if pressedAt.IsZero() {
				continue
			}
			held := time.Since(pressedAt)
			pressedAt = time.Time{}

			if held < minPressMs*time.Millisecond {
				continue
			}

			// Anything longer than a short press cannot be part of a double,
			// so it fires immediately and cancels anything waiting.
			if held >= longPressMs*time.Millisecond {
				if pendingShort != nil {
					pendingShort.Stop()
					pendingShort = nil
				}
				go handleHeldPress(held)
				continue
			}

			if pendingShort != nil {
				// The second of two short presses.
				pendingShort.Stop()
				pendingShort = nil
				go handleBinding("double")
				continue
			}

			pendingShort = time.AfterFunc(doublePressWindowMs*time.Millisecond, func() {
				pendingShort = nil
				handleBinding("short")
			})
		}
	}
}

func handleHeldPress(held time.Duration) {
	kind := "long"
	if held >= veryLongPressMs*time.Millisecond {
		kind = "veryLong"
	}
	handleBinding(kind)
}

func handleBinding(kind string) {
	cfg, err := loadConfig()
	if err != nil {
		return
	}

	var binding string
	switch kind {
	case "short":
		binding = cfg.Buttons.ShortPress
	case "double":
		binding = cfg.Buttons.DoublePress
	case "long":
		binding = cfg.Buttons.LongPress
	case "veryLong":
		binding = cfg.Buttons.VeryLongPress
	}

	if binding == "" {
		return
	}

	if err := RunByID(binding); err != nil {
		log.Errorf("button %s press action failed: %s", kind, err)
		return
	}

	log.Infof("button %s press ran its action", kind)
}
