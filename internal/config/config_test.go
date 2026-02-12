package config

import (
	"testing"
)

func TestLoadDefaultConfig(t *testing.T) {
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load default config: %v", err)
	}

	// Test basic structure
	if cfg == nil {
		t.Fatal("Config is nil")
	}

	if len(cfg.PortForwards) == 0 {
		t.Fatal("No port forwards configured")
	}

	// Test monitoring interval default
	if cfg.MonitoringInterval == 0 {
		t.Error("MonitoringInterval not set")
	}

	// Test UI options
	if cfg.UIOptions.RefreshRate == 0 {
		t.Error("UIOptions.RefreshRate not set")
	}
}

func TestConfigMerging(t *testing.T) {
	// Test that we can load default config successfully
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Default config should have embedded services
	if len(cfg.PortForwards) == 0 {
		t.Error("Default config should have port forwards")
	}

	// Check for some expected services from embedded config
	expectedServices := []string{"flyte-console", "flyte-admin-rpc", "api-gateway"}
	foundServices := 0
	for _, serviceName := range expectedServices {
		if _, exists := cfg.PortForwards[serviceName]; exists {
			foundServices++
		}
	}

	if foundServices == 0 {
		t.Error("Expected to find some default services in config")
	}
}

func TestConfigStructure(t *testing.T) {
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Test that all services have required fields
	for name, service := range cfg.PortForwards {
		if service.Target == "" {
			t.Errorf("Service %s missing target", name)
		}
		if service.TargetPort <= 0 {
			t.Errorf("Service %s has invalid target port: %d", name, service.TargetPort)
		}
		if service.LocalPort <= 0 {
			t.Errorf("Service %s has invalid local port: %d", name, service.LocalPort)
		}
		if service.Namespace == "" {
			t.Errorf("Service %s missing namespace", name)
		}
	}
}

func TestServiceValidation(t *testing.T) {
	tests := []struct {
		name    string
		service Service
		valid   bool
	}{
		{
			name: "valid service",
			service: Service{
				Target:     "service/test",
				TargetPort: 8080,
				LocalPort:  9080,
				Namespace:  "default",
				Type:       "web",
			},
			valid: true,
		},
		{
			name: "missing target",
			service: Service{
				TargetPort: 8080,
				LocalPort:  9080,
				Namespace:  "default",
			},
			valid: false,
		},
		{
			name: "invalid port",
			service: Service{
				Target:     "service/test",
				TargetPort: 0,
				LocalPort:  9080,
				Namespace:  "default",
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := validateService(tt.service)
			if valid != tt.valid {
				t.Errorf("Expected validation result %v, got %v", tt.valid, valid)
			}
		})
	}
}

// validateService is a helper function for testing service validation
func validateService(service Service) bool {
	if service.Target == "" {
		return false
	}
	if service.TargetPort <= 0 || service.TargetPort > 65535 {
		return false
	}
	if service.LocalPort <= 0 || service.LocalPort > 65535 {
		return false
	}
	if service.Namespace == "" {
		return false
	}
	return true
}
func TestDisabledServiceDeletion(t *testing.T) {
	defaultCfg := &Config{
		PortForwards: map[string]Service{
			"svc-a": {Target: "service/a", TargetPort: 80, LocalPort: 8080, Namespace: "ns"},
			"svc-b": {Target: "service/b", TargetPort: 80, LocalPort: 8081, Namespace: "ns"},
		},
		MonitoringInterval: 5_000_000_000,
	}

	userCfg := &Config{
		PortForwards: map[string]Service{
			"svc-b": {Target: "service/b", TargetPort: 80, LocalPort: 8081, Namespace: "ns", Disabled: true},
			"svc-c": {Target: "service/c", TargetPort: 80, LocalPort: 8082, Namespace: "ns"},
		},
	}

	merged := mergeConfigs(defaultCfg, userCfg)

	if _, ok := merged.PortForwards["svc-a"]; !ok {
		t.Error("svc-a should be present (not disabled)")
	}
	if _, ok := merged.PortForwards["svc-b"]; ok {
		t.Error("svc-b should be deleted (disabled=true)")
	}
	if _, ok := merged.PortForwards["svc-c"]; !ok {
		t.Error("svc-c should be present (new user service)")
	}
	if len(merged.PortForwards) != 2 {
		t.Errorf("expected 2 services after merge, got %d", len(merged.PortForwards))
	}
}

func TestDisabledServiceDeletionOptimized(t *testing.T) {
	loader := NewOptimizedConfigLoader()

	defaultCfg := &Config{
		PortForwards: map[string]Service{
			"svc-a": {Target: "service/a", TargetPort: 80, LocalPort: 8080, Namespace: "ns"},
			"svc-b": {Target: "service/b", TargetPort: 80, LocalPort: 8081, Namespace: "ns"},
		},
		MonitoringInterval: 5_000_000_000,
	}

	userCfg := &Config{
		PortForwards: map[string]Service{
			"svc-b": {Disabled: true},
		},
	}

	merged := loader.mergeConfigsOptimized(defaultCfg, userCfg)

	if _, ok := merged.PortForwards["svc-a"]; !ok {
		t.Error("svc-a should be present")
	}
	if _, ok := merged.PortForwards["svc-b"]; ok {
		t.Error("svc-b should be deleted (disabled=true)")
	}
	if len(merged.PortForwards) != 1 {
		t.Errorf("expected 1 service after merge, got %d", len(merged.PortForwards))
	}
}
