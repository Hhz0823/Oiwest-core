package dccp

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"sync"
	"time"
)

var (
	ErrHandshakeTimeout = errors.New("dccp: handshake timeout")
	ErrHandshakeFailed  = errors.New("dccp: handshake failed")
	ErrInvalidCookie    = errors.New("dccp: invalid cookie")
)

type HandshakeState byte

const (
	StateClosed     HandshakeState = 0
	StateRequest    HandshakeState = 1
	StateRespond    HandshakeState = 2
	StateParTOPEN   HandshakeState = 3
	StateOpen       HandshakeState = 4
	StateCloseReq   HandshakeState = 5
	StateClosing    HandshakeState = 6
	StateTimeWait   HandshakeState = 7
)

type ConnState struct {
	mu              sync.Mutex
	State           HandshakeState
	LocalPort       uint16
	RemotePort      uint16
	LocalSeq        uint64
	RemoteSeq       uint64
	LocalAck        uint64
	RemoteAck       uint64
	ServiceCode     ServiceCode
	CCID            CCID
	LastActivity    time.Time
	IsServer        bool
	NegotiatedCCID  CCID
}

func NewConnState(isServer bool) *ConnState {
	return &ConnState{
		State:        StateClosed,
		IsServer:     isServer,
		LastActivity: time.Now(),
		CCID:         CCID4,
	}
}

func (cs *ConnState) GenerateSequence() uint64 {
	b := make([]byte, 8)
	io.ReadFull(rand.Reader, b)
	return binary.BigEndian.Uint64(b)
}

func (cs *ConnState) AdvanceState(newState HandshakeState) bool {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	switch cs.State {
	case StateClosed:
		return newState == StateRequest || newState == StateRespond
	case StateRequest:
		return newState == StateRespond || newState == StateOpen
	case StateRespond:
		return newState == StateParTOPEN || newState == StateOpen
	case StateParTOPEN:
		return newState == StateOpen
	case StateOpen:
		return newState == StateCloseReq || newState == StateClosing
	case StateCloseReq:
		return newState == StateClosing || newState == StateTimeWait
	case StateClosing:
		return newState == StateTimeWait
	case StateTimeWait:
		return newState == StateClosed
	default:
		return false
	}
}

type HandshakeHandler struct {
	Config       *HandshakeConfig
	state        *ConnState
	readTimeout  time.Duration
	writeTimeout time.Duration
}

type HandshakeConfig struct {
	ServiceCode ServiceCode
	CCID        CCID
	MaxRetries  int
	RetryDelay  time.Duration
	Timeout     time.Duration
	InitCookie  []byte
}

func DefaultHandshakeConfig() *HandshakeConfig {
	return &HandshakeConfig{
		ServiceCode: ServiceCodeV2Ray,
		CCID:        CCID4,
		MaxRetries:  3,
		RetryDelay:  time.Second,
		Timeout:     15 * time.Second,
	}
}

func NewHandshakeHandler(config *HandshakeConfig, state *ConnState) *HandshakeHandler {
	if config == nil {
		config = DefaultHandshakeConfig()
	}
	return &HandshakeHandler{
		Config:       config,
		state:        state,
		readTimeout:  15 * time.Second,
		writeTimeout: 10 * time.Second,
	}
}

