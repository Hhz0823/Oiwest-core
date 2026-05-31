package dccp

import (
	"math"
	"sync"
	"time"
)

type CongestionControl interface {
	Name() string
	OnPacketSent(seq uint64, size int)
	OnPacketLost(seq uint64)
	OnPacketAcked(seq uint64, rtt time.Duration)
	OnRTO() time.Duration
	CanSend() bool
	CWND() float64
	SSTHRESH() float64
	MSS() int
	Reset()
}

type ccidFactory func() CongestionControl

var ccidRegistry = make(map[CCID]ccidFactory)

func registerCCID(id CCID, factory ccidFactory) {
	ccidRegistry[id] = factory
}

func GetCongestionControl(id CCID) CongestionControl {
	if factory, ok := ccidRegistry[id]; ok {
		return factory()
	}
	return newCCID4CC()
}

type BaseCongestionControl struct {
	mu       sync.Mutex
	cwnd     float64
	ssthresh float64
	mss      int
	lastRTT  time.Duration
	rttSum   time.Duration
	rttCnt   int
	minRTT   time.Duration
	maxRTT   time.Duration
}

func (b *BaseCongestionControl) CWND() float64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.cwnd
}

func (b *BaseCongestionControl) SSTHRESH() float64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.ssthresh
}

func (b *BaseCongestionControl) MSS() int {
	return b.mss
}

type ccid2CC struct {
	BaseCongestionControl
	inFlight   uint64
	srtt       time.Duration
	rttvar     time.Duration
	rto        time.Duration
	backoff    int
	pipe       uint64
	lossWindow float64
}

func newCCID2CC() CongestionControl {
	return &ccid2CC{
		BaseCongestionControl: BaseCongestionControl{
			cwnd:    2,
			ssthresh: math.MaxFloat64,
			mss:     536,
			minRTT:  time.Hour,
		},
		rto: time.Second,
	}
}

func (c *ccid2CC) Name() string { return "CCID2-TCP-Like" }

func (c *ccid2CC) OnPacketSent(seq uint64, size int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.inFlight++
	c.pipe++
}

func (c *ccid2CC) OnPacketLost(seq uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ssthresh = math.Max(2, c.cwnd/2)
	c.lossWindow = c.cwnd
	c.cwnd = math.Max(1, c.cwnd/2)
	c.backoff++
	rto := c.rto
	for i := 0; i < c.backoff && i < 6; i++ {
		rto *= 2
	}
	c.rto = rto
	if c.pipe > 0 {
		c.pipe--
	}
}

func (c *ccid2CC) OnPacketAcked(seq uint64, rtt time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.pipe > 0 {
		c.pipe--
	}

	if rtt < c.minRTT {
		c.minRTT = rtt
	}
	if rtt > c.maxRTT {
		c.maxRTT = rtt
	}

	if c.srtt == 0 {
		c.srtt = rtt
		c.rttvar = rtt / 2
	} else {
		delta := rtt - c.srtt
		if delta < 0 {
			delta = -delta
		}
		c.rttvar = (3*c.rttvar + delta) / 4
		c.srtt = (7*c.srtt + rtt) / 8
	}
	c.rto = c.srtt + 4*c.rttvar
	if c.rto < time.Millisecond*200 {
		c.rto = time.Millisecond * 200
	}

	if c.cwnd < c.ssthresh {
		c.cwnd += 1
	} else {
		c.cwnd += 1.0 / c.cwnd
	}
}

func (c *ccid2CC) OnRTO() time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()
	rto := c.rto
	for i := 0; i < c.backoff && i < 6; i++ {
		rto *= 2
	}
	return rto
}

func (c *ccid2CC) CanSend() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return float64(c.pipe) < c.cwnd
}

func (c *ccid2CC) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cwnd = 2
	c.ssthresh = math.MaxFloat64
	c.inFlight = 0
	c.pipe = 0
	c.backoff = 0
	c.rto = time.Second
}

type ccid3CC struct {
	BaseCongestionControl
	x            float64
	xRecv        float64
	tcpWMin      float64
	rtt          time.Duration
	tld          time.Duration
	lossRate     float64
	lastLoss     time.Time
	sendRate     float64
	bytesSent    uint64
	bytesAcked   uint64
	lossCount    uint64
	totalCount   uint64
	lastRateCalc time.Time
}

func newCCID3CC() CongestionControl {
	return &ccid3CC{
		BaseCongestionControl: BaseCongestionControl{
			cwnd:     4,
			ssthresh: math.MaxFloat64,
			mss:      536,
			minRTT:   time.Hour,
		},
		tcpWMin:      4,
		sendRate:     0,
		lastRateCalc: time.Now(),
	}
}

