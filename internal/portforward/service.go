package portforward

import (
	"context"
	"fmt"
	"os/exec"
	"sync"
	"time"

	"github.com/victorkazakov/kportforward/internal/config"
	"github.com/victorkazakov/kportforward/internal/utils"
)

// ServiceManager manages the lifecycle of a single port-forward service
type ServiceManager struct {
	name   string
	config config.Service
	status *config.ServiceStatus
	cmd    *exec.Cmd
	logger *utils.Logger
	mutex  sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc

	// Exponential backoff fields
	failureCount   int
	cooldownUntil  time.Time
	backoffSeconds []int
}

// NewServiceManager creates a new service manager
func NewServiceManager(name string, service config.Service, logger *utils.Logger) *ServiceManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &ServiceManager{
		name:           name,
		config:         service,
		logger:         logger,
		ctx:            ctx,
		cancel:         cancel,
		backoffSeconds: []int{5, 10, 20, 40, 60}, // Exponential backoff: 5s, 10s, 20s, 40s, 60s max
		status: &config.ServiceStatus{
			Name:         name,
			Status:       "Starting",
			LocalPort:    service.LocalPort,
			RestartCount: 0,
			InCooldown:   false,
		},
	}
}

// Start begins the port-forward process
func (sm *ServiceManager) Start() error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	// Check if we're in cooldown
	if sm.isInCooldown() {
		sm.status.Status = "Cooldown"
		sm.status.InCooldown = true
		return fmt.Errorf("service %s is in cooldown until %v", sm.name, sm.cooldownUntil)
	}

	// Resolve port conflicts
	actualPort, err := sm.resolvePort()
	if err != nil {
		sm.status.Status = "Failed"
		sm.status.LastError = err.Error()
		return fmt.Errorf("port resolution failed for %s: %w", sm.name, err)
	}
	sm.status.LocalPort = actualPort

	// Start kubectl port-forward
	cmd, err := utils.StartKubectlPortForward(
		sm.config.Namespace,
		sm.config.Target,
		actualPort,
		sm.config.TargetPort,
	)
	if err != nil {
		sm.status.Status = "Failed"
		sm.status.LastError = err.Error()
		sm.handleFailure()
		return fmt.Errorf("failed to start port-forward for %s: %w", sm.name, err)
	}

	sm.cmd = cmd
	sm.status.PID = cmd.Process.Pid
	sm.status.StartTime = time.Now()
	sm.status.Status = "Running"
	sm.status.LastError = ""
	sm.status.InCooldown = false

	sm.logger.Info("Started port-forward for %s: %s:%d -> %d",
		sm.name, sm.config.Target, sm.config.TargetPort, actualPort)

	return nil
}

// Stop terminates the port-forward process
func (sm *ServiceManager) Stop() error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if sm.cmd != nil && sm.cmd.Process != nil {
		if err := utils.KillProcess(sm.cmd.Process.Pid); err != nil {
			sm.logger.Warn("Failed to kill process for %s: %v", sm.name, err)
		}
		sm.cmd = nil
	}

	sm.status.Status = "Stopped"
	sm.status.PID = 0
	sm.logger.Info("Stopped port-forward for %s", sm.name)

	return nil
}

// Restart stops and starts the service
func (sm *ServiceManager) Restart() error {
	sm.logger.Info("Restarting service %s", sm.name)

	if err := sm.Stop(); err != nil {
		sm.logger.Warn("Error stopping service %s during restart: %v", sm.name, err)
	}

	sm.mutex.Lock()
	sm.status.RestartCount++
	sm.mutex.Unlock()

	return sm.Start()
}

// IsHealthy checks if the service is running and responding
func (sm *ServiceManager) IsHealthy() bool {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	// Check if process is running
	if sm.cmd == nil || sm.cmd.Process == nil {
		return false
	}

	if !utils.IsProcessRunning(sm.cmd.Process.Pid) {
		return false
	}

	// Check port connectivity
	return utils.CheckPortConnectivity(sm.status.LocalPort)
}

// GetStatus returns the current status of the service
func (sm *ServiceManager) GetStatus() config.ServiceStatus {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	// Update status based on health check, but allow grace period for startup
	if sm.status.Status == "Running" {
		// Give service 5 seconds grace period after startup before health checking
		gracePeriod := 5 * time.Second
		if time.Since(sm.status.StartTime) > gracePeriod && !sm.IsHealthy() {
			sm.status.Status = "Failed"
			sm.status.LastError = "Health check failed"
		}
	}

	return *sm.status
}

// SetStatusMessage sets a transient status message for the service
func (sm *ServiceManager) SetStatusMessage(message string) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.status.StatusMessage = message
}

// Shutdown gracefully shuts down the service manager
func (sm *ServiceManager) Shutdown() {
	sm.cancel()
	sm.Stop()
}

// resolvePort finds an available port, starting from the configured port
func (sm *ServiceManager) resolvePort() (int, error) {
	if utils.IsPortAvailable(sm.config.LocalPort) {
		return sm.config.LocalPort, nil
	}

	// Port is in use, find an alternative
	newPort, err := utils.FindAvailablePort(sm.config.LocalPort + 1)
	if err != nil {
		return 0, err
	}

	sm.logger.Warn("Port %d is in use for %s, using port %d instead",
		sm.config.LocalPort, sm.name, newPort)

	return newPort, nil
}

// handleFailure implements exponential backoff for failed services
func (sm *ServiceManager) handleFailure() {
	sm.failureCount++

	// Don't set cooldown for the first few failures
	if sm.failureCount < 3 {
		return
	}

	// Calculate backoff index (capped at max)
	backoffIndex := sm.failureCount - 3
	if backoffIndex >= len(sm.backoffSeconds) {
		backoffIndex = len(sm.backoffSeconds) - 1
	}

	cooldownDuration := time.Duration(sm.backoffSeconds[backoffIndex]) * time.Second
	sm.cooldownUntil = time.Now().Add(cooldownDuration)

	sm.logger.Warn("Service %s failed %d times, entering cooldown for %v",
		sm.name, sm.failureCount, cooldownDuration)
}

// isInCooldown checks if the service is currently in cooldown
func (sm *ServiceManager) isInCooldown() bool {
	return time.Now().Before(sm.cooldownUntil)
}

// resetFailureCount resets the failure count when service recovers
func (sm *ServiceManager) resetFailureCount() {
	if sm.failureCount > 0 {
		sm.logger.Info("Service %s recovered, resetting failure count", sm.name)
		sm.failureCount = 0
		sm.cooldownUntil = time.Time{}
	}
}
