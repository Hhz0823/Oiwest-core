package dccp

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"sync"
	"time"

	"github.com/Hhz0823/oiwest-core/app/transport/dccp/disguise"
)

var ErrTransportClosed = errors.New("dccp: transport closed")

// DCCPTransport implements the core DCCP transport with support for
// multiple wrapping methods and obfuscation disguises.
type DCCPTransport struct {
	settings   *StreamSettings
	connState  *ConnState
	handshake  *HandshakeHandler
	congestion CongestionControl
	disguise   disguise.Disguise
	conn       net.Conn
	rxBuf      []byte
	rxOffset   int
	rxLen      int
	readDeadline  time.Time
	writeDeadline time.Time
	mu         sync.Mutex
	closed     bool
	closeCh    chan struct{}
	ctx        context.Context
	cancel     context.CancelFunc
}

// StreamSettings is an alias for the transport-level settings used by DCCP.
type StreamSettings struct {
	Network        string
	Security       string
	DCCPSettings   *DCCPTransportSettings
	TLSSettings    *TLSSettings
}

type TLSSettings struct {
	ServerName    string
	AllowInsecure bool
	ALPN          []string
	Fingerprint   string
}

type DCCPTransportSettings struct {
	CCID             int
	ServiceCode      string
	MaxPacketSize    int
	HandshakeTimeout time.Duration
	MaxRetries       int
	Obfuscation      string
	Disguise         string
	DisguiseSettings *disguise.Config
}

func NewDCCPTransport(settings *StreamSettings) *DCCPTransport {
	ctx, cancel := context.WithCancel(context.Background())
	t := &DCCPTransport{
		settings: settings,
		rxBuf:    make([]byte, 64*1024),
		closeCh:  make(chan struct{}),
		ctx:      ctx,
		cancel:   cancel,
	}
	t.initDCCP()
	return t
}

func (t *DCCPTransport) initDCCP() {
	ds := t.settings.DCCPSettings
	if ds == nil {
		ds = &DCCPTransportSettings{CCID: 4, ServiceCode: "OIWE", MaxPacketSize: 1500}
	}

	t.connState = NewConnState(false)
	t.congestion = GetCongestionControl(CCID(ds.CCID))

	handshakeConfig := &HandshakeConfig{
		ServiceCode: t.getServiceCode(),
		CCID:        CCID(ds.CCID),
		MaxRetries:  ds.MaxRetries,
		Timeout:     ds.HandshakeTimeout,
	}
	t.handshake = NewHandshakeHandler(handshakeConfig, t.connState)

	// Initialize disguise if configured
	if ds.Disguise != "" && ds.DisguiseSettings != nil {
		t.disguise = disguise.NewDisguise(disguise.Method(ds.Disguise), ds.DisguiseSettings)
	}
}

func (t *DCCPTransport) getServiceCode() ServiceCode {
	var sc ServiceCode
	code := t.settings.DCCPSettings.ServiceCode
	if len(code) >= 4 {
		copy(sc[:], code[:4])
	} else {
		copy(sc[:], "OIWE")
	}
	return sc
}

// Dial connects to a remote address using DCCP.
func (t *DCCPTransport) Dial(ctx context.Context, addr net.Addr) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Apply disguise wrapping if configured
	var conn net.Conn
	var err error

	if t.disguise != nil {
		conn, err = t.disguise.Dial(ctx, addr)
		if err != nil {
			return err
		}
	} else {
		dialer := net.Dialer{Timeout: 15 * time.Second}
		conn, err = dialer.DialContext(ctx, "tcp", addr.String())
		if err != nil {
			return err
		}
	}

	t.conn = conn
	t.connState = NewConnState(false)
	t.initDCCP()

	tcpAddr := conn.RemoteAddr().(*net.TCPAddr)
	srcPort := uint16(tcpAddr.Port)
	dstPort := uint16(tcpAddr.Port)
	if a, ok := addr.(*net.TCPAddr); ok {
		dstPort = uint16(a.Port)
	}

	if err := t.handshake.ClientHandshake(conn, srcPort, dstPort); err != nil {
		conn.Close()
		return err
	}

	return nil
}

// DialWithTLS connects using DCCP over TLS.
func (t *DCCPTransport) DialWithTLS(ctx context.Context, addr net.Addr, tlsCfg *tls.Config) error {
	dialer := net.Dialer{Timeout: 15 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", addr.String())
	if err != nil {
		return err
	}

	tlsConn := tls.Client(conn, tlsCfg)
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		conn.Close()
		return err
	}

	t.conn = tlsConn
	t.connState = NewConnState(false)
	t.initDCCP()

	tcpAddr := conn.RemoteAddr().(*net.TCPAddr)
	srcPort := uint16(tcpAddr.Port)
	dstPort := uint16(tcpAddr.Port)
	if a, ok := addr.(*net.TCPAddr); ok {
		dstPort = uint16(a.Port)
	}

	if err := t.handshake.ClientHandshake(tlsConn, srcPort, dstPort); err != nil {
		tlsConn.Close()
		return err
	}

	return nil
}

