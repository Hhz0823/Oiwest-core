package bbr

import (
	"math"
	"sync"
	"time"
)

type BBRAlgorithm string

const (
	BBROriginal   BBRAlgorithm = "bbr"
	BBRv2         BBRAlgorithm = "bbrv2"
	BBRv3         BBRAlgorithm = "bbrv3"
	BBRPlus       BBRAlgorithm = "bbrplus"
	BBRECN        BBRAlgorithm = "bbr_ecn"
	BBRAdaptive   BBRAlgorithm = "bbr_adaptive"
	BBRProbeRTTAlgo BBRAlgorithm = "bbr_probert"
)

type BBRMode int

const (
	ModeStartup     BBRMode = 0
	ModeDrain       BBRMode = 1
	ModeProbeBW     BBRMode = 2
	ModeProbeRTT    BBRMode = 3
)

type BBRCongestionControl interface {
	Name() string
	Algorithm() BBRAlgorithm
	OnPacketSent(seq uint64, size int)
	OnPacketLost(seq uint64, ecn bool)
	OnPacketAcked(seq uint64, size int, rtt time.Duration, ecn bool)
	OnRTTUpdate(rtt time.Duration)
	CanSend() bool
	CWND() float64
	PacingRate() float64
	PacingGain() float64
	CWNDGain() float64
	Reset()
	MinRTT() time.Duration
	BW() float64
}

type BBRState struct {
	mu            sync.RWMutex
	algorithm     BBRAlgorithm
	mode          BBRMode
	bw            float64
	bwMax         float64
	bwLatest      float64
	bwFilter      []float64
	bwFilterIdx   int
	bwFilterLen   int
	minRTT        time.Duration
	minRTTFilter  []time.Duration
	minRTTIdx     int
	minRTTLen     int
	latestRTT     time.Duration
	cwnd          float64
	cwndGain      float64
	pacingGain    float64
	pacingRate    float64
	ackedBytes    uint64
	lostBytes     uint64
	inflight      uint64
	pipe          uint64
	packetsSent   uint64
	packetsLost   uint64
	packetsAcked  uint64
	bytesSent     uint64
	bytesAcked    uint64
	bytesLost     uint64
	ecnCount      uint64
	ecnThreshold  float64
	rtProp        time.Duration
	rtPropStamp   time.Time
	pktDeliveryRate float64
	probeRTTSince  time.Time
	probeRTTDoneAt time.Time
	probeRTTRound  uint64
	probeRTTMinRTT time.Duration
	probeRTTCount  uint64
	fullBW         bool
	fullBWCount    uint64
	fullBWTimestamp time.Time
	cycleCount     uint64
	roundCount     uint64
	roundStart     bool
	nextRoundSeq   uint64
	delivered      uint64
	deliveredTime  time.Time
	firstSentTime  time.Time
	appLimited     bool
	lossRound      uint64
	lossInRound    uint64
	bwAtLoss       float64
	recoverCount   uint64
	mss           int
	initCWND      float64
	targetCWND    float64
	priorCWND     float64
	bwLo          float64
	bwHi          float64
	roundsSinceLoss uint64
	startupRound      uint64
	startupFullSample bool
	startupBW         float64
	startupMinRTT     time.Duration
	tcpAwareECN  bool
	ecnBDP       float64
	startupPacingGain float64
	startupCWNDGain   float64
	drainPacingGain   float64
}

type BBRConfig struct {
	Algorithm     BBRAlgorithm  `json:"algorithm"`
	MSS           int           `json:"mss"`
	InitCWND      float64       `json:"initCwnd"`
	BWFilterLen   int           `json:"bwFilterLen"`
	RTTFilterLen  int           `json:"rttFilterLen"`
	ProbeRTTInterval time.Duration `json:"probeRttInterval"`
	ECNThreshold  float64       `json:"ecnThreshold"`
	StartupPacingGain float64   `json:"startupPacingGain"`
	StartupCWNDGain   float64   `json:"startupCwndGain"`
	DrainPacingGain   float64   `json:"drainPacingGain"`
	ProbeBWPacingGain []float64 `json:"probeBwPacingGain"`
	ProbeBWCWNDGain   float64   `json:"probeBwCwndGain"`
	BBRv2Alpha   float64       `json:"bbrv2Alpha"`
	BBRv2Beta    float64       `json:"bbrv2Beta"`
	BBRv2Gamma   float64       `json:"bbrv2Gamma"`
}

