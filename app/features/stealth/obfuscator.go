package stealth

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"io"
	"math/big"
	"sync"

	"github.com/Hhz0823/oiwest-core/app/common/crypto"
)

var (
	ErrInvalidObfuscation = errors.New("stealth: invalid obfuscation parameters")
	ErrKeyExchange        = errors.New("stealth: key exchange failed")
)

// Pool for obfuscation result buffers to reduce GC pressure
var obfBufPool = sync.Pool{
	New: func() interface{} {
		b := make([]byte, 0, 64*1024)
		return &b
	},
}

func getObfBuf(size int) []byte {
	bp := obfBufPool.Get().(*[]byte)
	buf := *bp
	if cap(buf) >= size {
		return buf[:size]
	}
	return make([]byte, size)
}

func putObfBuf(buf []byte) {
	if cap(buf) > 0 && cap(buf) <= 128*1024 {
		obfBufPool.Put(&buf)
	}
}

type ObfuscationMethod string

const (
	MethodNone        ObfuscationMethod = "none"
	MethodRandom      ObfuscationMethod = "random"
	MethodRandomPad   ObfuscationMethod = "random_padding"
	MethodXTLS        ObfuscationMethod = "xtls"
	MethodVision      ObfuscationMethod = "vision"
	MethodReality     ObfuscationMethod = "reality"
	MethodUDP         ObfuscationMethod = "udp_obfs"
	MethodDTLS        ObfuscationMethod = "dtls"
	MethodWireGuard   ObfuscationMethod = "wireguard"
)

type Obfuscator interface {
	Method() ObfuscationMethod
	Obfuscate(data []byte) ([]byte, error)
	Deobfuscate(data []byte) ([]byte, error)
	Reset()
}

type ObfuscatorConfig struct {
	Method     ObfuscationMethod
	Key        []byte
	IV         []byte
	PaddingMin int
	PaddingMax int
	XTLSFlow   string
	RealityConfig *RealityConfig
}

type RealityConfig struct {
	ServerName  string
	ServerPort  int
	Fingerprint string
	PublicKey   []byte
	PrivateKey  []byte
	ShortID     []byte
	SpiderX     string
}

type baseObfuscator struct {
	mu     sync.Mutex
	method ObfuscationMethod
	config *ObfuscatorConfig
}

func (b *baseObfuscator) Method() ObfuscationMethod {
	return b.method
}

type RandomObfuscator struct {
	baseObfuscator
	xorKey []byte
}

func NewRandomObfuscator(config *ObfuscatorConfig) *RandomObfuscator {
	return &RandomObfuscator{
		baseObfuscator: baseObfuscator{
			method: MethodRandom,
			config: config,
		},
		xorKey: config.Key,
	}
}

func (r *RandomObfuscator) Obfuscate(data []byte) ([]byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	result := make([]byte, len(data))
	keyLen := len(r.xorKey)
	for i := range data {
		result[i] = data[i] ^ r.xorKey[i%keyLen]
	}
	return result, nil
}

func (r *RandomObfuscator) Deobfuscate(data []byte) ([]byte, error) {
	return r.Obfuscate(data)
}

func (r *RandomObfuscator) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.xorKey {
		r.xorKey[i] = 0
	}
}

type RandomPaddingObfuscator struct {
	baseObfuscator
}

func NewRandomPaddingObfuscator(config *ObfuscatorConfig) *RandomPaddingObfuscator {
	return &RandomPaddingObfuscator{
		baseObfuscator: baseObfuscator{
			method: MethodRandomPad,
			config: config,
		},
	}
}

func (rp *RandomPaddingObfuscator) Obfuscate(data []byte) ([]byte, error) {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	padRange := int64(rp.config.PaddingMax - rp.config.PaddingMin + 1)
	padSize, _ := rand.Int(rand.Reader, big.NewInt(padRange))
	totalPad := int(padSize.Int64()) + rp.config.PaddingMin

	totalLen := 2 + totalPad + len(data)
	result := make([]byte, totalLen)
	binary.BigEndian.PutUint16(result[0:2], uint16(totalPad))

	if _, err := io.ReadFull(rand.Reader, result[2:2+totalPad]); err != nil {
		return nil, err
	}

	copy(result[2+totalPad:], data)
	return result, nil
}

func (rp *RandomPaddingObfuscator) Deobfuscate(data []byte) ([]byte, error) {
	if len(data) < 2 {
		return nil, ErrInvalidObfuscation
	}

	padSize := int(binary.BigEndian.Uint16(data[0:2]))
	if 2+padSize > len(data) {
		return nil, ErrInvalidObfuscation
	}

	return data[2+padSize:], nil
}

func (rp *RandomPaddingObfuscator) Reset() {}

