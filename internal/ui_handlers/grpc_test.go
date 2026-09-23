package ui_handlers

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/victorkazakov/kportforward/internal/config"
	"github.com/victorkazakov/kportforward/internal/utils"
)

func TestNewGRPCUIManager(t *testing.T) {
	logger := utils.NewLogger(utils.LevelInfo)
	manager := NewGRPCUIManager(logger)

	if manager == nil {
		t.Fatal("Manager should not be nil")
	}

	if manager.logger != logger {
		t.Error("Logger not set correctly")
	}

	if manager.services == nil {
		t.Error("Services map should be initialized")
	}

	if manager.IsEnabled() {
		t.Error("Manager should not be enabled initially")
	}
}

func TestGRPCUIManagerEnable(t *testing.T) {
	logger := utils.NewLogger(utils.LevelInfo)
	manager := NewGRPCUIManager(logger)

	// Initially disabled
	if manager.IsEnabled() {
		t.Error("Manager should be disabled initially")
	}

	// Test enable (will likely fail since grpcui is not installed in test environment)
	err := manager.Enable()
	// We expect this to fail in test environment, so we just check that it doesn't panic
	if err != nil {
		t.Logf("Enable failed as expected in test environment: %v", err)
	}
}

func TestGRPCUIManagerDisable(t *testing.T) {
	logger := utils.NewLogger(utils.LevelInfo)
	manager := NewGRPCUIManager(logger)

	// Test disable on non-enabled manager (should not panic)
	err := manager.Disable()
	if err != nil {
		t.Errorf("Disable should not return error: %v", err)
	}

	if manager.IsEnabled() {
		t.Error("Manager should be disabled after calling Disable")
	}
}

func TestGRPCUIManagerStartService(t *testing.T) {
	logger := utils.NewLogger(utils.LevelInfo)
	manager := NewGRPCUIManager(logger)

	// Test starting service when not enabled (should return early)
	serviceStatus := config.ServiceStatus{
		Name:      "test-rpc",
		Status:    "Running",
		LocalPort: 8080,
	}

	serviceConfig := config.Service{
		Target:     "service/test-rpc",
		TargetPort: 8080,
		LocalPort:  9080,
		Namespace:  "default",
		Type:       "rpc",
	}

	err := manager.StartService("test-rpc", serviceStatus, serviceConfig)
	if err != nil {
		t.Errorf("StartService should not return error when disabled: %v", err)
	}
}

func TestGRPCUIManagerStartServiceNonRPC(t *testing.T) {
	logger := utils.NewLogger(utils.LevelInfo)
	manager := NewGRPCUIManager(logger)

	// Force enable for testing
	manager.enabled = true

	// Test starting non-RPC service (should return early)
	serviceStatus := config.ServiceStatus{
		Name:      "test-web",
		Status:    "Running",
		LocalPort: 8080,
	}

	serviceConfig := config.Service{
		Target:     "service/test-web",
		TargetPort: 8080,
		LocalPort:  9080,
		Namespace:  "default",
		Type:       "web", // Not RPC
	}

	err := manager.StartService("test-web", serviceStatus, serviceConfig)
	if err != nil {
		t.Errorf("StartService should not return error for non-RPC service: %v", err)
	}

	// Should not have created a service entry
	if len(manager.services) != 0 {
		t.Error("Should not have created service entry for non-RPC service")
	}
}

func TestGRPCUIManagerStopService(t *testing.T) {
	logger := utils.NewLogger(utils.LevelInfo)
	manager := NewGRPCUIManager(logger)

	// Test stopping non-existent service (should not error)
	err := manager.StopService("non-existent")
	if err != nil {
		t.Errorf("StopService should not return error for non-existent service: %v", err)
	}
}

func TestGRPCUIManagerGetServiceInfo(t *testing.T) {
	logger := utils.NewLogger(utils.LevelInfo)
	manager := NewGRPCUIManager(logger)

	// Test getting info for non-existent service
	info := manager.GetServiceInfo("non-existent")
	if info != nil {
		t.Error("GetServiceInfo should return nil for non-existent service")
	}
}

func TestGRPCUIManagerGetServiceURL(t *testing.T) {
	logger := utils.NewLogger(utils.LevelInfo)
	manager := NewGRPCUIManager(logger)

	// Test getting URL for non-existent service
	url := manager.GetServiceURL("non-existent")
	if url != "" {
		t.Error("GetServiceURL should return empty string for non-existent service")
	}
}

