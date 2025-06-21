package utils

import (
	"fmt"
	"net"
	"sync"
	"time"
)

// IsPortAvailable checks if a port is available for binding
func IsPortAvailable(port int) bool {
	address := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return false
	}
	defer listener.Close()
	return true
}

// FindAvailablePort finds the next available port starting from the given port
func FindAvailablePort(startPort int) (int, error) {
	for port := startPort; port <= 65535; port++ {
		if IsPortAvailable(port) {
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available ports found starting from %d", startPort)
}

// CheckPortConnectivity tests if a service is responding on the given port
// Uses retry logic to be resilient against transient connectivity issues
func CheckPortConnectivity(port int) bool {
	// Use 3 retry attempts with 750ms delay and 2s timeout
	// This gives services more time to respond and handles more transient issues
	return CheckPortConnectivityWithRetries(port, 3, 750*time.Millisecond, 2*time.Second)
}

// CheckPortConnectivityWithRetries tests port connectivity with configurable retries
func CheckPortConnectivityWithRetries(port int, retries int, retryDelay time.Duration, timeout time.Duration) bool {
	address := fmt.Sprintf("localhost:%d", port)

	// Track the number of successful connections (require at least 2 successful connections)
	successCount := 0
	requiredSuccesses := 1 // For normal health checks, one success is enough

	// Try up to the specified number of times
	for attempt := 1; attempt <= retries; attempt++ {
		conn, err := net.DialTimeout("tcp", address, timeout)
		if err == nil {
			// Connection successful
			conn.Close()
			successCount++

			// If we got a successful connection, one is enough for normal health checks
			if successCount >= requiredSuccesses {
				return true
			}
		}

		// Don't sleep after the last attempt
		if attempt < retries {
			time.Sleep(retryDelay)
		}
	}

	return false
}

// ResolvePortConflicts checks for port conflicts in a service map and resolves them
func ResolvePortConflicts(services map[string]ServiceConfig) (map[string]int, error) {
	portAssignments := make(map[string]int)
	usedPorts := make(map[int]bool)

	// First pass: assign ports that are available
	for name, service := range services {
		if IsPortAvailable(service.LocalPort) && !usedPorts[service.LocalPort] {
			portAssignments[name] = service.LocalPort
			usedPorts[service.LocalPort] = true
		}
	}

	// Second pass: resolve conflicts by finding alternative ports
	for name, service := range services {
		if _, assigned := portAssignments[name]; !assigned {
			newPort, err := FindAvailablePort(service.LocalPort)
			if err != nil {
				return nil, fmt.Errorf("failed to find available port for service %s: %w", name, err)
			}
			portAssignments[name] = newPort
			usedPorts[newPort] = true
		}
	}

	return portAssignments, nil
}

// ServiceConfig represents a minimal service configuration for port resolution
type ServiceConfig struct {
	LocalPort int
}

// Global port allocator to prevent race conditions
var (
	allocatedPorts = make(map[int]bool)
	portMutex      sync.Mutex
)

// FindAvailablePortSafe finds the next available port starting from the given port
// in a thread-safe manner to prevent race conditions
func FindAvailablePortSafe(startPort int) (int, error) {
	portMutex.Lock()
	defer portMutex.Unlock()

	for port := startPort; port <= 65535; port++ {
		// Skip if already allocated by us
		if allocatedPorts[port] {
			continue
		}

		// Check if port is actually available
		if IsPortAvailable(port) {
			allocatedPorts[port] = true
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available ports found starting from %d", startPort)
}

// ReleasePort releases a previously allocated port
func ReleasePort(port int) {
	portMutex.Lock()
	defer portMutex.Unlock()
	delete(allocatedPorts, port)
}
