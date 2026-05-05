package transport

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrXHTTPClosed = errors.New("XHTTP: connection closed")
)

type XHTTPConfig struct {
	Path            string            `json:"path"`
	Host            string            `json:"host"`
	Method          string            `json:"method"`
	Headers         map[string]string `json:"headers"`
	MaxConcurrency  int               `json:"maxConcurrency"`
	MaxStreams      int               `json:"maxStreams"`
	IdleTimeout     time.Duration     `json:"idleTimeout"`
	KeepAlivePeriod time.Duration     `json:"keepAlivePeriod"`
	ReadBufferSize  int               `json:"readBufferSize"`
	WriteBufferSize int               `json:"writeBufferSize"`
}

func DefaultXHTTPConfig() *XHTTPConfig {
	return &XHTTPConfig{
		Path:            "/xhttp",
		Host:            "",
		Method:          "POST",
		MaxConcurrency:  256,
		MaxStreams:      128,
		IdleTimeout:     60 * time.Second,
		KeepAlivePeriod: 30 * time.Second,
		ReadBufferSize:  4 * 1024,
		WriteBufferSize: 4 * 1024,
	}
}

type XHTTPFrameType byte

const (
	XHTTPFrameData XHTTPFrameType = 0x01
	XHTTPFramePing XHTTPFrameType = 0x02
	XHTTPFramePong XHTTPFrameType = 0x03
	XHTTPFrameClose XHTTPFrameType = 0x04
	XHTTPFrameReset XHTTPFrameType = 0x05
	XHTTPFrameGoAway XHTTPFrameType = 0x06
)

type XHTTPFrameHeader struct {
	Type    XHTTPFrameType
	StreamID uint32
	Length  uint32
	Flags   byte
}

type XHTTPStream struct {
	id       uint32
	conn     *XHTTPConnection
	readBuf  []byte
	readPos  int
	readLen  int
	mu       sync.Mutex
	cond     *sync.Cond
	closed   int32
	writeCh  chan []byte
}

type XHTTPConnection struct {
	conn       net.Conn
	config     *XHTTPConfig
	streams    map[uint32]*XHTTPStream
	nextID     uint32
	mu         sync.RWMutex
	closed     int32
	readBuf    []byte
	reader     *bufio.Reader
	writer     *bufio.Writer
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	acceptCh   chan *XHTTPStream
}

func NewXHTTPConnection(conn net.Conn, config *XHTTPConfig) *XHTTPConnection {
	ctx, cancel := context.WithCancel(context.Background())
	xc := &XHTTPConnection{
		conn:     conn,
		config:   config,
		streams:  make(map[uint32]*XHTTPStream),
		nextID:   1,
		readBuf:  make([]byte, 64*1024),
		reader:   bufio.NewReaderSize(conn, config.ReadBufferSize),
		writer:   bufio.NewWriterSize(conn, config.WriteBufferSize),
		ctx:      ctx,
		cancel:   cancel,
		acceptCh: make(chan *XHTTPStream, 16),
	}
	xc.wg.Add(1)
	go xc.readLoop()
	return xc
}

func (xc *XHTTPConnection) OpenStream() (*XHTTPStream, error) {
	if atomic.LoadInt32(&xc.closed) != 0 {
		return nil, ErrXHTTPClosed
	}
	xc.mu.Lock()
	id := xc.nextID
	xc.nextID += 2
	stream := &XHTTPStream{
		id:      id,
		conn:    xc,
		writeCh: make(chan []byte, 64),
	}
	stream.cond = sync.NewCond(&stream.mu)
	xc.streams[id] = stream
	xc.mu.Unlock()
	return stream, nil
}

func (xc *XHTTPConnection) AcceptStream() (*XHTTPStream, error) {
	select {
	case stream := <-xc.acceptCh:
		return stream, nil
	case <-xc.ctx.Done():
		return nil, ErrXHTTPClosed
	}
}

func (xc *XHTTPConnection) readLoop() {
	defer xc.wg.Done()
	defer xc.Close()

	headerBuf := make([]byte, 12)
	for {
		if atomic.LoadInt32(&xc.closed) != 0 {
			return
		}
		n, err := io.ReadFull(xc.reader, headerBuf)
		if err != nil || n < 12 {
			return
		}
		frameType := XHTTPFrameType(headerBuf[0])
		streamID := binary.BigEndian.Uint32(headerBuf[1:5])
		length := binary.BigEndian.Uint32(headerBuf[5:9])
		_ = headerBuf[9]

		payload := make([]byte, length)
		if length > 0 {
			if _, err := io.ReadFull(xc.reader, payload); err != nil {
				return
			}
		}
		switch frameType {
		case XHTTPFrameData:
			xc.handleData(streamID, payload)
		case XHTTPFramePing:
			xc.writeFrame(0, XHTTPFramePong, 0, payload)
		case XHTTPFramePong:
		case XHTTPFrameClose:
			xc.handleClose(streamID)
		case XHTTPFrameReset:
			xc.handleReset(streamID)
		case XHTTPFrameGoAway:
			return
		}
	}
}

func (xc *XHTTPConnection) handleData(streamID uint32, data []byte) {
	xc.mu.RLock()
	stream, ok := xc.streams[streamID]
	xc.mu.RUnlock()
	if !ok {
		if streamID%2 == 0 {
			stream = &XHTTPStream{
				id:      streamID,
				conn:    xc,
				writeCh: make(chan []byte, 64),
			}
			stream.cond = sync.NewCond(&stream.mu)
			xc.mu.Lock()
			xc.streams[streamID] = stream
			xc.mu.Unlock()
			select {
			case xc.acceptCh <- stream:
			default:
			}
			stream.mu.Lock()
			stream.readBuf = append(stream.readBuf, data...)
			stream.cond.Signal()
			stream.mu.Unlock()
		}
		return
	}
	stream.mu.Lock()
	stream.readBuf = append(stream.readBuf, data...)
	stream.cond.Signal()
	stream.mu.Unlock()
}