type XTLSObfuscator struct {
	baseObfuscator
	aead       *crypto.AEADCipher
	counter    *crypto.SessionCounter
	flow       string
	command    byte
	splitting  bool
}

func NewXTLSObfuscator(config *ObfuscatorConfig) (*XTLSObfuscator, error) {
	aead, err := crypto.NewAEADCipher(crypto.CipherAES256GCM, config.Key)
	if err != nil {
		return nil, err
	}

	flow := config.XTLSFlow
	if flow == "" {
		flow = "xtls-rprx-vision"
	}

	return &XTLSObfuscator{
		baseObfuscator: baseObfuscator{
			method: MethodXTLS,
			config: config,
		},
		aead:      aead,
		counter:   crypto.NewSessionCounter(0),
		flow:      flow,
		command:   0x01,
		splitting: true,
	}, nil
}

func (x *XTLSObfuscator) Obfuscate(data []byte) ([]byte, error) {
	x.mu.Lock()
	defer x.mu.Unlock()

	payload := data

	if x.splitting && len(payload) > 900 {
		payload = payload[:900]
	}

	encrypted := x.aead.Seal(nil, x.counter.Next(), payload, nil)

	length := len(encrypted)
	padSize, _ := rand.Int(rand.Reader, big.NewInt(16))
	pad := int(padSize.Int64())

	result := make([]byte, 5+length+pad)
	result[0] = x.command
	binary.BigEndian.PutUint16(result[1:3], uint16(length))
	result[3] = byte(pad)
	result[4] = byte(0)
	copy(result[5:], encrypted)
	if pad > 0 {
		io.ReadFull(rand.Reader, result[5+length:])
	}

	return result, nil
}

func (x *XTLSObfuscator) Deobfuscate(data []byte) ([]byte, error) {
	if len(data) < 5 {
		return nil, ErrInvalidObfuscation
	}

	length := int(binary.BigEndian.Uint16(data[1:3]))
	pad := int(data[3])
	payloadStart := 5
	payloadEnd := payloadStart + length

	if payloadEnd > len(data) {
		return nil, ErrInvalidObfuscation
	}

	decrypted, err := x.aead.Open(nil, x.counter.Next(), data[payloadStart:payloadEnd], nil)
	if err != nil {
		return nil, err
	}

	_ = pad
	return decrypted, nil
}

func (x *XTLSObfuscator) Reset() {
	x.mu.Lock()
	defer x.mu.Unlock()
	x.counter = crypto.NewSessionCounter(0)
}

type VisionObfuscator struct {
	baseObfuscator
	aead    *crypto.AEADCipher
	counter *crypto.SessionCounter
}

func NewVisionObfuscator(config *ObfuscatorConfig) (*VisionObfuscator, error) {
	aead, err := crypto.NewAEADCipher(crypto.CipherAES256GCM, config.Key)
	if err != nil {
		return nil, err
	}

	return &VisionObfuscator{
		baseObfuscator: baseObfuscator{
			method: MethodVision,
			config: config,
		},
		aead:    aead,
		counter: crypto.NewSessionCounter(0),
	}, nil
}

func (v *VisionObfuscator) Obfuscate(data []byte) ([]byte, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	encrypted := v.aead.Seal(nil, v.counter.Next(), data, nil)

	length := len(encrypted)
	result := make([]byte, 2+length)
	binary.BigEndian.PutUint16(result[0:2], uint16(length))
	copy(result[2:], encrypted)

	return result, nil
}

func (v *VisionObfuscator) Deobfuscate(data []byte) ([]byte, error) {
	if len(data) < 2 {
		return nil, ErrInvalidObfuscation
	}

	length := int(binary.BigEndian.Uint16(data[0:2]))
	if 2+length > len(data) {
		return nil, ErrInvalidObfuscation
	}

	nonceSize := v.aead.NonceSize()
	if length < nonceSize {
		return nil, ErrInvalidObfuscation
	}

	nonce := data[2 : 2+nonceSize]
	ciphertext := data[2+nonceSize : 2+length]

	return v.aead.Open(nil, nonce, ciphertext, nil)
}

func (v *VisionObfuscator) Reset() {}

type RealityObfuscator struct {
	baseObfuscator
	aead       *crypto.AEADCipher
	serverName string
	serverPort int
	shortID    []byte
	publicKey  []byte
	privateKey []byte
	fingerprint string
}

