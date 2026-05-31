package api

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/Hhz0823/oiwest-core/O-ui/database"
	"github.com/Hhz0823/oiwest-core/O-ui/model"
)

// GET /api/nodes
func GetNodes(w http.ResponseWriter, r *http.Request) {
	db := database.GetDB()
	rows, err := db.Query("SELECT id, name, address, port, protocol, uuid, password, method, settings, group_name, enabled, traffic_up, traffic_down, last_check, latency, created_at FROM nodes ORDER BY id DESC")
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	defer rows.Close()

	var list []model.Node
	for rows.Next() {
		var n model.Node
		var enabled int
		var lastCheck, createdAt *string
		rows.Scan(&n.ID, &n.Name, &n.Address, &n.Port, &n.Protocol, &n.UUID, &n.Password, &n.Method, &n.Settings, &n.GroupName, &enabled, &n.TrafficUp, &n.TrafficDown, &lastCheck, &n.Latency, &createdAt)
		n.Enabled = enabled == 1
		if lastCheck != nil { n.LastCheck = *lastCheck }
		if createdAt != nil { n.CreatedAt = *createdAt }
		list = append(list, n)
	}
	success(w, list)
}

// POST /api/nodes
func AddNode(w http.ResponseWriter, r *http.Request) {
	var n model.Node
	if err := json.NewDecoder(r.Body).Decode(&n); err != nil {
		fail(w, 400, "invalid request")
		return
	}
	db := database.GetDB()
	enabled := 0
	if n.Enabled { enabled = 1 }
	_, err := db.Exec("INSERT INTO nodes (name, address, port, protocol, uuid, password, method, settings, group_name, enabled) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		n.Name, n.Address, n.Port, n.Protocol, n.UUID, n.Password, n.Method, n.Settings, n.GroupName, enabled)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	success(w, nil)
}

// PUT /api/nodes/:id
func UpdateNode(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	var n model.Node
	if err := json.NewDecoder(r.Body).Decode(&n); err != nil {
		fail(w, 400, "invalid request")
		return
	}
	db := database.GetDB()
	enabled := 0
	if n.Enabled { enabled = 1 }
	_, err := db.Exec("UPDATE nodes SET name=?, address=?, port=?, protocol=?, uuid=?, password=?, method=?, settings=?, group_name=?, enabled=? WHERE id=?",
		n.Name, n.Address, n.Port, n.Protocol, n.UUID, n.Password, n.Method, n.Settings, n.GroupName, enabled, id)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	success(w, nil)
}

// DELETE /api/nodes/:id
func DeleteNode(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	database.GetDB().Exec("DELETE FROM nodes WHERE id=?", id)
	success(w, nil)
}

// POST /api/nodes/:id/check
func CheckNode(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	db := database.GetDB()
	var addr string
	var port int
	err := db.QueryRow("SELECT address, port FROM nodes WHERE id=?", id).Scan(&addr, &port)
	if err != nil {
		fail(w, 404, "node not found")
		return
	}

	start := time.Now()
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", addr, port), 5*time.Second)
	latency := int(time.Since(start).Milliseconds())
	if err != nil {
		db.Exec("UPDATE nodes SET latency=-1, last_check=? WHERE id=?", time.Now(), id)
		fail(w, 500, fmt.Sprintf("connection failed: %v", err))
		return
	}
	conn.Close()

	db.Exec("UPDATE nodes SET latency=?, last_check=? WHERE id=?", latency, time.Now(), id)
	success(w, map[string]interface{}{"latency": latency})
}
