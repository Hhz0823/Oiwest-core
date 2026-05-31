package vmess

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"sync"
	"time"

	"github.com/Hhz0823/oiwest-core/app/common/buf"
	"github.com/Hhz0823/oiwest-core/app/common/crypto"
	"github.com/Hhz0823/oiwest-core/app/transport"
)

const (
	Version                              = 1
	CmdTCP                               = 1
	CmdUDP                               = 2
	AddrTypeIPv4                         = 1
	AddrTypeDomain                       = 2
	AddrTypeIPv6                         = 3
	SecurityNone                         = 0
	SecurityAES128GCM                    = 1
	SecurityChaCha20Poly1305             = 2
	SecurityZero                         = 3
	KDFSaltConstAuthIDEncryptionKey      = "AES Auth ID Encryption"
	KDFSaltConstAEADRespHeaderLenKey     = "AEAD Resp Header Len Key"
	KDFSaltConstAEADRespHeaderLenIV      = "AEAD Resp Header Len IV"
	KDFSaltConstAEADRespHeaderPayloadKey = "AEAD Resp Header Key"
	KDFSaltConstAEADRespHeaderPayloadIV  = "AEAD Resp Header IV"
)

var (
	ErrInvalidUser    = errors.New("vmess: invalid user")
	ErrInvalidRequest = errors.New("vmess: invalid request")
	ErrInvalidKey     = errors.New("vmess: invalid key")
)

type User struct {
	UUID  string
	Email string
	Level int
}

type Account struct {
	User    *User
	Cipher  crypto.CipherType
	UUIDKey []byte
}

func NewAccount(user *User, cipherType crypto.CipherType) (*Account, error) {
	uuidKey := GenerateUUIDKey(user.UUID)
	return &Account{User: user, Cipher: cipherType, UUIDKey: uuidKey}, nil
}

func GenerateUUIDKey(uuid string) []byte {
	clean := ""
	for _, c := range uuid {
		if c != '-' {
			clean += string(c)
		}
	}
	key := md5.Sum([]byte(clean))
	return key[:]
}

type VMessInboundHandler struct {
	tag            string
	port           uint16
	listen         string
	users          []*Account
	streamSettings *transport.StreamSettings
	timeout        time.Duration
	mu             sync.RWMutex
}

func NewVMessInbound(tag string, port uint16, listen string, accounts []*Account, ss *transport.StreamSettings) *VMessInboundHandler {
	return &VMessInboundHandler{
		tag:            tag,
		port:           port,
		listen:         listen,
		users:          accounts,
		streamSettings: ss,
		timeout:        30 * time.Second,
	}
}

func (h *VMessInboundHandler) Tag() string { return h.tag }
func (h *VMessInboundHandler) Network() []string {
	return []string{"tcp", "mKCP", "ws", "h2", "quic", "dccp"}
}

func (h *VMessInboundHandler) Process(ctx context.Context, conn net.Conn, dispatch func(context.Context, net.Conn)) error {
	conn.SetDeadline(time.Now().Add(h.timeout))
	defer conn.SetDeadline(time.Time{})

	account, bodyAuth, bodyOpt, err := h.parseHeader(conn)
	if err != nil {
		return err
	}

	reader, writer, err := h.decryptBody(conn, account, bodyAuth, bodyOpt)
	if err != nil {
		return err
	}

	wrapConn := &vmessConn{Conn: conn, reader: reader, writer: writer}
	dispatch(ctx, wrapConn)
	return nil
}

func (h *VMessInboundHandler) parseHeader(conn net.Conn) (*Account, []byte, byte, error) {
	buf := make([]byte, 64)
	_, err := io.ReadFull(conn, buf[:49])
	if err != nil {
		return nil, nil, 0, fmt.Errorf("vmess: read auth info: %w", err)
	}

	if buf[0] != Version {
		return nil, nil, 0, ErrInvalidRequest
	}

	authID := buf[1:17]
	var account *Account
	for _, u := range h.users {
		if hmac.Equal(authID, h.generateAuthID(u)) {
			account = u
			break
		}
	}
	if account == nil {
		return nil, nil, 0, ErrInvalidUser
	}

	return account, buf[17:33], buf[33], nil
}