// Write sends data over DCCP.
func (t *DCCPTransport) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return 0, ErrTransportClosed
	}

	srcPort := t.connState.LocalPort
	dstPort := t.connState.RemotePort
	if srcPort == 0 { srcPort = 1 }
	if dstPort == 0 { dstPort = 1 }

	var written int
	offset := 0
	maxPayload := t.settings.DCCPSettings.MaxPacketSize - HeaderSize - 12
	if maxPayload <= 0 { maxPayload = 1400 }

	for offset < len(p) {
		chunkSize := len(p) - offset
		if chunkSize > maxPayload { chunkSize = maxPayload }
		chunk := p[offset : offset+chunkSize]

		t.connState.LocalSeq++
		packet, err := BuildDataPacket(srcPort, dstPort, t.connState.LocalSeq, chunk)
		if err != nil { return written, err }

		if !t.writeDeadline.IsZero() {
			t.conn.SetWriteDeadline(t.writeDeadline)
		}

		n, err := t.conn.Write(packet)
		if err != nil { return written, err }
		written += chunkSize
		offset += chunkSize
		t.congestion.OnPacketSent(t.connState.LocalSeq, n)
	}

	return written, nil
}

// Read receives data from DCCP.
func (t *DCCPTransport) Read(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return 0, ErrTransportClosed
	}

	if t.rxLen == 0 {
		if err := t.readNextPacket(); err != nil {
			return 0, err
		}
	}

	n := copy(p, t.rxBuf[t.rxOffset:t.rxOffset+t.rxLen])
	t.rxOffset += n
	t.rxLen -= n
	if t.rxLen == 0 { t.rxOffset = 0 }
	return n, nil
}

func (t *DCCPTransport) readNextPacket() error {
	for {
		if !t.readDeadline.IsZero() {
			t.conn.SetReadDeadline(t.readDeadline)
		}

		hdr, _, payload, err := ReadFullDCCPPacket(t.conn)
		if err != nil {
			if err == io.EOF { t.markClosed() }
			return err
		}

		switch hdr.Type {
		case PktDCCPData, PktDCCPDataAck:
			t.rxOffset = 0
			t.rxLen = copy(t.rxBuf, payload)
			if hdr.Type == PktDCCPDataAck {
				go t.sendAck(hdr.SequenceNumber)
			} else {
				t.connState.RemoteAck = hdr.SequenceNumber + 1
			}
			return nil
		case PktDCCPAck:
			t.connState.LocalAck = hdr.SequenceNumber
		case PktDCCPClose, PktDCCPCloseReq:
			t.sendClose()
			t.markClosed()
			return io.EOF
		case PktDCCPReset:
			t.markClosed()
			return errors.New("connection reset")
		case PktDCCPSync, PktDCCPSyncAck:
		}
	}
}

func (t *DCCPTransport) sendAck(seq uint64) {
	srcPort := t.connState.LocalPort
	dstPort := t.connState.RemotePort
	if srcPort == 0 { srcPort = 1 }
	if dstPort == 0 { dstPort = 1 }
	packet, _ := BuildAckPacket(srcPort, dstPort, seq+1)
	t.conn.Write(packet)
}

func (t *DCCPTransport) sendClose() {
	packet, _ := BuildClosePacket(t.connState.LocalPort, t.connState.RemotePort, t.connState.LocalSeq)
	t.conn.Write(packet)
}

func (t *DCCPTransport) markClosed() {
	t.closed = true
	select {
	case <-t.closeCh:
	default:
		close(t.closeCh)
	}
}

func (t *DCCPTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed { return nil }
	t.sendClose()
	t.markClosed()
	t.cancel()
	if t.conn != nil { return t.conn.Close() }
	return nil
}

func (t *DCCPTransport) LocalAddr() net.Addr  { if t.conn != nil { return t.conn.LocalAddr() }; return nil }
func (t *DCCPTransport) RemoteAddr() net.Addr { if t.conn != nil { return t.conn.RemoteAddr() }; return nil }

func (t *DCCPTransport) SetDeadline(d time.Time) error {
	t.mu.Lock(); defer t.mu.Unlock()
	t.readDeadline = d; t.writeDeadline = d
	if t.conn != nil { return t.conn.SetDeadline(d) }; return nil
}
func (t *DCCPTransport) SetReadDeadline(d time.Time) error {
	t.mu.Lock(); defer t.mu.Unlock()
	t.readDeadline = d; return nil
}
func (t *DCCPTransport) SetWriteDeadline(d time.Time) error {
	t.mu.Lock(); defer t.mu.Unlock()
	t.writeDeadline = d; return nil
}
func (t *DCCPTransport) IsClosed() bool { t.mu.Lock(); defer t.mu.Unlock(); return t.closed }
func (t *DCCPTransport) GetCongestionControl() CongestionControl { return t.congestion }
func (t *DCCPTransport) GetConnState() *ConnState { return t.connState }
