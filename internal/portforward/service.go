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

	// Health check fields
	healthCheckFailures int
	consecutiveFailures int
	maxFailureThreshold int
	lastHealthCheckTime time.Time
}

// NewServiceManager creates a new service manager
func NewServiceManager(name string, service config.Service, logger *utils.Logger) *ServiceManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &ServiceManager{
		name:                name,
		config:              service,
		logger:              logger,
		ctx:                 ctx,
		cancel:              cancel,
		backoffSeconds:      []int{5, 10, 20, 40, 60}, // Exponential backoff: 5s, 10s, 20s, 40s, 60s max
		healthCheckFailures: 0,
		consecutiveFailures: 0,
		maxFailureThreshold: 3, // Require 3 consecutive failures before marking as failed
		lastHealthCheckTime: time.Now(),
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

	// Set initial status to "Connecting" until health checks confirm it's running
	// This provides better feedback during the connection establishment phase
	sm.status.Status = "Connecting"
	sm.status.StatusMessage = "Waiting for port-forward to establish"
	sm.status.LastError = ""
	sm.status.InCooldown = false

	// Reset health check counters
	sm.healthCheckFailures = 0
	sm.consecutiveFailures = 0
	sm.lastHealthCheckTime = time.Now()

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
// This is a simplified version as the main health tracking logic is now in GetStatus
// to avoid mutex deadlocks
func (sm *ServiceManager) IsHealthy() bool {
	// Check if process is running
	if sm.cmd == nil || sm.cmd.Process == nil {
		return false
	}

	if !utils.IsProcessRunning(sm.cmd.Process.Pid) {
		return false
	}

	// Check port connectivity with retries built into the CheckPortConnectivity function
	return utils.CheckPortConnectivity(sm.status.LocalPort)
}

// GetStatus returns the current status of the service
func (sm *ServiceManager) GetStatus() config.ServiceStatus {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	// Make a copy of the status to return
	statusCopy := *sm.status

	// Update status based on health check, but allow grace period for startup
	if sm.status.Status == "Running" || sm.status.Status == "Degraded" ||
		sm.status.Status == "Connecting" || sm.status.Status == "Reconnecting" {
		// Give service 5 seconds grace period after startup before health checking
		gracePeriod := 5 * time.Second
		if time.Since(sm.status.StartTime) > gracePeriod {
			// This check doesn't call IsHealthy() directly to avoid deadlock
			// as IsHealthy already acquires the lock

			// Check process running
			isProcessRunning := true
			if sm.cmd == nil || sm.cmd.Process == nil || !utils.IsProcessRunning(sm.cmd.Process.Pid) {
				isProcessRunning = false
			}

			// Check port connectivity - with retries built in
			isPortConnected := false
			if isProcessRunning {
				isPortConnected = utils.CheckPortConnectivity(sm.status.LocalPort)
				if !isPortConnected {
					sm.logger.Debug("Port connectivity check failed for %s on port %d", sm.name, sm.status.LocalPort)
				}
			}

			// Update consecutive failures
			isHealthy := isProcessRunning && isPortConnected

			// Update consecutive failure counter
			if isHealthy {
				// Only consider it truly recovered if we have multiple successful checks
				// This avoids flapping between Running/Failed for unstable connections
				if sm.status.Status == "Failed" {
					// For previously failed services, require 3 consecutive successful checks
					// before marking as recovered (stay in Failed state during this period)
					sm.consecutiveFailures--
					if sm.consecutiveFailures <= 0 {
						sm.logger.Info("Service %s confirmed recovered after multiple successful health checks",
							sm.name)
						sm.status.Status = "Running"
						sm.status.LastError = ""
						sm.status.StatusMessage = ""
						sm.resetFailureCount() // Reset exponential backoff

						// Update the copy we'll return
						statusCopy = *sm.status
					} else {
						sm.logger.Debug("Service %s shows signs of recovery (%d more checks needed)",
							sm.name, sm.consecutiveFailures)
					}
				} else if sm.status.Status == "Degraded" {
					// For services that were in Degraded state but now passing health checks
					sm.consecutiveFailures--
					if sm.consecutiveFailures <= 0 {
						sm.logger.Info("Service %s recovered from degraded state",
							sm.name)
						sm.status.Status = "Running"
						sm.status.StatusMessage = ""
						sm.status.LastError = ""

						// Update the copy we'll return
						statusCopy = *sm.status
					}
				} else if sm.status.Status == "Connecting" {
					// For services that just completed initial connection
					sm.logger.Info("Service %s successfully connected",
						sm.name)
					sm.status.Status = "Running"
					sm.status.StatusMessage = ""
					sm.status.LastError = ""

					// Update the copy we'll return
					statusCopy = *sm.status
				} else if sm.status.Status == "Reconnecting" {
					// For services that just completed reconnection
					sm.logger.Info("Service %s successfully reconnected",
						sm.name)
					sm.status.Status = "Running"
					sm.status.StatusMessage = ""
					sm.status.LastError = ""

					// Update the copy we'll return
					statusCopy = *sm.status
				} else {
					// For services that are running normally
					if sm.consecutiveFailures > 0 {
						sm.logger.Debug("Health check recovered for %s after %d consecutive failures",
							sm.name, sm.consecutiveFailures)
					}
					sm.consecutiveFailures = 0

					// Always clear any lingering status messages when the service is healthy
					if sm.status.StatusMessage != "" {
						sm.logger.Debug("Clearing status message for %s: \"%s\"", sm.name, sm.status.StatusMessage)
						sm.status.StatusMessage = ""
						// Update the copy we'll return
						statusCopy = *sm.status
					}
				}
			} else {
				sm.consecutiveFailures++
				sm.healthCheckFailures++

				// Log why the health check failed (process or port)
				if !isProcessRunning {
					sm.logger.Debug("Health check failed for %s: process not running (PID %d)",
						sm.name, sm.status.PID)
				} else if !isPortConnected {
					sm.logger.Debug("Health check failed for %s: port %d not responding (%d consecutive failures, %d total)",
						sm.name, sm.status.LocalPort, sm.consecutiveFailures, sm.healthCheckFailures)
				}

				// On first health check failure, update status appropriately
				// Handle each possible current state
				if sm.status.Status == "Running" {
					// Standard case - mark as Degraded
					sm.status.Status = "Degraded"
					sm.status.StatusMessage = "Port connectivity issues"
					sm.logger.Warn("Service %s is degraded - health check failing on port %d",
						sm.name, sm.status.LocalPort)

					// Set the consecutive failures to 2 so it takes 2 successful checks to recover
					sm.consecutiveFailures = 2

					// Update the copy we'll return
					statusCopy = *sm.status
				} else if sm.status.Status == "Connecting" {
					// For new connections, just leave as Connecting but update message
					// This provides better feedback during initial connection phase
					sm.status.StatusMessage = "Connection in progress..."

					// Update the copy we'll return
					statusCopy = *sm.status
				} else if sm.status.Status == "Reconnecting" {
					// For reconnections, just leave as Reconnecting but update message
					sm.status.StatusMessage = "Reconnection in progress..."

					// Update the copy we'll return
					statusCopy = *sm.status
				}
			}

			// Only mark as failed if we've exceeded the consecutive failure threshold
			if !isHealthy && sm.consecutiveFailures >= sm.maxFailureThreshold && sm.status.Status != "Failed" {
				// Set higher value to require more successful checks to recover
				// This creates a hysteresis effect to prevent status flapping
				sm.consecutiveFailures = 3

				sm.status.Status = "Failed"
				// Add more details about the failure reason
				if !isProcessRunning {
					sm.status.LastError = fmt.Sprintf("Process not running (PID %d)", sm.status.PID)
				} else if !isPortConnected {
					sm.status.LastError = fmt.Sprintf("Port %d not responding after multiple attempts", sm.status.LocalPort)
				} else {
					sm.status.LastError = fmt.Sprintf("Health check failed after %d consecutive failures", sm.consecutiveFailures)
				}

				sm.logger.Warn("Service %s marked as failed: %s", sm.name, sm.status.LastError)

				// Update the copy we'll return
				statusCopy = *sm.status
			}
			// Note: Recovery is now handled in the isHealthy block above to require multiple successful checks
		}
	}

	return statusCopy
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
