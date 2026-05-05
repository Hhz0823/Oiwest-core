package transport

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrKCPClosed = errors.New("mKCP: session closed")
	ErrKCPTimeout = errors.New("mKCP: connection timeout")
)

type KCPConfig struct {
	MTU            int           `json:"mtu"`
	TTI            time.Duration `json:"tti"`
	HeaderType     string        `json:"headerType"`
	ReadBufferSize int           `json:"readBufferSize"`
	WriteBufferSize int          `json:"writeBufferSize"`
	Congestion     bool          `json:"congestion"`
	DownlinkCapacity int         `json:"downlinkCapacity"`
	UplinkCapacity   int         `json:"uplinkCapacity"`
}

func DefaultKCPConfig() *KCPConfig {
	return &KCPConfig{
		MTU:             1350,
		TTI:             20 * time.Millisecond,
		HeaderType:      "none",
		ReadBufferSize:  4 * 1024 * 1024,
		WriteBufferSize: 4 * 1024 * 1024,
		Congestion:      false,
		DownlinkCapacity: 100,
		UplinkCapacity:   100,
	}
}

type KCPHeaderType int

const (
	KCPHeaderNone     KCPHeaderType = 0
	KCPHeaderMux      KCPHeaderType = 1
	KCPHeaderAuth     KCPHeaderType = 2
	KCPHeaderProto    KCPHeaderType = 3
)

type KCPSegment struct {
	Conv     uint32
	Cmd      byte
	Frgl     byte
	Wnd      uint16
	Ts       uint32
	Sn       uint32
	Una      uint32
	Len      uint32
	Data     []byte
	xmit     uint32
}

type KCPState struct {
	conv      uint32
	mtu       int
	mss       int
	state     int

	sndUna    uint32
	sndNxt    uint32
	rcvNxt    uint32

	sndWnd    uint16
	rcvWnd    uint16
	rmtWnd    uint16
	probe     int
	current   uint32
	interval  uint32
	tsFlush   int64
	xmit      uint32

	nodelay   int
	updated   bool
	tsProbe   uint64
	probeWait int64
	deadLink  uint32
	incr      uint32

	sndBuf    []KCPSegment
	rcvBuf    []KCPSegment
	sndQueue  []KCPSegment
	rcvQueue  []KCPSegment

	acklist   []ackItem

	buf       []byte
	reserved  int
}

type ackItem struct {
	sn uint32
	ts uint32
}

const (
	IKCP_RTO_NDL     = 30
	IKCP_RTO_MIN     = 100
	IKCP_RTO_DEF     = 200
	IKCP_RTO_MAX     = 60000
	IKCP_CMD_PUSH    = 81
	IKCP_CMD_ACK     = 82
	IKCP_CMD_WASK    = 83
	IKCP_CMD_WINS    = 84
	IKCP_ASK_SEND    = 1
	IKCP_ASK_TELL    = 2
	IKCP_WND_SND     = 32
	IKCP_WND_RCV     = 128
	IKCP_MTU_DEF     = 1350
	IKCP_ACK_FAST    = 3
	IKCP_INTERVAL    = 100
	IKCP_OVERHEAD    = 24
	IKCP_DEADLINK    = 20
	IKCP_THRESH_INIT = 2
	IKCP_THRESH_MIN  = 2
	IKCP_PROBE_INIT  = 7000
	IKCP_PROBE_LIMIT = 120000
)

func NewKCPState(conv uint32) *KCPState {
	kcp := &KCPState{
		conv:     conv,
		mtu:      IKCP_MTU_DEF,
		mss:      IKCP_MTU_DEF - IKCP_OVERHEAD,
		sndWnd:   IKCP_WND_SND,
		rcvWnd:   IKCP_WND_RCV,
		rmtWnd:   IKCP_WND_RCV,
		interval: IKCP_INTERVAL,
		deadLink: IKCP_DEADLINK,
		buf:      make([]byte, IKCP_MTU_DEF),
	}
	return kcp
}

func (k *KCPState) Recv(buffer []byte) (int, error) {
	if len(k.rcvQueue) == 0 {
		return 0, errors.New("mKCP: no data")
	}

	n := 0
	for _, seg := range k.rcvQueue {
		if n+len(seg.Data) > len(buffer) {
			break
		}
		n += copy(buffer[n:], seg.Data)
	}

	k.rcvQueue = k.rcvQueue[:0]
	return n, nil
}

func (k *KCPState) Send(buffer []byte) int {
	count := (len(buffer) + k.mss - 1) / k.mss
	if count > 0 {
		for i := 0; i < count; i++ {
			start := i * k.mss
			end := start + k.mss
			if end > len(buffer) {
				end = len(buffer)
			}
			seg := KCPSegment{
				Data: append([]byte{}, buffer[start:end]...),
				Len:  uint32(end - start),
			}
			k.sndQueue = append(k.sndQueue, seg)
		}
	}
	return len(buffer)
}

