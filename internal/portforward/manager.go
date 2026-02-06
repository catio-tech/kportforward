package portforward

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/victorkazakov/kportforward/internal/common"
	"github.com/victorkazakov/kportforward/internal/config"
	"github.com/victorkazakov/kportforward/internal/utils"
)

// UIHandler interface for UI managers
type UIHandler interface {
	StartService(serviceName string, serviceStatus config.ServiceStatus, serviceConfig config.Service) error
	StopService(serviceName string) error
	MonitorServices(services map[string]config.ServiceStatus, configs map[string]config.Service)
	SetStatusCallback(callback common.StatusCallback)
	IsEnabled() bool
	GetServiceURL(serviceName string) string
}

// Manager coordinates multiple port-forward services
type Manager struct {
	services          map[string]*ServiceManager
	config            *config.Config
	logger            *utils.Logger
	ctx               context.Context
	cancel            context.CancelFunc
	mutex             sync.RWMutex
	kubernetesContext string
	shuttingDown      bool

	// UI Handlers
	grpcUIHandler    UIHandler
	swaggerUIHandler UIHandler

	// Monitoring
	monitoringTicker *time.Ticker
	statusChan       chan map[string]config.ServiceStatus
	contextChan      chan string

	// Global access state
	globalAccessHealthy   bool
	globalAccessLastCheck time.Time
	globalAccessFailCount int
	globalAccessCooldown  time.Time
	globalAccessMutex     sync.RWMutex
}

func (m *Manager) isShuttingDown() bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.shuttingDown
}
func applyKubeconfigEnv(cmd *exec.Cmd) {
	// Respect existing KUBECONFIG if set
	if os.Getenv("KUBECONFIG") != "" {
		cmd.Env = os.Environ()
		return
	}

	homeDir, err := os.UserHomeDir()
	if err != nil || homeDir == "" {
		cmd.Env = os.Environ()
		return
	}

	kubeconfig := filepath.Join(homeDir, ".kube", "config")
	cmd.Env = append(os.Environ(), "KUBECONFIG="+kubeconfig)
}

// NewManager creates a new port-forward manager
func NewManager(cfg *config.Config, logger *utils.Logger) *Manager {
	ctx, cancel := context.WithCancel(context.Background())

	return &Manager{
		services:    make(map[string]*ServiceManager),
		config:      cfg,
		logger:      logger,
		ctx:         ctx,
		cancel:      cancel,
		statusChan:  make(chan map[string]config.ServiceStatus, 1),
		contextChan: make(chan string, 1),

		// Initialize global access state
		globalAccessHealthy:   true, // Start optimistically
		globalAccessLastCheck: time.Time{},
		globalAccessFailCount: 0,
		globalAccessCooldown:  time.Time{},
	}
}

// SetUIHandlers sets the UI handlers for the manager
func (m *Manager) SetUIHandlers(grpcUI, swaggerUI UIHandler) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.grpcUIHandler = grpcUI
	m.swaggerUIHandler = swaggerUI

	// Set the status callback for UI handlers
	// Check for nil interface values properly using reflection
	if grpcUI != nil {
		if reflect.ValueOf(grpcUI).IsNil() == false {
			grpcUI.SetStatusCallback(m)
		}
	}
	if swaggerUI != nil {
		if reflect.ValueOf(swaggerUI).IsNil() == false {
			swaggerUI.SetStatusCallback(m)
		}
	}
}

