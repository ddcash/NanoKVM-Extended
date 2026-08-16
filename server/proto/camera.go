package proto

type GetCameraRsp struct {
	Enabled bool `json:"enabled"`
	// Returned so the UI can show the URL to paste into go2rtc. Anyone who can
	// read this already holds a session that can view the screen anyway.
	Token string `json:"token"`
}

type SetCameraReq struct {
	// Enabling mints a fresh token, so toggling off and on invalidates any URL
	// handed out previously.
	Enabled *bool `json:"enabled" form:"enabled" validate:"required"`
}
