package multiplex

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sb-panel/dccp-kernel/common/buf"
)

var (
	ErrSessionClosed   = errors.New("mux: session closed")
	ErrStreamClosed    = errors.New("mux: stream closed")
	ErrStreamReset     = errors.New("mux: stream reset")
	ErrMaxStreams      = errors.New("mux: max streams reached")
)

const (
	DefaultMaxStreams  = 128
	DefaultBufferSize  = 32 * 1024
	FrameHeaderSize    = 8
	MaxFrameSize       = 64 * 1024
)

type FrameType byte

const (
	FrameDataType  FrameType = 0x0
	FrameCloseType FrameType = 0x1
	FrameResetType FrameType = 0x2
	FramePingType  FrameType = 0x3
	FramePongType  FrameType = 0x4
	FrameGoAwayType FrameType = 0x5
)

type FrameHeader struct {
	StreamID uint32
	Flags    byte
	Length   uint32
}

type MuxStream struct {
	id       uint32
	session  *MuxSession
	readBuf  *buf.Buffer
	readCh   chan struct{}
	mu       sync.Mutex
	closed   uint32
	reset    uint32
}

func (s *MuxStream) ID() uint32 {
	return s.id
}

func (s *MuxStream) Read(p []byte) (int, error) {
	if atomic.LoadUint32(&s.reset) == 1 {
		return 0, ErrStreamReset
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for s.readBuf.IsEmpty() {
		if atomic.LoadUint32(&s.closed) == 1 {
			return 0, io.EOF
		}
		s.mu.Unlock()
		select {
		case <-s.readCh:
		case <-s.session.ctx.Done():
			s.mu.Lock()
			return 0, ErrSessionClosed
		}
		s.mu.Lock()
	}

	return s.readBuf.Read(p)
}

func (s *MuxStream) Write(p []byte) (int, error) {
	if atomic.LoadUint32(&s.closed) == 1 {
		return 0, ErrStreamClosed
	}

	return s.session.WriteFrame(s.id, FrameDataType, p)
}

func (s *MuxStream) Close() error {
	if !atomic.CompareAndSwapUint32(&s.closed, 0, 1) {
		return nil
	}
	_, err := s.session.WriteFrame(s.id, FrameCloseType, nil)
	return err
}

func (s *MuxStream) Reset() error {
	atomic.StoreUint32(&s.reset, 1)
	_, err := s.session.WriteFrame(s.id, FrameResetType, nil)
	return err
}

type MuxSession struct {
	conn        net.Conn
	streams     map[uint32]*MuxStream
	nextID      uint32
	maxStreams  uint32
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	closed      int32
	acceptCh    chan *MuxStream
	wg          sync.WaitGroup
}

type MuxConfig struct {
	MaxStreams     uint32
	MaxFrameSize   uint32
	KeepAliveInterval time.Duration
	KeepAliveTimeout  time.Duration
}

func DefaultMuxConfig() *MuxConfig {
	return &MuxConfig{
		MaxStreams:        DefaultMaxStreams,
		MaxFrameSize:      MaxFrameSize,
		KeepAliveInterval: 30 * time.Second,
		KeepAliveTimeout:  90 * time.Second,
	}
}

func NewMuxSession(conn net.Conn, config *MuxConfig) *MuxSession {
	if config == nil {
		config = DefaultMuxConfig()
	}
	ctx, cancel := context.WithCancel(context.Background())
	s := &MuxSession{
		conn:       conn,
		streams:    make(map[uint32]*MuxStream),
		nextID:     1,
		maxStreams: config.MaxStreams,
		ctx:        ctx,
		cancel:     cancel,
		acceptCh:   make(chan *MuxStream, 16),
	}
	s.wg.Add(1)
	go s.readLoop()
	if config.KeepAliveInterval > 0 {
		s.wg.Add(1)
		go s.keepAliveLoop(config.KeepAliveInterval)
	}
	return s
}

func (s *MuxSession) OpenStream() (*MuxStream, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if atomic.LoadInt32(&s.closed) != 0 {
		return nil, ErrSessionClosed
	}

	if uint32(len(s.streams)) >= s.maxStreams {
		return nil, ErrMaxStreams
	}

	id := s.nextID
	s.nextID += 2

	stream := &MuxStream{
		id:      id,
		session: s,
		readBuf: buf.NewWithSize(DefaultBufferSize),
		readCh:  make(chan struct{}, 1),
	}
	s.streams[id] = stream
	return stream, nil
}

func (s *MuxSession) AcceptStream() (*MuxStream, error) {
	select {
	case stream, ok := <-s.acceptCh:
		if !ok {
			return nil, ErrSessionClosed
		}
		return stream, nil
	case <-s.ctx.Done():
		return nil, ErrSessionClosed
	}
}

func (s *MuxSession) WriteFrame(streamID uint32, frameType FrameType, data []byte) (int, error) {
	frameSize := FrameHeaderSize + len(data)
	frame := make([]byte, frameSize)

	frame[0] = byte(streamID >> 24)
	frame[1] = byte(streamID >> 16)
	frame[2] = byte(streamID >> 8)
	frame[3] = byte(streamID)
	frame[4] = byte(frameType)
	frame[5] = 0
	frame[6] = byte(len(data) >> 8)
	frame[7] = byte(len(data))

	copy(frame[FrameHeaderSize:], data)

	s.mu.RLock()
	if atomic.LoadInt32(&s.closed) != 0 {
		s.mu.RUnlock()
		return 0, ErrSessionClosed
	}
	s.mu.RUnlock()

	return s.conn.Write(frame)
}

func (s *MuxSession) readLoop() {
	defer s.wg.Done()
	defer s.Close()

	frameHeader := make([]byte, FrameHeaderSize)

	for {
		_, err := io.ReadFull(s.conn, frameHeader)
		if err != nil {
			return
		}

		streamID := uint32(frameHeader[0])<<24 | uint32(frameHeader[1])<<16 |
			uint32(frameHeader[2])<<8 | uint32(frameHeader[3])
		frameType := FrameType(frameHeader[4])
		dataLen := int(frameHeader[6])<<8 | int(frameHeader[7])

		var data []byte
		if dataLen > 0 {
			data = make([]byte, dataLen)
			if _, err := io.ReadFull(s.conn, data); err != nil {
				return
			}
		}

		switch frameType {
		case FrameDataType:
			s.handleData(streamID, data)
		case FrameCloseType:
			s.handleClose(streamID)
		case FrameResetType:
			s.handleReset(streamID)
		case FramePingType:
			s.WriteFrame(0, FramePongType, data)
		case FramePongType:
		case FrameGoAwayType:
			return
		}
	}
}

func (s *MuxSession) handleData(streamID uint32, data []byte) {
	s.mu.RLock()
	stream, ok := s.streams[streamID]
	s.mu.RUnlock()

	if !ok {
		if streamID%2 == 0 {
			s.mu.Lock()
			stream = &MuxStream{
				id:      streamID,
				session: s,
				readBuf: buf.NewWithSize(DefaultBufferSize),
				readCh:  make(chan struct{}, 1),
			}
			s.streams[streamID] = stream
			s.mu.Unlock()

			select {
			case s.acceptCh <- stream:
			default:
			}
		} else {
			return
		}
	}

	stream.mu.Lock()
	stream.readBuf.Write(data)
	stream.mu.Unlock()

	select {
	case stream.readCh <- struct{}{}:
	default:
	}
}

func (s *MuxSession) handleClose(streamID uint32) {
	s.mu.Lock()
	stream, ok := s.streams[streamID]
	if ok {
		atomic.StoreUint32(&stream.closed, 1)
		select {
		case stream.readCh <- struct{}{}:
		default:
		}
	}
	s.mu.Unlock()
}

func (s *MuxSession) handleReset(streamID uint32) {
	s.mu.Lock()
	stream, ok := s.streams[streamID]
	if ok {
		atomic.StoreUint32(&stream.reset, 1)
		atomic.StoreUint32(&stream.closed, 1)
		select {
		case stream.readCh <- struct{}{}:
		default:
		}
	}
	s.mu.Unlock()
}

func (s *MuxSession) keepAliveLoop(interval time.Duration) {
	defer s.wg.Done()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.WriteFrame(0, FramePingType, nil)
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *MuxSession) Close() error {
	if !atomic.CompareAndSwapInt32(&s.closed, 0, 1) {
		return nil
	}

	s.WriteFrame(0, FrameGoAwayType, nil)
	s.cancel()

	s.mu.Lock()
	for _, stream := range s.streams {
		atomic.StoreUint32(&stream.closed, 1)
		select {
		case stream.readCh <- struct{}{}:
		default:
		}
	}
	s.streams = nil
	s.mu.Unlock()

	close(s.acceptCh)
	s.wg.Wait()

	return s.conn.Close()
}

func (s *MuxSession) NumStreams() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.streams)
}

func (s *MuxSession) IsClosed() bool {
	return atomic.LoadInt32(&s.closed) != 0
}
