package model

import "time"

type User struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Password  string    `json:"-"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	LastLogin time.Time `json:"last_login"`
}

type Inbound struct {
	ID             int64  `json:"id"`
	Tag            string `json:"tag"`
	Protocol       string `json:"protocol"`
	Port           int    `json:"port"`
	Listen         string `json:"listen"`
	Settings       string `json:"settings"`
	StreamSettings string `json:"stream_settings"`
	Enabled        bool   `json:"enabled"`
	Remark         string `json:"remark"`
	TrafficUp      int64  `json:"traffic_up"`
	TrafficDown    int64  `json:"traffic_down"`
	CreatedAt      string `json:"created_at"`
}

type Outbound struct {
	ID             int64  `json:"id"`
	Tag            string `json:"tag"`
	Protocol       string `json:"protocol"`
	Settings       string `json:"settings"`
	StreamSettings string `json:"stream_settings"`
	Enabled        bool   `json:"enabled"`
	Remark         string `json:"remark"`
	CreatedAt      string `json:"created_at"`
}

type Node struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Address     string `json:"address"`
	Port        int    `json:"port"`
	Protocol    string `json:"protocol"`
	UUID        string `json:"uuid"`
	Password    string `json:"password"`
	Method      string `json:"method"`
	Settings    string `json:"settings"`
	GroupName   string `json:"group_name"`
	Enabled     bool   `json:"enabled"`
	TrafficUp   int64  `json:"traffic_up"`
	TrafficDown int64  `json:"traffic_down"`
	LastCheck   string `json:"last_check"`
	Latency     int    `json:"latency"`
	CreatedAt   string `json:"created_at"`
}

type TrafficLog struct {
	ID          int64  `json:"id"`
	InboundTag  string `json:"inbound_tag"`
	NodeID      int64  `json:"node_id"`
	Upload      int64  `json:"upload"`
	Download    int64  `json:"download"`
	RecordedAt  string `json:"recorded_at"`
}

type Setting struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type CoreStatus struct {
	Running    bool   `json:"running"`
	Version    string `json:"version"`
	Uptime     string `json:"uptime"`
	PID        int    `json:"pid"`
	MemUsage   int64  `json:"mem_usage"`
	CPUUsage   float64 `json:"cpu_usage"`
	Inbounds   int    `json:"inbounds"`
	Outbounds  int    `json:"outbounds"`
	TotalUp    int64  `json:"total_up"`
	TotalDown  int64  `json:"total_down"`
	ActiveConn int64  `json:"active_conn"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token    string `json:"token"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}
