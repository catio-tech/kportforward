package utils

import (
	"net"
	"strconv"
	"testing"
)

func TestIsPortAvailable(t *testing.T) {
	// Test with a port that should be available
	port, err := FindAvailablePort(40000)
	if err != nil {
		t.Fatalf("Failed to find available port: %v", err)
	}

	// The port should be available
	if !IsPortAvailable(port) {
		t.Errorf("Port %d should be available", port)
	}

	// Start a listener on 127.0.0.1 (matching kubectl's bind behavior)
	listener, err := net.Listen("tcp4", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil {
		t.Fatalf("Failed to start listener: %v", err)
	}
	defer listener.Close()

	// Now the port should not be available (127.0.0.1 is occupied)
	if IsPortAvailable(port) {
		t.Errorf("Port %d should not be available", port)
	}
}

func TestFindAvailablePort(t *testing.T) {
	// Test finding an available port starting from a high number
	port, err := FindAvailablePort(50000)
	if err != nil {
		t.Fatalf("Failed to find available port: %v", err)
	}

	if port < 50000 {
		t.Errorf("Expected port >= 50000, got %d", port)
	}

	// Verify the port is actually available
	if !IsPortAvailable(port) {
		t.Errorf("Found port %d is not actually available", port)
	}
}

func TestFindAvailablePortWithOccupiedPorts(t *testing.T) {
	// Find an available port
	basePort, err := FindAvailablePort(45000)
	if err != nil {
		t.Fatalf("Failed to find base port: %v", err)
	}

	// Occupy several consecutive ports on 127.0.0.1 (matching kubectl's bind behavior)
	var listeners []net.Listener
	for i := 0; i < 3; i++ {
		listener, err := net.Listen("tcp4", "127.0.0.1:"+strconv.Itoa(basePort+i))
		if err != nil {
			t.Fatalf("Failed to occupy port %d: %v", basePort+i, err)
		}
		listeners = append(listeners, listener)
	}
	defer func() {
		for _, l := range listeners {
			l.Close()
		}
	}()

	// Find available port starting from occupied range
	availablePort, err := FindAvailablePort(basePort)
	if err != nil {
		t.Fatalf("Failed to find available port: %v", err)
	}

	// Should find a port higher than the occupied ones
	if availablePort <= basePort+2 {
		t.Errorf("Expected port > %d, got %d", basePort+2, availablePort)
	}

	// Verify it's actually available
	if !IsPortAvailable(availablePort) {
		t.Errorf("Found port %d is not actually available", availablePort)
	}
}

func TestInvalidPorts(t *testing.T) {
	// Test invalid port numbers - negative ports should fail
	if IsPortAvailable(-1) {
		t.Error("Negative port should not be considered available")
	}

	if IsPortAvailable(65536) {
		t.Error("Port > 65535 should not be considered available")
	}

	// Port 0 is actually valid in Go (means "any available port")
	// so we don't test that as invalid
}

func TestFindAvailablePortEdgeCases(t *testing.T) {
	// Test starting from port close to the limit
	_, err := FindAvailablePort(65530)
	if err != nil {
		t.Errorf("Should be able to find port near the limit: %v", err)
	}

	// Test starting from invalid port
	_, err = FindAvailablePort(70000)
	if err == nil {
		t.Error("Should return error for start port > 65535")
	}
}
