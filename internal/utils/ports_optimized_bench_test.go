package utils

import (
	"fmt"
	"testing"
	"time"
)

// BenchmarkOptimizedPortChecker tests the optimized port checker
func BenchmarkOptimizedPortChecker(b *testing.B) {
	checker := NewPortChecker(5 * time.Second)
	port := 45000

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		checker.IsPortAvailableOptimized(port + (i % 100))
	}
}

// BenchmarkOptimizedPortResolver tests the optimized port resolver
func BenchmarkOptimizedPortResolver(b *testing.B) {
	resolver := NewOptimizedPortResolver()

	// Create test services
	services := make(map[string]ServiceConfig)
	basePort := 60000

	for i := range 50 {
		services[fmt.Sprintf("service-%d", i)] = ServiceConfig{
			LocalPort: basePort + (i % 10), // Create conflicts every 10 services
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := resolver.ResolvePortConflictsOptimized(services)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkBatchPortCheck tests batch port checking
func BenchmarkBatchPortCheck(b *testing.B) {
	checker := NewPortChecker(5 * time.Second)
	ports := make([]int, 50)
	for i := range ports {
		ports[i] = 40000 + i
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		results := checker.BatchPortCheck(ports)
		if len(results) != len(ports) {
			b.Fatalf("Expected %d results, got %d", len(ports), len(results))
		}
	}
}

// BenchmarkOptimizedPortFinder tests the optimized port finder
func BenchmarkOptimizedPortFinder(b *testing.B) {
	finder := NewOptimizedPortFinder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		port, err := finder.FindAvailablePortFast(50000 + (i % 1000))
		if err != nil {
			b.Fatal(err)
		}
		if port == 0 {
			b.Fatal("Invalid port returned")
		}
	}
}

// BenchmarkCachedVsUncachedPortCheck compares cached vs uncached performance
func BenchmarkCachedVsUncachedPortCheck(b *testing.B) {
	checker := NewPortChecker(5 * time.Second)
	port := 45000

	b.Run("Uncached", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			IsPortAvailable(port + (i % 100))
		}
	})

	b.Run("Cached", func(b *testing.B) {
		// Warm up cache
		for i := range 100 {
			checker.IsPortAvailableOptimized(port + i)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			checker.IsPortAvailableOptimized(port + (i % 100))
		}
	})
}

// BenchmarkOriginalVsOptimizedResolver compares old vs new implementations
func BenchmarkOriginalVsOptimizedResolver(b *testing.B) {
	services := make(map[string]ServiceConfig)
	basePort := 60000

	for i := range 50 {
		services[fmt.Sprintf("service-%d", i)] = ServiceConfig{
			LocalPort: basePort + (i % 10),
		}
	}

	b.Run("Original", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := ResolvePortConflicts(services)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Optimized", func(b *testing.B) {
		resolver := NewOptimizedPortResolver()
		for i := 0; i < b.N; i++ {
			_, err := resolver.ResolvePortConflictsOptimized(services)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkConcurrentOptimizedOperations tests concurrent optimized operations
func BenchmarkConcurrentOptimizedOperations(b *testing.B) {
	checker := NewPortChecker(5 * time.Second)
	finder := NewOptimizedPortFinder()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			port := 40000 + (i % 1000)

			// Test both operations
			checker.IsPortAvailableOptimized(port)
			finder.FindAvailablePortFast(port + 1000)

			i++
		}
	})
}
