package services

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type ServerProtocol string

const (
	ProtocolVMess       ServerProtocol = "vmess"
	ProtocolVLess       ServerProtocol = "vless"
	ProtocolTrojan      ServerProtocol = "trojan"
	ProtocolShadowsocks ServerProtocol = "shadowsocks"
	ProtocolSOCKS       ServerProtocol = "socks"
	ProtocolHTTP        ServerProtocol = "http"
)

type ServerNode struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Group       string         `json:"group"`
	Protocol    ServerProtocol `json:"protocol"`
	Address     string         `json:"address"`
	Port        int            `json:"port"`
	UUID        string         `json:"uuid,omitempty"`
	Password    string         `json:"password,omitempty"`
	Security    string         `json:"security,omitempty"`
	Flow        string         `json:"flow,omitempty"`
	Network     string         `json:"network,omitempty"`
	Path        string         `json:"path,omitempty"`
	Host        string         `json:"host,omitempty"`
	TLS         bool           `json:"tls"`
	SNI         string         `json:"sni,omitempty"`
	Fingerprint string         `json:"fingerprint,omitempty"`
	PublicKey   string         `json:"publicKey,omitempty"`
	ShortID     string         `json:"shortId,omitempty"`
	SpiderX     string         `json:"spiderX,omitempty"`
	AllowInsecure bool         `json:"allowInsecure"`
	Latency     int64          `json:"latency"`
	Upload      int64          `json:"upload"`
	Download    int64          `json:"download"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
}

type ServerGroup struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type NodeManager struct {
	mu        sync.RWMutex
	nodes     []*ServerNode
	dataPath  string
}

var nodeManager *NodeManager

func GetNodeManager() *NodeManager {
	if nodeManager == nil {
		nodeManager = &NodeManager{
			nodes:    make([]*ServerNode, 0),
			dataPath: "",
		}
	}
	return nodeManager
}

func (nm *NodeManager) SetDataPath(path string) {
	nm.dataPath = path
	nm.LoadNodes()
}

func (nm *NodeManager) configFilePath() string {
	return filepath.Join(nm.dataPath, "servers.json")
}

func (nm *NodeManager) LoadNodes() error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	data, err := os.ReadFile(nm.configFilePath())
	if err != nil {
		nm.nodes = make([]*ServerNode, 0)
		return nil
	}

	return json.Unmarshal(data, &nm.nodes)
}

func (nm *NodeManager) SaveNodes() error {
	nm.mu.RLock()
	data, err := json.MarshalIndent(nm.nodes, "", "  ")
	nm.mu.RUnlock()
	if err != nil {
		return err
	}
	return os.WriteFile(nm.configFilePath(), data, 0644)
}

func (nm *NodeManager) AddNode(node *ServerNode) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	if node.ID == "" {
		node.ID = fmt.Sprintf("node_%d", time.Now().UnixNano())
	}
	node.CreatedAt = time.Now()
	node.UpdatedAt = time.Now()
	if node.Group == "" {
		node.Group = "默认分组"
	}

	nm.nodes = append(nm.nodes, node)
	return nm.saveNodesUnsafe()
}

func (nm *NodeManager) UpdateNode(node *ServerNode) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	for i, n := range nm.nodes {
		if n.ID == node.ID {
			node.UpdatedAt = time.Now()
			node.CreatedAt = n.CreatedAt
			nm.nodes[i] = node
			return nm.saveNodesUnsafe()
		}
	}
	return fmt.Errorf("node %s not found", node.ID)
}

func (nm *NodeManager) DeleteNode(id string) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	for i, n := range nm.nodes {
		if n.ID == id {
			nm.nodes = append(nm.nodes[:i], nm.nodes[i+1:]...)
			return nm.saveNodesUnsafe()
		}
	}
	return fmt.Errorf("node %s not found", id)
}

func (nm *NodeManager) GetAllNodes() []*ServerNode {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	result := make([]*ServerNode, len(nm.nodes))
	copy(result, nm.nodes)
	return result
}

func (nm *NodeManager) GetNode(id string) *ServerNode {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	for _, n := range nm.nodes {
		if n.ID == id {
			cp := *n
			return &cp
		}
	}
	return nil
}

func (nm *NodeManager) GetGroups() []ServerGroup {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	groupMap := make(map[string]int)
	for _, n := range nm.nodes {
		groupMap[n.Group]++
	}
	groups := make([]ServerGroup, 0, len(groupMap))
	for name, count := range groupMap {
		groups = append(groups, ServerGroup{Name: name, Count: count})
	}
	return groups
}

func (nm *NodeManager) MoveNode(id string, newGroup string) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	for _, n := range nm.nodes {
		if n.ID == id {
			n.Group = newGroup
			n.UpdatedAt = time.Now()
			return nm.saveNodesUnsafe()
		}
	}
	return fmt.Errorf("node %s not found", id)
}

func (nm *NodeManager) UpdateNodeStats(id string, upload int64, download int64, latency int64) {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	for _, n := range nm.nodes {
		if n.ID == id {
			n.Upload += upload
			n.Download += download
			if latency > 0 {
				n.Latency = latency
			}
			return
		}
	}
}

func (nm *NodeManager) NodeCount() int {
	nm.mu.RLock()
	defer nm.mu.RUnlock()
	return len(nm.nodes)
}

func (nm *NodeManager) saveNodesUnsafe() error {
	data, err := json.MarshalIndent(nm.nodes, "", "  ")
	if err != nil {
		return err
	}
	if nm.dataPath == "" {
		return nil
	}
	return os.WriteFile(nm.configFilePath(), data, 0644)
}