// Start initializes and starts all port-forward services
func (m *Manager) Start() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Get current Kubernetes context
	if err := m.updateKubernetesContext(); err != nil {
		return fmt.Errorf("failed to get Kubernetes context: %w", err)
	}

	// Create service managers
	for name, serviceConfig := range m.config.PortForwards {
		sm := NewServiceManager(name, serviceConfig, m.logger)
		m.services[name] = sm
	}

	// Check global access BEFORE starting any services to prevent resource waste
	m.logger.Info("Checking global kubectl access before starting services")
	if !m.checkAndUpdateGlobalAccess() {
		m.logger.Warn("Global kubectl access failed at startup - services will remain suspended")
		// Don't start services if auth is failing - just create them in suspended state
		for _, sm := range m.services {
			sm.mutex.Lock()
			sm.status.Status = "Suspended"
			sm.status.StatusMessage = "Suspended due to global kubectl access failure at startup"
			sm.status.StartTime = time.Time{} // Clear start time for suspended services
			sm.status.PID = 0                 // No process running
			sm.mutex.Unlock()
		}
	} else {
		// Only start services if global access check passed
		m.logger.Info("Global access check passed - starting services")
		var startErrors []error
		for name, sm := range m.services {
			if err := sm.Start(); err != nil {
				m.logger.Error("Failed to start service %s: %v", name, err)
				startErrors = append(startErrors, err)
			}
		}

		if len(startErrors) > 0 {
			m.logger.Warn("Failed to start %d services, but continuing", len(startErrors))
		}
	}

	// Start monitoring
	m.startMonitoring()

	// Send immediate status update to populate TUI table
	go func() {
		// Send initial status immediately
		m.sendInitialStatus()

		// Give services a moment to start, then trigger UI handler check
		time.Sleep(2 * time.Second)
		m.logger.Info("Triggering initial UI handler check")
		m.monitorServices()
	}()

	// Get count of services that are actually running vs suspended
	runningCount := 0
	for _, sm := range m.services {
		status := sm.GetStatus()
		if status.Status != "Suspended" {
			runningCount++
		}
	}

	if m.GetGlobalAccessStatus() {
		m.logger.Info("Initialized %d services (%d running)", len(m.services), runningCount)
	} else {
		m.logger.Info("Initialized %d services (all suspended due to authentication failure)", len(m.services))
	}
	return nil
}

// Stop gracefully stops all services
func (m *Manager) Stop() error {
	m.mutex.Lock()
	m.shuttingDown = true
	defer m.mutex.Unlock()

	// Stop monitoring
	if m.monitoringTicker != nil {
		m.monitoringTicker.Stop()
	}

	// Stop UI handlers
	if m.grpcUIHandler != nil && !isNilInterface(m.grpcUIHandler) && m.grpcUIHandler.IsEnabled() {
		for serviceName := range m.services {
			if err := m.grpcUIHandler.StopService(serviceName); err != nil {
				m.logger.Error("Failed to stop gRPC UI for %s: %v", serviceName, err)
			}
		}
	}

	if m.swaggerUIHandler != nil && !isNilInterface(m.swaggerUIHandler) && m.swaggerUIHandler.IsEnabled() {
		for serviceName := range m.services {
			if err := m.swaggerUIHandler.StopService(serviceName); err != nil {
				m.logger.Error("Failed to stop Swagger UI for %s: %v", serviceName, err)
			}
		}
	}

	// Stop all services
	for name, sm := range m.services {
		if err := sm.Stop(); err != nil {
			m.logger.Error("Failed to stop service %s: %v", name, err)
		}
	}

	m.cancel()
	// close(m.statusChan)

	m.logger.Info("Stopped all port-forward services")
	return nil
}

// GetStatusChannel returns a channel that receives status updates
func (m *Manager) GetStatusChannel() <-chan map[string]config.ServiceStatus {
	return m.statusChan
}

// GetContextChannel returns a channel that receives context updates
func (m *Manager) GetContextChannel() <-chan string {
	return m.contextChan
}

// GetCurrentStatus returns the current status of all services
func (m *Manager) GetCurrentStatus() map[string]config.ServiceStatus {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	status := make(map[string]config.ServiceStatus)
	for name, sm := range m.services {
		status[name] = sm.GetStatus()
	}
	return status
}

// RestartService restarts a specific service
func (m *Manager) RestartService(name string) error {
	m.mutex.RLock()
	sm, exists := m.services[name]
	m.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("service %s not found", name)
	}

	return sm.Restart()
}

// GetKubernetesContext returns the current Kubernetes context
func (m *Manager) GetKubernetesContext() string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.kubernetesContext
}