func DefaultBBRConfig(algorithm BBRAlgorithm) *BBRConfig {
	return &BBRConfig{
		Algorithm:     algorithm,
		MSS:           1460,
		InitCWND:      10,
		BWFilterLen:   10,
		RTTFilterLen:  10,
		ProbeRTTInterval: 10 * time.Second,
		ECNThreshold:  0.08,
		StartupPacingGain: 2.77,
		StartupCWNDGain:   2.0,
		DrainPacingGain:   0.5,
		ProbeBWPacingGain: []float64{1.25, 0.75, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0},
		ProbeBWCWNDGain:   2.0,
		BBRv2Alpha:    0.85,
		BBRv2Beta:     0.7,
		BBRv2Gamma:    1.0,
	}
}

func newBBRState(config *BBRConfig) *BBRState {
	state := &BBRState{
		algorithm:     config.Algorithm,
		mss:           config.MSS,
		initCWND:      config.InitCWND,
		cwnd:          config.InitCWND,
		mode:          ModeStartup,
		pacingGain:    config.StartupPacingGain,
		cwndGain:      config.StartupCWNDGain,
		startupPacingGain: config.StartupPacingGain,
		startupCWNDGain:   config.StartupCWNDGain,
		drainPacingGain:   config.DrainPacingGain,
		bwFilter:      make([]float64, config.BWFilterLen),
		bwFilterLen:   config.BWFilterLen,
		minRTTFilter:  make([]time.Duration, config.RTTFilterLen),
		minRTTLen:     config.RTTFilterLen,
		ecnThreshold:  config.ECNThreshold,
		rtProp:        time.Duration(math.MaxInt64),
		probeRTTMinRTT: time.Hour,
		pacingRate:    10 * float64(config.MSS),
		minRTT:        time.Hour,
	}
	for i := range state.bwFilter {
		state.bwFilter[i] = 0
	}
	for i := range state.minRTTFilter {
		state.minRTTFilter[i] = time.Hour
	}
	return state
}

func (s *BBRState) updateBWFilter() {
	s.bwFilter[s.bwFilterIdx] = s.bwLatest
	s.bwFilterIdx = (s.bwFilterIdx + 1) % s.bwFilterLen
	s.bwMax = 0
	for _, bw := range s.bwFilter {
		if bw > s.bwMax {
			s.bwMax = bw
		}
	}
	if s.bwLatest > s.bw {
		s.bw = s.bwLatest
	}
}

func (s *BBRState) updateMinRTTFilter() {
	s.minRTTFilter[s.minRTTIdx] = s.latestRTT
	s.minRTTIdx = (s.minRTTIdx + 1) % s.minRTTLen
	s.minRTT = time.Hour
	for _, rtt := range s.minRTTFilter {
		if rtt < s.minRTT {
			s.minRTT = rtt
		}
	}
}

func (s *BBRState) bdp() float64 {
	if s.minRTT <= 0 {
		return s.initCWND
	}
	return s.bw * s.minRTT.Seconds()
}

func (s *BBRState) bdpBytes() float64 {
	return s.bdp() * float64(s.mss)
}

func (s *BBRState) updatePacingRate() {
	if s.bw > 0 && s.pacingGain > 0 {
		rate := s.bw * s.pacingGain * float64(s.mss)
		if rate > s.pacingRate || s.pacingRate == 0 {
			s.pacingRate = rate
		}
	}
}

func (s *BBRState) updateCWND() {
	if s.mode == ModeProbeRTT {
		s.cwnd = math.Min(s.cwnd, s.initCWND)
		return
	}
	bdp := s.bdpBytes()
	if s.mode == ModeStartup {
		s.cwnd = math.Min(s.cwnd+s.cwndGain, bdp*s.cwndGain)
	} else if s.mode == ModeProbeBW {
		s.cwnd = bdp * 2
	} else {
		s.cwnd = bdp
	}
	if s.cwnd < s.initCWND {
		s.cwnd = s.initCWND
	}
	if s.cwnd > 600 {
		s.cwnd = 600
	}
}

func (s *BBRState) advanceMode() {
	switch s.mode {
	case ModeStartup:
		if s.bwMax == s.bwLatest && s.fullBW {
			s.fullBWCount++
			if s.fullBWCount >= 3 {
				s.mode = ModeDrain
				s.pacingGain = s.drainPacingGain
				s.cwndGain = 1.0
				s.fullBW = false
			}
		} else {
			s.fullBW = true
			s.fullBWCount = 0
		}
	case ModeDrain:
		if float64(s.inflight) <= s.bdp() {
			s.mode = ModeProbeBW
			s.pacingGain = 1.25
			s.cwndGain = 2.0
			s.cycleCount = 0
		}
	case ModeProbeBW:
		s.cycleCount++
		gains := []float64{1.25, 0.75, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0}
		idx := int(s.cycleCount) % len(gains)
		s.pacingGain = gains[idx]
		if idx == 0 {
			s.cwndGain = 2.0
		} else {
			s.cwndGain = 1.0
		}
	case ModeProbeRTT:
		if time.Since(s.probeRTTDoneAt) > 200*time.Millisecond {
			s.mode = ModeProbeBW
			s.pacingGain = 1.25
			s.cwndGain = 2.0
		}
	}
}