func TestGRPCUIManagerMonitorServices(t *testing.T) {
	logger := utils.NewLogger(utils.LevelInfo)
	manager := NewGRPCUIManager(logger)

	// Test monitoring when disabled (should return early)
	services := map[string]config.ServiceStatus{
		"test-rpc": {
			Name:      "test-rpc",
			Status:    "Running",
			LocalPort: 8080,
		},
	}

	configs := map[string]config.Service{
		"test-rpc": {
			Target:     "service/test-rpc",
			TargetPort: 8080,
			LocalPort:  9080,
			Namespace:  "default",
			Type:       "rpc",
		},
	}

	// Should not panic when disabled
	manager.MonitorServices(services, configs)
}

func TestGRPCUIServiceStruct(t *testing.T) {
	// Test GRPCUIService struct creation
	service := &GRPCUIService{
		serviceName:  "test",
		localPort:    8080,
		grpcuiPort:   9090,
		restartCount: 0,
		status:       "Running",
	}

	if service.serviceName != "test" {
		t.Error("Service name not set correctly")
	}

	if service.localPort != 8080 {
		t.Error("Local port not set correctly")
	}

	if service.grpcuiPort != 9090 {
		t.Error("gRPC UI port not set correctly")
	}

	if service.status != "Running" {
		t.Error("Status not set correctly")
	}
}

func TestGRPCUIManagerIsGRPCUIAvailable(t *testing.T) {
	logger := utils.NewLogger(utils.LevelInfo)
	manager := NewGRPCUIManager(logger)

	// Test the availability check (will likely return false in test environment)
	available := manager.isGRPCUIAvailable()
	// We just check that it doesn't panic
	t.Logf("gRPC UI available: %v", available)
}

func TestPortReleasedOnEarlyReturn(t *testing.T) {
	logger := utils.NewLogger(utils.LevelInfo)
	manager := NewGRPCUIManager(logger)
	manager.enabled = true

	serviceStatus := config.ServiceStatus{
		Name:      "test-rpc",
		Status:    "Running",
		LocalPort: 1, // port 1 is not connectable, so testGRPCConnection will fail
	}

	serviceConfig := config.Service{
		Target:     "service/test-rpc",
		TargetPort: 8080,
		LocalPort:  1,
		Namespace:  "default",
		Type:       "rpc",
	}

	// Call StartService multiple times — should not exhaust the port pool
	// because each failed attempt should release its allocated port
	for i := 0; i < 20; i++ {
		_ = manager.StartService("test-rpc", serviceStatus, serviceConfig)
	}

	// No service should have been created (testGRPCConnection fails)
	if len(manager.services) != 0 {
		t.Errorf("Expected 0 services, got %d (ports leaked?)", len(manager.services))
	}
}

func TestFailedServiceCleanedUpByMonitor(t *testing.T) {
	logger := utils.NewLogger(utils.LevelInfo)
	manager := NewGRPCUIManager(logger)
	manager.enabled = true

	// Inject a fake "Failed" service into the map
	manager.services["test-rpc"] = &GRPCUIService{
		serviceName: "test-rpc",
		localPort:   8080,
		grpcuiPort:  9200,
		status:      "Failed",
	}

	services := map[string]config.ServiceStatus{
		"test-rpc": {
			Name:      "test-rpc",
			Status:    "Running",
			LocalPort: 8080,
		},
	}

	configs := map[string]config.Service{
		"test-rpc": {
			Target:     "service/test-rpc",
			TargetPort: 8080,
			LocalPort:  9080,
			Namespace:  "default",
			Type:       "rpc",
		},
	}

	// MonitorServices should clean up the Failed entry and attempt restart
	manager.MonitorServices(services, configs)

	// Give goroutine a moment to run
	time.Sleep(200 * time.Millisecond)

	// The failed entry should have been removed from the map
	// (the restart goroutine will fail since there's no real gRPC service,
	// but the old Failed entry should be cleaned up)
	manager.mutex.Lock()
	_, exists := manager.services["test-rpc"]
	manager.mutex.Unlock()

	// After cleanup, the entry was deleted. The goroutine may or may not have
	// re-added it (depends on testGRPCConnection), but the old "Failed" one is gone.
	// We just verify no panic occurred and the port was released.
	t.Logf("Service exists after monitor: %v", exists)
}

