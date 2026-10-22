package utils

import (
	"fmt"
	"maps"
	"sync"
	"time"
)

// PortChecker provides optimized port checking with connection pooling
type PortChecker struct {
	cache    sync.Map
	cacheTTL time.Duration
}

// PortCacheEntry represents a cached port check result
type PortCacheEntry struct {
	available bool
	timestamp time.Time
}

// NewPortChecker creates a new optimized port checker
func NewPortChecker(cacheTTL time.Duration) *PortChecker {
	return &PortChecker{
		cacheTTL: cacheTTL,
	}
}

// IsPortAvailableOptimized checks port availability with caching
func (pc *PortChecker) IsPortAvailableOptimized(port int) bool {
	// Check cache first
	if entry, ok := pc.cache.Load(port); ok {
		cacheEntry := entry.(PortCacheEntry)
		if time.Since(cacheEntry.timestamp) < pc.cacheTTL {
			return cacheEntry.available
		}
		// Cache expired, remove entry
		pc.cache.Delete(port)
	}

	// Check port availability
	available := IsPortAvailable(port)

	// Cache the result
	pc.cache.Store(port, PortCacheEntry{
		available: available,
		timestamp: time.Now(),
	})

	return available
}

// OptimizedPortResolver provides bulk port resolution with reduced allocations
type OptimizedPortResolver struct {
	checker    *PortChecker
	portPool   []int
	resultPool sync.Pool
}

// NewOptimizedPortResolver creates a new optimized port resolver
func NewOptimizedPortResolver() *OptimizedPortResolver {
	return &OptimizedPortResolver{
		checker: NewPortChecker(5 * time.Second), // 5 second cache
		resultPool: sync.Pool{
			New: func() any {
				return make(map[string]int, 50) // Pre-allocate for 50 services
			},
		},
	}
}

// ResolvePortConflictsOptimized resolves port conflicts with optimized memory usage
func (opr *OptimizedPortResolver) ResolvePortConflictsOptimized(services map[string]ServiceConfig) (map[string]int, error) {
	// Get result map from pool
	result := opr.resultPool.Get().(map[string]int)
	defer func() {
		// Clear and return to pool
		for k := range result {
			delete(result, k)
		}
		opr.resultPool.Put(result)
	}()

	usedPorts := make(map[int]bool, len(services))

	// First pass: assign available ports
	for name, service := range services {
		if opr.checker.IsPortAvailableOptimized(service.LocalPort) && !usedPorts[service.LocalPort] {
			result[name] = service.LocalPort
			usedPorts[service.LocalPort] = true
		}
	}

	// Second pass: resolve conflicts with optimized port finding
	for name, service := range services {
		if _, assigned := result[name]; !assigned {
			newPort := opr.findNextAvailablePortOptimized(service.LocalPort, usedPorts)
			if newPort == 0 {
				return nil, fmt.Errorf("failed to find available port for service %s", name)
			}
			result[name] = newPort
			usedPorts[newPort] = true
		}
	}

	// Create final result (caller owns this)
	finalResult := make(map[string]int, len(result))
	maps.Copy(finalResult, result)

	return finalResult, nil
}

// findNextAvailablePortOptimized finds the next available port with reduced allocations
func (opr *OptimizedPortResolver) findNextAvailablePortOptimized(startPort int, usedPorts map[int]bool) int {
	// Check in small increments first
	for port := startPort; port < startPort+100 && port <= 65535; port++ {
		if !usedPorts[port] && opr.checker.IsPortAvailableOptimized(port) {
			return port
		}
	}

	// If no port found in the first 100, do a broader search
	for port := startPort + 100; port <= 65535; port++ {
		if !usedPorts[port] && opr.checker.IsPortAvailableOptimized(port) {
			return port
		}
	}

	return 0
}

// BatchPortCheck checks multiple ports concurrently
func (pc *PortChecker) BatchPortCheck(ports []int) map[int]bool {
	results := make(map[int]bool, len(ports))
	resultsMutex := sync.Mutex{}

	// Use worker pool for concurrent checks
	const maxWorkers = 10
	portChan := make(chan int, len(ports))

	// Start workers
	var wg sync.WaitGroup
	for range maxWorkers {
		wg.Go(func() {
			for port := range portChan {
				available := pc.IsPortAvailableOptimized(port)
				resultsMutex.Lock()
				results[port] = available
				resultsMutex.Unlock()
			}
		})
	}

	// Send ports to workers
	for _, port := range ports {
		portChan <- port
	}
	close(portChan)

	// Wait for completion
	wg.Wait()

	return results
}

// OptimizedPortFinder provides fast port finding with smart algorithms
type OptimizedPortFinder struct {
	lastAssigned int
	mutex        sync.Mutex
}

// NewOptimizedPortFinder creates a new optimized port finder
func NewOptimizedPortFinder() *OptimizedPortFinder {
	return &OptimizedPortFinder{
		lastAssigned: 8080, // Start from a reasonable port
	}
}

// FindAvailablePortFast finds an available port using an optimized algorithm
func (opf *OptimizedPortFinder) FindAvailablePortFast(hint int) (int, error) {
	opf.mutex.Lock()
	defer opf.mutex.Unlock()

	// Try the hint first
	if IsPortAvailable(hint) {
		opf.lastAssigned = hint
		return hint, nil
	}

	// Try sequential search from last assigned + 1
	start := max(opf.lastAssigned+1, hint)

	for port := start; port <= 65535; port++ {
		if IsPortAvailable(port) {
			opf.lastAssigned = port
			return port, nil
		}
	}

	return 0, fmt.Errorf("no available ports found starting from %d", hint)
}