// startMonitoring begins the monitoring loop for all services
func (m *Manager) startMonitoring() {
	m.monitoringTicker = time.NewTicker(m.config.MonitoringInterval)

	go func() {
		defer m.monitoringTicker.Stop()

		for {
			select {
			case <-m.ctx.Done():
				return
			case <-m.monitoringTicker.C:
				m.monitorServices()
				m.checkKubernetesContext()
			}
		}
	}()
}

// monitorServices checks the health of all services and restarts failed ones
func (m *Manager) monitorServices() {

	if m.isShuttingDown() {
		return
	}
	// Check global access first - if this fails, suspend all services
	if !m.checkAndUpdateGlobalAccess() {
		m.logger.Warn("Global kubectl access failed, suspending all service operations")
		m.suspendAllServices()
		return
	}

	// If global access recovered, resume services if needed
	if m.resumeServicesIfNeeded() {
		m.logger.Info("Global access recovered, resuming service operations")
	}

	m.mutex.RLock()
	services := make(map[string]*ServiceManager, len(m.services))
	for name, sm := range m.services {
		services[name] = sm
	}
	m.mutex.RUnlock()

	statusMap := make(map[string]config.ServiceStatus)

	for name, sm := range services {
		// Get status (this runs health checks internally)
		status := sm.GetStatus()

		// If status is Running but still has a status message about connectivity issues,
		// perform an explicit health check and clear message if service is actually healthy
		if status.Status == "Running" && status.StatusMessage == "Port connectivity issues" {
			// Do a direct health check (bypassing status caching)
			if sm.IsHealthy() {
				// If it's really healthy, clear the status message
				sm.SetStatusMessage("")
				// Get updated status
				status = sm.GetStatus()
				m.logger.Debug("Cleared lingering status message for healthy service: %s", name)
			}
		}

		// Enhance status with global information
		status.GlobalStatus = m.getGlobalStatusString()
		statusMap[name] = status

		// Check if service needs to be restarted
		if status.Status == "Failed" && !status.InCooldown {
			m.logger.Info("Restarting failed service: %s", name)
			go func(serviceName string, serviceManager *ServiceManager) {
				if m.isShuttingDown() {
					return
				}
				if err := serviceManager.Restart(); err != nil {
					m.logger.Error("Failed to restart service %s: %v", serviceName, err)
				}
			}(name, sm)
		}
	}

	// Monitor UI handlers
	m.monitorUIHandlers(statusMap)

	// Send status update (non-blocking)
	select {
	case m.statusChan <- statusMap:
	default:
		// Channel is full, skip this update
	}
}

// monitorUIHandlers monitors UI handlers and manages their lifecycle
func (m *Manager) monitorUIHandlers(statusMap map[string]config.ServiceStatus) {
	m.mutex.RLock()
	grpcHandler := m.grpcUIHandler
	swaggerHandler := m.swaggerUIHandler
	m.mutex.RUnlock()

	// Monitor gRPC UI handler - check both nil interface and nil concrete value
	if grpcHandler != nil && !isNilInterface(grpcHandler) && grpcHandler.IsEnabled() {
		grpcHandler.MonitorServices(statusMap, m.config.PortForwards)
	}

	// Monitor Swagger UI handler - check both nil interface and nil concrete value
	if swaggerHandler != nil && !isNilInterface(swaggerHandler) && swaggerHandler.IsEnabled() {
		swaggerHandler.MonitorServices(statusMap, m.config.PortForwards)
	}
}

// isNilInterface checks if an interface contains a nil concrete value
func isNilInterface(handler UIHandler) bool {
	if handler == nil {
		return true
	}

	// Use reflection to check if the interface contains a nil pointer
	v := reflect.ValueOf(handler)
	if v.Kind() == reflect.Ptr {
		return v.IsNil()
	}

	return false
}