func (h *VMessInboundHandler) generateAuthID(account *Account) []byte {
	ts := time.Now().Unix()
	buf := make([]byte, 16)
	binary.BigEndian.PutUint64(buf[:8], uint64(ts))
	copy(buf[8:], account.UUIDKey[:8])

	mac := hmac.New(md5.New, []byte(KDFSaltConstAuthIDEncryptionKey))
	mac.Write(buf)
	return mac.Sum(nil)[:16]
}

func (h *VMessInboundHandler) decryptBody(conn net.Conn, account *Account, bodyAuth []byte, bodyOpt byte) (io.Reader, io.Writer, error) {
	respHeaderLenKey := kdf(account.UUIDKey, KDFSaltConstAEADRespHeaderLenKey)
	respHeaderLenIV := kdf(account.UUIDKey, KDFSaltConstAEADRespHeaderLenIV)
	respHeaderPayloadKey := kdf(account.UUIDKey, KDFSaltConstAEADRespHeaderPayloadKey)
	respHeaderPayloadIV := kdf(account.UUIDKey, KDFSaltConstAEADRespHeaderPayloadIV)

	aeadKey := kdfSha256(account.UUIDKey, bodyAuth[:])
	nonce := kdfSha256(account.UUIDKey, []byte{bodyOpt})

	var aead cipher.AEAD
	switch account.Cipher {
	case crypto.CipherAES128GCM:
		block, _ := aes.NewCipher(aeadKey[:16])
		aead, _ = cipher.NewGCM(block)
	case crypto.CipherChacha20Poly1305:
		aead, _ = crypto.NewAEADCipher(crypto.CipherChacha20Poly1305, aeadKey[:32])
	default:
		block, _ := aes.NewCipher(aeadKey[:16])
		aead, _ = cipher.NewGCM(block)
	}

	reader := &vmessAEADReader{conn: conn, aead: aead, nonce: nonce}
	writer := &vmessAEADWriter{
		conn:                 conn,
		respHeaderLenKey:     respHeaderLenKey,
		respHeaderLenIV:      respHeaderLenIV,
		respHeaderPayloadKey: respHeaderPayloadKey,
		respHeaderPayloadIV:  respHeaderPayloadIV,
	}

	return reader, writer, nil
}

func kdf(key []byte, salt string) []byte {
	mac := hmac.New(sha256.New, []byte(salt))
	mac.Write(key)
	return mac.Sum(nil)
}

func kdfSha256(key, path []byte) []byte {
	hasher := sha256.New()
	hasher.Write(key)
	hasher.Write(path)
	return hasher.Sum(nil)
}

type vmessConn struct {
	net.Conn
	reader io.Reader
	writer io.Writer
}

func (c *vmessConn) Read(p []byte) (int, error)  { return c.reader.Read(p) }
func (c *vmessConn) Write(p []byte) (int, error) { return c.writer.Write(p) }

type vmessAEADReader struct {
	conn    net.Conn
	aead    cipher.AEAD
	nonce   []byte
	buf     bytes.Buffer
	payload []byte
	count   uint16
}

func (r *vmessAEADReader) Read(p []byte) (int, error) {
	if r.buf.Len() > 0 {
		return r.buf.Read(p)
	}

	header := make([]byte, 18)
	_, err := io.ReadFull(r.conn, header)
	if err != nil {
		return 0, err
	}

	lenBuf := header[:2]
	r.count++
	nonce := make([]byte, 12)
	copy(nonce, r.nonce[:8])
	binary.BigEndian.PutUint16(nonce[8:10], r.count)
	copy(nonce[10:], r.nonce[10:])

	decLen, err := r.aead.Open(lenBuf[:0], nonce, lenBuf, nil)
	if err != nil {
		return 0, err
	}
	length := binary.BigEndian.Uint16(decLen[:2])

	payload := make([]byte, int(length)+16)
	_, err = io.ReadFull(r.conn, payload)
	if err != nil {
		return 0, err
	}

	r.count++
	nonce2 := make([]byte, 12)
	copy(nonce2, r.nonce[:8])
	binary.BigEndian.PutUint16(nonce2[8:10], r.count)
	copy(nonce2[10:], r.nonce[10:])

	decPayload, err := r.aead.Open(payload[:0], nonce2, payload, nil)
	if err != nil {
		return 0, err
	}

	r.buf.Write(decPayload)
	return r.buf.Read(p)
}