type BBR struct {
	state *BBRState
	config *BBRConfig
}

func NewBBR(config *BBRConfig) BBRCongestionControl {
	if config == nil {
		config = DefaultBBRConfig(BBROriginal)
	}
	return &BBR{
		state: newBBRState(config),
		config: config,
	}
}

func NewBBRv2(config *BBRConfig) BBRCongestionControl {
	if config == nil {
		config = DefaultBBRConfig(BBRv2)
	}
	state := newBBRState(config)
	state.tcpAwareECN = true
	return &BBR{state: state, config: config}
}

func NewBBRv3(config *BBRConfig) BBRCongestionControl {
	if config == nil {
		config = DefaultBBRConfig(BBRv3)
	}
	state := newBBRState(config)
	state.tcpAwareECN = true
	state.ecnThreshold = 0.05
	return &BBR{state: state, config: config}
}

func newBBR(algorithm BBRAlgorithm, config *BBRConfig) BBRCongestionControl {
	if config == nil {
		config = DefaultBBRConfig(algorithm)
	}
	switch algorithm {
	case BBROriginal:
		return NewBBR(config)
	case BBRv2:
		return NewBBRv2(config)
	case BBRv3:
		return NewBBRv3(config)
	case BBRPlus:
		bbr := NewBBR(config)
		bbr.(*BBR).state.startupPacingGain = 2.89
		return bbr
	case BBRECN:
		bbr := NewBBR(config)
		bbr.(*BBR).state.tcpAwareECN = true
		return bbr
	case BBRAdaptive:
		bbr := NewBBR(config)
		bbr.(*BBR).state.cwndGain = 3.0
		return bbr
	case BBRProbeRTTAlgo:
		bbr := NewBBR(config)
		bbr.(*BBR).state.probeRTTSince = time.Now()
		return bbr
	default:
		return NewBBR(config)
	}
}

func (b *BBR) Name() string {
	switch b.state.algorithm {
	case BBROriginal:
		return "BBR"
	case BBRv2:
		return "BBRv2"
	case BBRv3:
		return "BBRv3"
	case BBRPlus:
		return "BBRPlus"
	case BBRECN:
		return "BBR-ECN"
	case BBRAdaptive:
		return "BBR-Adaptive"
	case BBRProbeRTTAlgo:
		return "BBR-ProbeRTT"
	default:
		return "BBR"
	}
}

func (b *BBR) Algorithm() BBRAlgorithm { return b.state.algorithm }

func (b *BBR) OnPacketSent(seq uint64, size int) {
	b.state.mu.Lock()
	defer b.state.mu.Unlock()
	b.state.packetsSent++
	b.state.inflight++
	b.state.bytesSent += uint64(size)
	if b.state.firstSentTime.IsZero() {
		b.state.firstSentTime = time.Now()
	}
}

func (b *BBR) OnPacketLost(seq uint64, ecn bool) {
	b.state.mu.Lock()
	defer b.state.mu.Unlock()
	b.state.packetsLost++
	b.state.lostBytes += uint64(b.state.mss)
	if b.state.inflight > 0 {
		b.state.inflight--
	}
	if ecn {
		b.state.ecnCount++
	}
	switch b.state.algorithm {
	case BBRv2, BBRv3:
		if ecn && float64(b.state.ecnCount) > b.state.ecnThreshold*float64(b.state.packetsSent) {
			b.state.cwnd *= b.config.BBRv2Alpha
			if b.state.cwnd < b.state.initCWND {
				b.state.cwnd = b.state.initCWND
			}
		} else {
			b.state.cwnd *= b.config.BBRv2Beta
		}
	default:
		b.state.cwnd *= 0.8
	}
	if b.state.cwnd < b.state.initCWND {
		b.state.cwnd = b.state.initCWND
	}
}

func (b *BBR) OnPacketAcked(seq uint64, size int, rtt time.Duration, ecn bool) {
	b.state.mu.Lock()
	defer b.state.mu.Unlock()
	b.state.packetsAcked++
	b.state.bytesAcked += uint64(size)
	if b.state.inflight > 0 {
		b.state.inflight--
	}
	b.state.latestRTT = rtt
	if rtt < b.state.minRTT && rtt > 0 {
		b.state.minRTT = rtt
	}
	b.state.updateMinRTTFilter()
	if size > 0 {
		rate := float64(size) / rtt.Seconds()
		if rate > 0 {
			b.state.bwLatest = rate
			b.state.updateBWFilter()
		}
	}
	if b.state.mode == ModeStartup {
		b.state.advanceMode()
	}
	b.state.updatePacingRate()
	b.state.updateCWND()
}

