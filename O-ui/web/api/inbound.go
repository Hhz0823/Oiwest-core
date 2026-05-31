package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Hhz0823/oiwest-core/O-ui/database"
	"github.com/Hhz0823/oiwest-core/O-ui/model"
)

// GET /api/inbounds
func GetInbounds(w http.ResponseWriter, r *http.Request) {
	db := database.GetDB()
	rows, err := db.Query("SELECT id, tag, protocol, port, listen, settings, stream_settings, enabled, remark, traffic_up, traffic_down, created_at FROM inbounds ORDER BY id DESC")
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	defer rows.Close()

	var list []model.Inbound
	for rows.Next() {
		var in model.Inbound
		var enabled int
		rows.Scan(&in.ID, &in.Tag, &in.Protocol, &in.Port, &in.Listen, &in.Settings, &in.StreamSettings, &enabled, &in.Remark, &in.TrafficUp, &in.TrafficDown, &in.CreatedAt)
		in.Enabled = enabled == 1
		list = append(list, in)
	}
	success(w, list)
}

// POST /api/inbounds
func AddInbound(w http.ResponseWriter, r *http.Request) {
	var in model.Inbound
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		fail(w, 400, "invalid request")
		return
	}
	db := database.GetDB()
	enabled := 0
	if in.Enabled { enabled = 1 }
	_, err := db.Exec("INSERT INTO inbounds (tag, protocol, port, listen, settings, stream_settings, enabled, remark) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		in.Tag, in.Protocol, in.Port, in.Listen, in.Settings, in.StreamSettings, enabled, in.Remark)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	success(w, nil)
}

// PUT /api/inbounds/:id
func UpdateInbound(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	var in model.Inbound
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		fail(w, 400, "invalid request")
		return
	}
	db := database.GetDB()
	enabled := 0
	if in.Enabled { enabled = 1 }
	_, err := db.Exec("UPDATE inbounds SET tag=?, protocol=?, port=?, listen=?, settings=?, stream_settings=?, enabled=?, remark=? WHERE id=?",
		in.Tag, in.Protocol, in.Port, in.Listen, in.Settings, in.StreamSettings, enabled, in.Remark, id)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	success(w, nil)
}

// DELETE /api/inbounds/:id
func DeleteInbound(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	db := database.GetDB()
	db.Exec("DELETE FROM inbounds WHERE id=?", id)
	success(w, nil)
}

// GET /api/outbounds
func GetOutbounds(w http.ResponseWriter, r *http.Request) {
	db := database.GetDB()
	rows, err := db.Query("SELECT id, tag, protocol, settings, stream_settings, enabled, remark, created_at FROM outbounds ORDER BY id DESC")
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	defer rows.Close()

	var list []model.Outbound
	for rows.Next() {
		var o model.Outbound
		var enabled int
		rows.Scan(&o.ID, &o.Tag, &o.Protocol, &o.Settings, &o.StreamSettings, &enabled, &o.Remark, &o.CreatedAt)
		o.Enabled = enabled == 1
		list = append(list, o)
	}
	success(w, list)
}

// POST /api/outbounds
func AddOutbound(w http.ResponseWriter, r *http.Request) {
	var o model.Outbound
	if err := json.NewDecoder(r.Body).Decode(&o); err != nil {
		fail(w, 400, "invalid request")
		return
	}
	db := database.GetDB()
	enabled := 0
	if o.Enabled { enabled = 1 }
	_, err := db.Exec("INSERT INTO outbounds (tag, protocol, settings, stream_settings, enabled, remark) VALUES (?, ?, ?, ?, ?, ?)",
		o.Tag, o.Protocol, o.Settings, o.StreamSettings, enabled, o.Remark)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	success(w, nil)
}

// DELETE /api/outbounds/:id
func DeleteOutbound(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	db := database.GetDB()
	db.Exec("DELETE FROM outbounds WHERE id=?", id)
	success(w, nil)
}
