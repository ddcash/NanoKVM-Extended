package proto

type SwitcherKey struct {
	Code  string `json:"code"`
	Label string `json:"label"`
}

// SwitcherStep is one press-and-release. Keys within a step are held together,
// so a step models both a single tap and a chord such as Ctrl+Alt+1.
type SwitcherStep struct {
	Keys []SwitcherKey `json:"keys"`
}

// SwitcherTarget is one labelled machine behind the KVM switch. Steps play in
// order, which is what switch hotkeys usually need: they are sequential taps
// (ScrollLock, ScrollLock, 2) rather than one chord.
type SwitcherTarget struct {
	Id    string         `json:"id"`
	Name  string         `json:"name"`
	Steps []SwitcherStep `json:"steps"`
}

type GetSwitcherRsp struct {
	Targets []SwitcherTarget `json:"targets"`
	// Milliseconds between steps. Switch firmware often drops taps that
	// arrive too quickly.
	StepDelayMs int `json:"stepDelayMs"`
}

type SetSwitcherReq struct {
	Targets     []SwitcherTarget `json:"targets"`
	StepDelayMs int              `json:"stepDelayMs"`
}