type vmessAEADWriter struct {
	conn                 net.Conn
	respHeaderLenKey     []byte
	respHeaderLenIV      []byte
	respHeaderPayloadKey []byte
	respHeaderPayloadIV  []byte
	headerSent           bool
	mu                   sync.Mutex
}

func (w *vmessAEADWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.headerSent {
		if err := w.writeResponseHeader(); err != nil {
			return 0, err
		}
		w.headerSent = true
	}

	return w.conn.Write(p)
}

func (w *vmessAEADWriter) writeResponseHeader() error {
	block, _ := aes.NewCipher(w.respHeaderLenKey)
	aeadLen, _ := cipher.NewGCM(block)

	nonce := make([]byte, 12)
	io.ReadFull(rand.Reader, nonce[:4])
	copy(nonce[4:], w.respHeaderLenIV[4:12])

	plainLen := []byte{0x00}
	encLen := aeadLen.Seal(nil, nonce, plainLen, nil)

	blockPayload, _ := aes.NewCipher(w.respHeaderPayloadKey)
	aeadPayload, _ := cipher.NewGCM(blockPayload)

	noncePayload := make([]byte, 12)
	io.ReadFull(rand.Reader, noncePayload[:4])
	copy(noncePayload[4:], w.respHeaderPayloadIV[4:12])

	resp := []byte{0x00}
	encPayload := aeadPayload.Seal(nil, noncePayload, resp, nil)

	_, err := w.conn.Write(encLen)
	if err != nil {
		return err
	}
	_, err = w.conn.Write(encPayload)
	return err
}

type VMessOutboundHandler struct {
	tag            string
	address        string
	port           uint16
	uuid           string
	security       crypto.CipherType
	streamSettings *transport.StreamSettings
	timeout        time.Duration
}

func NewVMessOutbound(tag string, address string, port uint16, uuid string, security crypto.CipherType, ss *transport.StreamSettings) *VMessOutboundHandler {
	return &VMessOutboundHandler{
		tag:            tag,
		address:        address,
		port:           port,
		uuid:           uuid,
		security:       security,
		streamSettings: ss,
		timeout:        30 * time.Second,
	}
}

func (h *VMessOutboundHandler) Tag() string { return h.tag }

