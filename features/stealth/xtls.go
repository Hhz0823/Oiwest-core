package stealth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/binary"
	"encoding/pem"
	"io"
	"math/big"
	"net"
	"sync"
	"time"

	"github.com/Hhz0823/oiwest-core/common/crypto"
)

type XTLSFlowController struct {
	mu          sync.Mutex
	cipher      *crypto.AEADCipher
	counter     *crypto.SessionCounter
	flow        string
	enableVision bool
	readBuf     []byte
	writeBuf    []byte
	direct      bool
}

func NewXTLSFlowController(cipher *crypto.AEADCipher, flow string) *XTLSFlowController {
	return &XTLSFlowController{
		cipher:       cipher,
		counter:      crypto.NewSessionCounter(0),
		flow:         flow,
		enableVision: true,
		readBuf:      make([]byte, 0, 64*1024),
		writeBuf:     make([]byte, 0, 64*1024),
		direct:       false,
	}
}

func (x *XTLSFlowController) EnableDirect() {
	x.mu.Lock()
	defer x.mu.Unlock()
	x.direct = true
}

func (x *XTLSFlowController) IsDirect() bool {
	x.mu.Lock()
	defer x.mu.Unlock()
	return x.direct
}

func (x *XTLSFlowController) EncryptFrame(data []byte) ([]byte, error) {
	x.mu.Lock()
	defer x.mu.Unlock()

	nonce := x.counter.Next()
	encrypted := x.cipher.Seal(nil, nonce, data, nil)

	frameSize := len(encrypted)
	frame := make([]byte, 2+frameSize)
	binary.BigEndian.PutUint16(frame[0:2], uint16(frameSize))
	copy(frame[2:], encrypted)

	return frame, nil
}

func (x *XTLSFlowController) DecryptFrame(data []byte) ([]byte, error) {
	if len(data) < 2 {
		return nil, io.ErrUnexpectedEOF
	}

	frameSize := int(binary.BigEndian.Uint16(data[0:2]))
	if 2+frameSize > len(data) {
		return nil, io.ErrUnexpectedEOF
	}

	nonce := x.counter.Next()
	return x.cipher.Open(nil, nonce, data[2:2+frameSize], nil)
}

type XTLSConn struct {
	conn      net.Conn
	controller *XTLSFlowController
	readBuf    []byte
	writeBuf   []byte
	mu         sync.Mutex
}

func NewXTLSConn(conn net.Conn, controller *XTLSFlowController) *XTLSConn {
	return &XTLSConn{
		conn:       conn,
		controller: controller,
		readBuf:    make([]byte, 64*1024),
		writeBuf:   make([]byte, 64*1024),
	}
}

func (x *XTLSConn) Read(b []byte) (int, error) {
	if x.controller.IsDirect() {
		return x.conn.Read(b)
	}

	n, err := x.conn.Read(x.readBuf)
	if err != nil {
		return 0, err
	}

	plaintext, err := x.controller.DecryptFrame(x.readBuf[:n])
	if err != nil {
		return 0, err
	}

	return copy(b, plaintext), nil
}

func (x *XTLSConn) Write(b []byte) (int, error) {
	if x.controller.IsDirect() {
		return x.conn.Write(b)
	}

	frame, err := x.controller.EncryptFrame(b)
	if err != nil {
		return 0, err
	}

	_, err = x.conn.Write(frame)
	if err != nil {
		return 0, err
	}

	return len(b), nil
}

func (x *XTLSConn) Close() error {
	return x.conn.Close()
}

func (x *XTLSConn) LocalAddr() net.Addr {
	return x.conn.LocalAddr()
}

func (x *XTLSConn) RemoteAddr() net.Addr {
	return x.conn.RemoteAddr()
}

func (x *XTLSConn) SetDeadline(t time.Time) error {
	return x.conn.SetDeadline(t)
}

func (x *XTLSConn) SetReadDeadline(t time.Time) error {
	return x.conn.SetReadDeadline(t)
}

func (x *XTLSConn) SetWriteDeadline(t time.Time) error {
	return x.conn.SetWriteDeadline(t)
}

