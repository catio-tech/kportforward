package ui

import (
	"fmt"
	"testing"
	"time"

	"github.com/victorkazakov/kportforward/internal/config"
)

// MockUIManagerProvider implements the UIManagerProvider interface for testing
type MockUIManagerProvider struct {
	globalAccessHealthy bool
	grpcUIURL           string
	swaggerUIURL        string
}

func (m *MockUIManagerProvider) GetGRPCUIURL(serviceName string) string {
	return m.grpcUIURL
}

func (m *MockUIManagerProvider) GetSwaggerUIURL(serviceName string) string {
	return m.swaggerUIURL
}

func (m *MockUIManagerProvider) GetGlobalAccessStatus() bool {
	return m.globalAccessHealthy
}

// TestModelGlobalStatusUpdate tests that the model correctly updates global status
func TestModelGlobalStatusUpdate(t *testing.T) {
	// Create mock manager
	mockManager := &MockUIManagerProvider{
		globalAccessHealthy: true,
		grpcUIURL:           "http://localhost:8080",
		swaggerUIURL:        "http://localhost:9090",
	}

	// Create mock status channel
	statusChan := make(chan map[string]config.ServiceStatus, 1)

	// Create service configs
	serviceConfigs := map[string]config.Service{
		"test-service": {
			Target:     "service/test",
			TargetPort: 8080,
			LocalPort:  8080,
			Namespace:  "default",
			Type:       "rest",
		},
	}

	// Create model
	model := NewModel(statusChan, serviceConfigs, mockManager)

	// Test initial state
	if !model.globalAccessHealthy {
		t.Error("Expected initial global access to be healthy")
	}

	// Test status update with healthy global access
	serviceStatuses := map[string]config.ServiceStatus{
		"test-service": {
			Name:         "test-service",
			Status:       "Running",
			LocalPort:    8080,
			PID:          12345,
			StartTime:    time.Now(),
			RestartCount: 0,
			GlobalStatus: "healthy",
		},
	}

	// Send status update
	statusUpdate := StatusUpdateMsg(serviceStatuses)
	updatedModel, _ := model.Update(statusUpdate)

	updatedModelTyped := updatedModel.(*Model)
	if !updatedModelTyped.globalAccessHealthy {
		t.Error("Expected global access to remain healthy after status update")
	}

	// Test with unhealthy global access
	mockManager.globalAccessHealthy = false

	serviceStatuses["test-service"] = config.ServiceStatus{
		Name:         "test-service",
		Status:       "Suspended",
		LocalPort:    8080,
		PID:          0,
		StartTime:    time.Now(),
		RestartCount: 0,
		GlobalStatus: "auth_failure",
	}

	statusUpdate = StatusUpdateMsg(serviceStatuses)
	updatedModel, _ = updatedModelTyped.Update(statusUpdate)

	updatedModelTyped = updatedModel.(*Model)
	if updatedModelTyped.globalAccessHealthy {
		t.Error("Expected global access to be unhealthy after manager status change")
	}
}

// TestModelManagerInterface tests that the model correctly uses the manager interface
func TestModelManagerInterface(t *testing.T) {
	mockManager := &MockUIManagerProvider{
		globalAccessHealthy: true,
		grpcUIURL:           "http://localhost:8080/grpc",
		swaggerUIURL:        "http://localhost:9090/swagger",
	}

	statusChan := make(chan map[string]config.ServiceStatus, 1)
	serviceConfigs := map[string]config.Service{
		"test-service": {
			Target:     "service/test",
			TargetPort: 8080,
			LocalPort:  8080,
			Namespace:  "default",
			Type:       "rest",
		},
	}

	model := NewModel(statusChan, serviceConfigs, mockManager)

	// Test that the model correctly stores the manager reference
	if model.manager == nil {
		t.Error("Expected model to have manager reference")
	}

	// Test that the model can access manager methods (indirectly via status updates)
	serviceStatuses := map[string]config.ServiceStatus{
		"test-service": {
			Name:      "test-service",
			Status:    "Running",
			LocalPort: 8080,
		},
	}

	statusUpdate := StatusUpdateMsg(serviceStatuses)
	updatedModel, _ := model.Update(statusUpdate)

	// The update should successfully access the manager's global status
	updatedModelTyped := updatedModel.(*Model)
	if !updatedModelTyped.globalAccessHealthy {
		t.Error("Expected model to correctly get global status from manager")
	}
}

