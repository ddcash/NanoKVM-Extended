package common

/*
	#cgo CFLAGS: -I../include
	// libkvm.so leaves its OpenCV and mmf symbols to the sibling libraries
	// shipped beside it in dl_lib. Those are found at runtime through the
	// binary's RPATH but are not on the link path, so the linker rejects the
	// real library unless undefined symbols in a shared object are permitted.
	// Without this only a stripped link stub can be used, which caps what the
	// C API is able to expose.
	#cgo LDFLAGS: -L../dl_lib -lkvm -Wl,--allow-shlib-undefined
	#include "kvm_vision.h"
*/
import "C"
import (
	"sync"
	"unsafe"

	log "github.com/sirupsen/logrus"
)

var (
	kvmVision     *KvmVision
	kvmVisionOnce sync.Once
)

type KvmVision struct {
	mutex  sync.RWMutex
	closed bool
}

func GetKvmVision() *KvmVision {
	kvmVisionOnce.Do(func() {
		kvmVision = &KvmVision{}

		logLevel := C.uint8_t(0)
		C.kvmv_init(logLevel)
		log.Debugf("kvm vision initialized")
	})

	return kvmVision
}

func (k *KvmVision) ReadMjpeg(width uint16, height uint16, quality uint16) (data []byte, result int) {
	k.mutex.RLock()
	defer k.mutex.RUnlock()
	if k.closed {
		return nil, -1
	}

	var (
		kvmData  *C.uint8_t
		dataSize C.uint32_t
	)

	result = int(C.kvmv_read_img(
		C.uint16_t(width),
		C.uint16_t(height),
		C.uint8_t(0),
		C.uint16_t(quality),
		&kvmData,
		&dataSize,
	))
	if result < 0 {
		log.Errorf("failed to read kvm image: %v", result)
		return
	}
	defer C.free_kvmv_data(&kvmData)

	data = C.GoBytes(unsafe.Pointer(kvmData), C.int(dataSize))
	return
}

func (k *KvmVision) ReadH264(width uint16, height uint16, bitRate uint16) (data []byte, result int) {
	k.mutex.RLock()
	defer k.mutex.RUnlock()
	if k.closed {
		return nil, -1
	}

	var (
		kvmData  *C.uint8_t
		dataSize C.uint32_t
	)

	result = int(C.kvmv_read_img(
		C.uint16_t(width),
		C.uint16_t(height),
		C.uint8_t(1),
		C.uint16_t(bitRate),
		&kvmData,
		&dataSize,
	))
	if result < 0 {
		log.Errorf("failed to read kvm image: %v", result)
		return
	}
	defer C.free_kvmv_data(&kvmData)

	data = C.GoBytes(unsafe.Pointer(kvmData), C.int(dataSize))
	return
}

func (k *KvmVision) SetHDMI(enable bool) int {
	k.mutex.RLock()
	defer k.mutex.RUnlock()
	if k.closed {
		return -1
	}

	hdmiEnable := C.uint8_t(0)
	if enable {
		hdmiEnable = C.uint8_t(1)
	}

	result := int(C.kvmv_hdmi_control(hdmiEnable))
	if result < 0 {
		log.Errorf("failed to set hdmi to %t", enable)
		return result
	}

	return result
}

func (k *KvmVision) HasHDMISignal() bool {
	k.mutex.RLock()
	defer k.mutex.RUnlock()
	if k.closed {
		return false
	}

	return C.kvmv_hdmi_signal_active() != 0
}

func (k *KvmVision) SetGop(gop uint8) {
	k.mutex.RLock()
	defer k.mutex.RUnlock()
	if k.closed {
		return
	}

	_gop := C.uint8_t(gop)
	C.set_h264_gop(_gop)
}

func (k *KvmVision) SetFrameDetect(frame uint8) {
	k.mutex.RLock()
	defer k.mutex.RUnlock()
	if k.closed {
		return
	}

	_frame := C.uint8_t(frame)
	C.set_frame_detact(_frame)
}

// Video codec, in the encoder's own numbering.
const (
	CodecH265 uint8 = 1
	CodecH264 uint8 = 2
)

// SetCodec selects the video codec. It takes effect on the next frame, which
// rebuilds the encoder channel.
func (k *KvmVision) SetCodec(codec uint8) {
	k.mutex.RLock()
	defer k.mutex.RUnlock()
	if k.closed {
		return
	}

	C.kvmv_set_codec(C.uint8_t(codec))
}

// GetCodec reports the codec the encoder is currently configured for.
func (k *KvmVision) GetCodec() uint8 {
	k.mutex.RLock()
	defer k.mutex.RUnlock()
	if k.closed {
		return CodecH264
	}

	return uint8(C.kvmv_get_codec())
}

func (k *KvmVision) Close() {
	k.mutex.Lock()
	defer k.mutex.Unlock()
	if k.closed {
		return
	}

	k.closed = true
	C.kvmv_deinit()
	log.Debugf("stop kvm vision...")
}
