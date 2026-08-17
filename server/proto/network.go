package proto

type WakeOnLANReq struct {
	Mac string `form:"mac" validate:"required"`
}

type GetMacRsp struct {
	Macs []string `json:"macs"`
}

type DeleteMacReq struct {
	Mac string `form:"mac" validate:"required"`
}

type SetMacNameReq struct {
	Mac  string `form:"mac" validate:"required"`
	Name string `form:"name" validate:"required"`
}

// Adding a device only needs the address. Naming is a separate call, so
// reusing SetMacNameReq here would reject every add for want of a name.
type AddMacReq struct {
	Mac string `json:"mac" form:"mac" validate:"required"`
}

type GetWifiRsp struct {
	Supported bool   `json:"supported"`
	ApMode    bool   `json:"apMode"`
	Connected bool   `json:"connected"`
	Ssid      string `json:"ssid"`
}

type ConnectWifiReq struct {
	Ssid     string `validate:"required"`
	Password string `validate:"required"`
}

type GetDNSRsp struct {
	Mode      string   `json:"mode"`
	Servers   []string `json:"servers"`
	Effective []string `json:"effective"`
	DHCP      []string `json:"dhcp"`
	Info      DNSInfo  `json:"info"`
}

type SetDNSReq struct {
	Mode    string   `json:"mode" validate:"required,oneof=manual dhcp"`
	Servers []string `json:"servers"`
}

type DNSInfo struct {
	Interface     string   `json:"interface"`
	Type          string   `json:"type"`
	Address       string   `json:"address"`
	SubnetMask    string   `json:"subnetMask"`
	Gateway       string   `json:"gateway"`
	SearchDomains []string `json:"searchDomains"`
}

type GetTimeSyncRsp struct {
	NtpEnabled bool     `json:"ntpEnabled"`
	NtpServers []string `json:"ntpServers"`
	// "disable" means WebRTC uses host candidates only.
	Stun string `json:"stun"`
}

type SetTimeSyncReq struct {
	NtpEnabled *bool    `json:"ntpEnabled" form:"ntpEnabled"`
	NtpServers []string `json:"ntpServers" form:"ntpServers"`
	Stun       string   `json:"stun" form:"stun"`
}
