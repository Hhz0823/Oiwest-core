package services

import (
	"net"
	"time"
)

type LatencyResult struct {
	NodeID  string `json:"nodeId"`
	Latency int64  `json:"latency"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func TestNodeLatency(address string, port int, timeout time.Duration) LatencyResult {
	result := LatencyResult{Success: false}

	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	start := time.Now()
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(address, itoa(port)), timeout)
	elapsed := time.Since(start).Milliseconds()

	if err != nil {
		result.Latency = -1
		result.Error = err.Error()
		result.Success = false
	} else {
		conn.Close()
		result.Latency = elapsed
		result.Success = true
	}

	return result
}

func ResolveNodeIP(address string) ([]string, error) {
	ips, err := net.LookupIP(address)
	if err != nil {
		return nil, err
	}
	result := make([]string, 0, len(ips))
	for _, ip := range ips {
		result = append(result, ip.String())
	}
	return result, nil
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	result := ""
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		result = string(rune('0'+n%10)) + result
		n /= 10
	}
	if neg {
		result = "-" + result
	}
	return result
}