func (b *BBR) OnRTTUpdate(rtt time.Duration) {
	b.state.mu.Lock()
	defer b.state.mu.Unlock()
	b.state.latestRTT = rtt
	if rtt < b.state.minRTT && rtt > 0 {
		b.state.minRTT = rtt
	}
	b.state.updateMinRTTFilter()
	if b.state.minRTT > 0 && (time.Since(b.state.probeRTTDoneAt) > 10*time.Second) {
		b.state.mode = ModeProbeRTT
		b.state.probeRTTSince = time.Now()
	}
}

func (b *BBR) CanSend() bool {
	b.state.mu.RLock()
	defer b.state.mu.RUnlock()
	return float64(b.state.inflight) < b.state.cwnd
}

func (b *BBR) CWND() float64 {
	b.state.mu.RLock()
	defer b.state.mu.RUnlock()
	return b.state.cwnd
}

func (b *BBR) PacingRate() float64 {
	b.state.mu.RLock()
	defer b.state.mu.RUnlock()
	return b.state.pacingRate
}

func (b *BBR) PacingGain() float64 {
	b.state.mu.RLock()
	defer b.state.mu.RUnlock()
	return b.state.pacingGain
}

func (b *BBR) CWNDGain() float64 {
	b.state.mu.RLock()
	defer b.state.mu.RUnlock()
	return b.state.cwndGain
}

func (b *BBR) Reset() {
	b.state.mu.Lock()
	defer b.state.mu.Unlock()
	b.state.cwnd = b.state.initCWND
	b.state.bw = 0
	b.state.bwMax = 0
	b.state.mode = ModeStartup
	b.state.pacingGain = b.state.startupPacingGain
	b.state.cwndGain = b.state.startupCWNDGain
	b.state.packetsSent = 0
	b.state.packetsLost = 0
	b.state.packetsAcked = 0
	b.state.inflight = 0
	b.state.bytesSent = 0
	b.state.bytesAcked = 0
}

func (b *BBR) MinRTT() time.Duration {
	b.state.mu.RLock()
	defer b.state.mu.RUnlock()
	return b.state.minRTT
}

func (b *BBR) BW() float64 {
	b.state.mu.RLock()
	defer b.state.mu.RUnlock()
	return b.state.bw
}

type BBRFactory struct {
	variants map[BBRAlgorithm]func(*BBRConfig) BBRCongestionControl
}

var globalBBRFactory *BBRFactory

func GetBBRFactory() *BBRFactory {
	if globalBBRFactory == nil {
		globalBBRFactory = &BBRFactory{
			variants: make(map[BBRAlgorithm]func(*BBRConfig) BBRCongestionControl),
		}
		globalBBRFactory.Register(BBROriginal, func(cfg *BBRConfig) BBRCongestionControl {
			return NewBBR(cfg)
		})
		globalBBRFactory.Register(BBRv2, func(cfg *BBRConfig) BBRCongestionControl {
			return NewBBRv2(cfg)
		})
		globalBBRFactory.Register(BBRv3, func(cfg *BBRConfig) BBRCongestionControl {
			return NewBBRv3(cfg)
		})
		globalBBRFactory.Register(BBRPlus, func(cfg *BBRConfig) BBRCongestionControl {
			return newBBR(BBRPlus, cfg)
		})
		globalBBRFactory.Register(BBRECN, func(cfg *BBRConfig) BBRCongestionControl {
			return newBBR(BBRECN, cfg)
		})
		globalBBRFactory.Register(BBRAdaptive, func(cfg *BBRConfig) BBRCongestionControl {
			return newBBR(BBRAdaptive, cfg)
		})
		globalBBRFactory.Register(BBRProbeRTTAlgo, func(cfg *BBRConfig) BBRCongestionControl {
			return newBBR(BBRProbeRTTAlgo, cfg)
		})
	}
	return globalBBRFactory
}

func (f *BBRFactory) Register(algorithm BBRAlgorithm, factory func(*BBRConfig) BBRCongestionControl) {
	f.variants[algorithm] = factory
}

func (f *BBRFactory) Create(algorithm BBRAlgorithm, config *BBRConfig) BBRCongestionControl {
	if factory, ok := f.variants[algorithm]; ok {
		return factory(config)
	}
	return NewBBR(config)
}

func (f *BBRFactory) ListAlgorithms() []BBRAlgorithm {
	algorithms := make([]BBRAlgorithm, 0, len(f.variants))
	for algo := range f.variants {
		algorithms = append(algorithms, algo)
	}
	return algorithms
}



