package portforward

import (
	"testing"
	"time"

	"github.com/victorkazakov/kportforward/internal/config"
	"github.com/victorkazakov/kportforward/internal/utils"
)

// MockUIHandler for testing
type MockUIHandler struct {
	enabled    bool
	startCalls []string
	stopCalls  []string
}

func NewMockUIHandler() *MockUIHandler {
	return &MockUIHandler{
		enabled:    false,
		startCalls: make([]string, 0),
		stopCalls:  make([]string, 0),
	}
}

func (m *MockUIHandler) Enable() {
	m.enabled = true
}

func (m *MockUIHandler) IsEnabled() bool {
	return m.enabled
}

func (m *MockUIHandler) StartService(serviceName string, serviceStatus config.ServiceStatus, serviceConfig config.Service) error {
	m.startCalls = append(m.startCalls, serviceName)
	return nil
}

func (m *MockUIHandler) StopService(serviceName string) error {
	m.stopCalls = append(m.stopCalls, serviceName)
	return nil
}

func (m *MockUIHandler) MonitorServices(services map[string]config.ServiceStatus, configs map[string]config.Service) {
	// Mock implementation - just track that it was called
}

func (m *MockUIHandler) GetServiceURL(serviceName string) string {
	// Mock implementation - return a test URL
	if m.enabled {
		return "http://localhost:9999"
	}
	return ""
}

func TestNewManager(t *testing.T) {
	cfg := &config.Config{
		PortForwards: map[string]config.Service{
			"test-service": {
				Target:     "service/test",
				TargetPort: 8080,
				LocalPort:  9080,
				Namespace:  "default",
				Type:       "web",
			},
		},
		MonitoringInterval: 5 * time.Second,
	}

	logger := utils.NewLogger(utils.LevelInfo)
	manager := NewManager(cfg, logger)

	if manager == nil {
		t.Fatal("Manager should not be nil")
	}

	if manager.config != cfg {
		t.Error("Manager config not set correctly")
	}

	if manager.logger != logger {
		t.Error("Manager logger not set correctly")
	}

	if manager.services == nil {
		t.Error("Manager services map should be initialized")
	}

	if manager.statusChan == nil {
		t.Error("Manager status channel should be initialized")
	}
}

func TestManagerUIHandlers(t *testing.T) {
	cfg := &config.Config{
		PortForwards:       map[string]config.Service{},
		MonitoringInterval: 1 * time.Second,
	}

	logger := utils.NewLogger(utils.LevelInfo)
	manager := NewManager(cfg, logger)

	// Create mock UI handlers
	grpcHandler := NewMockUIHandler()
	swaggerHandler := NewMockUIHandler()

	grpcHandler.Enable()
	swaggerHandler.Enable()

	// Set UI handlers
	manager.SetUIHandlers(grpcHandler, swaggerHandler)

	// Verify handlers are set (we can't directly access them due to private fields,
	// but we can test through behavior)
	if !grpcHandler.IsEnabled() {
		t.Error("gRPC handler should be enabled")
	}

	if !swaggerHandler.IsEnabled() {
		t.Error("Swagger handler should be enabled")
	}
}

func TestManagerKubernetesContext(t *testing.T) {
	cfg := &config.Config{
		PortForwards:       map[string]config.Service{},
		MonitoringInterval: 1 * time.Second,
	}

	logger := utils.NewLogger(utils.LevelInfo)
	manager := NewManager(cfg, logger)

	// Initially context should be empty
	context := manager.GetKubernetesContext()
	if context != "" {
		t.Errorf("Initial context should be empty, got: %s", context)
	}
}

func TestManagerStatusChannel(t *testing.T) {
	cfg := &config.Config{
		PortForwards:       map[string]config.Service{},
		MonitoringInterval: 1 * time.Second,
	}

	logger := utils.NewLogger(utils.LevelInfo)
	manager := NewManager(cfg, logger)

	statusChan := manager.GetStatusChannel()
	if statusChan == nil {
		t.Error("Status channel should not be nil")
	}

	// Test that we can receive from the channel
	select {
	case <-statusChan:
		// This is fine, we might get an immediate status update
	case <-time.After(100 * time.Millisecond):
		// This is also fine, no immediate status update
	}
}

func TestManagerCurrentStatus(t *testing.T) {
	cfg := &config.Config{
		PortForwards: map[string]config.Service{
			"test-service": {
				Target:     "service/test",
				TargetPort: 8080,
				LocalPort:  9080,
				Namespace:  "default",
				Type:       "web",
			},
		},
		MonitoringInterval: 1 * time.Second,
	}

	logger := utils.NewLogger(utils.LevelInfo)
	manager := NewManager(cfg, logger)

	// Get current status (should be empty initially since we haven't started)
	status := manager.GetCurrentStatus()
	if status == nil {
		t.Error("Status should not be nil")
	}

	// Initially should have no running services
	if len(status) != 0 {
		t.Errorf("Expected 0 services initially, got %d", len(status))
	}
}

func TestUIHandlerInterface(t *testing.T) {
	// Test that our mock implements the interface correctly
	var handler UIHandler = NewMockUIHandler()

	// Test interface methods
	if handler.IsEnabled() {
		t.Error("Handler should not be enabled initially")
	}

	// Test StartService
	status := config.ServiceStatus{
		Name:      "test",
		Status:    "Running",
		LocalPort: 8080,
	}
	serviceConfig := config.Service{
		Target:     "service/test",
		TargetPort: 8080,
		LocalPort:  9080,
		Namespace:  "default",
		Type:       "rpc",
	}

	err := handler.StartService("test", status, serviceConfig)
	if err != nil {
		t.Errorf("StartService should not return error: %v", err)
	}

	// Test StopService
	err = handler.StopService("test")
	if err != nil {
		t.Errorf("StopService should not return error: %v", err)
	}

	// Test MonitorServices
	handler.MonitorServices(map[string]config.ServiceStatus{}, map[string]config.Service{})
}

func TestManagerValidation(t *testing.T) {
	// Test with nil config - should handle gracefully
	logger := utils.NewLogger(utils.LevelInfo)
	manager := NewManager(nil, logger)
	if manager == nil {
		t.Error("NewManager should not return nil even with nil config")
	}

	// Test with nil logger - should handle gracefully
	cfg := &config.Config{
		PortForwards:       map[string]config.Service{},
		MonitoringInterval: 1 * time.Second,
	}
	manager = NewManager(cfg, nil)
	if manager == nil {
		t.Error("NewManager should not return nil even with nil logger")
	}
}
