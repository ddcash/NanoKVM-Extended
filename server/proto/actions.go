package proto

type ActionGPIO struct {
	Pin int `json:"pin"`
	// "high", "low" or "pulse".
	Mode       string `json:"mode"`
	DurationMs int    `json:"durationMs"`
	// For relay boards, which usually activate when pulled low.
	ActiveLow bool `json:"activeLow"`
	// Non-empty when the firmware already drives this pin. Advisory only.
	ReservedFor string `json:"reservedFor,omitempty"`
}

type ActionInfo struct {
	Id   string `json:"id"`
	Name string `json:"name"`
	// "gpio" or "command".
	Type       string      `json:"type"`
	GPIO       *ActionGPIO `json:"gpio,omitempty"`
	Command    string      `json:"command,omitempty"`
	ShowInMenu bool        `json:"showInMenu"`
}

type GetActionsRsp struct {
	Actions    []ActionInfo `json:"actions"`
	ShortPress string       `json:"shortPress"`
	LongPress  string       `json:"longPress"`
	// The daemon's own button handling continues alongside a custom action.
	KeepDefaults bool `json:"keepDefaults"`
	// The usable gpio range on this hardware, so the UI can bound its input.
	MinGPIO int `json:"minGpio"`
	MaxGPIO int `json:"maxGpio"`
}

type SetActionsReq struct {
	Actions      []ActionInfo `json:"actions"`
	ShortPress   string       `json:"shortPress"`
	LongPress    string       `json:"longPress"`
	KeepDefaults bool         `json:"keepDefaults"`
}

type RunActionReq struct {
	Id string `json:"id" form:"id" validate:"required"`
}
