package vless

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/Hhz0823/oiwest-core/app/transport"
)

const (
	Version = 0
	CmdTCP  = 1
	CmdUDP  = 2
	CmdMux  = 3

	AddrTypeIPv4   = 1
	AddrTypeDomain = 2
	AddrTypeIPv6   = 3

	FlowNone  = "none"
	FlowXTLS  = "xtls-rprx-vision"
	FlowXRXU  = "xtls-rprx-vision-udp443"
)

var (
	ErrInvalidFlow  = errors.New("vless: invalid flow")
	ErrAuthFailed   = errors.New("vless: authentication failed")
)

type User struct {
	UUID    string
	Email   string
	Flow    string
	Level   int
	Encryption string
}

type VLESSInboundHandler struct {
	tag            string
	port           uint16
	listen         string
	users          map[string]*User
	streamSettings *transport.StreamSettings
	timeout        time.Duration
	mu             sync.RWMutex
}

func NewVLESSInbound(tag string, port uint16, listen string, users []*User, ss *transport.StreamSettings) *VLESSInboundHandler {
	userMap := make(map[string]*User)
	for _, u := range users {
		userMap[u.UUID] = u
	}
	return &VLESSInboundHandler{
		tag:            tag,
		port:           port,
		listen:         listen,
		users:          userMap,
		streamSettings: ss,
		timeout:        30 * time.Second,
	}
}

func (h *VLESSInboundHandler) Tag() string      { return h.tag }
func (h *VLESSInboundHandler) Network() []string { return []string{"tcp", "mKCP", "ws", "h2", "quic", "grpc", "xhttp", "dccp"} }

func (h *VLESSInboundHandler) AddUser(user *User) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.users[user.UUID] = user
}

func (h *VLESSInboundHandler) RemoveUser(uuid string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.users, uuid)
}

func (h *VLESSInboundHandler) Process(ctx context.Context, conn net.Conn, dispatch func(context.Context, net.Conn)) error {
	conn.SetDeadline(time.Now().Add(h.timeout))
	defer conn.SetDeadline(time.Time{})

	vlessConn, target, err := h.Handshake(conn)
	if err != nil {
		return err
	}

	_ = target

	dispatch(ctx, vlessConn)
	return nil
}

func (h *VLESSInboundHandler) Handshake(conn net.Conn) (net.Conn, string, error) {
	buf := make([]byte, 64)
	n, err := io.ReadFull(conn, buf[:1])
	if err != nil || n < 1 {
		return nil, "", fmt.Errorf("vless: read version: %w", err)
	}
	if buf[0] != Version {
		return nil, "", fmt.Errorf("vless: invalid version %d", buf[0])
	}

	n, err = io.ReadFull(conn, buf[:16])
	if err != nil || n < 16 {
		return nil, "", fmt.Errorf("vless: read UUID: %w", err)
	}
	uuidStr := fmt.Sprintf("%x-%x-%x-%x-%x", buf[0:4], buf[4:6], buf[6:8], buf[8:10], buf[10:16])

	if _, ok := h.users[uuidStr]; !ok {
		return nil, "", ErrAuthFailed
	}

	n, err = io.ReadFull(conn, buf[:17])
	if err != nil || n < 17 {
		return nil, "", fmt.Errorf("vless: read opts: %w", err)
	}

	cmd := buf[16]
	if cmd != CmdTCP && cmd != CmdUDP && cmd != CmdMux {
		return nil, "", fmt.Errorf("vless: unknown command %d", cmd)
	}

	targetAddr, err := h.parseTarget(conn)
	if err != nil {
		return nil, "", err
	}

	respHeader := []byte{Version, 0}
	conn.Write(respHeader)

	return conn, targetAddr, nil
}

func (h *VLESSInboundHandler) parseTarget(conn net.Conn) (string, error) {
	buf := make([]byte, 1)
	io.ReadFull(conn, buf)
	addrType := buf[0]

	var host string
	switch addrType {
	case AddrTypeIPv4:
		ip := make([]byte, 4)
		io.ReadFull(conn, ip)
		host = net.IP(ip).String()
	case AddrTypeDomain:
		io.ReadFull(conn, buf[:1])
		domainLen := int(buf[0])
		domain := make([]byte, domainLen)
		io.ReadFull(conn, domain)
		host = string(domain)
	case AddrTypeIPv6:
		ip := make([]byte, 16)
		io.ReadFull(conn, ip)
		host = net.IP(ip).String()
	default:
		return "", fmt.Errorf("vless: unknown address type %d", addrType)
	}

	portBuf := make([]byte, 2)
	io.ReadFull(conn, portBuf)
	port := binary.BigEndian.Uint16(portBuf)

	return net.JoinHostPort(host, fmt.Sprintf("%d", port)), nil
}