// checkKubernetesContext monitors for Kubernetes context changes
func (m *Manager) checkKubernetesContext() {
	newContext, err := m.getCurrentKubernetesContext()
	if err != nil {
		m.logger.Error("Failed to get Kubernetes context: %v", err)
		return
	}

	m.mutex.RLock()
	currentContext := m.kubernetesContext
	m.mutex.RUnlock()

	// Log context check for debugging purposes
	m.logger.Debug("Checking Kubernetes context - Current: %s, New: %s", currentContext, newContext)

	// Always update context in TUI even if it hasn't changed (to ensure it's displayed)
	select {
	case m.contextChan <- newContext:
		m.logger.Debug("Sent context update to TUI: %s", newContext)
	default:
		// Channel is full, skip this update
	}

	if newContext != currentContext && newContext != "N/A" {
		m.logger.Info("Kubernetes context changed from %s to %s, restarting all services",
			currentContext, newContext)

		m.mutex.Lock()
		m.kubernetesContext = newContext
		m.mutex.Unlock()

		// Restart all services in the new context
		go m.restartAllServices()
	}
}

// restartAllServices restarts all services (typically after context change)
func (m *Manager) restartAllServices() {
	m.logger.Info("Context changed - tearing down all connections and recreating with new context")

	// Reset global access state - assume unhealthy until proven otherwise
	m.globalAccessMutex.Lock()
	m.globalAccessHealthy = false // Assume unhealthy on context change
	m.globalAccessFailCount = 0
	m.globalAccessCooldown = time.Time{}
	m.globalAccessLastCheck = time.Time{} // Force immediate check
	m.globalAccessMutex.Unlock()

	m.mutex.RLock()
	services := make([]*ServiceManager, 0, len(m.services))
	for _, sm := range m.services {
		services = append(services, sm)
	}
	m.mutex.RUnlock()

	// STEP 1: Immediately stop ALL existing processes from old context
	m.logger.Info("Stopping all services and UI processes from previous context")

	// Stop all UI handlers first
	if m.grpcUIHandler != nil && !isNilInterface(m.grpcUIHandler) && m.grpcUIHandler.IsEnabled() {
		for _, sm := range services {
			if err := m.grpcUIHandler.StopService(sm.name); err != nil {
				m.logger.Warn("Failed to stop gRPC UI for %s: %v", sm.name, err)
			}
		}
	}

	if m.swaggerUIHandler != nil && !isNilInterface(m.swaggerUIHandler) && m.swaggerUIHandler.IsEnabled() {
		for _, sm := range services {
			if err := m.swaggerUIHandler.StopService(sm.name); err != nil {
				m.logger.Warn("Failed to stop Swagger UI for %s: %v", sm.name, err)
			}
		}
	}

	// Stop all port-forward services
	for _, sm := range services {
		sm.mutex.Lock()
		sm.status.Status = "Reconnecting"
		sm.status.StatusMessage = "Reconnecting due to context change"
		sm.mutex.Unlock()

		if err := sm.Stop(); err != nil {
			m.logger.Warn("Failed to stop service %s during context change: %v", sm.name, err)
		}
	}

	// Small delay to allow processes to fully terminate
	time.Sleep(500 * time.Millisecond)

	// STEP 2: Check if new context is accessible
	if !m.checkAndUpdateGlobalAccess() {
		m.logger.Warn("New context has authentication issues - services will remain suspended")
		// Services are already stopped, just mark them as suspended and ensure they stay that way
		for _, sm := range services {
			sm.mutex.Lock()
			m.logger.Info("Suspending service %s due to context change auth failure", sm.name)
			sm.status.Status = "Suspended"
			sm.status.StatusMessage = "Suspended due to global kubectl access failure after context change"
			sm.status.PID = 0
			sm.status.StartTime = time.Time{}
			sm.mutex.Unlock()
		}
		// Immediately suspend to prevent any monitoring from trying to restart
		m.logger.Info("Calling suspendAllServices() to ensure services stay suspended")
		m.suspendAllServices()

		// Send immediate status update to reflect suspended state
		m.sendInitialStatus()
		return
	}

	// STEP 3: Start all services fresh with new context (only if auth passed)
	m.logger.Info("New context is accessible - starting all services fresh")
	for _, sm := range services {
		if err := sm.Start(); err != nil {
			m.logger.Error("Failed to start service %s in new context: %v", sm.name, err)
		}
		// Small delay between starts to avoid overwhelming the system
		time.Sleep(100 * time.Millisecond)
	}

	// Give services a moment to establish, then trigger UI handler check
	time.Sleep(2 * time.Second)
	m.logger.Info("Triggering UI handler restart for new context")
	// Only start monitoring if the context is healthy - suspended services shouldn't be monitored yet
	if m.GetGlobalAccessStatus() {
		go m.monitorServices()
	} else {
		m.logger.Info("Skipping monitoring restart - global access is unhealthy")
	}
}