func (h *HandshakeHandler) ClientHandshake(conn net.Conn, srcPort, dstPort uint16) error {
	h.state.mu.Lock()
	h.state.IsServer = false
	h.state.LocalPort = srcPort
	h.state.RemotePort = dstPort
	h.state.State = StateRequest
	h.state.LocalSeq = h.state.GenerateSequence()
	localSeq := h.state.LocalSeq
	h.state.mu.Unlock()

	conn.SetDeadline(time.Now().Add(h.Config.Timeout))
	defer conn.SetDeadline(time.Time{})

	reqPacket, err := BuildRequestPacket(srcPort, dstPort, h.Config.ServiceCode, localSeq)
	if err != nil {
		return err
	}

	for attempt := 0; attempt <= h.Config.MaxRetries; attempt++ {
		if _, err := conn.Write(reqPacket); err != nil {
			if attempt == h.Config.MaxRetries {
				return err
			}
			time.Sleep(h.Config.RetryDelay)
			continue
		}

		hdr, opts, _, err := ReadFullDCCPPacket(conn)
		if err != nil {
			if attempt == h.Config.MaxRetries {
				return ErrHandshakeFailed
			}
			time.Sleep(h.Config.RetryDelay)
			continue
		}

		if hdr.Type != PktDCCPResponse {
			if attempt == h.Config.MaxRetries {
				return ErrHandshakeFailed
			}
			time.Sleep(h.Config.RetryDelay)
			continue
		}

		cookieValid := false
		for _, opt := range opts {
			if opt.Type == OptInitCookie && len(opt.Data) >= 4 {
				cookieValid = true
				break
			}
		}
		if !cookieValid {
			return ErrInvalidCookie
		}

		h.state.mu.Lock()
		h.state.RemoteSeq = hdr.SequenceNumber
		h.state.LocalAck = hdr.SequenceNumber
		h.state.RemoteAck = localSeq + 1
		h.state.State = StateParTOPEN
		h.state.mu.Unlock()

		ackPacket, err := BuildAckPacket(srcPort, dstPort, h.state.LocalAck)
		if err != nil {
			return err
		}

		h.state.mu.Lock()
		h.state.LocalSeq++
		h.state.State = StateOpen
		h.state.mu.Unlock()

		if _, err := conn.Write(ackPacket); err != nil {
			return err
		}

		for _, opt := range opts {
			if opt.Type == OptFeature && len(opt.Data) >= 1 {
				h.state.mu.Lock()
				h.state.NegotiatedCCID = CCID(opt.Data[0])
				h.state.mu.Unlock()
			}
		}

		return nil
	}

	return ErrHandshakeFailed
}

func (h *HandshakeHandler) ServerHandshake(conn net.Conn) (uint16, uint16, error) {
	h.state.mu.Lock()
	h.state.IsServer = true
	h.state.State = StateClosed
	h.state.mu.Unlock()

	conn.SetDeadline(time.Now().Add(h.Config.Timeout))
	defer conn.SetDeadline(time.Time{})

	hdr, opts, _, err := ReadFullDCCPPacket(conn)
	if err != nil {
		return 0, 0, err
	}

	if hdr.Type != PktDCCPRequest {
		return 0, 0, ErrHandshakeFailed
	}

	srcPort := hdr.SourcePort
	dstPort := hdr.DestPort
	clientSeq := hdr.SequenceNumber

	cookieValid := false
	var clientServiceCode ServiceCode
	for _, opt := range opts {
		if opt.Type == OptInitCookie && len(opt.Data) >= 4 {
			copy(clientServiceCode[:], opt.Data[2:6])
			cookieValid = true
		}
	}
	if !cookieValid {
		return 0, 0, ErrInvalidCookie
	}

	h.state.mu.Lock()
	h.state.LocalPort = dstPort
	h.state.RemotePort = srcPort
	h.state.ServiceCode = clientServiceCode
	h.state.RemoteSeq = clientSeq
	h.state.LocalSeq = h.state.GenerateSequence()
	h.state.RemoteAck = h.state.LocalSeq
	localSeq := h.state.LocalSeq
	h.state.State = StateRespond
	h.state.mu.Unlock()

	respPacket, err := BuildResponsePacket(dstPort, srcPort, clientServiceCode, localSeq, clientSeq)
	if err != nil {
		return 0, 0, err
	}

	if _, err := conn.Write(respPacket); err != nil {
		return 0, 0, err
	}

	ackHdr, _, _, err := ReadFullDCCPPacket(conn)
	if err != nil {
		return 0, 0, err
	}

	if ackHdr.Type != PktDCCPAck {
		return 0, 0, ErrHandshakeFailed
	}

	h.state.mu.Lock()
	h.state.LocalAck = ackHdr.SequenceNumber
	h.state.State = StateOpen
	h.state.mu.Unlock()

	return srcPort, dstPort, nil
}

func (h *HandshakeHandler) Close(conn net.Conn) error {
	h.state.mu.Lock()
	srcPort := h.state.LocalPort
	dstPort := h.state.RemotePort
	seq := h.state.LocalSeq
	h.state.State = StateClosing
	h.state.mu.Unlock()

	closePacket, err := BuildClosePacket(srcPort, dstPort, seq)
	if err != nil {
		return err
	}

	conn.SetWriteDeadline(time.Now().Add(h.writeTimeout))
	_, err = conn.Write(closePacket)

	h.state.mu.Lock()
	h.state.State = StateClosed
	h.state.mu.Unlock()

	return err
}

func (h *HandshakeHandler) State() HandshakeState {
	h.state.mu.Lock()
	defer h.state.mu.Unlock()
	return h.state.State
}
