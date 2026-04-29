package stealth

import (
	"crypto/rand"
	"io"
	"sync"
)

type VisionServerConfig struct {
	Enabled           bool
	FlowControl       string
	PaddingSize       int
	MinPaddingSize    int
	MaxPaddingSize    int
	EnableSplice      bool
	EnableDirectCopy  bool
	BufferSize        int
}

func DefaultVisionConfig() *VisionServerConfig {
	return &VisionServerConfig{
		Enabled:          true,
		FlowControl:      "xtls-rprx-vision",
		PaddingSize:      0,
		MinPaddingSize:   0,
		MaxPaddingSize:   64,
		EnableSplice:     false,
		EnableDirectCopy: true,
		BufferSize:       64 * 1024,
	}
}

type VisionConn struct {
	conn      io.ReadWriter
	config    *VisionServerConfig
	readBuf   []byte
	writeBuf  []byte
	mu        sync.Mutex
}

func NewVisionConn(conn io.ReadWriter, config *VisionServerConfig) *VisionConn {
	if config == nil {
		config = DefaultVisionConfig()
	}
	return &VisionConn{
		conn:     conn,
		config:   config,
		readBuf:  make([]byte, config.BufferSize),
		writeBuf: make([]byte, config.BufferSize),
	}
}

func (vc *VisionConn) Read(p []byte) (int, error) {
	n, err := vc.conn.Read(p)
	if err != nil {
		return n, err
	}

	if vc.config.Enabled && vc.config.PaddingSize > 0 {
		return n, nil
	}

	return n, nil
}

func (vc *VisionConn) Write(p []byte) (int, error) {
	if !vc.config.Enabled {
		return vc.conn.Write(p)
	}

	vc.mu.Lock()
	defer vc.mu.Unlock()

	padSize := 0
	if vc.config.MaxPaddingSize > vc.config.MinPaddingSize {
		padSize = vc.config.MinPaddingSize + int(randByte())%(vc.config.MaxPaddingSize-vc.config.MinPaddingSize)
	}

	if padSize > 0 {
		padded := make([]byte, len(p)+padSize)
		copy(padded, p)
		io.ReadFull(rand.Reader, padded[len(p):])

		_, err := vc.conn.Write(padded)
		if err != nil {
			return 0, err
		}

		return len(p), nil
	}

	return vc.conn.Write(p)
}

func randByte() byte {
	b := make([]byte, 1)
	rand.Read(b)
	return b[0]
}

type VisionFlow struct {
	config    *VisionServerConfig
	flowCtrl  *VisionFlowController
	validated bool
	mu        sync.Mutex
}

func NewVisionFlow(config *VisionServerConfig) *VisionFlow {
	return &VisionFlow{
		config:   config,
		flowCtrl: NewVisionFlowController(),
	}
}

func (vf *VisionFlow) OnClientHello(data []byte) []byte {
	if !vf.config.Enabled {
		return data
	}

	vf.mu.Lock()
	defer vf.mu.Unlock()

	vf.flowCtrl.PadTraffic(data, len(data)+vf.config.MinPaddingSize)

	vf.validated = true
	return data
}

func (vf *VisionFlow) OnServerHello(data []byte) []byte {
	if !vf.config.Enabled || !vf.validated {
		return data
	}

	return vf.flowCtrl.PadTraffic(data, len(data))
}

func (vf *VisionFlow) IsValidated() bool {
	vf.mu.Lock()
	defer vf.mu.Unlock()
	return vf.validated
}

func (vf *VisionFlow) Reset() {
	vf.mu.Lock()
	defer vf.mu.Unlock()
	vf.validated = false
	vf.flowCtrl = NewVisionFlowController()
}
