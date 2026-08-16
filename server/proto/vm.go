package proto

type IP struct {
	Name    string `json:"name"`
	Addr    string `json:"addr"`
	Version string `json:"version"`
	Type    string `json:"type"`
}

type GetInfoRsp struct {
	IPs         []IP   `json:"ips"`
	Mdns        string `json:"mdns"`
	Image       string `json:"image"`
	Application string `json:"application"`
	DeviceKey   string `json:"deviceKey"`
}

type GetHardwareRsp struct {
	Version string `json:"version"`
}

type SetGpioReq struct {
	Type     string `validate:"required"`  // reset / power
	Duration uint   `validate:"omitempty"` // press time (unit: milliseconds)
}

type GetGpioRsp struct {
	PWR bool `json:"pwr"` // power led
	HDD bool `json:"hdd"` // hdd led
}

type SetScreenReq struct {
	Type  string `validate:"required"` // resolution / fps / quality
	Value int    `validate:"number"`   // value
}

type GetScriptsRsp struct {
	Files []string `json:"files"`
}

type UploadScriptRsp struct {
	File string `json:"file"`
}

type RunScriptReq struct {
	Name string `validate:"required"`
	Type string `validate:"required"` // foreground | background
}

type RunScriptRsp struct {
	Log string `json:"log"`
}

type DeleteScriptReq struct {
	Name string `validate:"required"`
}

// autostart
type GetAutostartRsp struct {
	Files []string `json:"files"`
}

type UploadAutostartReq struct {
	Content string `json:"content"`
}

type GetVirtualDeviceRsp struct {
	Network bool `json:"network"`
	Media   bool `json:"media"`
	Disk    bool `json:"disk"`
}

type UpdateVirtualDeviceReq struct {
	Device string `validate:"required"`
}

type UpdateVirtualDeviceRsp struct {
	On bool `json:"on"`
}

type SetMemoryLimitReq struct {
	Enabled bool  `validate:"omitempty"`
	Limit   int64 `validate:"omitempty"`
}

type GetMemoryLimitRsp struct {
	Enabled bool  `json:"enabled"`
	Limit   int64 `json:"limit"`
}

type SetOledReq struct {
	Sleep int `validate:"omitempty"`
}

type GetOLEDRsp struct {
	Exist bool `json:"exist"`
	Sleep int  `json:"sleep"`
}

type GetGetHdmiStateRsp struct {
	Enabled     bool `json:"enabled"`
	Signal      bool `json:"signal"`
	IdleTimeout int  `json:"idleTimeout"`
}

type SetHdmiIdleTimeoutReq struct {
	Minutes int `validate:"gte=0,lte=10080"`
}

type GetSSHStateRsp struct {
	Enabled bool `json:"enabled"`
}

type GetSwapRsp struct {
	Size int64 `json:"size"` // unit: MB
}

type SetSwapReq struct {
	Size int64 `validate:"omitempty"` // unit: MB
}

type GetMouseJigglerRsp struct {
	Enabled bool   `json:"enabled"`
	Mode    string `json:"mode"`
}

type SetMouseJigglerReq struct {
	Enabled bool   `validate:"omitempty"`
	Mode    string `validate:"omitempty"`
}

type GetMdnsStateRsp struct {
	Enabled bool `json:"enabled"`
}

type SetHostnameReq struct {
	Hostname string `validate:"required"`
}

type GetHostnameRsp struct {
	Hostname string `json:"hostname"`
}

type SetWebTitleReq struct {
	Title string `validate:"omitempty"`
}

type GetWebTitleRsp struct {
	Title string `json:"title"`
}

type SetTlsReq struct {
	Enabled bool `validate:"omitempty"`
}

type GetResourcesRsp struct {
	CpuPercent float64 `json:"cpuPercent"`
	// MemAvailable rather than MemFree: the kernel keeps most of the remainder
	// as reclaimable cache, so MemFree looks alarming while nothing is wrong.
	MemoryTotal     uint64  `json:"memoryTotal"`
	MemoryAvailable uint64  `json:"memoryAvailable"`
	MemoryPercent   float64 `json:"memoryPercent"`
	DiskTotal       uint64  `json:"diskTotal"`
	DiskFree        uint64  `json:"diskFree"`
	DiskPercent     float64 `json:"diskPercent"`
	Load1           float64 `json:"load1"`
	Load5           float64 `json:"load5"`
	Load15          float64 `json:"load15"`
	// Degrees Celsius, or 0 when the SoC exposes no thermal zone.
	Temperature   float64 `json:"temperature"`
	UptimeSeconds int64   `json:"uptimeSeconds"`
}
