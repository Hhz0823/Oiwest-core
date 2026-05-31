package stealth

import (
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"io"
	"sync"
	"time"
)

var (
	ErrRealityVerification = errors.New("reality: verification failed")
	ErrRealityHandshake    = errors.New("reality: handshake failed")
)

type RealityHandshakeConfig struct {
	ServerName  string
	ServerPort  int
	Fingerprint string
	ShortID     []byte
	PublicKey   []byte
	PrivateKey  []byte
	SpiderX     string
	TimeDiff    time.Duration
}

type RealityClient struct {
	config *RealityHandshakeConfig
	pubKey ed25519.PublicKey
	priKey ed25519.PrivateKey
	mu     sync.Mutex
}

func NewRealityClient(config *RealityHandshakeConfig) (*RealityClient, error) {
	if config.ShortID == nil {
		config.ShortID = make([]byte, 8)
		io.ReadFull(rand.Reader, config.ShortID)
	}

	var pubKey ed25519.PublicKey
	var priKey ed25519.PrivateKey

	if len(config.PublicKey) > 0 {
		pubKey = ed25519.PublicKey(config.PublicKey)
	}

	if len(config.PrivateKey) > 0 {
		priKey = ed25519.PrivateKey(config.PrivateKey)
	} else {
		var err error
		_, priKey, err = ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return nil, err
		}
		pubKey = priKey.Public().(ed25519.PublicKey)
	}

	return &RealityClient{
		config: config,
		pubKey: pubKey,
		priKey: priKey,
	}, nil
}

func (rc *RealityClient) BuildClientHello(conn io.ReadWriter) ([]byte, error) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	timeStamp := time.Now().Add(rc.config.TimeDiff).Unix()
	timeBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(timeBytes, uint64(timeStamp))

	randomBytes := make([]byte, 32)
	io.ReadFull(rand.Reader, randomBytes)

	hello := make([]byte, 0, 64+len(rc.config.ShortID))
	hello = append(hello, timeBytes...)
	hello = append(hello, randomBytes...)
	hello = append(hello, rc.config.ShortID...)

	digest := sha256.Sum256(hello)
	sig := ed25519.Sign(rc.priKey, digest[:])

	verification := &RealityVerification{
		TimeStamp: timeStamp,
		ShortID:   rc.config.ShortID,
		Signature: sig,
		PublicKey: rc.pubKey,
	}

	return encodeVerification(verification)
}

type RealityVerification struct {
	TimeStamp int64
	ShortID   []byte
	Signature []byte
	PublicKey ed25519.PublicKey
}

func encodeVerification(v *RealityVerification) ([]byte, error) {
	pubKeyLen := len(v.PublicKey)
	shortIDLen := len(v.ShortID)
	sigLen := len(v.Signature)

	totalLen := 10 + 8 + 1 + shortIDLen + 2 + sigLen + 2 + pubKeyLen
	buf := make([]byte, totalLen)

	binary.BigEndian.PutUint64(buf[0:8], uint64(v.TimeStamp))
	buf[8] = byte(shortIDLen)
	buf[9] = 0
	copy(buf[10:10+shortIDLen], v.ShortID)
	offset := 10 + shortIDLen
	binary.BigEndian.PutUint16(buf[offset:], uint16(sigLen))
	offset += 2
	copy(buf[offset:offset+sigLen], v.Signature)
	offset += sigLen
	binary.BigEndian.PutUint16(buf[offset:], uint16(pubKeyLen))
	offset += 2
	copy(buf[offset:], v.PublicKey)

	return buf, nil
}

func decodeVerification(data []byte) (*RealityVerification, error) {
	if len(data) < 10 {
		return nil, ErrRealityVerification
	}

	v := &RealityVerification{
		TimeStamp: int64(binary.BigEndian.Uint64(data[0:8])),
	}

	shortIDLen := int(data[8])
	if 10+shortIDLen > len(data) {
		return nil, ErrRealityVerification
	}

	v.ShortID = make([]byte, shortIDLen)
	copy(v.ShortID, data[10:10+shortIDLen])

	offset := 10 + shortIDLen
	if offset+2 > len(data) {
		return nil, ErrRealityVerification
	}

	sigLen := int(binary.BigEndian.Uint16(data[offset:]))
	offset += 2

	if offset+sigLen > len(data) {
		return nil, ErrRealityVerification
	}

	v.Signature = make([]byte, sigLen)
	copy(v.Signature, data[offset:offset+sigLen])
	offset += sigLen

	if offset+2 > len(data) {
		return nil, ErrRealityVerification
	}

	pubKeyLen := int(binary.BigEndian.Uint16(data[offset:]))
	offset += 2

	if offset+pubKeyLen > len(data) {
		return nil, ErrRealityVerification
	}

	v.PublicKey = make([]byte, pubKeyLen)
	copy(v.PublicKey, data[offset:])

	return v, nil
}