func NewRealityObfuscator(config *ObfuscatorConfig) (*RealityObfuscator, error) {
	if config.RealityConfig == nil {
		return nil, ErrInvalidObfuscation
	}

	aead, err := crypto.NewAEADCipher(crypto.CipherChacha20Poly1305, config.Key)
	if err != nil {
		return nil, err
	}

	return &RealityObfuscator{
		baseObfuscator: baseObfuscator{
			method: MethodReality,
			config: config,
		},
		aead:        aead,
		serverName:  config.RealityConfig.ServerName,
		serverPort:  config.RealityConfig.ServerPort,
		shortID:     config.RealityConfig.ShortID,
		publicKey:   config.RealityConfig.PublicKey,
		privateKey:  config.RealityConfig.PrivateKey,
		fingerprint: config.RealityConfig.Fingerprint,
	}, nil
}

func (r *RealityObfuscator) Obfuscate(data []byte) ([]byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	encrypted, err := r.aead.Encrypt(data)
	if err != nil {
		return nil, err
	}

	length := len(encrypted)
	shortIDLen := len(r.shortID)
	result := make([]byte, 1+shortIDLen+2+length)

	result[0] = byte(shortIDLen)
	copy(result[1:1+shortIDLen], r.shortID)
	binary.BigEndian.PutUint16(result[1+shortIDLen:], uint16(length))
	copy(result[3+shortIDLen:], encrypted)

	return result, nil
}

func (r *RealityObfuscator) Deobfuscate(data []byte) ([]byte, error) {
	if len(data) < 3 {
		return nil, ErrInvalidObfuscation
	}

	shortIDLen := int(data[0])
	start := 1 + shortIDLen
	if start+2 > len(data) {
		return nil, ErrInvalidObfuscation
	}

	length := int(binary.BigEndian.Uint16(data[start:]))
	payloadStart := start + 2

	if payloadStart+length > len(data) {
		return nil, ErrInvalidObfuscation
	}

	return r.aead.Decrypt(data[payloadStart : payloadStart+length])
}

func (r *RealityObfuscator) Reset() {}

func NewObfuscator(method ObfuscationMethod, config *ObfuscatorConfig) (Obfuscator, error) {
	switch method {
	case MethodRandom:
		if len(config.Key) == 0 {
			config.Key = make([]byte, 32)
			io.ReadFull(rand.Reader, config.Key)
		}
		return NewRandomObfuscator(config), nil

	case MethodRandomPad:
		if config.PaddingMin == 0 {
			config.PaddingMin = 4
		}
		if config.PaddingMax == 0 {
			config.PaddingMax = 64
		}
		return NewRandomPaddingObfuscator(config), nil

	case MethodXTLS:
		if len(config.Key) == 0 {
			config.Key = make([]byte, 32)
			io.ReadFull(rand.Reader, config.Key)
		}
		return NewXTLSObfuscator(config)

	case MethodVision:
		if len(config.Key) == 0 {
			config.Key = make([]byte, 32)
			io.ReadFull(rand.Reader, config.Key)
		}
		return NewVisionObfuscator(config)

	case MethodReality:
		if len(config.Key) == 0 {
			config.Key = make([]byte, 32)
			io.ReadFull(rand.Reader, config.Key)
		}
		return NewRealityObfuscator(config)

	default:
		return nil, ErrInvalidObfuscation
	}
}

type StreamObfuscator struct {
	obfuscator Obfuscator
	reader     io.Reader
	writer     io.Writer
	readBuf    []byte
	readPos    int
	readLen    int
	mu         sync.Mutex
}

func NewStreamObfuscator(obfuscator Obfuscator, r io.Reader, w io.Writer) *StreamObfuscator {
	return &StreamObfuscator{
		obfuscator: obfuscator,
		reader:     r,
		writer:     w,
		readBuf:    make([]byte, 64*1024),
	}
}

func (s *StreamObfuscator) Read(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.readLen > 0 {
		n := copy(p, s.readBuf[s.readPos:s.readPos+s.readLen])
		s.readPos += n
		s.readLen -= n
		if s.readLen == 0 {
			s.readPos = 0
		}
		return n, nil
	}

	n, err := s.reader.Read(s.readBuf)
	if err != nil {
		return 0, err
	}

	raw := s.readBuf[:n]

	plaintext, err := s.obfuscator.Deobfuscate(raw)
	if err != nil {
		return 0, err
	}

	copied := copy(p, plaintext)
	if copied < len(plaintext) {
		s.readPos = copied
		s.readLen = len(plaintext) - copied
		// Grow readBuf if needed rather than allocating fixed 64KB
		if cap(s.readBuf) < 64*1024 {
			s.readBuf = make([]byte, 64*1024)
		}
		copy(s.readBuf, plaintext[copied:])
	}

	return copied, nil
}

func (s *StreamObfuscator) Write(p []byte) (int, error) {
	obfuscated, err := s.obfuscator.Obfuscate(p)
	if err != nil {
		return 0, err
	}

	_, err = s.writer.Write(obfuscated)
	if err != nil {
		return 0, err
	}

	return len(p), nil
}
