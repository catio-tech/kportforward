package portforward

import (
	"context"
	"fmt"
	"os/exec"
	"reflect"
	"sync"
	"time"

	"github.com/victorkazakov/kportforward/internal/config"
	"github.com/victorkazakov/kportforward/internal/utils"
)

// UIHandler interface for UI managers
type UIHandler interface {
	StartService(serviceName string, serviceStatus config.ServiceStatus, serviceConfig config.Service) error
	StopService(serviceName string) error
	MonitorServices(services map[string]config.ServiceStatus, configs map[string]config.Service)
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

	// UI Handlers
	grpcUIHandler    UIHandler
	swaggerUIHandler UIHandler

	// Monitoring
	monitoringTicker *time.Ticker
	statusChan       chan map[string]config.ServiceStatus
}

// NewManager creates a new port-forward manager
func NewManager(cfg *config.Config, logger *utils.Logger) *Manager {
	ctx, cancel := context.WithCancel(context.Background())

	return &Manager{
		services:   make(map[string]*ServiceManager),
		config:     cfg,
		logger:     logger,
		ctx:        ctx,
		cancel:     cancel,
		statusChan: make(chan map[string]config.ServiceStatus, 1),
	}
}

// SetUIHandlers sets the UI handlers for the manager
func (m *Manager) SetUIHandlers(grpcUI, swaggerUI UIHandler) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.grpcUIHandler = grpcUI
	m.swaggerUIHandler = swaggerUI
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

	// Start all services
	var startErrors []error
	for name, sm := range m.services {
		if err := sm.Start(); err != nil {
			m.logger.Error("Failed to start service %s: %v", name, err)
			startErrors = append(startErrors, err)
		}
	}

	// Start monitoring
	m.startMonitoring()

	if len(startErrors) > 0 {
		return fmt.Errorf("failed to start %d services", len(startErrors))
	}

	m.logger.Info("Started %d port-forward services", len(m.services))
	return nil
}

// Stop gracefully stops all services
func (m *Manager) Stop() error {
	m.mutex.Lock()
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
	close(m.statusChan)

	m.logger.Info("Stopped all port-forward services")
	return nil
}

// GetStatusChannel returns a channel that receives status updates
func (m *Manager) GetStatusChannel() <-chan map[string]config.ServiceStatus {
	return m.statusChan
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
	m.mutex.RLock()
	services := make(map[string]*ServiceManager, len(m.services))
	for name, sm := range m.services {
		services[name] = sm
	}
	m.mutex.RUnlock()

	statusMap := make(map[string]config.ServiceStatus)

	for name, sm := range services {
		status := sm.GetStatus()
		statusMap[name] = status

		// Check if service needs to be restarted
		if status.Status == "Failed" && !status.InCooldown {
			m.logger.Info("Restarting failed service: %s", name)
			go func(serviceName string, serviceManager *ServiceManager) {
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

	if newContext != currentContext {
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
	m.mutex.RLock()
	services := make([]*ServiceManager, 0, len(m.services))
	for _, sm := range m.services {
		services = append(services, sm)
	}
	m.mutex.RUnlock()

	for _, sm := range services {
		if err := sm.Restart(); err != nil {
			m.logger.Error("Failed to restart service during context change: %v", err)
		}
		// Small delay between restarts to avoid overwhelming the system
		time.Sleep(100 * time.Millisecond)
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
	cmd := exec.Command("kubectl", "config", "current-context")
	output, err := cmd.Output()
	if err != nil {
		return "N/A", err
	}

	// Remove trailing newline
	context := string(output)
	if len(context) > 0 && context[len(context)-1] == '\n' {
		context = context[:len(context)-1]
	}

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