func (k *KCPState) Update(current uint32) {
	k.current = current
	if !k.updated {
		k.updated = true
		k.tsFlush = int64(current)
	}

	slap := int64(current) - k.tsFlush
	if slap >= 10000 || slap < -10000 {
		k.tsFlush = int64(current)
		slap = 0
	}
	if slap < 0 {
		slap = 0
	}
	if slap >= int64(k.interval) {
		k.tsFlush = int64(current) - (int64(k.interval) - int64(slap)%int64(k.interval))
		k.flush()
	}
}

func (k *KCPState) flush() {
	current := k.current
	buffer := k.buf

	if k.rmtWnd == 0 {
		if k.probeWait == 0 {
			k.probeWait = IKCP_PROBE_INIT
			k.tsProbe = uint64(current) + uint64(k.probeWait)
		} else {
			k.tsProbe = uint64(current) + uint64(k.probeWait)
		}
	} else {
		k.probeWait = 0
		k.tsProbe = 0
	}

	if k.tsProbe != 0 && uint64(current) >= k.tsProbe {
		k.probe = IKCP_ASK_TELL
		k.tsProbe = 0
		k.probeWait += IKCP_PROBE_INIT
		if k.probeWait > IKCP_PROBE_LIMIT {
			k.probeWait = IKCP_PROBE_LIMIT
		}
	}

	for i := 0; i < len(k.sndQueue); i++ {
		seg := &k.sndQueue[i]
		seg.Conv = k.conv
		seg.Sn = k.sndNxt
		seg.Ts = current
		seg.Cmd = IKCP_CMD_PUSH
		seg.Wnd = k.rcvWnd
		seg.Una = k.rcvNxt
		k.sndNxt++
		k.sndBuf = append(k.sndBuf, k.sndQueue[i])
	}
	k.sndQueue = k.sndQueue[:0]

	for i := 0; i < len(k.sndBuf); i++ {
		seg := &k.sndBuf[i]
		needsend := false
		if seg.xmit == 0 {
			needsend = true
			seg.xmit++
			seg.Ts = current
			seg.Wnd = k.rmtWnd
			seg.Una = k.rcvNxt
		}
		if seg.xmit >= k.deadLink {
			k.state = -1
		}
		if needsend {
			seg.Len = uint32(len(seg.Data))
			offset := k.reserved
			k.encodeSegment(buffer[offset:], seg)
		}
	}

	for i := range k.acklist {
		item := &k.acklist[i]
		seg := KCPSegment{
			Conv: k.conv,
			Cmd:  IKCP_CMD_ACK,
			Sn:   item.sn,
			Ts:   item.ts,
			Wnd:  k.rcvWnd,
			Una:  k.rcvNxt,
		}
		offset := k.reserved
		k.encodeSegment(buffer[offset:], &seg)
	}
	k.acklist = k.acklist[:0]

	if k.rmtWnd == 0 {
		if k.probe&IKCP_ASK_SEND != 0 {
			seg := KCPSegment{
				Conv: k.conv,
				Cmd:  IKCP_CMD_WASK,
			}
			offset := k.reserved
			k.encodeSegment(buffer[offset:], &seg)
		}
		if k.probe&IKCP_ASK_TELL != 0 {
			seg := KCPSegment{
				Conv: k.conv,
				Cmd:  IKCP_CMD_WINS,
				Wnd:  k.rcvWnd,
			}
			offset := k.reserved
			k.encodeSegment(buffer[offset:], &seg)
		}
	}
	k.probe = 0
}

func (k *KCPState) encodeSegment(buf []byte, seg *KCPSegment) int {
	offset := 0
	binary.LittleEndian.PutUint32(buf[offset:], seg.Conv); offset += 4
	buf[offset] = seg.Cmd; offset++
	buf[offset] = seg.Frgl; offset++
	binary.LittleEndian.PutUint16(buf[offset:], seg.Wnd); offset += 2
	binary.LittleEndian.PutUint32(buf[offset:], seg.Ts); offset += 4
	binary.LittleEndian.PutUint32(buf[offset:], seg.Sn); offset += 4
	binary.LittleEndian.PutUint32(buf[offset:], seg.Una); offset += 4
	binary.LittleEndian.PutUint32(buf[offset:], seg.Len); offset += 4
	if len(seg.Data) > 0 {
		copy(buf[offset:], seg.Data)
		offset += len(seg.Data)
	}
	return offset
}

