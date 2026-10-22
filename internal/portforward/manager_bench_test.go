package portforward

import (
	"fmt"
	"testing"
	"time"

	"github.com/victorkazakov/kportforward/internal/config"
	"github.com/victorkazakov/kportforward/internal/utils"
)

// BenchmarkManagerCreation tests the performance of creating a manager
func BenchmarkManagerCreation(b *testing.B) {
	cfg := &config.Config{
		PortForwards:       make(map[string]config.Service),
		MonitoringInterval: 1 * time.Second,
	}

	// Add services to the config
	for i := range 20 {
		serviceName := fmt.Sprintf("service-%d", i)
		cfg.PortForwards[serviceName] = config.Service{
			Target:     fmt.Sprintf("service/%s", serviceName),
			TargetPort: 8000 + i,
			LocalPort:  9000 + i,
			Namespace:  "default",
			Type:       "web",
		}
	}

	logger := utils.NewLogger(utils.LevelInfo)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager := NewManager(cfg, logger)
		if manager == nil {
			b.Fatal("Manager creation failed")
		}
	}
}

// BenchmarkGetCurrentStatus tests the performance of status retrieval
func BenchmarkGetCurrentStatus(b *testing.B) {
	cfg := &config.Config{
		PortForwards:       make(map[string]config.Service),
		MonitoringInterval: 1 * time.Second,
	}

	// Add many services to test with
	for i := range 100 {
		serviceName := fmt.Sprintf("service-%d", i)
		cfg.PortForwards[serviceName] = config.Service{
			Target:     fmt.Sprintf("service/%s", serviceName),
			TargetPort: 8000 + i,
			LocalPort:  9000 + i,
			Namespace:  "default",
			Type:       "web",
		}
	}

	logger := utils.NewLogger(utils.LevelInfo)
	manager := NewManager(cfg, logger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		status := manager.GetCurrentStatus()
		if status == nil {
			b.Fatal("Status retrieval failed")
		}
	}
}

// BenchmarkUIHandlerIntegration tests the performance of UI handler operations
func BenchmarkUIHandlerIntegration(b *testing.B) {
	cfg := &config.Config{
		PortForwards:       make(map[string]config.Service),
		MonitoringInterval: 1 * time.Second,
	}

	// Add services with different types
	for i := range 50 {
		serviceName := fmt.Sprintf("service-%d", i)
		serviceType := "web"
		if i%3 == 0 {
			serviceType = "rpc"
		} else if i%3 == 1 {
			serviceType = "rest"
		}

		cfg.PortForwards[serviceName] = config.Service{
			Target:     fmt.Sprintf("service/%s", serviceName),
			TargetPort: 8000 + i,
			LocalPort:  9000 + i,
			Namespace:  "default",
			Type:       serviceType,
		}
	}

	logger := utils.NewLogger(utils.LevelInfo)
	manager := NewManager(cfg, logger)

	// Create mock UI handlers
	grpcHandler := NewMockUIHandler()
	swaggerHandler := NewMockUIHandler()
	grpcHandler.Enable()
	swaggerHandler.Enable()

	manager.SetUIHandlers(grpcHandler, swaggerHandler)

	// Simulate status map
	statusMap := make(map[string]config.ServiceStatus)
	for serviceName := range cfg.PortForwards {
		statusMap[serviceName] = config.ServiceStatus{
			Name:      serviceName,
			Status:    "Running",
			LocalPort: 9000,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.monitorUIHandlers(statusMap)
	}
}

// BenchmarkConcurrentManagerOperations tests concurrent access to manager
func BenchmarkConcurrentManagerOperations(b *testing.B) {
	cfg := &config.Config{
		PortForwards:       make(map[string]config.Service),
		MonitoringInterval: 1 * time.Second,
	}

	// Add services
	for i := range 30 {
		serviceName := fmt.Sprintf("service-%d", i)
		cfg.PortForwards[serviceName] = config.Service{
			Target:     fmt.Sprintf("service/%s", serviceName),
			TargetPort: 8000 + i,
			LocalPort:  9000 + i,
			Namespace:  "default",
			Type:       "web",
		}
	}

	logger := utils.NewLogger(utils.LevelInfo)
	manager := NewManager(cfg, logger)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Simulate concurrent operations
			status := manager.GetCurrentStatus()
			context := manager.GetKubernetesContext()
			_ = status
			_ = context
		}
	})
}

// BenchmarkLargeServiceSet tests performance with many services
func BenchmarkLargeServiceSet(b *testing.B) {
	cfg := &config.Config{
		PortForwards:       make(map[string]config.Service),
		MonitoringInterval: 1 * time.Second,
	}

	// Add a large number of services
	for i := range 500 {
		serviceName := fmt.Sprintf("service-%d", i)
		cfg.PortForwards[serviceName] = config.Service{
			Target:     fmt.Sprintf("service/%s", serviceName),
			TargetPort: 8000 + i,
			LocalPort:  9000 + i,
			Namespace:  fmt.Sprintf("namespace-%d", i%10),
			Type:       []string{"web", "rest", "rpc"}[i%3],
		}
	}

	logger := utils.NewLogger(utils.LevelInfo)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager := NewManager(cfg, logger)
		status := manager.GetCurrentStatus()
		if len(status) != 0 { // Should be 0 since we haven't started
			b.Fatalf("Expected 0 running services, got %d", len(status))
		}
	}
}
