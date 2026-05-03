package stealth

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"io"
	"math/big"
	"sync"

	"github.com/Hhz0823/oiwest-core/common/crypto"
)

var (
	ErrInvalidObfuscation = errors.New("stealth: invalid obfuscation parameters")
	ErrKeyExchange        = errors.New("stealth: key exchange failed")
)

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
	for i := range data {
		result[i] = data[i] ^ r.xorKey[i%len(r.xorKey)]
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

	padSize, _ := rand.Int(rand.Reader, big.NewInt(int64(rp.config.PaddingMax-rp.config.PaddingMin+1)))
	totalPad := int(padSize.Int64()) + rp.config.PaddingMin

	result := make([]byte, 2+totalPad+len(data))
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
	readBuf    []byte
	writeBuf   []byte
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
		readBuf:   make([]byte, 0, 64*1024),
		writeBuf:  make([]byte, 0, 64*1024),
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
	result := make([]byte, 5+length)

	result[0] = x.command
	binary.BigEndian.PutUint16(result[1:3], uint16(length))

	paddingSize, _ := rand.Int(rand.Reader, big.NewInt(16))
	pad := int(paddingSize.Int64())
	result[3] = byte(pad)
	result[4] = byte(0)

	copy(result[5:], encrypted)

	randomPad := make([]byte, pad)
	io.ReadFull(rand.Reader, randomPad)
	result = append(result, randomPad...)

	return result, nil
}

func (x *XTLSObfuscator) Deobfuscate(data []byte) ([]byte, error) {
	if len(data) < 5 {
		return nil, ErrInvalidObfuscation
	}

	length := int(binary.BigEndian.Uint16(data[1:3]))
	padSize := int(data[3])

	if 5+length+padSize > len(data) {
		return nil, ErrInvalidObfuscation
	}

	ciphertext := data[5 : 5+length]

	plaintext, err := x.aead.Open(nil, x.counter.Next(), ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

func (x *XTLSObfuscator) Reset() {
	x.mu.Lock()
	defer x.mu.Unlock()
	x.counter = crypto.NewSessionCounter(0)
}

type VisionObfuscator struct {
	baseObfuscator
	aead    *crypto.AEADCipher
}

func NewVisionObfuscator(config *ObfuscatorConfig) (*VisionObfuscator, error) {
	aead, err := crypto.NewAEADCipher(crypto.CipherChacha20Poly1305, config.Key)
	if err != nil {
		return nil, err
	}

	return &VisionObfuscator{
		baseObfuscator: baseObfuscator{
			method: MethodVision,
			config: config,
		},
		aead: aead,
	}, nil
}

func (v *VisionObfuscator) Obfuscate(data []byte) ([]byte, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	nonce := make([]byte, v.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	encrypted := v.aead.Seal(nonce, nonce, data, nil)

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
	result := make([]byte, 1+len(r.shortID)+2+length)

	result[0] = byte(len(r.shortID))
	copy(result[1:1+len(r.shortID)], r.shortID)
	binary.BigEndian.PutUint16(result[1+len(r.shortID):], uint16(length))
	copy(result[3+len(r.shortID):], encrypted)

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
		s.readBuf = make([]byte, 64*1024)
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
