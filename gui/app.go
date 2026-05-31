package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Hhz0823/oiwest-core/gui/services"
	"github.com/Hhz0823/oiwest-core/app/common/platform"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	wailsWindows "github.com/wailsapp/wails/v2/pkg/options/windows"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

type App struct {
	ctx      context.Context
	nodeMgr  *services.NodeManager
	coreMgr  *services.CoreManager
	proxyMgr *services.ProxyManager
	statsMgr *services.StatsManager
	netMgr   *services.NetworkConfigManager
	sysMgr   *services.SysInfoManager
	tlsMgr   *services.TLSCertManager
	dataPath string
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	p := platform.Get()
	a.dataPath = p.AppDir
	os.MkdirAll(a.dataPath, 0755)

	a.nodeMgr = services.GetNodeManager()
	a.coreMgr = services.GetCoreManager()
	a.proxyMgr = services.GetProxyManager()
	a.statsMgr = services.GetStatsManager()
	a.netMgr = services.GetNetworkConfigManager()
	a.sysMgr = services.GetSysInfoManager()
	a.tlsMgr = services.GetTLSCertManager()

	a.nodeMgr.SetDataPath(a.dataPath)
	a.coreMgr.SetPaths("", "", a.dataPath)
	a.netMgr.SetDataPath(a.dataPath)
	a.tlsMgr.SetDataPath(a.dataPath)
	a.statsMgr.StartMonitoring()

	exePath, _ := os.Executable()
	exeDir := filepath.Dir(exePath)

	exeName := platform.Get().ExeName()
	searchPaths := []string{
		filepath.Join(exeDir, exeName),
		filepath.Join(filepath.Dir(exeDir), exeName),
		filepath.Join(a.dataPath, exeName),
		exeName,
	}

	for _, corePath := range searchPaths {
		if _, err := os.Stat(corePath); err == nil {
			a.coreMgr.SetPaths(corePath, "", a.dataPath)
			break
		}
	}

	a.coreMgr.SetLogCallback(func(msg string) {
		runtime.EventsEmit(a.ctx, "core-log", msg)
	})

	a.statsMgr.SetUpdateCallback(func(stats services.TrafficStats) {
		runtime.EventsEmit(a.ctx, "stats-update", stats)
	})
}

func (a *App) shutdown(ctx context.Context) {
	a.coreMgr.Stop()
	a.statsMgr.StopMonitoring()
	a.nodeMgr.SaveNodes()
}

func (a *App) IsKernelInstalled() bool {
	exePath, _ := os.Executable()
	exeDir := filepath.Dir(exePath)
	exeName := platform.Get().ExeName()

	searchPaths := []string{
		filepath.Join(exeDir, exeName),
		filepath.Join(filepath.Dir(exeDir), exeName),
		filepath.Join(a.dataPath, exeName),
		exeName,
	}

	for _, p := range searchPaths {
		if _, err := os.Stat(p); err == nil {
			return true
		}
	}
	return false
}