// sendInitialStatus sends initial service status to TUI without UI handler checks
func (m *Manager) sendInitialStatus() {
	m.mutex.RLock()
	services := make(map[string]*ServiceManager, len(m.services))
	for name, sm := range m.services {
		services[name] = sm
	}
	m.mutex.RUnlock()

	statusMap := make(map[string]config.ServiceStatus)
	for name, sm := range services {
		statusMap[name] = sm.GetStatus()
	}

	// Send status update (non-blocking)
	select {
	case m.statusChan <- statusMap:
		m.logger.Debug("Sent initial service status to TUI")
	default:
		// Channel is full, skip this update
	}
}

// updateKubernetesContext gets and stores the current Kubernetes context
func (m *Manager) updateKubernetesContext() error {
	context, err := m.getCurrentKubernetesContext()
	if err != nil {
		return err
	}
	m.kubernetesContext = context
	return nil
}

// getCurrentKubernetesContext retrieves the current kubectl context
func (m *Manager) getCurrentKubernetesContext() (string, error) {
	// Create command with timeout context to ensure it doesn't hang
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "kubectl", "config", "current-context")

	// Add environment variables to ensure kubectl uses the right config
	applyKubeconfigEnv(cmd)

	// Capture both stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run the command
	err := cmd.Run()
	if err != nil {
		m.logger.Error("Failed to get current kubectl context: %v, stderr: %s", err, stderr.String())
		return "N/A", err
	}

	// Remove trailing newline
	context := stdout.String()
	if len(context) > 0 && context[len(context)-1] == '\n' {
		context = context[:len(context)-1]
	}

	m.logger.Debug("Current kubectl context: %s", context)
	return context, nil
}

// GetGRPCUIURL returns the gRPC UI URL for a service
func (m *Manager) GetGRPCUIURL(serviceName string) string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if m.grpcUIHandler != nil && !isNilInterface(m.grpcUIHandler) && m.grpcUIHandler.IsEnabled() {
		return m.grpcUIHandler.GetServiceURL(serviceName)
	}
	return ""
}

// GetSwaggerUIURL returns the Swagger UI URL for a service
func (m *Manager) GetSwaggerUIURL(serviceName string) string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if m.swaggerUIHandler != nil && !isNilInterface(m.swaggerUIHandler) && m.swaggerUIHandler.IsEnabled() {
		return m.swaggerUIHandler.GetServiceURL(serviceName)
	}
	return ""
}

// UpdateServiceStatusMessage updates the status message for a service
func (m *Manager) UpdateServiceStatusMessage(serviceName, message string) {
	m.mutex.RLock()
	sm, exists := m.services[serviceName]
	m.mutex.RUnlock()

	if exists {
		sm.SetStatusMessage(message)
	}
}

// checkGlobalAccess performs a lightweight kubectl connectivity test
func (m *Manager) checkGlobalAccess() error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// Test basic kubectl connectivity using a lightweight command
	cmd := exec.CommandContext(ctx, "kubectl", "get", "nodes", "--request-timeout=15s")

	// Add environment variables to ensure kubectl uses the right config
	applyKubeconfigEnv(cmd)

	// Capture both stdout and stderr
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	// Run the command
	err := cmd.Run()
	if err != nil {
		errorOutput := stderr.String()
		m.logger.Debug("Global access check failed: %v, stderr: %s", err, errorOutput)

		// Check both the command error and stderr for auth failures
		combinedError := fmt.Sprintf("%v %s", err, errorOutput)

		// Detect specific auth failures
		if isAuthError(fmt.Errorf("%s", combinedError)) {
			return fmt.Errorf("authentication failed: %s", errorOutput)
		}

		// Detect network failures
		if isNetworkError(fmt.Errorf("%s", combinedError)) {
			return fmt.Errorf("network connectivity failed: %s", errorOutput)
		}

		return fmt.Errorf("kubectl access failed: %s", errorOutput)
	}

	m.logger.Debug("Global access check successful")
	return nil
}