func (k *KCPState) Input(data []byte) int {
	if len(data) < IKCP_OVERHEAD {
		return -1
	}
	offset := 0
	for offset <= len(data)-IKCP_OVERHEAD {
		seg := KCPSegment{}
		seg.Conv = binary.LittleEndian.Uint32(data[offset:]); offset += 4
		seg.Cmd = data[offset]; offset++
		seg.Frgl = data[offset]; offset++
		seg.Wnd = binary.LittleEndian.Uint16(data[offset:]); offset += 2
		seg.Ts = binary.LittleEndian.Uint32(data[offset:]); offset += 4
		seg.Sn = binary.LittleEndian.Uint32(data[offset:]); offset += 4
		seg.Una = binary.LittleEndian.Uint32(data[offset:]); offset += 4
		seg.Len = binary.LittleEndian.Uint32(data[offset:]); offset += 4
		if seg.Len > 0 {
			seg.Data = make([]byte, seg.Len)
			copy(seg.Data, data[offset:offset+int(seg.Len)])
			offset += int(seg.Len)
		}
		if seg.Cmd == IKCP_CMD_PUSH {
			k.rcvBuf = append(k.rcvBuf, seg)
			k.rcvQueue = append(k.rcvQueue, seg)
		} else if seg.Cmd == IKCP_CMD_ACK {
		} else if seg.Cmd == IKCP_CMD_WASK {
		} else if seg.Cmd == IKCP_CMD_WINS {
			k.rmtWnd = seg.Wnd
		}
	}
	return 0
}

type KCPConn struct {
	state      *KCPState
	conn       net.PacketConn
	remote     net.Addr
	config     *KCPConfig
	readBuf    []byte
	readCh     chan []byte
	done       chan struct{}
	mu         sync.Mutex
	closed     int32
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
}

func NewKCPConn(conn net.PacketConn, remote net.Addr, config *KCPConfig) *KCPConn {
	ctx, cancel := context.WithCancel(context.Background())
	conv := uint32(0)
	binary.Read(rand.Reader, binary.LittleEndian, &conv)
	kcp := &KCPConn{
		state:   NewKCPState(conv),
		conn:    conn,
		remote:  remote,
		config:  config,
		readBuf: make([]byte, config.MTU),
		readCh:  make(chan []byte, 1024),
		done:    make(chan struct{}),
		ctx:     ctx,
		cancel:  cancel,
	}
	kcp.wg.Add(2)
	go kcp.readLoop()
	go kcp.updateLoop()
	return kcp
}

func (k *KCPConn) readLoop() {
	defer k.wg.Done()
	buf := make([]byte, k.config.MTU)
	for {
		select {
		case <-k.done:
			return
		default:
		}
		n, _, err := k.conn.ReadFrom(buf)
		if err != nil {
			return
		}
		k.state.Input(buf[:n])
	}
}

func (k *KCPConn) updateLoop() {
	defer k.wg.Done()
	ticker := time.NewTicker(k.config.TTI)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			k.state.Update(uint32(time.Now().UnixMilli()))
		case <-k.done:
			return
		}
	}
}

func (k *KCPConn) Read(p []byte) (int, error) {
	if atomic.LoadInt32(&k.closed) != 0 {
		return 0, ErrKCPClosed
	}
	return k.state.Recv(p)
}

func (k *KCPConn) Write(p []byte) (int, error) {
	if atomic.LoadInt32(&k.closed) != 0 {
		return 0, ErrKCPClosed
	}
	k.state.Send(p)
	return len(p), nil
}

func (k *KCPConn) Close() error {
	if atomic.CompareAndSwapInt32(&k.closed, 0, 1) {
		close(k.done)
		k.cancel()
		k.wg.Wait()
	}
	return nil
}

func (k *KCPConn) LocalAddr() net.Addr  { return k.conn.LocalAddr() }
func (k *KCPConn) RemoteAddr() net.Addr { return k.remote }

func (k *KCPConn) SetDeadline(t time.Time) error {
	return nil
}

func (k *KCPConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (k *KCPConn) SetWriteDeadline(t time.Time) error {
	return nil
}

type KCPTransport struct {
	config *KCPConfig
	conn   net.PacketConn
}

func NewKCPTransport(config *KCPConfig) *KCPTransport {
	if config == nil {
		config = DefaultKCPConfig()
	}
	return &KCPTransport{config: config}
}

func (t *KCPTransport) Dial(ctx context.Context, addr string) (net.Conn, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("mKCP: resolve addr: %w", err)
	}
	udpConn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return nil, fmt.Errorf("mKCP: dial udp: %w", err)
	}
	return NewKCPConn(udpConn, udpAddr, t.config), nil
}

func (t *KCPTransport) Listen(addr string) (net.Listener, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, err
	}
	return &KCPListener{
		transport: t,
		conn:      conn,
		acceptCh:  make(chan *KCPConn, 16),
		done:      make(chan struct{}),
	}, nil
}

type KCPListener struct {
	transport *KCPTransport
	conn      *net.UDPConn
	acceptCh  chan *KCPConn
	done      chan struct{}
	mu        sync.Mutex
}

func (l *KCPListener) Accept() (net.Conn, error) {
	select {
	case conn := <-l.acceptCh:
		return conn, nil
	case <-l.done:
		return nil, ErrKCPClosed
	}
}

func (l *KCPListener) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	select {
	case <-l.done:
	default:
		close(l.done)
	}
	return l.conn.Close()
}

func (l *KCPListener) Addr() net.Addr {
	return l.conn.LocalAddr()
}