func (xc *XHTTPConnection) handleClose(streamID uint32) {
	xc.mu.Lock()
	if stream, ok := xc.streams[streamID]; ok {
		atomic.StoreInt32(&stream.closed, 1)
		stream.cond.Signal()
	}
	xc.mu.Unlock()
}

func (xc *XHTTPConnection) handleReset(streamID uint32) {
	xc.mu.Lock()
	if stream, ok := xc.streams[streamID]; ok {
		atomic.StoreInt32(&stream.closed, 1)
		stream.cond.Signal()
	}
	xc.mu.Unlock()
}

func (xc *XHTTPConnection) writeFrame(streamID uint32, frameType XHTTPFrameType, flags byte, data []byte) error {
	xc.mu.Lock()
	defer xc.mu.Unlock()
	if atomic.LoadInt32(&xc.closed) != 0 {
		return ErrXHTTPClosed
	}
	header := make([]byte, 12)
	header[0] = byte(frameType)
	binary.BigEndian.PutUint32(header[1:5], streamID)
	binary.BigEndian.PutUint32(header[5:9], uint32(len(data)))
	header[9] = flags
	if _, err := xc.writer.Write(header); err != nil {
		return err
	}
	if len(data) > 0 {
		if _, err := xc.writer.Write(data); err != nil {
			return err
		}
	}
	return xc.writer.Flush()
}

func (xc *XHTTPConnection) Close() error {
	if atomic.CompareAndSwapInt32(&xc.closed, 0, 1) {
		xc.cancel()
		xc.wg.Wait()
		xc.conn.Close()
	}
	return nil
}

func (s *XHTTPStream) Read(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for len(s.readBuf) == 0 {
		if atomic.LoadInt32(&s.closed) != 0 {
			return 0, io.EOF
		}
		s.cond.Wait()
	}
	n := copy(p, s.readBuf)
	s.readBuf = s.readBuf[n:]
	return n, nil
}

func (s *XHTTPStream) Write(p []byte) (int, error) {
	if atomic.LoadInt32(&s.closed) != 0 {
		return 0, io.EOF
	}
	if err := s.conn.writeFrame(s.id, XHTTPFrameData, 0, p); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (s *XHTTPStream) Close() error {
	if atomic.CompareAndSwapInt32(&s.closed, 0, 1) {
		s.conn.writeFrame(s.id, XHTTPFrameClose, 0, nil)
	}
	return nil
}

type XHTTPTransport struct {
	config *XHTTPConfig
}

func NewXHTTPTransport(config *XHTTPConfig) *XHTTPTransport {
	if config == nil {
		config = DefaultXHTTPConfig()
	}
	return &XHTTPTransport{config: config}
}

func (t *XHTTPTransport) Dial(ctx context.Context, addr string) (net.Conn, error) {
	tcpConn, err := net.DialTimeout("tcp", addr, 15*time.Second)
	if err != nil {
		return nil, fmt.Errorf("XHTTP: dial: %w", err)
	}
	xc := NewXHTTPConnection(tcpConn, t.config)
	return &XHTTPConnWrapper{conn: tcpConn, xc: xc, stream: nil}, nil
}

func (t *XHTTPTransport) Listen(addr string) (net.Listener, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	return &XHTTPListener{
		transport: t,
		listener:  ln,
	}, nil
}

type XHTTPConnWrapper struct {
	conn   net.Conn
	xc     *XHTTPConnection
	stream *XHTTPStream
	mu     sync.Mutex
}

func (w *XHTTPConnWrapper) Read(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.stream == nil {
		var err error
		w.stream, err = w.xc.AcceptStream()
		if err != nil {
			return 0, err
		}
	}
	return w.stream.Read(p)
}

func (w *XHTTPConnWrapper) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.stream == nil {
		var err error
		w.stream, err = w.xc.OpenStream()
		if err != nil {
			return 0, err
		}
	}
	return w.stream.Write(p)
}

func (w *XHTTPConnWrapper) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.stream != nil {
		w.stream.Close()
	}
	return w.xc.Close()
}

func (w *XHTTPConnWrapper) LocalAddr() net.Addr  { return w.conn.LocalAddr() }
func (w *XHTTPConnWrapper) RemoteAddr() net.Addr { return w.conn.RemoteAddr() }
func (w *XHTTPConnWrapper) SetDeadline(t time.Time) error     { return w.conn.SetDeadline(t) }
func (w *XHTTPConnWrapper) SetReadDeadline(t time.Time) error  { return w.conn.SetReadDeadline(t) }
func (w *XHTTPConnWrapper) SetWriteDeadline(t time.Time) error { return w.conn.SetWriteDeadline(t) }

type XHTTPListener struct {
	transport *XHTTPTransport
	listener  net.Listener
	mu        sync.Mutex
}

func (l *XHTTPListener) Accept() (net.Conn, error) {
	tcpConn, err := l.listener.Accept()
	if err != nil {
		return nil, err
	}
	xc := NewXHTTPConnection(tcpConn, l.transport.config)
	return &XHTTPConnWrapper{conn: tcpConn, xc: xc}, nil
}

func (l *XHTTPListener) Close() error { return l.listener.Close() }
func (l *XHTTPListener) Addr() net.Addr { return l.listener.Addr() }
