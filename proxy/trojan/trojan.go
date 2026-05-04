package trojan

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/Hhz0823/oiwest-core/transport"
)

const (
	MaxPacketSize      = 8192
	NetworkTCP         = "tcp"
	NetworkUDP         = "udp"
	MinPasswordLength  = 8
)

var (
	ErrInvalidPassword   = errors.New("trojan: invalid password")
	ErrAuthFailed        = errors.New("trojan: authentication failed")
	ErrUnsupportedNetwork = errors.New("trojan: unsupported network")
)

type User struct {
	Password string
	Hash     string
	Level    int
}

func HashPassword(password string) string {
	hash := sha256.Sum224([]byte(password))
	return hex.EncodeToString(hash[:])
}

type TrojanInboundHandler struct {
	tag            string
	port           uint16
	listen         string
	users          map[string]*User
	streamSettings *transport.StreamSettings
	fallbackAddr   string
	fallbackPort   uint16
	timeout        time.Duration
	mu             sync.RWMutex
}

func NewTrojanInbound(tag string, port uint16, listen string, passwords []string, ss *transport.StreamSettings) *TrojanInboundHandler {
	users := make(map[string]*User)
	for _, pass := range passwords {
		hash := HashPassword(pass)
		users[hash] = &User{Password: pass, Hash: hash}
	}
	return &TrojanInboundHandler{
		tag:            tag,
		port:           port,
		listen:         listen,
		users:          users,
		streamSettings: ss,
		timeout:        30 * time.Second,
	}
}

func (h *TrojanInboundHandler) Tag() string       { return h.tag }
func (h *TrojanInboundHandler) Network() []string  { return []string{"tcp", "ws", "h2", "grpc", "dccp"} }

func (h *TrojanInboundHandler) SetFallback(addr string, port uint16) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.fallbackAddr = addr
	h.fallbackPort = port
}

func (h *TrojanInboundHandler) GetFallback() (string, uint16) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.fallbackAddr, h.fallbackPort
}

func (h *TrojanInboundHandler) Process(ctx context.Context, conn net.Conn, dispatch func(context.Context, net.Conn)) error {
	conn.SetDeadline(time.Now().Add(h.timeout))
	defer conn.SetDeadline(time.Time{})

	trojanConn, target, err := h.handshake(ctx, conn)
	if err != nil {
		if h.fallbackAddr != "" {
			h.handleFallback(conn)
			return nil
		}
		return err
	}

	_ = target
	dispatch(ctx, trojanConn)
	return nil
}

func (h *TrojanInboundHandler) handshake(ctx context.Context, conn net.Conn) (net.Conn, string, error) {
	buf := make([]byte, 128)
	n, err := io.ReadFull(conn, buf[:56])
	if err != nil {
		return nil, "", fmt.Errorf("trojan: read password: %w", err)
	}

	passwordHex := hex.EncodeToString(buf[:56])

	h.mu.RLock()
	user, ok := h.users[passwordHex]
	h.mu.RUnlock()
	if !ok {
		return nil, "", ErrAuthFailed
	}

	_ = user

	n, err = io.ReadFull(conn, buf[:2])
	if err != nil || n < 2 {
		return nil, "", fmt.Errorf("trojan: read crlf: %w", err)
	}
	if buf[0] != 0x0D || buf[1] != 0x0A {
		return nil, "", fmt.Errorf("trojan: invalid CRLF")
	}

	cmd, err := readByte(conn)
	if err != nil {
		return nil, "", err
	}

	target, err := h.parseAddress(conn, cmd)
	if err != nil {
		return nil, "", err
	}

	crlf2 := make([]byte, 2)
	io.ReadFull(conn, crlf2)

	return conn, target, nil
}

