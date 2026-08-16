package stream

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	"NanoKVM-Server/common"
	"NanoKVM-Server/proto"
)

// The selected codec is persisted so it survives a reboot; the encoder itself
// only holds it in memory.
const CodecFile = "/etc/kvm/codec"

const (
	codecH264 = "h264"
	codecH265 = "h265"
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

// ApplyStoredCodec restores the saved codec at startup. H.264 is used when
// nothing is stored, which is also what the encoder defaults to.
func ApplyStoredCodec() {
	data, err := os.ReadFile(CodecFile)
	if err != nil {
		return
	}

	if strings.TrimSpace(string(data)) == codecH265 {
		common.GetKvmVision().SetCodec(common.CodecH265)
		log.Info("video codec: h265")
	}
}

func (s *Service) GetCodec(c *gin.Context) {
	var rsp proto.Response

	codec := codecH264
	if common.GetKvmVision().GetCodec() == common.CodecH265 {
		codec = codecH265
	}

	rsp.OkRspWithData(c, &proto.GetCodecRsp{Codec: codec})
}

func (s *Service) SetCodec(c *gin.Context) {
	var req proto.SetCodecReq
	var rsp proto.Response

	if err := proto.ParseFormRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}

	var value uint8
	switch req.Codec {
	case codecH264:
		value = common.CodecH264
	case codecH265:
		value = common.CodecH265
	default:
		rsp.ErrRsp(c, -2, "unsupported codec")
		return
	}

	common.GetKvmVision().SetCodec(value)

	if err := persistCodec(req.Codec); err != nil {
		log.Errorf("failed to persist codec: %s", err)
	}

	rsp.OkRsp(c)
}

func persistCodec(codec string) error {
	if err := os.MkdirAll(filepath.Dir(CodecFile), 0o755); err != nil {
		return err
	}
	return os.WriteFile(CodecFile, []byte(codec+"\n"), 0o644)
}