type VLESSOutboundHandler struct {
	tag            string
	address        string
	port           uint16
	uuid           string
	flow           string
	streamSettings *transport.StreamSettings
	timeout        time.Duration
}

func NewVLESSOutbound(tag, address string, port uint16, uuid, flow string, ss *transport.StreamSettings) *VLESSOutboundHandler {
	if flow == "" {
		flow = FlowNone
	}
	return &VLESSOutboundHandler{
		tag:            tag,
		address:        address,
		port:           port,
		uuid:           uuid,
		flow:           flow,
		streamSettings: ss,
		timeout:        30 * time.Second,
	}
}

func (h *VLESSOutboundHandler) Tag() string { return h.tag }

func (h *VLESSOutboundHandler) Process(ctx context.Context, link *VLESSLink) error {
	var conn net.Conn
	var err error

	if h.streamSettings != nil {
		switch h.streamSettings.Network {
		case transport.TransportWebSocket:
			conn, err = dialWebSocket(ctx, h.streamSettings, h.address, int(h.port))
		case transport.TransportQUIC:
			conn, err = dialQUIC(ctx, h.streamSettings, h.address, int(h.port))
		case transport.TransportGRPC:
			conn, err = dialGRPC(ctx, h.streamSettings, h.address, int(h.port))
		default:
			conn, err = net.DialTimeout("tcp", fmt.Sprintf("%s:%d", h.address, h.port), h.timeout)
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

	request := h.buildRequest(link.Target, link.IsUDP)
	if _, err := conn.Write(request); err != nil {
		return err
	}

	resp := make([]byte, 2)
	io.ReadFull(conn, resp)

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

func (h *VLESSOutboundHandler) buildRequest(target string, isUDP bool) []byte {
	uuidBytes := parseUUID(h.uuid)

	cmd := byte(CmdTCP)
	if isUDP {
		cmd = CmdUDP
	}

	host, portStr, _ := net.SplitHostPort(target)
	portVal := uint16(0)
	fmt.Sscanf(portStr, "%d", &portVal)

	var addrHeader []byte
	ip := net.ParseIP(host)
	if ip != nil {
		if ip.To4() != nil {
			addrHeader = append([]byte{AddrTypeIPv4}, ip.To4()...)
		} else {
			addrHeader = append([]byte{AddrTypeIPv6}, ip.To16()...)
		}
	} else {
		addrHeader = append([]byte{AddrTypeDomain, byte(len(host))}, []byte(host)...)
	}

	portBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(portBytes, portVal)
	addrHeader = append(addrHeader, portBytes...)

	var buf []byte
	buf = append(buf, Version)
	buf = append(buf, uuidBytes...)
	buf = append(buf, 0)
	buf = append(buf, addrHeader...)
	buf = append(buf, cmd)

	return buf
}

func parseUUID(uuid string) []byte {
	clean := ""
	for _, c := range uuid {
		if c != '-' {
			clean += string(c)
		}
	}
	bytes := make([]byte, 16)
	for i := 0; i < 32; i += 2 {
		var b byte
		fmt.Sscanf(clean[i:i+2], "%02x", &b)
		bytes[i/2] = b
	}
	return bytes
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
	return transport.NewWebSocketConn(tcpConn, addr, path, ss)
}

func dialQUIC(ctx context.Context, ss *transport.StreamSettings, addr string, port int) (net.Conn, error) {
	return transport.NewQUICConn(ctx, addr, port, ss)
}

func dialGRPC(ctx context.Context, ss *transport.StreamSettings, addr string, port int) (net.Conn, error) {
	return transport.NewGRPCConn(ctx, addr, port, ss)
}

type VLESSLink struct {
	Reader io.Reader
	Writer io.Writer
	Target string
	IsUDP  bool
}
