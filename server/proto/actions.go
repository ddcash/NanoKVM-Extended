package proto

type ActionStepGPIO struct {
	Pin int `json:"pin"`
	// "high", "low", "pulse" or "toggle".
	Mode       string `json:"mode"`
	DurationMs int    `json:"durationMs"`
	ActiveLow  bool   `json:"activeLow"`
	// Non-empty when the firmware already drives this pin. Advisory only.
	ReservedFor string `json:"reservedFor,omitempty"`
}

type ActionStepPWM struct {
	Chip        int  `json:"chip"`
	Channel     int  `json:"channel"`
	PeriodNs    int  `json:"periodNs"`
	DutyPercent int  `json:"dutyPercent"`
	Enable      bool `json:"enable"`
}

type ActionStepCommand struct {
	Command string `json:"command"`
	// Return immediately rather than waiting for the command to finish.
	Background bool `json:"background"`
	TimeoutSec int  `json:"timeoutSec"`
}

type ActionStep struct {
	// "gpio", "pwm", "command" or "delay".
	Type    string             `json:"type"`
	GPIO    *ActionStepGPIO    `json:"gpio,omitempty"`
	PWM     *ActionStepPWM     `json:"pwm,omitempty"`
	Command *ActionStepCommand `json:"command,omitempty"`
	DelayMs int                `json:"delayMs,omitempty"`
}

type ActionInfo struct {
	Id         string       `json:"id"`
	Name       string       `json:"name"`
	Steps      []ActionStep `json:"steps"`
	ShowInMenu bool         `json:"showInMenu"`
}

type GetActionsRsp struct {
	Actions []ActionInfo `json:"actions"`

	ShortPress    string `json:"shortPress"`
	DoublePress   string `json:"doublePress"`
	LongPress     string `json:"longPress"`
	VeryLongPress string `json:"veryLongPress"`
	// The daemon's own button handling continues alongside a custom action.
	KeepDefaults bool `json:"keepDefaults"`

	// Bounds and thresholds, so the UI does not restate what the server knows.
	MinGPIO         int `json:"minGpio"`
	MaxGPIO         int `json:"maxGpio"`
	LongPressMs     int `json:"longPressMs"`
	VeryLongPressMs int `json:"veryLongPressMs"`
}

type SetActionsReq struct {
	Actions []ActionInfo `json:"actions"`

	ShortPress    string `json:"shortPress"`
	DoublePress   string `json:"doublePress"`
	LongPress     string `json:"longPress"`
	VeryLongPress string `json:"veryLongPress"`
	KeepDefaults  bool   `json:"keepDefaults"`
}

type RunActionReq struct {
	Id string `json:"id" form:"id" validate:"required"`
}

type GPIOState struct {
	Pin   int `json:"pin"`
	Value int `json:"value"`
	// False when the pin has never been driven, so the UI shows "unknown"
	// rather than claiming it is off.
	Known bool `json:"known"`
}

type GetGPIOStateRsp struct {
	States []GPIOState `json:"states"`
}
