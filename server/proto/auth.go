package proto

type LoginReq struct {
	Username string `validate:"required"`
	Password string `validate:"required"`
	// Only consulted when TOTP is enabled. Accepts a 6-digit code or one of
	// the single-use backup codes.
	Code string `json:"code" form:"code"`
}

type GetTotpRsp struct {
	Enabled bool `json:"enabled"`
	// True once a secret has been generated but not yet confirmed by a code.
	Pending bool `json:"pending"`
}

type SetupTotpRsp struct {
	// otpauth:// URI for an authenticator app to scan.
	Uri    string `json:"uri"`
	Secret string `json:"secret"`
}

type EnableTotpReq struct {
	Code string `json:"code" form:"code" validate:"required"`
}

type EnableTotpRsp struct {
	// Shown once, at enrolment. Only hashes are stored.
	BackupCodes []string `json:"backupCodes"`
}

type DisableTotpReq struct {
	Password string `json:"password" form:"password" validate:"required"`
	Code     string `json:"code" form:"code" validate:"required"`
}

type LoginRsp struct {
	Token string `json:"token"`
}

type GetAccountRsp struct {
	Username string `json:"username"`
}

type ChangePasswordReq struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type IsPasswordUpdatedRsp struct {
	IsUpdated bool `json:"isUpdated"`
}