// isAuthError detects authentication-related errors
func isAuthError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "unauthorized") ||
		strings.Contains(errStr, "authentication") ||
		strings.Contains(errStr, "token") ||
		strings.Contains(errStr, "credential") ||
		strings.Contains(errStr, "forbidden") ||
		strings.Contains(errStr, "invalid user") ||
		strings.Contains(errStr, "access denied") ||
		strings.Contains(errStr, "unable to load aws credentials") ||
		strings.Contains(errStr, "expired") ||
		strings.Contains(errStr, "sso") ||
		strings.Contains(errStr, "login") ||
		strings.Contains(errStr, "auth") ||
		strings.Contains(errStr, "permission denied") ||
		strings.Contains(errStr, "invalid_grant") ||
		strings.Contains(errStr, "session") ||
		strings.Contains(errStr, "getting credentials") ||
		strings.Contains(errStr, "refresh failed") ||
		strings.Contains(errStr, "executable aws failed") ||
		strings.Contains(errStr, "unable to connect to the server")
}

// isNetworkError detects network-related errors
func isNetworkError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "network") ||
		strings.Contains(errStr, "no route to host") ||
		strings.Contains(errStr, "connection timed out") ||
		strings.Contains(errStr, "dial tcp") ||
		strings.Contains(errStr, "i/o timeout")
}

// GetGlobalAccessStatus returns the current global access status
func (m *Manager) GetGlobalAccessStatus() bool {
	m.globalAccessMutex.RLock()
	defer m.globalAccessMutex.RUnlock()
	return m.globalAccessHealthy
}

// checkAndUpdateGlobalAccess checks global access and updates state with smart cooldown
func (m *Manager) checkAndUpdateGlobalAccess() bool {
	m.globalAccessMutex.Lock()
	defer m.globalAccessMutex.Unlock()

	now := time.Now()

	// If in cooldown, check if we should allow early recovery
	if now.Before(m.globalAccessCooldown) {
		// When services are suspended, check every 5 seconds for faster recovery
		timeSinceLastCheck := now.Sub(m.globalAccessLastCheck)
		if !m.globalAccessHealthy && timeSinceLastCheck < 5*time.Second {
			return m.globalAccessHealthy
		}
		// Allow recovery check after 5 seconds when authentication is failing
		if !m.globalAccessHealthy {
			m.logger.Debug("Allowing recovery check after %v (cooldown has %v remaining)",
				timeSinceLastCheck, m.globalAccessCooldown.Sub(now))
		} else {
			// If healthy, respect the full cooldown
			return m.globalAccessHealthy
		}
	}

	// Perform access check
	err := m.checkGlobalAccess()
	m.globalAccessLastCheck = now

	if err != nil {
		m.globalAccessFailCount++
		wasHealthy := m.globalAccessHealthy
		m.globalAccessHealthy = false

		// Log state change
		if wasHealthy {
			m.logger.Warn("Global kubectl access failed: %v", err)
		}

		// Exponential backoff based on error type
		var cooldowns []time.Duration
		var errorType string

		if strings.Contains(err.Error(), "authentication") {
			// Long cooldown for auth failures (5, 10, 30 minutes)
			cooldowns = []time.Duration{5 * time.Minute, 10 * time.Minute, 30 * time.Minute}
			errorType = "authentication"
		} else {
			// Shorter cooldown for network issues (30s, 1m, 2m)
			cooldowns = []time.Duration{30 * time.Second, 1 * time.Minute, 2 * time.Minute}
			errorType = "network"
		}

		// Apply cooldown based on failure count
		cooldownIndex := m.globalAccessFailCount - 1
		if cooldownIndex >= len(cooldowns) {
			cooldownIndex = len(cooldowns) - 1
		}

		cooldownDuration := cooldowns[cooldownIndex]
		m.globalAccessCooldown = now.Add(cooldownDuration)

		m.logger.Error("Global access check failed (%s failure #%d), cooldown for %v",
			errorType, m.globalAccessFailCount, cooldownDuration)

		return false
	}

	// Reset on success
	if !m.globalAccessHealthy || m.globalAccessFailCount > 0 {
		m.logger.Info("Global kubectl access recovered after %d failures", m.globalAccessFailCount)
		m.globalAccessHealthy = true
		m.globalAccessFailCount = 0
		m.globalAccessCooldown = time.Time{}
	}

	return true
}