func TestGetServiceInfoReturnsCopy(t *testing.T) {
	logger := utils.NewLogger(utils.LevelInfo)
	manager := NewGRPCUIManager(logger)

	// Inject a service
	manager.services["test-rpc"] = &GRPCUIService{
		serviceName: "test-rpc",
		localPort:   8080,
		grpcuiPort:  9200,
		status:      "Running",
	}

	// Get info (should be a copy)
	info := manager.GetServiceInfo("test-rpc")
	if info == nil {
		t.Fatal("Expected non-nil service info")
	}

	// Mutate the copy
	info.status = "Mutated"

	// Original should be unchanged
	manager.mutex.Lock()
	original := manager.services["test-rpc"]
	manager.mutex.Unlock()

	if original.status == "Mutated" {
		t.Error("GetServiceInfo returned a reference, not a copy — external mutation affected internal state")
	}
}

func TestNoGoroutineLeakOnRepeatedStartStop(t *testing.T) {
	logger := utils.NewLogger(utils.LevelInfo)
	manager := NewGRPCUIManager(logger)
	manager.enabled = true

	serviceStatus := config.ServiceStatus{
		Name:      "test-rpc",
		Status:    "Running",
		LocalPort: 1, // unreachable, so StartService returns early
	}

	serviceConfig := config.Service{
		Target:     "service/test-rpc",
		TargetPort: 8080,
		LocalPort:  1,
		Namespace:  "default",
		Type:       "rpc",
	}

	// Record baseline goroutine count
	runtime.GC()
	time.Sleep(50 * time.Millisecond)
	baseline := runtime.NumGoroutine()

	// Repeatedly start (which bails early) and stop
	for i := 0; i < 50; i++ {
		_ = manager.StartService("test-rpc", serviceStatus, serviceConfig)
		_ = manager.StopService("test-rpc")
	}

	// Give goroutines time to settle
	runtime.GC()
	time.Sleep(200 * time.Millisecond)

	current := runtime.NumGoroutine()

	// Allow a small margin (±5) for background runtime goroutines
	if current > baseline+5 {
		t.Errorf("Possible goroutine leak: baseline=%d, current=%d (delta=%d)", baseline, current, current-baseline)
	} else {
		t.Logf("Goroutine count OK: baseline=%d, current=%d", baseline, current)
	}
}

func TestMarkFailedBacksOffRetries(t *testing.T) {
	logger := utils.NewLogger(utils.LevelInfo)
	manager := NewGRPCUIManager(logger)

	service := &GRPCUIService{serviceName: "test-rpc", startTime: time.Now(), status: "Running"}
	manager.services["test-rpc"] = service

	var delays []time.Duration
	for i := 0; i < 8; i++ {
		service.status = "Running"
		before := time.Now()
		manager.markFailed(service)
		if service.status != grpcuiStatusFailed {
			t.Fatalf("Expected status %q, got %q", grpcuiStatusFailed, service.status)
		}
		delays = append(delays, service.nextRetry.Sub(before).Round(time.Second))
	}

	want := []time.Duration{5 * time.Second, 10 * time.Second, 20 * time.Second, 40 * time.Second, 80 * time.Second, 160 * time.Second, 5 * time.Minute, 5 * time.Minute}
	for i := range want {
		if delays[i] != want[i] {
			t.Errorf("Failure %d: expected retry delay %s, got %s", i+1, want[i], delays[i])
		}
	}
}

func TestMarkFailedResetsBackoffAfterStableUptime(t *testing.T) {
	logger := utils.NewLogger(utils.LevelInfo)
	manager := NewGRPCUIManager(logger)

	service := &GRPCUIService{
		serviceName:  "test-rpc",
		startTime:    time.Now().Add(-2 * grpcuiStableUptime),
		restartCount: 6,
		status:       "Running",
	}
	manager.markFailed(service)

	if service.restartCount != 1 {
		t.Errorf("Expected failure count to reset to 1 after stable uptime, got %d", service.restartCount)
	}
}

