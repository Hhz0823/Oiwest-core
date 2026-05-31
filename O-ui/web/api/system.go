package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/Hhz0823/oiwest-core/O-ui/database"
	"github.com/Hhz0823/oiwest-core/O-ui/model"
)

var coreStartTime time.Time
var coreRunning bool

func init() {
	coreStartTime = time.Now()
}

func GetStats(w http.ResponseWriter, r *http.Request) {
	db := database.GetDB()
	var totalUp, totalDown int64
	db.QueryRow("SELECT COALESCE(SUM(traffic_up),0), COALESCE(SUM(traffic_down),0) FROM inbounds").Scan(&totalUp, &totalDown)
	var nodeUp, nodeDown int64
	db.QueryRow("SELECT COALESCE(SUM(traffic_up),0), COALESCE(SUM(traffic_down),0) FROM nodes").Scan(&nodeUp, &nodeDown)
	success(w, map[string]interface{}{
		"inbound_up": totalUp, "inbound_down": totalDown,
		"node_up": nodeUp, "node_down": nodeDown,
		"total_up": totalUp + nodeUp, "total_down": totalDown + nodeDown,
	})
}

func GetTrafficHistory(w http.ResponseWriter, r *http.Request) {
	hours := 24
	if h := r.URL.Query().Get("hours"); h != "" {
		fmt.Sscanf(h, "%d", &hours)
	}
	db := database.GetDB()
	rows, err := db.Query("SELECT strftime('%H:00', recorded_at) as hour, SUM(upload), SUM(download) FROM traffic_log WHERE recorded_at > datetime('now', ?) GROUP BY hour ORDER BY hour", fmt.Sprintf("-%d hours", hours))
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	defer rows.Close()
	var history []map[string]interface{}
	for rows.Next() {
		var hour string
		var up, down int64
		rows.Scan(&hour, &up, &down)
		history = append(history, map[string]interface{}{"hour": hour, "upload": up, "download": down})
	}
	success(w, history)
}

func GetSystemInfo(w http.ResponseWriter, r *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	hostname, _ := os.Hostname()
	success(w, map[string]interface{}{
		"hostname": hostname, "os": runtime.GOOS, "arch": runtime.GOARCH,
		"go_version": runtime.Version(), "num_cpu": runtime.NumCPU(),
		"num_goroutine": runtime.NumGoroutine(), "mem_alloc": m.Alloc,
		"mem_sys": m.Sys, "mem_total": m.TotalAlloc, "heap_alloc": m.HeapAlloc,
		"heap_sys": m.HeapSys, "gc_count": m.NumGC, "uptime": time.Since(coreStartTime).String(),
	})
}

func GetCoreStatus(w http.ResponseWriter, r *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	inboundCount, outboundCount := 0, 0
	db := database.GetDB()
	db.QueryRow("SELECT COUNT(*) FROM inbounds WHERE enabled=1").Scan(&inboundCount)
	db.QueryRow("SELECT COUNT(*) FROM outbounds WHERE enabled=1").Scan(&outboundCount)
	success(w, model.CoreStatus{
		Running: coreRunning, Version: "2.1.0", Uptime: time.Since(coreStartTime).String(),
		PID: os.Getpid(), MemUsage: int64(m.Alloc), Inbounds: inboundCount, Outbounds: outboundCount,
	})
}

func StartCore(w http.ResponseWriter, r *http.Request) {
	coreRunning = true
	coreStartTime = time.Now()
	success(w, map[string]string{"status": "started"})
}

func StopCore(w http.ResponseWriter, r *http.Request) {
	coreRunning = false
	success(w, map[string]string{"status": "stopped"})
}

func RestartCore(w http.ResponseWriter, r *http.Request) {
	coreRunning = false
	time.Sleep(100 * time.Millisecond)
	coreRunning = true
	coreStartTime = time.Now()
	success(w, map[string]string{"status": "restarted"})
}

func GetSettings(w http.ResponseWriter, r *http.Request) {
	db := database.GetDB()
	rows, err := db.Query("SELECT key, value FROM settings")
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	defer rows.Close()
	settings := make(map[string]string)
	for rows.Next() {
		var k, v string
		rows.Scan(&k, &v)
		settings[k] = v
	}
	success(w, settings)
}

func UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var settings map[string]string
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		fail(w, 400, "invalid request")
		return
	}
	db := database.GetDB()
	for k, v := range settings {
		db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)", k, v)
	}
	success(w, nil)
}

func GetSubscription(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	if token == "" {
		fail(w, 400, "missing token")
		return
	}
	db := database.GetDB()
	rows, err := db.Query("SELECT name, address, port, protocol, uuid, password, method FROM nodes WHERE enabled=1")
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	defer rows.Close()
	var links []string
	for rows.Next() {
		var n model.Node
		rows.Scan(&n.Name, &n.Address, &n.Port, &n.Protocol, &n.UUID, &n.Password, &n.Method)
		switch n.Protocol {
		case "vmess":
			links = append(links, fmt.Sprintf("vmess://%s@%s:%d#%s", n.UUID, n.Address, n.Port, n.Name))
		case "vless":
			links = append(links, fmt.Sprintf("vless://%s@%s:%d#%s", n.UUID, n.Address, n.Port, n.Name))
		case "trojan":
			links = append(links, fmt.Sprintf("trojan://%s@%s:%d#%s", n.Password, n.Address, n.Port, n.Name))
		case "shadowsocks":
			links = append(links, fmt.Sprintf("ss://%s@%s:%d#%s", n.Method, n.Address, n.Port, n.Name))
		}
	}
	w.Header().Set("Content-Type", "text/plain")
	for _, link := range links {
		fmt.Fprintln(w, link)
	}
}

