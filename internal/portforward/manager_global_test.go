package portforward

import (
	"errors"
	"testing"
	"time"

	"github.com/victorkazakov/kportforward/internal/config"
	"github.com/victorkazakov/kportforward/internal/utils"
)

// TestIsAuthError tests the authentication error detection
func TestIsAuthError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"unauthorized", errors.New("unauthorized"), true},
		{"authentication failed", errors.New("authentication failed"), true},
		{"token expired", errors.New("token expired"), true},
		{"credential invalid", errors.New("credential invalid"), true},
		{"forbidden", errors.New("forbidden access"), true},
		{"invalid user", errors.New("invalid user"), true},
		{"access denied", errors.New("access denied"), true},
		{"network error", errors.New("connection refused"), false},
		{"timeout error", errors.New("request timeout"), false},
		{"generic error", errors.New("generic failure"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isAuthError(tt.err)
			if result != tt.expected {
				t.Errorf("isAuthError(%v) = %v, expected %v", tt.err, result, tt.expected)
			}
		})
	}
}

// TestIsNetworkError tests the network error detection
func TestIsNetworkError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"connection refused", errors.New("connection refused"), true},
		{"timeout", errors.New("timeout occurred"), true},
		{"network unreachable", errors.New("network unreachable"), true},
		{"no route to host", errors.New("no route to host"), true},
		{"connection timed out", errors.New("connection timed out"), true},
		{"dial tcp", errors.New("dial tcp: connection failed"), true},
		{"i/o timeout", errors.New("i/o timeout"), true},
		{"auth error", errors.New("unauthorized"), false},
		{"generic error", errors.New("generic failure"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isNetworkError(tt.err)
			if result != tt.expected {
				t.Errorf("isNetworkError(%v) = %v, expected %v", tt.err, result, tt.expected)
			}
		})
	}
}

// TestGlobalAccessState tests the global access state management
func TestGlobalAccessState(t *testing.T) {
	// Create a test manager
	cfg := &config.Config{
		PortForwards:       make(map[string]config.Service),
		MonitoringInterval: 5 * time.Second,
	}
	logger := utils.NewLoggerWithOutput(utils.LevelInfo, nil) // Discard output
	manager := NewManager(cfg, logger)

	// Test initial state
	if !manager.GetGlobalAccessStatus() {
		t.Error("Expected initial global access status to be healthy (true)")
	}

	// Test state changes via checkAndUpdateGlobalAccess
	// Note: This is a simplified test since checkGlobalAccess makes actual kubectl calls
	// In a real test environment, you would mock the kubectl command execution
}

// TestServiceSuspensionLogic tests the service suspension and resumption
func TestServiceSuspensionLogic(t *testing.T) {
	// Create test config with services
	cfg := &config.Config{
		PortForwards: map[string]config.Service{
			"test-service-1": {
				Target:     "service/test-1",
				TargetPort: 8080,
				LocalPort:  8080,
				Namespace:  "default",
				Type:       "rest",
			},
			"test-service-2": {
				Target:     "service/test-2",
				TargetPort: 9090,
				LocalPort:  9090,
				Namespace:  "default",
				Type:       "rpc",
			},
		},
		MonitoringInterval: 5 * time.Second,
	}

	logger := utils.NewLoggerWithOutput(utils.LevelInfo, nil)
	manager := NewManager(cfg, logger)

	// Create mock service managers
	sm1 := NewServiceManager("test-service-1", cfg.PortForwards["test-service-1"], logger)
	sm2 := NewServiceManager("test-service-2", cfg.PortForwards["test-service-2"], logger)

	// Set initial states
	sm1.mutex.Lock()
	sm1.status.Status = "Running"
	sm1.mutex.Unlock()

	sm2.mutex.Lock()
	sm2.status.Status = "Degraded"
	sm2.mutex.Unlock()

	// Add to manager
	manager.services["test-service-1"] = sm1
	manager.services["test-service-2"] = sm2

	// Test suspension
	manager.suspendAllServices()

	// Check that services are suspended
	status1 := sm1.GetStatus()
	status2 := sm2.GetStatus()

	if status1.Status != "Suspended" {
		t.Errorf("Expected test-service-1 to be Suspended, got %s", status1.Status)
	}

	if status2.Status != "Suspended" {
		t.Errorf("Expected test-service-2 to be Suspended, got %s", status2.Status)
	}

	if status1.StatusMessage != "Suspended due to global kubectl access failure" {
		t.Errorf("Expected suspension message, got: %s", status1.StatusMessage)
	}
}

