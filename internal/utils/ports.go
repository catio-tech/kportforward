package utils

import (
	"fmt"
	"net"
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
func CheckPortConnectivity(port int) bool {
	address := fmt.Sprintf("localhost:%d", port)
	conn, err := net.DialTimeout("tcp", address, 1*time.Second)
	if err != nil {
		return false
	}
	defer conn.Close()
	return true
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