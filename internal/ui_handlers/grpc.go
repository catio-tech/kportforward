package ui_handlers

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/victorkazakov/kportforward/internal/config"
	"github.com/victorkazakov/kportforward/internal/utils"
)

// GRPCUIManager manages gRPC UI processes for RPC services
type GRPCUIManager struct {
	services map[string]*GRPCUIService
	logger   *utils.Logger
	mutex    sync.RWMutex
	enabled  bool
}

// GRPCUIService represents a single gRPC UI instance
type GRPCUIService struct {
	serviceName  string
	localPort    int
	grpcuiPort   int
	cmd          *exec.Cmd
	logFile      string
	startTime    time.Time
	restartCount int
	status       string
}

// NewGRPCUIManager creates a new gRPC UI manager
func NewGRPCUIManager(logger *utils.Logger) *GRPCUIManager {
	return &GRPCUIManager{
		services: make(map[string]*GRPCUIService),
		logger:   logger,
		enabled:  false,
	}
}

// Enable enables gRPC UI management
func (gm *GRPCUIManager) Enable() error {
	// Check if grpcui is available
	if !gm.isGRPCUIAvailable() {
		return fmt.Errorf("grpcui not found in PATH. Install with: go install github.com/fullstorydev/grpcui/cmd/grpcui@latest")
	}

	gm.enabled = true
	gm.logger.Info("gRPC UI manager enabled")
	return nil
}

// Disable disables gRPC UI management and stops all instances
func (gm *GRPCUIManager) Disable() error {
	gm.mutex.Lock()
	defer gm.mutex.Unlock()

	for serviceName := range gm.services {
		if err := gm.stopService(serviceName); err != nil {
			gm.logger.Error("Failed to stop gRPC UI for %s: %v", serviceName, err)
		}
	}

	gm.enabled = false
	gm.logger.Info("gRPC UI manager disabled")
	return nil
}

// StartService starts a gRPC UI instance for the given service
func (gm *GRPCUIManager) StartService(serviceName string, serviceStatus config.ServiceStatus, serviceConfig config.Service) error {
	if !gm.enabled {
		return nil
	}

	// Only start for RPC services that are running
	if serviceConfig.Type != "rpc" || serviceStatus.Status != "Running" {
		return nil
	}

	gm.mutex.Lock()
	defer gm.mutex.Unlock()

	// Check if already running
	if service, exists := gm.services[serviceName]; exists && service.status == "Running" {
		return nil
	}

	// Find available port for gRPC UI (thread-safe)
	grpcuiPort, err := utils.FindAvailablePortSafe(9090)
	if err != nil {
		return fmt.Errorf("failed to find available port for gRPC UI: %w", err)
	}

	// Create log file
	logFile := gm.getLogFilePath(serviceName)
	if err := gm.ensureLogDir(logFile); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Start grpcui process
	cmd, err := gm.startGRPCUIProcess(serviceName, serviceStatus.LocalPort, grpcuiPort, logFile)
	if err != nil {
		return fmt.Errorf("failed to start grpcui process: %w", err)
	}

	// Create service entry
	gm.services[serviceName] = &GRPCUIService{
		serviceName:  serviceName,
		localPort:    serviceStatus.LocalPort,
		grpcuiPort:   grpcuiPort,
		cmd:          cmd,
		logFile:      logFile,
		startTime:    time.Now(),
		restartCount: 0,
		status:       "Running",
	}

	gm.logger.Info("Started gRPC UI for %s on port %d", serviceName, grpcuiPort)
	return nil
}

// StopService stops the gRPC UI instance for the given service
func (gm *GRPCUIManager) StopService(serviceName string) error {
	gm.mutex.Lock()
	defer gm.mutex.Unlock()

	return gm.stopService(serviceName)
}

// stopService stops a service (internal method, assumes lock is held)
func (gm *GRPCUIManager) stopService(serviceName string) error {
	service, exists := gm.services[serviceName]
	if !exists {
		return nil
	}

	if service.cmd != nil && service.cmd.Process != nil {
		if err := utils.KillProcess(service.cmd.Process.Pid); err != nil {
			gm.logger.Warn("Failed to kill gRPC UI process for %s: %v", serviceName, err)
		}
	}

	// Release the allocated port
	utils.ReleasePort(service.grpcuiPort)

	service.status = "Stopped"
	delete(gm.services, serviceName)

	gm.logger.Info("Stopped gRPC UI for %s", serviceName)
	return nil
}