func (h *TrojanInboundHandler) parseAddress(conn net.Conn, cmd byte) (string, error) {
	if cmd != 0x01 {
		return "", fmt.Errorf("trojan: unsupported command %d", cmd)
	}

	addrType, err := readByte(conn)
	if err != nil {
		return "", err
	}

	var host string
	switch addrType {
	case 0x01:
		ip := make([]byte, 4)
		io.ReadFull(conn, ip)
		host = net.IP(ip).String()
	case 0x03:
		lenBuf, _ := readByte(conn)
		domain := make([]byte, int(lenBuf))
		io.ReadFull(conn, domain)
		host = string(domain)
	case 0x04:
		ip := make([]byte, 16)
		io.ReadFull(conn, ip)
		host = net.IP(ip).String()
	default:
		return "", fmt.Errorf("trojan: unknown address type %d", addrType)
	}

	portBuf := make([]byte, 2)
	io.ReadFull(conn, portBuf)
	port := binary.BigEndian.Uint16(portBuf)

	return net.JoinHostPort(host, fmt.Sprintf("%d", port)), nil
}

func (h *TrojanInboundHandler) handleFallback(conn net.Conn) {
	defer conn.Close()
	fallback, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", h.fallbackAddr, h.fallbackPort), 5*time.Second)
	if err != nil {
		return
	}
	defer fallback.Close()

	go io.Copy(fallback, conn)
	io.Copy(conn, fallback)
}

func readByte(r io.Reader) (byte, error) {
	var b [1]byte
	_, err := io.ReadFull(r, b[:])
	return b[0], err
}

type TrojanOutboundHandler struct {
	tag            string
	address        string
	port           uint16
	password       string
	streamSettings *transport.StreamSettings
	timeout        time.Duration
}

func NewTrojanOutbound(tag, address string, port uint16, password string, ss *transport.StreamSettings) *TrojanOutboundHandler {
	return &TrojanOutboundHandler{
		tag:            tag,
		address:        address,
		port:           port,
		password:       password,
		streamSettings: ss,
		timeout:        30 * time.Second,
	}
}

func (h *TrojanOutboundHandler) Tag() string { return h.tag }

func (h *TrojanOutboundHandler) Process(ctx context.Context, link *TrojanLink) error {
	var conn net.Conn
	var err error

	if h.streamSettings != nil && h.streamSettings.Network == transport.TransportWebSocket {
		tcpConn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", h.address, h.port), h.timeout)
		if err != nil {
			return err
		}
		path := "/"
		if h.streamSettings.WSSettings != nil && h.streamSettings.WSSettings.Path != "" {
			path = h.streamSettings.WSSettings.Path
		}
		conn, err = transport.NewWebSocketConn(tcpConn, h.address, path, h.streamSettings)
		if err != nil {
			tcpConn.Close()
			return err
		}
	} else {
		conn, err = net.DialTimeout("tcp", fmt.Sprintf("%s:%d", h.address, h.port), h.timeout)
	}
	if err != nil {
		return err
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(h.timeout))
	defer conn.SetDeadline(time.Time{})

	request := h.buildRequest(link.Target)
	if _, err := conn.Write(request); err != nil {
		return err
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

func (h *TrojanOutboundHandler) buildRequest(target string) []byte {
	passwordHash := sha256.Sum224([]byte(h.password))

	host, portStr, _ := net.SplitHostPort(target)
	portVal := uint16(0)
	fmt.Sscanf(portStr, "%d", &portVal)

	var addrBuf bytes.Buffer
	ip := net.ParseIP(host)
	if ip != nil {
		if ip.To4() != nil {
			addrBuf.WriteByte(0x01)
			addrBuf.Write(ip.To4())
		} else {
			addrBuf.WriteByte(0x04)
			addrBuf.Write(ip.To16())
		}
	} else {
		addrBuf.WriteByte(0x03)
		addrBuf.WriteByte(byte(len(host)))
		addrBuf.WriteString(host)
	}
	binary.Write(&addrBuf, binary.BigEndian, portVal)
	addrBytes := addrBuf.Bytes()

	var buf bytes.Buffer
	buf.Write(passwordHash[:])
	buf.Write([]byte{0x0D, 0x0A})
	buf.WriteByte(0x01)
	buf.Write(addrBytes)
	buf.Write([]byte{0x0D, 0x0A})

	return buf.Bytes()
}

type TrojanLink struct {
	Reader io.Reader
	Writer io.Writer
	Target string
}
