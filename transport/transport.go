package transport

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"time"

	"github.com/Hhz0823/oiwest-core/transport/dccp"
)

var (
	ErrUnsupportedTransport = errors.New("unsupported transport protocol")
	ErrListenerClosed       = errors.New("listener closed")
)

type DCCPTransport struct {
	settings     *StreamSettings
	connState    *dccp.ConnState
	handshake    *dccp.HandshakeHandler
	congestion   dccp.CongestionControl
	conn         net.Conn
	rxBuf        []byte
	rxOffset     int
	rxLen        int
	readDeadline time.Time
	writeDeadline time.Time
	mu           sync.Mutex
	closed       bool
	closeCh      chan struct{}
	ctx          context.Context
	cancel       context.CancelFunc
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
	t.connState = dccp.NewConnState(false)
	t.congestion = dccp.GetCongestionControl(t.settings.DCCPSettings.CCID)
	handshakeConfig := &dccp.HandshakeConfig{
		ServiceCode: t.getServiceCode(),
		CCID:        t.settings.DCCPSettings.CCID,
		MaxRetries:  t.settings.DCCPSettings.MaxRetries,
		Timeout:     t.settings.DCCPSettings.HandshakeTimeout,
	}
	t.handshake = dccp.NewHandshakeHandler(handshakeConfig, t.connState)
}

func (t *DCCPTransport) getServiceCode() dccp.ServiceCode {
	var sc dccp.ServiceCode
	code := t.settings.DCCPSettings.ServiceCode
	if len(code) >= 4 {
		copy(sc[:], code[:4])
	} else {
		copy(sc[:], "V2RY")
	}
	return sc
}

func (t *DCCPTransport) Dial(ctx context.Context, addr net.Addr) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	dialer := net.Dialer{Timeout: 15 * time.Second}
	conn, err := dialer.DialContext(ctx, addr.Network(), addr.String())
	if err != nil {
		return err
	}

	t.conn = conn
	t.connState = dccp.NewConnState(false)
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

func (t *DCCPTransport) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return 0, ErrTransportClosed
	}

	srcPort := t.connState.LocalPort
	dstPort := t.connState.RemotePort
	if srcPort == 0 {
		srcPort = 1
	}
	if dstPort == 0 {
		dstPort = 1
	}

	var written int
	offset := 0
	maxPayload := t.settings.DCCPSettings.MaxPacketSize - dccp.HeaderSize - 12

	for offset < len(p) {
		chunkSize := len(p) - offset
		if chunkSize > maxPayload {
			chunkSize = maxPayload
		}

		chunk := p[offset : offset+chunkSize]

		t.connState.LocalSeq++
		packet, err := dccp.BuildDataPacket(srcPort, dstPort, t.connState.LocalSeq, chunk)
		if err != nil {
			return written, err
		}

		if t.writeDeadline.IsZero() == false {
			t.conn.SetWriteDeadline(t.writeDeadline)
		}

		n, err := t.conn.Write(packet)
		if err != nil {
			return written, err
		}
		written += chunkSize
		offset += chunkSize

		t.congestion.OnPacketSent(t.connState.LocalSeq, n)
	}

	return written, nil
}

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
	if t.rxLen == 0 {
		t.rxOffset = 0
	}

	return n, nil
}

func (t *DCCPTransport) readNextPacket() error {
	for {
		if t.readDeadline.IsZero() == false {
			t.conn.SetReadDeadline(t.readDeadline)
		}

		hdr, _, payload, err := dccp.ReadFullDCCPPacket(t.conn)
		if err != nil {
			if err == io.EOF {
				t.markClosed()
			}
			return err
		}

		switch hdr.Type {
		case dccp.PktDCCPData, dccp.PktDCCPDataAck:
			t.rxOffset = 0
			t.rxLen = copy(t.rxBuf, payload)

			if hdr.Type == dccp.PktDCCPDataAck {
				go t.sendAck(hdr.SequenceNumber)
			} else {
				t.connState.RemoteAck = hdr.SequenceNumber + 1
			}

			return nil

		case dccp.PktDCCPAck:
			t.connState.LocalAck = hdr.SequenceNumber

		case dccp.PktDCCPClose, dccp.PktDCCPCloseReq:
			t.sendClose()
			t.markClosed()
			return io.EOF

		case dccp.PktDCCPReset:
			t.markClosed()
			return errors.New("connection reset")

		case dccp.PktDCCPSync, dccp.PktDCCPSyncAck:
		}
	}
}

func (t *DCCPTransport) sendAck(seq uint64) {
	srcPort := t.connState.LocalPort
	dstPort := t.connState.RemotePort
	if srcPort == 0 {
		srcPort = 1
	}
	if dstPort == 0 {
		dstPort = 1
	}
	packet, err := dccp.BuildAckPacket(srcPort, dstPort, seq+1)
	if err != nil {
		return
	}
	t.conn.Write(packet)
}

func (t *DCCPTransport) sendClose() {
	srcPort := t.connState.LocalPort
	dstPort := t.connState.RemotePort
	packet, _ := dccp.BuildClosePacket(srcPort, dstPort, t.connState.LocalSeq)
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

	if t.closed {
		return nil
	}

	t.sendClose()
	t.markClosed()
	t.cancel()

	if t.conn != nil {
		return t.conn.Close()
	}
	return nil
}

func (t *DCCPTransport) LocalAddr() net.Addr {
	if t.conn != nil {
		return t.conn.LocalAddr()
	}
	return nil
}

func (t *DCCPTransport) RemoteAddr() net.Addr {
	if t.conn != nil {
		return t.conn.RemoteAddr()
	}
	return nil
}

func (t *DCCPTransport) SetDeadline(d time.Time) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.readDeadline = d
	t.writeDeadline = d
	if t.conn != nil {
		return t.conn.SetDeadline(d)
	}
	return nil
}

func (t *DCCPTransport) SetReadDeadline(d time.Time) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.readDeadline = d
	return nil
}

func (t *DCCPTransport) SetWriteDeadline(d time.Time) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.writeDeadline = d
	return nil
}

func (t *DCCPTransport) IsClosed() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.closed
}

func (t *DCCPTransport) GetCongestionControl() dccp.CongestionControl {
	return t.congestion
}

func (t *DCCPTransport) GetConnState() *dccp.ConnState {
	return t.connState
}
