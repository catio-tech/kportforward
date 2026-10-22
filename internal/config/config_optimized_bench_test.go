package config

import (
	"fmt"
	"testing"
)

// BenchmarkOptimizedConfigLoader tests the optimized config loader
func BenchmarkOptimizedConfigLoader(b *testing.B) {
	loader := NewOptimizedConfigLoader()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cfg, err := loader.LoadConfigOptimized()
		if err != nil {
			b.Fatal(err)
		}
		if len(cfg.PortForwards) == 0 {
			b.Fatal("No port forwards loaded")
		}
	}
}

// BenchmarkOptimizedConfigWithStats tests the stats-enabled config loader
func BenchmarkOptimizedConfigWithStats(b *testing.B) {
	loader := NewOptimizedConfigWithStats()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cfg, stats, err := loader.LoadConfigWithStats()
		if err != nil {
			b.Fatal(err)
		}
		if len(cfg.PortForwards) == 0 {
			b.Fatal("No port forwards loaded")
		}
		_ = stats // Use stats to avoid unused variable warning
	}
}

// BenchmarkConfigCaching tests the performance of cached vs uncached loading
func BenchmarkConfigCaching(b *testing.B) {
	b.Run("Uncached", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			cfg, err := LoadConfig()
			if err != nil {
				b.Fatal(err)
			}
			if len(cfg.PortForwards) == 0 {
				b.Fatal("No port forwards loaded")
			}
		}
	})

	b.Run("Cached", func(b *testing.B) {
		loader := NewOptimizedConfigLoader()

		// Warm up cache
		_, err := loader.LoadConfigOptimized()
		if err != nil {
			b.Fatal(err)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			cfg, err := loader.LoadConfigOptimized()
			if err != nil {
				b.Fatal(err)
			}
			if len(cfg.PortForwards) == 0 {
				b.Fatal("No port forwards loaded")
			}
		}
	})
}

// BenchmarkLoadConfigFast tests the drop-in replacement function
func BenchmarkLoadConfigFast(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cfg, err := LoadConfigFast()
		if err != nil {
			b.Fatal(err)
		}
		if len(cfg.PortForwards) == 0 {
			b.Fatal("No port forwards loaded")
		}
	}
}

// BenchmarkOriginalVsOptimizedConfig compares original vs optimized config loading
func BenchmarkOriginalVsOptimizedConfig(b *testing.B) {
	b.Run("Original", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			cfg, err := LoadConfig()
			if err != nil {
				b.Fatal(err)
			}
			if len(cfg.PortForwards) == 0 {
				b.Fatal("No port forwards loaded")
			}
		}
	})

	b.Run("Optimized", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			cfg, err := LoadConfigFast()
			if err != nil {
				b.Fatal(err)
			}
			if len(cfg.PortForwards) == 0 {
				b.Fatal("No port forwards loaded")
			}
		}
	})
}

// BenchmarkConcurrentConfigLoading tests concurrent config loading
func BenchmarkConcurrentConfigLoading(b *testing.B) {
	loader := NewOptimizedConfigLoader()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cfg, err := loader.LoadConfigOptimized()
			if err != nil {
				b.Fatal(err)
			}
			if len(cfg.PortForwards) == 0 {
				b.Fatal("No port forwards loaded")
			}
		}
	})
}

// BenchmarkConfigCopyOperations tests the performance of config copying
func BenchmarkConfigCopyOperations(b *testing.B) {
	loader := NewOptimizedConfigLoader()
	cfg, err := loader.LoadConfigOptimized()
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		copy := loader.copyConfig(cfg)
		if len(copy.PortForwards) != len(cfg.PortForwards) {
			b.Fatal("Copy failed")
		}
	}
}

// BenchmarkOptimizedMerging tests the optimized config merging
func BenchmarkOptimizedMerging(b *testing.B) {
	loader := NewOptimizedConfigLoader()

	// Get default config
	defaultConfig, err := loader.getDefaultConfigOptimized()
	if err != nil {
		b.Fatal(err)
	}

	// Create a user config
	userConfig := &Config{
		PortForwards: make(map[string]Service),
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
		merged := loader.mergeConfigsOptimized(defaultConfig, userConfig)
		expectedServices := len(defaultConfig.PortForwards) + len(userConfig.PortForwards)
		if len(merged.PortForwards) != expectedServices {
			b.Fatalf("Expected %d services, got %d", expectedServices, len(merged.PortForwards))
		}
	}
}