func (c *ccid3CC) Name() string { return "CCID3-TFRC" }

func (c *ccid3CC) OnPacketSent(seq uint64, size int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.bytesSent += uint64(size)
	c.totalCount++
}

func (c *ccid3CC) OnPacketLost(seq uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lossCount++
	now := time.Now()
	if !c.lastLoss.IsZero() {
		interval := now.Sub(c.lastLoss)
		if interval > 0 {
			c.lossRate = 1.0 / interval.Seconds()
		}
	}
	c.lastLoss = now
	c.updateSendRate()
}

func (c *ccid3CC) OnPacketAcked(seq uint64, rtt time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.bytesAcked += uint64(c.mss)
	if rtt < c.minRTT {
		c.minRTT = rtt
	}
	if rtt > c.maxRTT {
		c.maxRTT = rtt
	}
	c.rtt = (7*c.rtt + rtt) / 8

	now := time.Now()
	if now.Sub(c.lastRateCalc) > time.Second {
		c.updateSendRate()
		c.lastRateCalc = now
	}
}

func (c *ccid3CC) updateSendRate() {
	if c.lossCount == 0 {
		c.sendRate = math.MaxFloat64
		return
	}

	p := float64(c.lossCount) / float64(c.totalCount+1)
	if c.rtt > 0 {
		rttSec := c.rtt.Seconds()
		b := 1.0
		c.x = math.Sqrt(2.0*b/3.0) / (rttSec * math.Sqrt(p))
		c.sendRate = c.x * float64(c.mss)
		c.cwnd = math.Max(c.tcpWMin, c.x*rttSec)
	}
}

func (c *ccid3CC) OnRTO() time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.rtt == 0 {
		return time.Second
	}
	return 4 * c.rtt
}

func (c *ccid3CC) CanSend() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.sendRate > 0
}

func (c *ccid3CC) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cwnd = 4
	c.ssthresh = math.MaxFloat64
	c.x = 0
	c.lossRate = 0
	c.lossCount = 0
	c.totalCount = 0
	c.sendRate = 0
}

type ccid4CC struct {
	BaseCongestionControl
	rtt        time.Duration
	rttVar     time.Duration
	rto        time.Duration
	inFlight   int
	maxInFlight int
	backoff    int
}

func newCCID4CC() CongestionControl {
	return &ccid4CC{
		BaseCongestionControl: BaseCongestionControl{
			cwnd:     10,
			ssthresh: math.MaxFloat64,
			mss:      1200,
			minRTT:   time.Hour,
		},
		rto:         time.Second,
		maxInFlight: 10,
	}
}

func (c *ccid4CC) Name() string { return "CCID4-Experimental" }

func (c *ccid4CC) OnPacketSent(seq uint64, size int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.inFlight++
}

func (c *ccid4CC) OnPacketLost(seq uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cwnd = math.Max(2, c.cwnd*0.7)
	c.maxInFlight = int(c.cwnd)
	c.backoff++
	c.inFlight--
}

func (c *ccid4CC) OnPacketAcked(seq uint64, rtt time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.inFlight--
	if c.inFlight < 0 {
		c.inFlight = 0
	}

	if rtt < c.minRTT {
		c.minRTT = rtt
	}
	if rtt > c.maxRTT {
		c.maxRTT = rtt
	}

	if c.rtt == 0 {
		c.rtt = rtt
		c.rttVar = rtt / 2
	} else {
		delta := rtt - c.rtt
		if delta < 0 {
			delta = -delta
		}
		c.rttVar = (3*c.rttVar + delta) / 4
		c.rtt = (7*c.rtt + rtt) / 8
	}

	if c.cwnd < c.ssthresh {
		c.cwnd *= 1.5
	} else {
		c.cwnd += 0.5
	}
	if c.cwnd > 1000 {
		c.cwnd = 1000
	}
	c.maxInFlight = int(c.cwnd)
	c.rto = c.rtt/2 + 4*c.rttVar
	c.backoff = 0
}

func (c *ccid4CC) OnRTO() time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()
	rto := c.rto
	for i := 0; i < c.backoff && i < 6; i++ {
		rto *= 2
	}
	return rto
}

func (c *ccid4CC) CanSend() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.inFlight < c.maxInFlight
}

func (c *ccid4CC) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cwnd = 10
	c.ssthresh = math.MaxFloat64
	c.inFlight = 0
	c.maxInFlight = 10
	c.backoff = 0
	c.rto = time.Second
}

func init() {
	registerCCID(CCID2, newCCID2CC)
	registerCCID(CCID3, newCCID3CC)
	registerCCID(CCID4, newCCID4CC)
}
