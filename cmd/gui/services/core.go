package services

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

type CoreStatus string

const (
	CoreStatusRunning  CoreStatus = "running"
	CoreStatusStopped  CoreStatus = "stopped"
	CoreStatusStarting CoreStatus = "starting"
	CoreStatusError    CoreStatus = "error"
)

type CoreManager struct {
	mu          sync.Mutex
	exePath     string
	configPath  string
	dataPath    string
	cmd         *exec.Cmd
	cancel      context.CancelFunc
	status      CoreStatus
	running     int32
	startTime   time.Time
	logBuffer   []string
	logMaxSize  int
	logCallback func(string)
}

var coreManager *CoreManager

func GetCoreManager() *CoreManager {
	if coreManager == nil {
		coreManager = &CoreManager{
			status:     CoreStatusStopped,
			logBuffer:  make([]string, 0),
			logMaxSize: 1000,
		}
	}
	return coreManager
}

func (cm *CoreManager) SetPaths(exePath, configPath, dataPath string) {
	cm.exePath = exePath
	cm.configPath = configPath
	cm.dataPath = dataPath
}

func (cm *CoreManager) SetLogCallback(cb func(string)) {
	cm.logCallback = cb
}

func (cm *CoreManager) GetStatus() CoreStatus {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	return cm.status
}

func (cm *CoreManager) Start() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.status == CoreStatusRunning {
		return fmt.Errorf("核心已在运行中")
	}

	if cm.exePath == "" {
		exe, err := os.Executable()
		if err == nil {
			base := filepath.Join(filepath.Dir(exe), "oiwest-core")
			if runtime.GOOS == "windows" {
				base += ".exe"
			}
			cm.exePath = base
		}
	}

	if cm.configPath == "" {
		cm.configPath = filepath.Join(cm.dataPath, "config.json")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cm.cancel = cancel

	cm.status = CoreStatusStarting
	atomic.StoreInt32(&cm.running, 0)

	go cm.runCore(ctx)

	return nil
}

func (cm *CoreManager) runCore(ctx context.Context) {
	args := []string{
		"-config", cm.configPath,
		"-test",
	}

	cmd := exec.CommandContext(ctx, cm.exePath, args...)
	cmd.Dir = cm.dataPath

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		cm.mu.Lock()
		cm.status = CoreStatusError
		cm.mu.Unlock()
		cm.addLog(fmt.Sprintf("[错误] 启动失败: %v", err))
		return
	}

	cm.mu.Lock()
	cm.cmd = cmd
	cm.status = CoreStatusRunning
	cm.startTime = time.Now()
	atomic.StoreInt32(&cm.running, 1)
	cm.mu.Unlock()

	cm.addLog("[信息] Oiwest Core 已启动")

	go cm.readLogs(stdout)
	go cm.readLogs(stderr)

	if err := cmd.Wait(); err != nil {
		cm.addLog(fmt.Sprintf("[信息] 核心进程已退出: %v", err))
	}

	cm.mu.Lock()
	if cm.status == CoreStatusRunning {
		cm.status = CoreStatusStopped
	}
	atomic.StoreInt32(&cm.running, 0)
	cm.mu.Unlock()
}

func (cm *CoreManager) readLogs(reader io.Reader) {
	buf := make([]byte, 1024)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			cm.addLog(string(buf[:n]))
		}
		if err != nil {
			break
		}
	}
}

func (cm *CoreManager) Stop() error {
	cm.mu.Lock()
	if cm.status != CoreStatusRunning {
		cm.mu.Unlock()
		return nil
	}
	cancelFn := cm.cancel
	var cmdProc *os.Process
	if cm.cmd != nil {
		cmdProc = cm.cmd.Process
	}
	cm.status = CoreStatusStopped
	atomic.StoreInt32(&cm.running, 0)
	cm.mu.Unlock()

	if cancelFn != nil {
		cancelFn()
	}
	if cmdProc != nil {
		cmdProc.Signal(os.Interrupt)
		time.Sleep(500 * time.Millisecond)
		cmdProc.Kill()
	}

	cm.addLog("[信息] Oiwest Core 已停止")
	return nil
}

func (cm *CoreManager) Restart() error {
	if err := cm.Stop(); err != nil {
		log.Printf("Stop error during restart: %v", err)
	}
	time.Sleep(800 * time.Millisecond)
	return cm.Start()
}

func (cm *CoreManager) IsRunning() bool {
	return atomic.LoadInt32(&cm.running) == 1
}

func (cm *CoreManager) Uptime() time.Duration {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	if cm.status != CoreStatusRunning {
		return 0
	}
	return time.Since(cm.startTime)
}

func (cm *CoreManager) StartTime() time.Time {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	return cm.startTime
}

func (cm *CoreManager) addLog(msg string) {
	cm.mu.Lock()
	cm.logBuffer = append(cm.logBuffer, msg)
	if len(cm.logBuffer) > cm.logMaxSize {
		cm.logBuffer = cm.logBuffer[len(cm.logBuffer)-cm.logMaxSize:]
	}
	cb := cm.logCallback
	cm.mu.Unlock()

	if cb != nil {
		cb(msg)
	}
}

func (cm *CoreManager) GetLogs(count int) []string {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	total := len(cm.logBuffer)
	if count <= 0 || count > total {
		count = total
	}
	start := total - count
	if start < 0 {
		start = 0
	}
	result := make([]string, count)
	copy(result, cm.logBuffer[start:])
	return result
}

func (cm *CoreManager) ClearLogs() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.logBuffer = make([]string, 0)
}

func (cm *CoreManager) GetConfigPath() string {
	return cm.configPath
}

func (cm *CoreManager) SetConfigPath(path string) {
	cm.configPath = path
}

func (cm *CoreManager) SaveConfig(configJSON string) error {
	if cm.configPath == "" {
		return fmt.Errorf("配置文件路径未设置")
	}
	return os.WriteFile(cm.configPath, []byte(configJSON), 0644)
}

func (cm *CoreManager) LoadConfig() (string, error) {
	if cm.configPath == "" {
		return "", fmt.Errorf("配置文件路径未设置")
	}
	data, err := os.ReadFile(cm.configPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (cm *CoreManager) GenerateConfig(nodeID string) (string, error) {
	n := GetNodeManager().GetNode(nodeID)
	if n == nil {
		return "", fmt.Errorf("节点不存在: %s", nodeID)
	}
	return GenerateConfigJSON(n), nil
}
