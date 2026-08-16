package proto

type UpdateFrameDetectReq struct {
	Enabled bool `validate:"omitempty"`
}

type StopFrameDetectReq struct {
	Duration int `validate:"omitempty"`
}

type GetCodecRsp struct {
	Codec string `json:"codec"`
}

type SetCodecReq struct {
	// "h264" or "h265". H.265 needs Chrome 136+ or Safari 18+ to decode.
	Codec string `json:"codec" form:"codec" validate:"required"`
}
