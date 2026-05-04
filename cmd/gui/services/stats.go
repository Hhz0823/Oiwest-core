package services

import (
	"sync"
	"sync/atomic"
	"time"
)

type TrafficStats struct {
	UploadSpeed   int64 `json:"uploadSpeed"`
	DownloadSpeed int64 `json:"downloadSpeed"`
	TotalUpload   int64 `json:"totalUpload"`
	TotalDownload int64 `json:"totalDownload"`
	ActiveConns   int   `json:"activeConns"`
	PacketsSent   int64 `json:"packetsSent"`
	PacketsRecv   int64 `json:"packetsRecv"`
}

type StatsManager struct {
	mu             sync.Mutex
	current        TrafficStats
	lastUpload     int64
	lastDownload   int64
	lastCalcTime   time.Time
	running        int32
	stopCh         chan struct{}
	updateCallback func(TrafficStats)
}

var statsManager *StatsManager

func GetStatsManager() *StatsManager {
	if statsManager == nil {
		statsManager = &StatsManager{
			lastCalcTime: time.Now(),
			stopCh:       make(chan struct{}),
		}
	}
	return statsManager
}

func (sm *StatsManager) SetUpdateCallback(cb func(TrafficStats)) {
	sm.updateCallback = cb
}

func (sm *StatsManager) StartMonitoring() {
	if atomic.LoadInt32(&sm.running) == 1 {
		return
	}
	atomic.StoreInt32(&sm.running, 1)
	go sm.monitorLoop()
}

func (sm *StatsManager) StopMonitoring() {
	if atomic.LoadInt32(&sm.running) == 0 {
		return
	}
	atomic.StoreInt32(&sm.running, 0)
	select {
	case sm.stopCh <- struct{}{}:
	default:
	}
}

func (sm *StatsManager) monitorLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sm.calculateSpeed()
		case <-sm.stopCh:
			return
		}
	}
}

func (sm *StatsManager) calculateSpeed() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(sm.lastCalcTime).Seconds()
	if elapsed <= 0 {
		return
	}

	uploadDelta := sm.current.TotalUpload - sm.lastUpload
	downloadDelta := sm.current.TotalDownload - sm.lastDownload

	sm.current.UploadSpeed = int64(float64(uploadDelta) / elapsed)
	sm.current.DownloadSpeed = int64(float64(downloadDelta) / elapsed)

	sm.lastUpload = sm.current.TotalUpload
	sm.lastDownload = sm.current.TotalDownload
	sm.lastCalcTime = now

	if sm.updateCallback != nil {
		cb := sm.updateCallback
		go cb(sm.current)
	}
}

func (sm *StatsManager) AddUpload(bytes int64) {
	atomic.AddInt64(&sm.current.TotalUpload, bytes)
	atomic.AddInt64(&sm.current.PacketsSent, 1)
}

func (sm *StatsManager) AddDownload(bytes int64) {
	atomic.AddInt64(&sm.current.TotalDownload, bytes)
	atomic.AddInt64(&sm.current.PacketsRecv, 1)
}

func (sm *StatsManager) SetActiveConns(n int) {
	sm.mu.Lock()
	sm.current.ActiveConns = n
	sm.mu.Unlock()
}

func (sm *StatsManager) GetStats() TrafficStats {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.current
}

func (sm *StatsManager) ResetStats() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.current = TrafficStats{}
	sm.lastUpload = 0
	sm.lastDownload = 0
	sm.lastCalcTime = time.Now()
}

func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return formatInt(bytes) + " B"
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return formatFloat(float64(bytes)/float64(div)) + " " + string("KMGTPE"[exp]) + "B"
}

func FormatSpeed(bytesPerSec int64) string {
	return FormatBytes(bytesPerSec) + "/s"
}

func formatInt(n int64) string {
	return formatNum(n, 0)
}

func formatFloat(f float64) string {
	s := formatNum(int64(f*100), 2)
	if len(s) >= 3 {
		return s[:len(s)-2] + "." + s[len(s)-2:]
	}
	return "0." + s
}

func formatNum(n int64, decimals int) string {
	if n < 0 {
		return "-" + formatNum(-n, decimals)
	}
	result := ""
	count := 0
	for n > 0 || count == 0 {
		if count > 0 && count%3 == 0 {
			result = "," + result
		}
		result = string(rune('0'+n%10)) + result
		n /= 10
		count++
	}
	if decimals > 0 {
		for len(result) <= decimals {
			result = "0" + result
		}
	}
	return result
}