func (a *App) GetKernelPath() string {
	exePath, _ := os.Executable()
	exeDir := filepath.Dir(exePath)
	exeName := platform.Get().ExeName()

	searchPaths := []string{
		filepath.Join(exeDir, exeName),
		filepath.Join(filepath.Dir(exeDir), exeName),
		filepath.Join(a.dataPath, exeName),
	}

	for _, p := range searchPaths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func (a *App) GetKernelStatus() map[string]interface{} {
	installed := a.IsKernelInstalled()
	path := a.GetKernelPath()
	status := string(a.coreMgr.GetStatus())
	running := a.coreMgr.IsRunning()

	return map[string]interface{}{
		"installed": installed,
		"path":      path,
		"status":    status,
		"running":   running,
	}
}

func (a *App) DownloadKernel(url string) error {
	runtime.EventsEmit(a.ctx, "core-log", "[info] Kernel download requires manual placement of oiwest-core binary")
	return fmt.Errorf("please manually place oiwest-core binary in the program directory")
}

func (a *App) GetNodes() []*services.ServerNode {
	return a.nodeMgr.GetAllNodes()
}

func (a *App) AddNode(node *services.ServerNode) error {
	return a.nodeMgr.AddNode(node)
}

func (a *App) UpdateNode(node *services.ServerNode) error {
	return a.nodeMgr.UpdateNode(node)
}

func (a *App) DeleteNode(id string) error {
	return a.nodeMgr.DeleteNode(id)
}

func (a *App) GetGroups() []services.ServerGroup {
	return a.nodeMgr.GetGroups()
}

func (a *App) MoveNode(id string, newGroup string) error {
	return a.nodeMgr.MoveNode(id, newGroup)
}

func (a *App) ImportFromLink(link string) (*services.ServerNode, error) {
	node, err := services.ParseVmessLink(link)
	if err != nil {
		return nil, err
	}
	if err := a.nodeMgr.AddNode(node); err != nil {
		return nil, err
	}
	return node, nil
}

func (a *App) TestNodeLatency(nodeID string) services.LatencyResult {
	node := a.nodeMgr.GetNode(nodeID)
	if node == nil {
		return services.LatencyResult{NodeID: nodeID, Success: false, Error: "node not found"}
	}
	result := services.TestNodeLatency(node.Address, node.Port, 5*time.Second)
	result.NodeID = nodeID
	if result.Success {
		a.nodeMgr.UpdateNodeStats(nodeID, 0, 0, result.Latency)
	}
	return result
}

func (a *App) TestAllNodesLatency() []services.LatencyResult {
	nodes := a.nodeMgr.GetAllNodes()
	results := make([]services.LatencyResult, 0, len(nodes))

	type testResult struct {
		result services.LatencyResult
	}
	ch := make(chan testResult, len(nodes))

	for _, node := range nodes {
		go func(n *services.ServerNode) {
			r := services.TestNodeLatency(n.Address, n.Port, 5*time.Second)
			r.NodeID = n.ID
			if r.Success {
				a.nodeMgr.UpdateNodeStats(n.ID, 0, 0, r.Latency)
			}
			ch <- testResult{result: r}
		}(node)
	}

	for range nodes {
		r := <-ch
		runtime.EventsEmit(a.ctx, "latency-result", r.result)
		results = append(results, r.result)
	}

	return results
}

func (a *App) GetNodeIPs(nodeID string) []string {
	node := a.nodeMgr.GetNode(nodeID)
	if node == nil {
		return nil
	}
	ips, err := services.ResolveNodeIP(node.Address)
	if err != nil {
		return []string{err.Error()}
	}
	return ips
}

func (a *App) StartCore() error      { return a.coreMgr.Start() }
func (a *App) StopCore() error       { return a.coreMgr.Stop() }
func (a *App) RestartCore() error    { return a.coreMgr.Restart() }
func (a *App) GetCoreStatus() string { return string(a.coreMgr.GetStatus()) }

func (a *App) GetCoreUptime() string {
	d := a.coreMgr.Uptime()
	if d <= 0 {
		return "00:00:00"
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

func (a *App) SelectNode(nodeID string) error {
	node := a.nodeMgr.GetNode(nodeID)
	if node == nil {
		return fmt.Errorf("node not found: %s", nodeID)
	}

	baseConfig := services.GenerateConfigJSON(node)
	netConfig := a.netMgr.BuildNetworkConfigJSON()

	var baseMap, netMap map[string]interface{}
	json.Unmarshal([]byte(baseConfig), &baseMap)
	json.Unmarshal([]byte(netConfig), &netMap)

	if inbounds, ok := netMap["inbounds"]; ok {
		baseMap["inbounds"] = inbounds
	}
	if routing, ok := netMap["routing"]; ok {
		baseMap["routing"] = routing
	}
	if dns, ok := netMap["dns"]; ok {
		baseMap["dns"] = dns
	}

	if tp := a.netMgr.GetTransparentProxyConfig(); tp.Enabled {
		if routing, ok := baseMap["routing"].(map[string]interface{}); ok {
			rules := routing["rules"].([]interface{})
			rules = append(rules, map[string]interface{}{
				"type":        "field",
				"inboundTag":  []string{"transparent"},
				"outboundTag": "proxy",
			})
			routing["rules"] = rules
		}
	}

	merged, _ := json.MarshalIndent(baseMap, "", "  ")
	if err := a.coreMgr.SaveConfig(string(merged)); err != nil {
		return err
	}

	a.coreMgr.Restart()
	return nil
}

func (a *App) GetActiveNodeID() string {
	data, err := a.coreMgr.LoadConfig()
	if err != nil {
		return ""
	}
	type vnextItem struct {
		Address string `json:"address"`
		Port    int    `json:"port"`
		Users   []struct {
			ID string `json:"id"`
		} `json:"users"`
	}
	type outboundItem struct {
		Tag      string `json:"tag"`
		Settings struct {
			Vnext []vnextItem `json:"vnext"`
		} `json:"settings"`
	}
	type configWrapper struct {
		Outbounds []outboundItem `json:"outbounds"`
	}
	var cfg configWrapper
	json.Unmarshal([]byte(data), &cfg)
	for _, ob := range cfg.Outbounds {
		for _, vn := range ob.Settings.Vnext {
			for _, node := range a.nodeMgr.GetAllNodes() {
				if node.Address == vn.Address && node.Port == vn.Port {
					return node.ID
				}
			}
		}
	}
	return ""
}

func (a *App) GetProxySettings() services.ProxySettings {
	return a.proxyMgr.GetSettings()
}

func (a *App) SetProxySettings(settings services.ProxySettings) error {
	a.proxyMgr.UpdateSettings(settings)
	return nil
}

func (a *App) EnableProxy() error                     { return a.proxyMgr.Enable() }
func (a *App) DisableProxy() error                    { return a.proxyMgr.Disable() }
func (a *App) ToggleProxy() error                     { return a.proxyMgr.Toggle() }
func (a *App) IsProxyEnabled() bool                   { return a.proxyMgr.IsEnabled() }
func (a *App) GetTrafficStats() services.TrafficStats { return a.statsMgr.GetStats() }

func (a *App) ResetTrafficStats() {
	a.statsMgr.ResetStats()
}

func (a *App) GetCoreLogs(count int) []string {
	return a.coreMgr.GetLogs(count)
}

func (a *App) GetFilteredLogs(category string, count int) []string {
	allLogs := a.coreMgr.GetLogs(count)
	if category == "" || category == "all" {
		return allLogs
	}
	prefix := ""
	switch category {
	case "error":
		prefix = "[error]"
	case "warn":
		prefix = "[warn]"
	case "info":
		prefix = "[info]"
	case "debug":
		prefix = "[debug]"
	default:
		return allLogs
	}
	filtered := make([]string, 0)
	for _, log := range allLogs {
		if len(log) >= len(prefix) && log[:len(prefix)] == prefix {
			filtered = append(filtered, log)
		}
	}
	return filtered
}

func (a *App) ClearCoreLogs() {
	a.coreMgr.ClearLogs()
}

func (a *App) CopyLogs(logs []string) {
	var text string
	for _, l := range logs {
		text += l + "\n"
	}
	runtime.ClipboardSetText(a.ctx, text)
}

func (a *App) GetInbounds() []services.InboundRule {
	return a.netMgr.GetInbounds()
}

func (a *App) AddInbound(rule services.InboundRule) error {
	return a.netMgr.AddInbound(rule)
}

func (a *App) UpdateInbound(rule services.InboundRule) error {
	return a.netMgr.UpdateInbound(rule)
}

func (a *App) DeleteInbound(id string) error {
	return a.netMgr.DeleteInbound(id)
}

func (a *App) ToggleInbound(id string, enabled bool) error {
	return a.netMgr.ToggleInbound(id, enabled)
}

func (a *App) GetRoutingRules() []services.RoutingRule {
	return a.netMgr.GetRoutingRules()
}

func (a *App) AddRoutingRule(rule services.RoutingRule) error {
	return a.netMgr.AddRoutingRule(rule)
}

func (a *App) UpdateRoutingRule(rule services.RoutingRule) error {
	return a.netMgr.UpdateRoutingRule(rule)
}

func (a *App) DeleteRoutingRule(id string) error {
	return a.netMgr.DeleteRoutingRule(id)
}

func (a *App) ReorderRoutingRules(orderedIDs []string) error {
	return a.netMgr.ReorderRoutingRules(orderedIDs)
}

func (a *App) GetDNSConfig() services.DNSConfig {
	return a.netMgr.GetDNSConfig()
}

func (a *App) SetDNSConfig(cfg services.DNSConfig) error {
	return a.netMgr.SetDNSConfig(cfg)
}

func (a *App) AddDNSServer(server services.DNSServerItem) error {
	return a.netMgr.AddDNSServer(server)
}

func (a *App) RemoveDNSServer(index int) error {
	return a.netMgr.RemoveDNSServer(index)
}

func (a *App) GetTransparentProxyConfig() services.TransparentProxyConfig {
	return a.netMgr.GetTransparentProxyConfig()
}

func (a *App) SetTransparentProxyConfig(cfg services.TransparentProxyConfig) error {
	return a.netMgr.SetTransparentProxyConfig(cfg)
}

func (a *App) GetAppVersion() string {
	return "1.1.0"
}

func (a *App) GetDeviceInfo() services.DeviceInfo   { return a.sysMgr.GetDeviceInfo() }
func (a *App) GetSystemUsage() services.SystemUsage { return a.sysMgr.GetSystemUsage() }
func (a *App) GetPublicIP() string                  { return a.sysMgr.GetPublicIP() }
func (a *App) RefreshPublicIP() string              { return a.sysMgr.RefreshPublicIP() }

func (a *App) GenerateTLS(path string) services.TLSKeyPair {
	result, err := a.tlsMgr.GenerateCertificate(path)
	if err != nil {
		return services.TLSKeyPair{}
	}
	return *result
}

func (a *App) GenerateRealityKeys() map[string]string {
	result, _ := a.tlsMgr.GenerateRealityKeys()
	return result
}

func (a *App) GenerateQRCode(nodeID string) string {
	node := a.nodeMgr.GetNode(nodeID)
	if node == nil {
		return ""
	}
	return fmt.Sprintf("vmess://%s:%d@%s", node.UUID, node.Port, node.Address)
}

func (a *App) SelectFile(title string, filter string) string {
	file, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: title,
		Filters: []runtime.FileFilter{
			{DisplayName: filter, Pattern: "*.json"},
		},
	})
	if err != nil {
		return ""
	}
	return file
}

func (a *App) OpenExternalURL(url string) {
	runtime.BrowserOpenURL(a.ctx, url)
}

func main() {
	p := platform.Get()

	if p.IsHeadless {
		fmt.Println("Oiwest Core GUI is not available in headless mode.")
		fmt.Println("Please use the daemon mode: oiwest-daemon")
		os.Exit(1)
	}

	app := NewApp()

	var windowOptions *wailsWindows.Options
	if p.IsWindows() {
		windowOptions = &wailsWindows.Options{
			WebviewIsTransparent: true,
			WindowIsTranslucent:  false,
		}
	}

	err := wails.Run(&options.App{
		Title:            "Oiwest Core",
		Width:            1100,
		Height:           700,
		MinWidth:         860,
		MinHeight:        520,
		AssetServer:      &assetserver.Options{Assets: assets},
		BackgroundColour: &options.RGBA{R: 27, G: 27, B: 27, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind:             []interface{}{app},
		Windows:          windowOptions,
	})

	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err.Error())
	}
}