// GetServiceInfo returns information about a gRPC UI service
func (gm *GRPCUIManager) GetServiceInfo(serviceName string) *GRPCUIService {
	gm.mutex.RLock()
	defer gm.mutex.RUnlock()

	service, exists := gm.services[serviceName]
	if !exists {
		return nil
	}

	// Check if process is still running
	if service.cmd != nil && service.cmd.Process != nil {
		if !utils.IsProcessRunning(service.cmd.Process.Pid) {
			service.status = "Failed"
		}
	}

	return service
}

// GetServiceURL returns the URL for accessing the gRPC UI
func (gm *GRPCUIManager) GetServiceURL(serviceName string) string {
	service := gm.GetServiceInfo(serviceName)
	if service == nil || service.status != "Running" {
		return ""
	}

	return fmt.Sprintf("http://localhost:%d", service.grpcuiPort)
}

// IsEnabled returns whether gRPC UI management is enabled
func (gm *GRPCUIManager) IsEnabled() bool {
	return gm.enabled
}

// isGRPCUIAvailable checks if grpcui is available in PATH
func (gm *GRPCUIManager) isGRPCUIAvailable() bool {
	_, err := exec.LookPath("grpcui")
	return err == nil
}

// startGRPCUIProcess starts the grpcui process
func (gm *GRPCUIManager) startGRPCUIProcess(serviceName string, targetPort, grpcuiPort int, logFile string) (*exec.Cmd, error) {
	// grpcui arguments
	args := []string{
		"-bind", "localhost",
		"-port", fmt.Sprintf("%d", grpcuiPort),
		"-plaintext",
		fmt.Sprintf("localhost:%d", targetPort),
	}

	cmd := exec.Command("grpcui", args...)

	// Set up logging
	logFileHandle, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	// Platform-specific process setup
	if err := gm.startGRPCUIProcessPlatform(cmd, logFileHandle); err != nil {
		return nil, err
	}

	return cmd, nil
}

// getLogFilePath returns the log file path for a service
func (gm *GRPCUIManager) getLogFilePath(serviceName string) string {
	logDir := "/tmp"
	if runtime.GOOS == "windows" {
		logDir = os.TempDir()
	}

	filename := fmt.Sprintf("kpf_grpcui_%s.log", strings.ReplaceAll(serviceName, "-", "_"))
	return filepath.Join(logDir, filename)
}

// ensureLogDir ensures the log directory exists
func (gm *GRPCUIManager) ensureLogDir(logFile string) error {
	logDir := filepath.Dir(logFile)
	return os.MkdirAll(logDir, 0755)
}

// MonitorServices monitors all gRPC UI services and restarts failed ones
func (gm *GRPCUIManager) MonitorServices(services map[string]config.ServiceStatus, configs map[string]config.Service) {
	if !gm.enabled {
		return
	}

	gm.mutex.Lock()
	defer gm.mutex.Unlock()

	// Start gRPC UI for new RPC services
	for serviceName, serviceStatus := range services {
		if serviceConfig, exists := configs[serviceName]; exists {
			if serviceConfig.Type == "rpc" && serviceStatus.Status == "Running" {
				if _, uiExists := gm.services[serviceName]; !uiExists {
					go func(name string, status config.ServiceStatus, config config.Service) {
						if err := gm.StartService(name, status, config); err != nil {
							gm.logger.Error("Failed to start gRPC UI for %s: %v", name, err)
						}
					}(serviceName, serviceStatus, serviceConfig)
				}
			}
		}
	}

	// Stop gRPC UI for services that are no longer running
	for serviceName := range gm.services {
		serviceStatus, exists := services[serviceName]
		if !exists || serviceStatus.Status != "Running" {
			go func(name string) {
				if err := gm.StopService(name); err != nil {
					gm.logger.Error("Failed to stop gRPC UI for %s: %v", name, err)
				}
			}(serviceName)
		}
	}
}