func (h *VMessOutboundHandler) Process(ctx context.Context, link *VMessLink) error {
	var conn net.Conn
	var err error

	if h.streamSettings != nil {
		conn, err = dialByTransport(ctx, h.streamSettings, h.address, int(h.port))
	} else {
		conn, err = net.DialTimeout("tcp", fmt.Sprintf("%s:%d", h.address, h.port), h.timeout)
	}
	if err != nil {
		return err
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(h.timeout))
	defer conn.SetDeadline(time.Time{})

	uuidKey := GenerateUUIDKey(h.uuid)
	authID := generateOutboundAuthID(uuidKey)

	cmd := byte(CmdTCP)
	if link.IsUDP {
		cmd = CmdUDP
	}

	header := buildVMessRequestHeader(authID, uuidKey, h.security, link.Target, cmd)
	if _, err := conn.Write(header); err != nil {
		return err
	}

	if h.security != SecurityNone && h.security != SecurityZero {
		bodyKey := make([]byte, 16)
		bodyIV := make([]byte, 16)
		io.ReadFull(rand.Reader, bodyKey)
		io.ReadFull(rand.Reader, bodyIV)
		conn.Write(bodyKey)
		conn.Write(bodyIV)

		var aead cipher.AEAD
		switch h.security {
		case crypto.CipherAES128GCM:
			block, _ := aes.NewCipher(bodyKey)
			aead, _ = cipher.NewGCM(block)
		case crypto.CipherChacha20Poly1305:
			aead, _ = crypto.NewAEADCipher(crypto.CipherChacha20Poly1305, bodyKey)
		}

		reader := &vmessAEADReader{conn: conn, aead: aead, nonce: bodyIV}
		writer := &vmessSimpleWriter{conn: conn}

		errCh := make(chan error, 2)
		go func() { _, err := io.Copy(writer, link.Reader); errCh <- err }()
		go func() { _, err := io.Copy(link.Writer, reader); errCh <- err }()

		select {
		case err := <-errCh:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	errCh := make(chan error, 2)
	go func() { _, err := io.Copy(conn, link.Reader); errCh <- err }()
	go func() { _, err := io.Copy(link.Writer, conn); errCh <- err }()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func generateOutboundAuthID(uuidKey []byte) []byte {
	ts := time.Now().Unix()
	buf := make([]byte, 16)
	binary.BigEndian.PutUint64(buf[:8], uint64(ts))
	copy(buf[8:], uuidKey[:8])

	mac := hmac.New(md5.New, []byte(KDFSaltConstAuthIDEncryptionKey))
	mac.Write(buf)
	return mac.Sum(nil)[:16]
}

func buildVMessRequestHeader(authID, uuidKey []byte, security crypto.CipherType, target string, cmd byte) []byte {
	var header bytes.Buffer
	header.WriteByte(Version)
	header.Write(authID)
	bodyKey := make([]byte, 16)
	bodyIV := make([]byte, 16)
	io.ReadFull(rand.Reader, bodyKey)
	io.ReadFull(rand.Reader, bodyIV)

	opt := byte(security) | (byte(security) << 4)
	header.Write(bodyKey)
	header.WriteByte(opt)

	p := byte(0)
	paddingLen := p & 0x0F
	header.WriteByte(p)

	if cmd == CmdUDP {
		header.WriteByte(0)
	}

	commandVal := byte(cmd)
	header.WriteByte(commandVal)

	header.WriteByte(0x00)

	host, portStr, _ := net.SplitHostPort(target)
	var port int
	fmt.Sscanf(portStr, "%d", &port)

	ip := net.ParseIP(host)
	if ip != nil {
		if ip.To4() != nil {
			header.WriteByte(AddrTypeIPv4)
			header.Write(ip.To4())
		} else {
			header.WriteByte(AddrTypeIPv6)
			header.Write(ip.To16())
		}
	} else {
		header.WriteByte(AddrTypeDomain)
		header.WriteByte(byte(len(host)))
		header.WriteString(host)
	}

	binary.Write(&header, binary.BigEndian, uint16(port))

	if paddingLen > 0 {
		padding := make([]byte, paddingLen)
		io.ReadFull(rand.Reader, padding)
		header.Write(padding)
	}

	fnvHash := fnv1a32(header.Bytes())
	result := make([]byte, 0, header.Len()+4)
	result = append(result, header.Bytes()...)
	fnvBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(fnvBuf, fnvHash)
	result = append(result, fnvBuf...)

	return result
}

func fnv1a32(data []byte) uint32 {
	var hash uint32 = 0x811c9dc5
	for _, b := range data {
		hash ^= uint32(b)
		hash *= 0x01000193
	}
	return hash
}

func dialByTransport(ctx context.Context, ss *transport.StreamSettings, addr string, port int) (net.Conn, error) {
	switch ss.Network {
	case transport.TransportWebSocket:
		return dialWebSocket(ctx, ss, addr, port)
	case transport.TransportQUIC:
		return dialQUIC(ctx, ss, addr, port)
	default:
		return net.DialTimeout("tcp", fmt.Sprintf("%s:%d", addr, port), 15*time.Second)
	}
}

func dialWebSocket(ctx context.Context, ss *transport.StreamSettings, addr string, port int) (net.Conn, error) {
	tcpConn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", addr, port), 15*time.Second)
	if err != nil {
		return nil, err
	}
	path := "/"
	if ss.WSSettings != nil && ss.WSSettings.Path != "" {
		path = ss.WSSettings.Path
	}
	wsConn, err := transport.NewWebSocketConn(tcpConn, addr, path, ss)
	if err != nil {
		tcpConn.Close()
		return nil, err
	}
	return wsConn, nil
}

func dialQUIC(ctx context.Context, ss *transport.StreamSettings, addr string, port int) (net.Conn, error) {
	return transport.NewQUICConn(ctx, addr, port, ss)
}

type vmessSimpleWriter struct {
	conn net.Conn
}

func (w *vmessSimpleWriter) Write(p []byte) (int, error) {
	return w.conn.Write(p)
}

type VMessLink struct {
	Reader io.Reader
	Writer io.Writer
	Target string
	IsUDP  bool
}

var (
	_ = crypto.AEADCipher{}
	_ = buf.Buffer{}
	_ = math.MaxFloat64
)
