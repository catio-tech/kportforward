package utils

import (
	"fmt"
	"testing"
)

// BenchmarkIsPortAvailable tests the performance of port availability checking
func BenchmarkIsPortAvailable(b *testing.B) {
	// Test with a port that's likely to be available
	port := 45000

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsPortAvailable(port + (i % 100)) // Vary the port to avoid caching effects
	}
}

// BenchmarkFindAvailablePort tests the performance of finding available ports
func BenchmarkFindAvailablePort(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		startPort := 50000 + (i % 1000) // Vary starting port
		_, err := FindAvailablePort(startPort)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkResolvePortConflicts tests bulk port conflict resolution
func BenchmarkResolvePortConflicts(b *testing.B) {
	// Create a set of services with potential conflicts
	services := make(map[string]ServiceConfig)
	basePort := 60000

	for i := range 50 {
		services[fmt.Sprintf("service-%d", i)] = ServiceConfig{
			LocalPort: basePort + (i % 10), // Create conflicts every 10 services
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ResolvePortConflicts(services)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkConcurrentPortChecks tests concurrent port availability checking
func BenchmarkConcurrentPortChecks(b *testing.B) {
	ports := make([]int, 100)
	for i := range ports {
		ports[i] = 40000 + i
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			IsPortAvailable(ports[i%len(ports)])
			i++
		}
	})
}