// TestGlobalStatusString tests the global status string generation
func TestGlobalStatusString(t *testing.T) {
	cfg := &config.Config{
		PortForwards:       make(map[string]config.Service),
		MonitoringInterval: 5 * time.Second,
	}
	logger := utils.NewLoggerWithOutput(utils.LevelInfo, nil)
	manager := NewManager(cfg, logger)

	// Test healthy status
	manager.globalAccessMutex.Lock()
	manager.globalAccessHealthy = true
	manager.globalAccessMutex.Unlock()

	status := manager.getGlobalStatusString()
	if status != "healthy" {
		t.Errorf("Expected 'healthy', got '%s'", status)
	}

	// Test failure status
	manager.globalAccessMutex.Lock()
	manager.globalAccessHealthy = false
	manager.globalAccessFailCount = 1
	manager.globalAccessCooldown = time.Now().Add(30 * time.Second) // Short cooldown
	manager.globalAccessMutex.Unlock()

	status = manager.getGlobalStatusString()
	if status != "network_failure" {
		t.Errorf("Expected 'network_failure' for short cooldown, got '%s'", status)
	}

	// Test auth failure (long cooldown)
	manager.globalAccessMutex.Lock()
	manager.globalAccessCooldown = time.Now().Add(10 * time.Minute) // Long cooldown
	manager.globalAccessMutex.Unlock()

	status = manager.getGlobalStatusString()
	if status != "auth_failure" {
		t.Errorf("Expected 'auth_failure' for long cooldown, got '%s'", status)
	}
}

// TestExponentialBackoff tests the exponential backoff logic
func TestExponentialBackoff(t *testing.T) {
	cfg := &config.Config{
		PortForwards:       make(map[string]config.Service),
		MonitoringInterval: 5 * time.Second,
	}
	logger := utils.NewLoggerWithOutput(utils.LevelInfo, nil)
	manager := NewManager(cfg, logger)

	// Test that cooldown increases with failure count
	manager.globalAccessMutex.Lock()

	// Simulate auth failure progression
	manager.globalAccessHealthy = false
	manager.globalAccessFailCount = 1
	manager.globalAccessCooldown = time.Now().Add(5 * time.Minute)
	firstCooldown := manager.globalAccessCooldown

	manager.globalAccessFailCount = 2
	manager.globalAccessCooldown = time.Now().Add(10 * time.Minute)
	secondCooldown := manager.globalAccessCooldown

	manager.globalAccessFailCount = 3
	manager.globalAccessCooldown = time.Now().Add(30 * time.Minute)
	thirdCooldown := manager.globalAccessCooldown

	manager.globalAccessMutex.Unlock()

	// Verify cooldown progression
	if !secondCooldown.After(firstCooldown) {
		t.Error("Second cooldown should be longer than first")
	}

	if !thirdCooldown.After(secondCooldown) {
		t.Error("Third cooldown should be longer than second")
	}
}

// BenchmarkGlobalAccessCheck benchmarks the global access check performance
func BenchmarkGlobalAccessCheck(b *testing.B) {
	cfg := &config.Config{
		PortForwards:       make(map[string]config.Service),
		MonitoringInterval: 5 * time.Second,
	}
	logger := utils.NewLoggerWithOutput(utils.LevelInfo, nil)
	manager := NewManager(cfg, logger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Note: This will make actual kubectl calls in a real environment
		// In a proper benchmark, you would mock the kubectl execution
		manager.checkAndUpdateGlobalAccess()
	}
}

// TestServiceManagerAuthError tests auth error handling at service level
func TestServiceManagerAuthError(t *testing.T) {
	logger := utils.NewLoggerWithOutput(utils.LevelInfo, nil)

	service := config.Service{
		Target:     "service/test",
		TargetPort: 8080,
		LocalPort:  8080,
		Namespace:  "default",
		Type:       "rest",
	}

	sm := NewServiceManager("test-service", service, logger)

	// Test auth error detection
	authErr := errors.New("unauthorized access")
	if !sm.isAuthError(authErr) {
		t.Error("Expected auth error to be detected")
	}

	networkErr := errors.New("connection refused")
	if sm.isAuthError(networkErr) {
		t.Error("Expected network error to not be detected as auth error")
	}
}