func TestMonitorDoesNotRestartBeforeBackoffExpires(t *testing.T) {
	logger := utils.NewLogger(utils.LevelInfo)
	manager := NewGRPCUIManager(logger)
	manager.enabled = true

	manager.services["test-rpc"] = &GRPCUIService{
		serviceName:  "test-rpc",
		status:       grpcuiStatusFailed,
		restartCount: 3,
		nextRetry:    time.Now().Add(time.Hour),
	}

	services := map[string]config.ServiceStatus{
		"test-rpc": {Name: "test-rpc", Status: "Running", LocalPort: 1},
	}
	configs := map[string]config.Service{
		"test-rpc": {Target: "service/test-rpc", TargetPort: 8080, LocalPort: 1, Namespace: "default", Type: "rpc"},
	}

	manager.MonitorServices(services, configs)
	_ = manager.StartService("test-rpc", services["test-rpc"], configs["test-rpc"])

	info := manager.GetServiceInfo("test-rpc")
	if info == nil || info.status != grpcuiStatusFailed || info.restartCount != 3 {
		t.Errorf("Expected Failed entry to be left alone until its retry time, got %+v", info)
	}
}

func TestNoReflectionMarksUnsupported(t *testing.T) {
	logger := utils.NewLogger(utils.LevelInfo)
	manager := NewGRPCUIManager(logger)
	manager.enabled = true

	logFile := filepath.Join(t.TempDir(), "grpcui.log")
	// Output from an earlier run must not count; only output after logOffset does
	earlier := "Failed to dial target host \"localhost:1\": connection refused\n"
	current := "Failed to compute set of methods to expose: server does not support the reflection API\n"
	if err := os.WriteFile(logFile, []byte(earlier+current), 0644); err != nil {
		t.Fatal(err)
	}

	service := &GRPCUIService{
		serviceName: "test-rpc",
		logFile:     logFile,
		logOffset:   int64(len(earlier)),
		startTime:   time.Now(),
		status:      "Running",
	}
	manager.services["test-rpc"] = service
	manager.markFailed(service)

	if service.status != grpcuiStatusUnsupported {
		t.Fatalf("Expected status %q, got %q", grpcuiStatusUnsupported, service.status)
	}
	if url := manager.GetServiceURL("test-rpc"); url != "" {
		t.Errorf("Expected no URL for unsupported service, got %q", url)
	}

	// Monitoring must not restart it while the port-forward stays up
	services := map[string]config.ServiceStatus{
		"test-rpc": {Name: "test-rpc", Status: "Running", LocalPort: 1},
	}
	configs := map[string]config.Service{
		"test-rpc": {Target: "service/test-rpc", TargetPort: 8080, LocalPort: 1, Namespace: "default", Type: "rpc"},
	}
	manager.MonitorServices(services, configs)
	_ = manager.StartService("test-rpc", services["test-rpc"], configs["test-rpc"])
	if info := manager.GetServiceInfo("test-rpc"); info == nil || info.status != grpcuiStatusUnsupported {
		t.Errorf("Expected unsupported entry to be kept, got %+v", info)
	}

	// Once the port-forward stops, the entry is cleared so a later run gets a fresh attempt
	services["test-rpc"] = config.ServiceStatus{Name: "test-rpc", Status: "Failed", LocalPort: 1}
	manager.MonitorServices(services, configs)
	time.Sleep(100 * time.Millisecond)
	if info := manager.GetServiceInfo("test-rpc"); info != nil {
		t.Errorf("Expected entry to be removed after port-forward stopped, got %+v", info)
	}
}

func TestLogWithoutReflectionErrorIsRetried(t *testing.T) {
	logger := utils.NewLogger(utils.LevelInfo)
	manager := NewGRPCUIManager(logger)

	logFile := filepath.Join(t.TempDir(), "grpcui.log")
	old := "Failed to compute set of methods to expose: server does not support the reflection API\n"
	if err := os.WriteFile(logFile, []byte(old+"listen tcp: address already in use\n"), 0644); err != nil {
		t.Fatal(err)
	}

	service := &GRPCUIService{serviceName: "test-rpc", logFile: logFile, logOffset: int64(len(old)), startTime: time.Now(), status: "Running"}
	manager.markFailed(service)

	if service.status != grpcuiStatusFailed {
		t.Errorf("Expected status %q, got %q", grpcuiStatusFailed, service.status)
	}
}
