package actions

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// Running a step sequence. Steps run in order and stop at the first failure,
// so a sequence that pulses a relay and then reports success cannot claim to
// have worked when the relay never moved.
const (
	pwmBase = "/sys/class/pwm"

	maxStepsPerAction = 16
	maxDelayMs        = 60000
	maxCommandTimeout = 300
)

// Run performs every step of an action in order.
func Run(action *Action) error {
	if len(action.Steps) == 0 {
		return errors.New("action has no steps")
	}

	for i, step := range action.Steps {
		if err := runStep(step); err != nil {
			return fmt.Errorf("step %d (%s): %w", i+1, step.Type, err)
		}
	}

	return nil
}

func runStep(step Step) error {
	switch step.Type {
	case "gpio":
		if step.GPIO == nil {
			return errors.New("no gpio settings")
		}
		return driveGPIO(*step.GPIO)
	case "pwm":
		if step.PWM == nil {
			return errors.New("no pwm settings")
		}
		return drivePWM(*step.PWM)
	case "command":
		if step.Command == nil {
			return errors.New("no command settings")
		}
		return runCommand(*step.Command)
	case "delay":
		if step.DelayMs > 0 {
			time.Sleep(time.Duration(step.DelayMs) * time.Millisecond)
		}
		return nil
	default:
		return fmt.Errorf("unknown step type %q", step.Type)
	}
}

// ReadGPIO reports a pin's current value, so the UI can show whether a relay
// is on without the user having to look at the hardware.
func ReadGPIO(pin int) (int, error) {
	if pin < minGPIO || pin > maxGPIO {
		return 0, fmt.Errorf("gpio %d is outside the range this device provides", pin)
	}

	pinDir, err := exportGPIO(pin)
	if err != nil {
		return 0, err
	}

	data, err := os.ReadFile(filepath.Join(pinDir, "value"))
	if err != nil {
		return 0, err
	}

	value, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, err
	}

	return value, nil
}

// exportGPIO makes the pin available and returns its sysfs directory. The pin
// is left exported: unexporting resets the line, which would drop a relay that
// was just switched on.
func exportGPIO(pin int) (string, error) {
	pinDir := filepath.Join(gpioBase, fmt.Sprintf("gpio%d", pin))
	if _, err := os.Stat(pinDir); err == nil {
		return pinDir, nil
	}

	if err := os.WriteFile(gpioExportPath, []byte(strconv.Itoa(pin)), 0o200); err != nil {
		return "", fmt.Errorf("export gpio %d: %w", pin, err)
	}

	// udev creates the attributes asynchronously.
	for attempt := 0; attempt < 20; attempt++ {
		if _, err := os.Stat(filepath.Join(pinDir, "value")); err == nil {
			return pinDir, nil
		}
		time.Sleep(25 * time.Millisecond)
	}

	return pinDir, nil
}

func driveGPIO(spec StepGPIO) error {
	if spec.Pin < minGPIO || spec.Pin > maxGPIO {
		return fmt.Errorf("gpio %d is outside the range this device provides", spec.Pin)
	}

	pinDir, err := exportGPIO(spec.Pin)
	if err != nil {
		return err
	}

	valuePath := filepath.Join(pinDir, "value")

	// Toggle has to read before it can decide, and reading needs the direction
	// left alone until then.
	if spec.Mode == "toggle" {
		data, readErr := os.ReadFile(valuePath)
		current := "0"
		if readErr == nil {
			current = strings.TrimSpace(string(data))
		}

		if err := setDirectionOut(pinDir); err != nil {
			return err
		}

		next := "1"
		if current == "1" {
			next = "0"
		}
		return os.WriteFile(valuePath, []byte(next), 0o644)
	}

	if err := setDirectionOut(pinDir); err != nil {
		return err
	}

	active, idle := "1", "0"
	if spec.ActiveLow {
		active, idle = "0", "1"
	}

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

func setDirectionOut(pinDir string) error {
	// Already an output on a pin exported earlier; writing it again is
	// harmless and keeps this idempotent.
	if err := os.WriteFile(filepath.Join(pinDir, "direction"), []byte("out"), 0o644); err != nil {
		return fmt.Errorf("set direction: %w", err)
	}
	return nil
}

// drivePWM exports the channel if needed, then sets period, duty and enable in
// that order, which is what the sysfs interface requires.
func drivePWM(spec StepPWM) error {
	chipDir := filepath.Join(pwmBase, fmt.Sprintf("pwmchip%d", spec.Chip))
	if _, err := os.Stat(chipDir); err != nil {
		return fmt.Errorf("pwmchip%d is not available", spec.Chip)
	}

	channelDir := filepath.Join(chipDir, fmt.Sprintf("pwm%d", spec.Channel))
	if _, err := os.Stat(channelDir); err != nil {
		if err := os.WriteFile(filepath.Join(chipDir, "export"),
			[]byte(strconv.Itoa(spec.Channel)), 0o200); err != nil {
			return fmt.Errorf("export pwm channel: %w", err)
		}
		for attempt := 0; attempt < 20; attempt++ {
			if _, err := os.Stat(filepath.Join(channelDir, "period")); err == nil {
				break
			}
			time.Sleep(25 * time.Millisecond)
		}
	}

	period := spec.PeriodNs
	if period <= 0 {
		period = 1000000 // 1 kHz, a sensible default for an LED
	}

	duty := spec.DutyPercent
	if duty < 0 {
		duty = 0
	}
	if duty > 100 {
		duty = 100
	}

	// Duty must never exceed period, so lower it before changing period and
	// raise it after; writing them the other way round is rejected.
	if err := os.WriteFile(filepath.Join(channelDir, "duty_cycle"), []byte("0"), 0o644); err != nil {
		return fmt.Errorf("reset duty cycle: %w", err)
	}
	if err := os.WriteFile(filepath.Join(channelDir, "period"),
		[]byte(strconv.Itoa(period)), 0o644); err != nil {
		return fmt.Errorf("set period: %w", err)
	}

	dutyNs := period * duty / 100
	if err := os.WriteFile(filepath.Join(channelDir, "duty_cycle"),
		[]byte(strconv.Itoa(dutyNs)), 0o644); err != nil {
		return fmt.Errorf("set duty cycle: %w", err)
	}

	enable := "0"
	if spec.Enable {
		enable = "1"
	}
	return os.WriteFile(filepath.Join(channelDir, "enable"), []byte(enable), 0o644)
}

func runCommand(spec StepCommand) error {
	command := strings.TrimSpace(spec.Command)
	if command == "" {
		return errors.New("no command configured")
	}

	cmd := exec.Command("sh", "-c", command)

	if spec.Background {
		// Detached on purpose: the caller does not wait, and the process
		// outlives the request.
		if err := cmd.Start(); err != nil {
			return err
		}
		go func() {
			if err := cmd.Wait(); err != nil {
				log.Debugf("background command finished with: %s", err)
			}
		}()
		return nil
	}

	timeout := time.Duration(spec.TimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = commandTimeout
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		_ = cmd.Process.Kill()
		return fmt.Errorf("command timed out after %s", timeout)
	}
}
