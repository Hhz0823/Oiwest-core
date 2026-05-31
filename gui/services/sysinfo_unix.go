//go:build !windows

package services

import (
	"os"
	"strconv"
	"strings"
)

func platformGetTotalMemory() uint64 {
	return platformParseMemInfo("MemTotal:") * 1024
}

func platformGetMemoryUsage() (used, total uint64) {
	totalKb := platformParseMemInfo("MemTotal:")
	availKb := platformParseMemInfo("MemAvailable:")
	if availKb == 0 {
		freeKb := platformParseMemInfo("MemFree:")
		bufKb := platformParseMemInfo("Buffers:")
		cacheKb := platformParseMemInfo("Cached:")
		availKb = freeKb + bufKb + cacheKb
	}
	total = totalKb * 1024
	used = total - (availKb * 1024)
	return
}

func platformParseMemInfo(key string) uint64 {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, key) {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				val, _ := strconv.ParseUint(parts[1], 10, 64)
				return val
			}
		}
	}
	return 0
}

type platformCPUState struct {
	lastIdle  uint64
	lastTotal uint64
}

func platformCPUInit() *platformCPUState {
	return &platformCPUState{}
}

func platformGetCPUPercent(state *platformCPUState) float64 {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "cpu ") {
			fields := strings.Fields(line)
			if len(fields) < 8 {
				return 0
			}
			var total, idle uint64
			for i := 1; i < len(fields); i++ {
				val, _ := strconv.ParseUint(fields[i], 10, 64)
				total += val
				if i == 4 {
					idle = val
				}
			}
			if state.lastTotal == 0 {
				state.lastTotal = total
				state.lastIdle = idle
				return 0
			}
			totalDiff := total - state.lastTotal
			idleDiff := idle - state.lastIdle
			state.lastTotal = total
			state.lastIdle = idle
			if totalDiff == 0 {
				return 0
			}
			return float64(totalDiff-idleDiff) / float64(totalDiff) * 100
		}
	}
	return 0
}
