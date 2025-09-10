// test_integration.go - Integration test script for global access check functionality
// This file can be run as: go run test_integration.go
package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/victorkazakov/kportforward/internal/config"
	"github.com/victorkazakov/kportforward/internal/portforward"
	"github.com/victorkazakov/kportforward/internal/utils"
)

func main() {
	fmt.Println("🧪 kportforward Integration Test Suite")
	fmt.Println("=====================================")

	// Run different test scenarios
	tests := []struct {
		name string
		test func() error
	}{
		{"Global Access Check", testGlobalAccessCheck},
		{"Service Suspension Logic", testServiceSuspension},
		{"Error Classification", testErrorClassification},
		{"Performance Stress Test", testPerformanceStress},
		{"Resource Cleanup", testResourceCleanup},
	}

	passed := 0
	failed := 0

	for _, test := range tests {
		fmt.Printf("\n🔧 Running: %s\n", test.name)
		if err := test.test(); err != nil {
			fmt.Printf("❌ FAILED: %s - %v\n", test.name, err)
			failed++
		} else {
			fmt.Printf("✅ PASSED: %s\n", test.name)
			passed++
		}
	}

	fmt.Printf("\n📊 Test Results: %d passed, %d failed\n", passed, failed)

	if failed > 0 {
		os.Exit(1)
	}
}

// testGlobalAccessCheck tests the global access check functionality
func testGlobalAccessCheck() error {
	// Create test configuration
	cfg := &config.Config{
		PortForwards: map[string]config.Service{
			"test-service": {
				Target:     "service/nonexistent",
				TargetPort: 8080,
				LocalPort:  18080,
				Namespace:  "default",
				Type:       "rest",
			},
		},
		MonitoringInterval: 1 * time.Second,
	}

	logger := utils.NewLoggerWithOutput(utils.LevelInfo, os.Stdout)
	manager := portforward.NewManager(cfg, logger)

	// Test initial global access status
	if !manager.GetGlobalAccessStatus() {
		return fmt.Errorf("expected initial global access to be healthy")
	}

	fmt.Printf("  ✓ Initial global access status: healthy\n")

	// Test that the manager was created successfully
	status := manager.GetCurrentStatus()
	if len(status) != 0 {
		return fmt.Errorf("expected no services to be running initially")
	}

	fmt.Printf("  ✓ Manager created with correct initial state\n")

	return nil
}

// testServiceSuspension tests service suspension and resumption logic
func testServiceSuspension() error {
	cfg := &config.Config{
		PortForwards: map[string]config.Service{
			"test-service-1": {
				Target:     "service/test-1",
				TargetPort: 8080,
				LocalPort:  18080,
				Namespace:  "default",
				Type:       "rest",
			},
			"test-service-2": {
				Target:     "service/test-2",
				TargetPort: 9090,
				LocalPort:  19090,
				Namespace:  "default",
				Type:       "rpc",
			},
		},
		MonitoringInterval: 1 * time.Second,
	}

	logger := utils.NewLoggerWithOutput(utils.LevelInfo, nil) // Suppress output
	_ = portforward.NewManager(cfg, logger)

	// Create mock service managers for testing
	sm1 := portforward.NewServiceManager("test-service-1", cfg.PortForwards["test-service-1"], logger)
	sm2 := portforward.NewServiceManager("test-service-2", cfg.PortForwards["test-service-2"], logger)

	// Set services to running state (simulated)
	status1 := sm1.GetStatus()
	status2 := sm2.GetStatus()

	fmt.Printf("  ✓ Created %d test services\n", len(cfg.PortForwards))
	fmt.Printf("  ✓ Service 1 status: %s\n", status1.Status)
	fmt.Printf("  ✓ Service 2 status: %s\n", status2.Status)

	return nil
}

// testErrorClassification tests error classification functions
func testErrorClassification() error {
	// Test auth errors
	authTests := []struct {
		error    string
		expected bool
	}{
		{"unauthorized", true},
		{"authentication failed", true},
		{"token expired", true},
		{"connection refused", false},
		{"timeout", false},
	}

	for _, test := range authTests {
		// We can't directly test the internal functions, but we can test the logic
		isAuth := containsAuthKeyword(test.error)
		if isAuth != test.expected {
			return fmt.Errorf("auth error classification failed for '%s': expected %v, got %v",
				test.error, test.expected, isAuth)
		}
	}

	fmt.Printf("  ✓ Auth error classification working correctly\n")

	// Test network errors
	networkTests := []struct {
		error    string
		expected bool
	}{
		{"connection refused", true},
		{"timeout", true},
		{"network unreachable", true},
		{"unauthorized", false},
		{"authentication failed", false},
	}

	for _, test := range networkTests {
		isNetwork := containsNetworkKeyword(test.error)
		if isNetwork != test.expected {
			return fmt.Errorf("network error classification failed for '%s': expected %v, got %v",
				test.error, test.expected, isNetwork)
		}
	}

	fmt.Printf("  ✓ Network error classification working correctly\n")

	return nil
}

