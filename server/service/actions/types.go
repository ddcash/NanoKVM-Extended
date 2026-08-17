package actions

// An action is a sequence of steps rather than a single operation, so one
// button or menu entry can do something useful: pulse a relay, wait, then run
// a command, or drive several pins in order.

type StepGPIO struct {
	Pin int `json:"pin"`
	// "high", "low", "pulse" or "toggle". Toggle reads the current value and
	// writes the opposite, which is what a light switch wants.
	Mode       string `json:"mode"`
	DurationMs int    `json:"durationMs"`
	// Relay boards usually activate when the line is pulled low.
	ActiveLow bool `json:"activeLow"`
}

// StepPWM drives one of the SoC's PWM channels, for brightness or fan speed.
// The chips are exposed as pwmchip0, 4, 8 and 12.
type StepPWM struct {
	Chip    int `json:"chip"`
	Channel int `json:"channel"`
	// Period in nanoseconds. 20000000 (20ms, 50Hz) suits servos; something
	// nearer 1000000 (1kHz) suits LEDs.
	PeriodNs    int  `json:"periodNs"`
	DutyPercent int  `json:"dutyPercent"`
	Enable      bool `json:"enable"`
}

type StepCommand struct {
	Command string `json:"command"`
	// Return immediately instead of waiting. For anything long-running that
	// should not hold up the rest of the sequence.
	Background bool `json:"background"`
	// Seconds. Ignored when Background is set.
	TimeoutSec int `json:"timeoutSec"`
}

type Step struct {
	// "gpio", "pwm", "command" or "delay".
	Type    string       `json:"type"`
	GPIO    *StepGPIO    `json:"gpio,omitempty"`
	PWM     *StepPWM     `json:"pwm,omitempty"`
	Command *StepCommand `json:"command,omitempty"`
	DelayMs int          `json:"delayMs,omitempty"`
}

type Action struct {
	Id         string `json:"id"`
	Name       string `json:"name"`
	Steps      []Step `json:"steps"`
	ShowInMenu bool   `json:"showInMenu"`

	// Superseded by Steps. Read when loading an older config so a device that
	// already has actions saved does not lose them, and never written back.
	LegacyType    string    `json:"type,omitempty"`
	LegacyGPIO    *StepGPIO `json:"gpio,omitempty"`
	LegacyCommand string    `json:"command,omitempty"`
}

// ButtonMap binds the NanoKVM's single button. The thresholds match the C++
// daemon's own, so "long" means the same thing whichever handler reacts.
type ButtonMap struct {
	ShortPress  string `json:"shortPress"`
	DoublePress string `json:"doublePress"`
	LongPress   string `json:"longPress"`
	// At and beyond nine seconds the daemon resets the password. An action
	// bound here runs alongside that; it does not replace it.
	VeryLongPress string `json:"veryLongPress"`
	KeepDefaults  bool   `json:"keepDefaults"`
}

type Config struct {
	Actions []Action  `json:"actions"`
	Buttons ButtonMap `json:"buttons"`
}

// migrate converts an action saved in the older single-operation shape into a
// one-step sequence.
func (a *Action) migrate() {
	if len(a.Steps) > 0 || a.LegacyType == "" {
		return
	}

	switch a.LegacyType {
	case "gpio":
		if a.LegacyGPIO != nil {
			a.Steps = []Step{{Type: "gpio", GPIO: a.LegacyGPIO}}
		}
	case "command":
		if a.LegacyCommand != "" {
			a.Steps = []Step{{
				Type:    "command",
				Command: &StepCommand{Command: a.LegacyCommand, TimeoutSec: int(commandTimeout.Seconds())},
			}}
		}
	}

	a.LegacyType = ""
	a.LegacyGPIO = nil
	a.LegacyCommand = ""
}