type RealityHandshakeHandler struct {
	config    *RealityHandshakeConfig
	client    *RealityClient
	allowList map[string]bool
	mu        sync.RWMutex
}

func NewRealityHandshakeHandler(config *RealityHandshakeConfig) (*RealityHandshakeHandler, error) {
	client, err := NewRealityClient(config)
	if err != nil {
		return nil, err
	}

	return &RealityHandshakeHandler{
		config:    config,
		client:    client,
		allowList: make(map[string]bool),
	}, nil
}

func (rh *RealityHandshakeHandler) VerifyConnection(conn io.Reader) (*RealityVerification, error) {
	buf := make([]byte, 256)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, err
	}

	v, err := decodeVerification(buf[:n])
	if err != nil {
		return nil, err
	}

	now := time.Now().Add(rh.config.TimeDiff).Unix()
	diff := now - v.TimeStamp
	if diff < 0 {
		diff = -diff
	}
	if diff > 120 {
		return nil, ErrRealityVerification
	}

	rh.mu.RLock()
	allowed, exists := rh.allowList[hex.EncodeToString(v.ShortID)]
	rh.mu.RUnlock()

	if exists && !allowed {
		return nil, ErrRealityVerification
	}

	digest := sha256.Sum256(buf[:n-len(v.PublicKey)-2])
	if !ed25519.Verify(v.PublicKey, digest[:], v.Signature) {
		return nil, ErrRealityVerification
	}

	return v, nil
}

func (rh *RealityHandshakeHandler) AddAllowedShortID(shortID []byte) {
	rh.mu.Lock()
	defer rh.mu.Unlock()
	rh.allowList[hex.EncodeToString(shortID)] = true
}

func (rh *RealityHandshakeHandler) RemoveAllowedShortID(shortID []byte) {
	rh.mu.Lock()
	defer rh.mu.Unlock()
	delete(rh.allowList, hex.EncodeToString(shortID))
}

type ECDHKeyExchange struct {
	curve     ecdh.Curve
	privateKey *ecdh.PrivateKey
	publicKey  *ecdh.PublicKey
	peerKey   *ecdh.PublicKey
	sharedSecret []byte
	mu        sync.Mutex
}

func NewECDHKeyExchange() (*ECDHKeyExchange, error) {
	curve := ecdh.X25519()
	privateKey, err := curve.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	return &ECDHKeyExchange{
		curve:      curve,
		privateKey: privateKey,
		publicKey:  privateKey.PublicKey(),
	}, nil
}

func (e *ECDHKeyExchange) PublicKeyBytes() []byte {
	return e.publicKey.Bytes()
}

func (e *ECDHKeyExchange) SetPeerPublicKey(peerKey []byte) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	pk, err := e.curve.NewPublicKey(peerKey)
	if err != nil {
		return err
	}

	e.peerKey = pk
	secret, err := e.privateKey.ECDH(pk)
	if err != nil {
		return err
	}
	e.sharedSecret = secret
	return nil
}

func (e *ECDHKeyExchange) SharedSecret() []byte {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.sharedSecret
}

func (e *ECDHKeyExchange) DeriveKey(length int) ([]byte, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.sharedSecret == nil {
		return nil, errors.New("no shared secret")
	}

	key := make([]byte, length)
	h := sha256.New()
	h.Write(e.sharedSecret)
	copy(key, h.Sum(nil))

	for len(key) < length {
		h.Reset()
		h.Write(e.sharedSecret)
		h.Write(key[:len(key)/2])
		copy(key[len(key)/2:], h.Sum(nil))
	}

	return key[:length], nil
}