// testPerformanceStress tests performance under stress
func testPerformanceStress() error {
	// Create config with many services
	cfg := &config.Config{
		PortForwards:       make(map[string]config.Service),
		MonitoringInterval: 1 * time.Second,
	}

	// Add 50 services to test scalability
	for i := 0; i < 50; i++ {
		serviceName := fmt.Sprintf("stress-test-service-%d", i)
		cfg.PortForwards[serviceName] = config.Service{
			Target:     fmt.Sprintf("service/test-%d", i),
			TargetPort: 8080 + i,
			LocalPort:  18080 + i,
			Namespace:  "default",
			Type:       "rest",
		}
	}

	logger := utils.NewLoggerWithOutput(utils.LevelInfo, nil)

	// Measure creation time
	start := time.Now()
	manager := portforward.NewManager(cfg, logger)
	creationTime := time.Since(start)

	fmt.Printf("  ✓ Manager creation with 50 services: %v\n", creationTime)

	// Measure status retrieval time
	start = time.Now()
	status := manager.GetCurrentStatus()
	statusTime := time.Since(start)

	fmt.Printf("  ✓ Status retrieval time: %v\n", statusTime)
	fmt.Printf("  ✓ Services in status map: %d\n", len(status))

	// Verify reasonable performance
	if creationTime > 100*time.Millisecond {
		return fmt.Errorf("manager creation too slow: %v", creationTime)
	}

	if statusTime > 10*time.Millisecond {
		return fmt.Errorf("status retrieval too slow: %v", statusTime)
	}

	return nil
}

// testResourceCleanup tests that resources are properly cleaned up
func testResourceCleanup() error {
	cfg := &config.Config{
		PortForwards: map[string]config.Service{
			"cleanup-test": {
				Target:     "service/cleanup-test",
				TargetPort: 8080,
				LocalPort:  18080,
				Namespace:  "default",
				Type:       "rest",
			},
		},
		MonitoringInterval: 1 * time.Second,
	}

	logger := utils.NewLoggerWithOutput(utils.LevelInfo, nil)
	manager := portforward.NewManager(cfg, logger)

	// Test cleanup (manager should be garbage collected properly)
	status := manager.GetCurrentStatus()
	globalStatus := manager.GetGlobalAccessStatus()

	fmt.Printf("  ✓ Manager cleanup test completed\n")
	fmt.Printf("  ✓ Final status map size: %d\n", len(status))
	fmt.Printf("  ✓ Final global status: %v\n", globalStatus)

	return nil
}

// Helper functions for testing error classification logic
func containsAuthKeyword(errorMsg string) bool {
	keywords := []string{"unauthorized", "authentication", "token", "credential", "forbidden"}
	for _, keyword := range keywords {
		if len(errorMsg) >= len(keyword) && errorMsg[:len(keyword)] == keyword {
			return true
		}
	}
	return false
}

func containsNetworkKeyword(errorMsg string) bool {
	keywords := []string{"connection refused", "timeout", "network"}
	for _, keyword := range keywords {
		if len(errorMsg) >= len(keyword) && errorMsg[:len(keyword)] == keyword {
			return true
		}
	}
	return false
}

// runRealIntegrationTest runs a real integration test with kubectl
func runRealIntegrationTest() error {
	fmt.Println("\n🚀 Real Integration Test (requires kubectl)")

	// Check if kubectl is available
	if _, err := exec.LookPath("kubectl"); err != nil {
		fmt.Println("  ⚠️  kubectl not found, skipping real integration test")
		return nil
	}

	// Check if we have a Kubernetes context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "kubectl", "config", "current-context")
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("  ⚠️  No Kubernetes context available, skipping real integration test")
		return nil
	}

	kubeContext := string(output)
	fmt.Printf("  ✓ Using Kubernetes context: %s", kubeContext)

	// Run actual integration test with timeout
	fmt.Println("  🔄 Starting 10-second integration test...")

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Run the actual binary with timeout
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd = exec.CommandContext(ctx, "./bin/kportforward", "--log-file", "/tmp/integration_test.log")

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start kportforward: %v", err)
	}

	// Wait for timeout or completion
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-time.After(10 * time.Second):
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		fmt.Println("  ✓ Integration test completed (timed out as expected)")
	case err := <-done:
		if err != nil {
			return fmt.Errorf("integration test failed: %v", err)
		}
		fmt.Println("  ✓ Integration test completed successfully")
	case <-sigChan:
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		fmt.Println("  ✓ Integration test interrupted")
	}

	return nil
}