// suspendAllServices marks all services as suspended due to global access failure
func (m *Manager) suspendAllServices() {
	m.mutex.RLock()
	services := make(map[string]*ServiceManager, len(m.services))
	for name, sm := range m.services {
		services[name] = sm
	}
	m.mutex.RUnlock()

	for name, sm := range services {
		sm.mutex.Lock()
		// Only suspend services that are currently running or in other active states
		if sm.status.Status == "Running" || sm.status.Status == "Degraded" ||
			sm.status.Status == "Connecting" || sm.status.Status == "Reconnecting" {

			m.logger.Debug("Suspending service %s (was %s)", name, sm.status.Status)

			// Actually stop the service process, don't just change status
			if sm.cmd != nil && sm.cmd.Process != nil {
				m.logger.Debug("Killing kubectl process for suspended service %s (PID %d)", name, sm.cmd.Process.Pid)
				if err := utils.KillProcess(sm.cmd.Process.Pid); err != nil {
					m.logger.Warn("Failed to kill process for suspended service %s: %v", name, err)
				}
				sm.cmd = nil
			}

			sm.status.Status = "Suspended"
			sm.status.StatusMessage = "Suspended due to global kubectl access failure"
			sm.status.PID = 0
			sm.status.StartTime = time.Time{}
		}
		sm.mutex.Unlock()
	}
}

// resumeServicesIfNeeded resumes suspended services when global access recovers
func (m *Manager) resumeServicesIfNeeded() bool {
	m.mutex.RLock()
	services := make(map[string]*ServiceManager, len(m.services))
	for name, sm := range m.services {
		services[name] = sm
	}
	m.mutex.RUnlock()

	resumed := false
	for name, sm := range services {
		sm.mutex.Lock()
		if sm.status.Status == "Suspended" {
			m.logger.Debug("Resuming suspended service %s", name)
			// Mark as reconnecting and clear suspension message
			sm.status.Status = "Reconnecting"
			sm.status.StatusMessage = "Resuming after global access recovery"
			sm.mutex.Unlock()

			// Restart service in a goroutine to avoid blocking
			go func(serviceName string, serviceManager *ServiceManager) {
				if m.isShuttingDown() {
					return
				}

				if err := serviceManager.Restart(); err != nil {
					m.logger.Error("Failed to resume service %s: %v", serviceName, err)
				} else {
					m.logger.Info("Successfully resumed service %s", serviceName)
				}
			}(name, sm)

			resumed = true
		} else {
			sm.mutex.Unlock()
		}
	}

	return resumed
}

// getGlobalStatusString returns a string representation of the current global status
func (m *Manager) getGlobalStatusString() string {
	m.globalAccessMutex.RLock()
	defer m.globalAccessMutex.RUnlock()

	if m.globalAccessHealthy {
		return "healthy"
	}

	// Determine failure type based on recent error pattern
	// This is a simplified approach - in practice we could store the last error type
	if m.globalAccessFailCount > 0 {
		// If we're in a long cooldown (>2 minutes), likely auth failure
		now := time.Now()
		if now.Before(m.globalAccessCooldown) {
			cooldownRemaining := m.globalAccessCooldown.Sub(now)
			if cooldownRemaining > 2*time.Minute {
				return "auth_failure"
			}
		}
	}

	return "network_failure"
}