// TestModelNilManagerHandling tests that the model handles nil manager gracefully
func TestModelNilManagerHandling(t *testing.T) {
	statusChan := make(chan map[string]config.ServiceStatus, 1)
	serviceConfigs := map[string]config.Service{
		"test-service": {
			Target:     "service/test",
			TargetPort: 8080,
			LocalPort:  8080,
			Namespace:  "default",
			Type:       "rest",
		},
	}

	// Create model with nil manager
	model := NewModel(statusChan, serviceConfigs, nil)

	// Test that status updates don't panic with nil manager
	serviceStatuses := map[string]config.ServiceStatus{
		"test-service": {
			Name:      "test-service",
			Status:    "Running",
			LocalPort: 8080,
		},
	}

	statusUpdate := StatusUpdateMsg(serviceStatuses)

	// This should not panic
	updatedModel, _ := model.Update(statusUpdate)
	updatedModelTyped := updatedModel.(*Model)

	// Global status should remain at initial value (true) since manager is nil
	if !updatedModelTyped.globalAccessHealthy {
		t.Error("Expected global access to remain at initial value with nil manager")
	}
}

// TestModelServiceStatusWithGlobalStatus tests handling of services with global status
func TestModelServiceStatusWithGlobalStatus(t *testing.T) {
	mockManager := &MockUIManagerProvider{
		globalAccessHealthy: false, // Simulate unhealthy global access
	}

	statusChan := make(chan map[string]config.ServiceStatus, 1)
	serviceConfigs := map[string]config.Service{
		"suspended-service": {
			Target:     "service/suspended",
			TargetPort: 8080,
			LocalPort:  8080,
			Namespace:  "default",
			Type:       "rest",
		},
	}

	model := NewModel(statusChan, serviceConfigs, mockManager)

	// Test service with suspended status and global status info
	serviceStatuses := map[string]config.ServiceStatus{
		"suspended-service": {
			Name:          "suspended-service",
			Status:        "Suspended",
			LocalPort:     8080,
			PID:           0,
			StartTime:     time.Now(),
			RestartCount:  0,
			LastError:     "",
			StatusMessage: "Suspended due to global kubectl access failure",
			GlobalStatus:  "auth_failure",
		},
	}

	statusUpdate := StatusUpdateMsg(serviceStatuses)
	updatedModel, _ := model.Update(statusUpdate)

	updatedModelTyped := updatedModel.(*Model)

	// Verify the model correctly processes the suspended service
	if updatedModelTyped.globalAccessHealthy {
		t.Error("Expected global access to be unhealthy")
	}

	// Verify service is properly stored
	if len(updatedModelTyped.services) != 1 {
		t.Errorf("Expected 1 service, got %d", len(updatedModelTyped.services))
	}

	service, exists := updatedModelTyped.services["suspended-service"]
	if !exists {
		t.Error("Expected suspended-service to exist in model")
	}

	if service.Status != "Suspended" {
		t.Errorf("Expected service status to be Suspended, got %s", service.Status)
	}

	if service.GlobalStatus != "auth_failure" {
		t.Errorf("Expected global status to be auth_failure, got %s", service.GlobalStatus)
	}
}

// BenchmarkModelUpdate benchmarks the model update performance
func BenchmarkModelUpdate(b *testing.B) {
	mockManager := &MockUIManagerProvider{
		globalAccessHealthy: true,
	}

	statusChan := make(chan map[string]config.ServiceStatus, 1)

	// Create multiple services to simulate real usage
	serviceConfigs := make(map[string]config.Service)
	serviceStatuses := make(map[string]config.ServiceStatus)

	for i := 0; i < 25; i++ { // Simulate 25 services (similar to default config)
		serviceName := fmt.Sprintf("service-%d", i)
		serviceConfigs[serviceName] = config.Service{
			Target:     fmt.Sprintf("service/test-%d", i),
			TargetPort: 8080 + i,
			LocalPort:  8080 + i,
			Namespace:  "default",
			Type:       "rest",
		}

		serviceStatuses[serviceName] = config.ServiceStatus{
			Name:         serviceName,
			Status:       "Running",
			LocalPort:    8080 + i,
			PID:          12345 + i,
			StartTime:    time.Now(),
			RestartCount: 0,
			GlobalStatus: "healthy",
		}
	}

	model := NewModel(statusChan, serviceConfigs, mockManager)
	statusUpdate := StatusUpdateMsg(serviceStatuses)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		model.Update(statusUpdate)
	}
}
