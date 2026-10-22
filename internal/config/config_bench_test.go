package config

import (
	"fmt"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

// BenchmarkLoadConfig tests the performance of loading the embedded configuration
func BenchmarkLoadConfig(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cfg, err := LoadConfig()
		if err != nil {
			b.Fatal(err)
		}
		if len(cfg.PortForwards) == 0 {
			b.Fatal("No port forwards loaded")
		}
	}
}

// BenchmarkConfigMerging tests the performance of merging user config with defaults
func BenchmarkConfigMerging(b *testing.B) {
	// Create a test config
	defaultConfig := &Config{
		PortForwards:       make(map[string]Service),
		MonitoringInterval: 5 * time.Second,
		UIOptions: UIConfig{
			RefreshRate: 1 * time.Second,
			Theme:       "light",
		},
	}

	// Add some default services
	for i := range 20 {
		serviceName := fmt.Sprintf("default-service-%d", i)
		defaultConfig.PortForwards[serviceName] = Service{
			Target:     fmt.Sprintf("service/%s", serviceName),
			TargetPort: 8080 + i,
			LocalPort:  9000 + i,
			Namespace:  "default",
			Type:       "web",
		}
	}

	// Create user config
	userConfig := &Config{
		PortForwards:       make(map[string]Service),
		MonitoringInterval: 3 * time.Second,
		UIOptions: UIConfig{
			RefreshRate: 500 * time.Millisecond,
			Theme:       "dark",
		},
	}

	// Add some user services
	for i := range 10 {
		serviceName := fmt.Sprintf("user-service-%d", i)
		userConfig.PortForwards[serviceName] = Service{
			Target:     fmt.Sprintf("service/%s", serviceName),
			TargetPort: 7000 + i,
			LocalPort:  8000 + i,
			Namespace:  "user",
			Type:       "rest",
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		merged := mergeConfigs(defaultConfig, userConfig)
		if len(merged.PortForwards) != 30 {
			b.Fatalf("Expected 30 services, got %d", len(merged.PortForwards))
		}
	}
}

// BenchmarkYAMLUnmarshal tests the performance of YAML parsing
func BenchmarkYAMLUnmarshal(b *testing.B) {
	// Test with the embedded default config
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		config := &Config{}
		err := yaml.Unmarshal(DefaultConfigYAML, config)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkLargeConfigMerging tests performance with many services
func BenchmarkLargeConfigMerging(b *testing.B) {
	defaultConfig := &Config{
		PortForwards:       make(map[string]Service),
		MonitoringInterval: 5 * time.Second,
	}

	userConfig := &Config{
		PortForwards:       make(map[string]Service),
		MonitoringInterval: 3 * time.Second,
	}

	// Create 100 default services
	for i := range 100 {
		serviceName := fmt.Sprintf("default-service-%d", i)
		defaultConfig.PortForwards[serviceName] = Service{
			Target:     fmt.Sprintf("service/%s", serviceName),
			TargetPort: 8000 + i,
			LocalPort:  9000 + i,
			Namespace:  "default",
			Type:       "web",
		}
	}

	// Create 50 user services
	for i := range 50 {
		serviceName := fmt.Sprintf("user-service-%d", i)
		userConfig.PortForwards[serviceName] = Service{
			Target:     fmt.Sprintf("service/%s", serviceName),
			TargetPort: 7000 + i,
			LocalPort:  8000 + i,
			Namespace:  "user",
			Type:       "rest",
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		merged := mergeConfigs(defaultConfig, userConfig)
		if len(merged.PortForwards) != 150 {
			b.Fatalf("Expected 150 services, got %d", len(merged.PortForwards))
		}
	}
}

// BenchmarkConfigValidation tests service validation performance
func BenchmarkConfigValidation(b *testing.B) {
	cfg, err := LoadConfig()
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, service := range cfg.PortForwards {
			// Simulate validation
			if service.Target == "" || service.TargetPort <= 0 ||
				service.LocalPort <= 0 || service.Namespace == "" {
				b.Fatal("Invalid service configuration")
			}
		}
	}
}