type RealityServer struct {
	config     *RealityConfig
	privateKey *rsa.PrivateKey
	tlsConfig  *tls.Config
	cert       tls.Certificate
	listener   net.Listener
	mu         sync.Mutex
	running    bool
}

func NewRealityServer(config *RealityConfig) (*RealityServer, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	cert, err := generateSelfSignedCert(config.ServerName, privateKey)
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
		MaxVersion:   tls.VersionTLS13,
		ServerName:   config.ServerName,
	}

	return &RealityServer{
		config:     config,
		privateKey: privateKey,
		tlsConfig:  tlsConfig,
		cert:       cert,
	}, nil
}

func (rs *RealityServer) Listen(addr string) (net.Listener, error) {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	if rs.running {
		return nil, nil
	}

	listener, err := tls.Listen("tcp", addr, rs.tlsConfig)
	if err != nil {
		return nil, err
	}

	rs.listener = listener
	rs.running = true

	return &realityListener{server: rs, listener: listener}, nil
}

func (rs *RealityServer) Close() error {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	rs.running = false
	if rs.listener != nil {
		return rs.listener.Close()
	}
	return nil
}

type realityListener struct {
	server   *RealityServer
	listener net.Listener
}

func (rl *realityListener) Accept() (net.Conn, error) {
	return rl.listener.Accept()
}

func (rl *realityListener) Close() error {
	return rl.listener.Close()
}

func (rl *realityListener) Addr() net.Addr {
	return rl.listener.Addr()
}

func generateSelfSignedCert(serverName string, privateKey *rsa.PrivateKey) (tls.Certificate, error) {
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:   serverName,
			Organization: []string{"DCCP Kernel"},
		},
		NotBefore:             time.Now().Add(-24 * time.Hour),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{serverName},
	}

	if serverName == "" {
		template.DNSNames = []string{"localhost"}
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return tls.Certificate{}, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})

	return tls.X509KeyPair(certPEM, keyPEM)
}

type FingerprintGenerator struct {
	fingerprints map[string]string
	mu           sync.RWMutex
}

var supportedFingerprints = map[string]string{
	"chrome":     "chrome_120",
	"firefox":    "firefox_120",
	"safari":     "safari_17",
	"ios":        "ios_17",
	"android":    "android_14",
	"edge":       "edge_120",
	"360":        "360_13",
	"qq":         "qq_11",
	"random":     "randomized",
	"randomized": "randomized",
}

func NewFingerprintGenerator() *FingerprintGenerator {
	return &FingerprintGenerator{
		fingerprints: supportedFingerprints,
	}
}

func (fg *FingerprintGenerator) GetFingerprint(name string) string {
	fg.mu.RLock()
	defer fg.mu.RUnlock()
	if fp, ok := fg.fingerprints[name]; ok {
		return fp
	}
	return "randomized"
}

func (fg *FingerprintGenerator) GenerateRandomFingerprint() string {
	return "randomized_" + randomString(8)
}

type VisionFlowController struct {
	mu           sync.Mutex
	encryptedLen int
	rawLen       int
	readCount    int
	writeCount   int
	paddingSize  int
	enabled      bool
}

func NewVisionFlowController() *VisionFlowController {
	return &VisionFlowController{
		enabled:     true,
		paddingSize: 0,
	}
}

func (v *VisionFlowController) PadTraffic(data []byte, targetSize int) []byte {
	v.mu.Lock()
	defer v.mu.Unlock()

	if !v.enabled || len(data) >= targetSize {
		return data
	}

	padLen := targetSize - len(data)
	if padLen > 256 {
		padLen = 256
	}

	result := make([]byte, len(data)+padLen)
	copy(result, data)
	io.ReadFull(rand.Reader, result[len(data):])

	return result
}

func (v *VisionFlowController) UnpadTraffic(data []byte) []byte {
	return data
}

func (v *VisionFlowController) Enable() {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.enabled = true
}

func (v *VisionFlowController) Disable() {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.enabled = false
}

func sha256Sum(data []byte) []byte {
	h := sha256.Sum256(data)
	return h[:]
}

func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		idx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		b[i] = charset[idx.Int64()]
	}
	return string(b)
}
