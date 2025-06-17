package ui_handlers

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/victorkazakov/kportforward/internal/common"
	"github.com/victorkazakov/kportforward/internal/config"
	"github.com/victorkazakov/kportforward/internal/utils"
)

// SwaggerUIManager manages Swagger UI containers for REST services
type SwaggerUIManager struct {
	services       map[string]*SwaggerUIService
	logger         *utils.Logger
	mutex          sync.RWMutex
	enabled        bool
	statusCallback common.StatusCallback
}

// SwaggerUIService represents a single Swagger UI instance
type SwaggerUIService struct {
	serviceName   string
	localPort     int
	swaggerPort   int
	containerID   string
	containerName string
	startTime     time.Time
	restartCount  int
	status        string
	swaggerPath   string
	apiPath       string
}

// NewSwaggerUIManager creates a new Swagger UI manager
func NewSwaggerUIManager(logger *utils.Logger) *SwaggerUIManager {
	return &SwaggerUIManager{
		services: make(map[string]*SwaggerUIService),
		logger:   logger,
		enabled:  false,
	}
}

// Enable enables Swagger UI management
func (sm *SwaggerUIManager) Enable() error {
	// Check if Docker is available
	if !sm.isDockerAvailable() {
		return fmt.Errorf("docker not found or not running. Please install and start Docker Desktop")
	}

	sm.enabled = true
	sm.logger.Info("Swagger UI manager enabled")
	return nil
}

// Disable disables Swagger UI management and stops all containers
func (sm *SwaggerUIManager) Disable() error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	for serviceName := range sm.services {
		if err := sm.stopService(serviceName); err != nil {
			sm.logger.Error("Failed to stop Swagger UI for %s: %v", serviceName, err)
		}
	}

	sm.enabled = false
	sm.logger.Info("Swagger UI manager disabled")
	return nil
}

// StartService starts a Swagger UI container for the given service
func (sm *SwaggerUIManager) StartService(serviceName string, serviceStatus config.ServiceStatus, serviceConfig config.Service) error {
	if !sm.enabled {
		return nil
	}

	// Only start for REST services that are running and have a swaggerPath configured
	if serviceConfig.Type != "rest" || serviceStatus.Status != "Running" {
		return nil
	}

	// Skip if no swaggerPath is configured
	if serviceConfig.SwaggerPath == "" {
		sm.logger.Debug("Skipping Swagger UI for %s: no swaggerPath configured", serviceName)
		return nil
	}

	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	// Check if already running
	if service, exists := sm.services[serviceName]; exists && service.status == "Running" {
		return nil
	}

	// Find available port for Swagger UI (thread-safe)
	swaggerPort, err := utils.FindAvailablePortSafe(9100)
	if err != nil {
		return fmt.Errorf("failed to find available port for Swagger UI: %w", err)
	}

	// Get swagger configuration
	swaggerPath := serviceConfig.SwaggerPath
	if swaggerPath == "" {
		swaggerPath = "configuration/swagger" // Default path
	}

	apiPath := serviceConfig.APIPath
	if apiPath == "" {
		apiPath = "api" // Default API path
	}

	// Send status update that we're starting Swagger UI
	if sm.statusCallback != nil {
		sm.statusCallback.UpdateServiceStatusMessage(serviceName, "Starting Swagger UI...")
	}

	// Wait for port-forward to establish and verify it's working
	sm.logger.Info("Starting Swagger UI for REST service %s (port %d)", serviceName, serviceStatus.LocalPort)

	// Give port-forward time to establish
	time.Sleep(1 * time.Second)
	sm.logger.Info("Checking if port-forward is established for %s on port %d", serviceName, serviceStatus.LocalPort)

	// Verify the port-forward is actually working before starting Swagger UI
	if !sm.isPortReachable(serviceStatus.LocalPort) {
		sm.logger.Info("Port-forward not ready for %s, Swagger UI not started", serviceName)
		if sm.statusCallback != nil {
			sm.statusCallback.UpdateServiceStatusMessage(serviceName, "")
		}
		return fmt.Errorf("port-forward not ready on port %d", serviceStatus.LocalPort)
	}

	// Start Docker container
	sm.logger.Info("Starting Swagger UI for %s: connecting to localhost:%d, serving on port %d", serviceName, serviceStatus.LocalPort, swaggerPort)
	containerID, containerName, err := sm.startSwaggerContainer(serviceName, serviceStatus.LocalPort, swaggerPort, swaggerPath, apiPath)
	if err != nil {
		sm.logger.Error("Failed to start Swagger UI container for %s: %v", serviceName, err)
		return fmt.Errorf("failed to start Swagger UI container: %w", err)
	}

	// Create service entry
	sm.services[serviceName] = &SwaggerUIService{
		serviceName:   serviceName,
		localPort:     serviceStatus.LocalPort,
		swaggerPort:   swaggerPort,
		containerID:   containerID,
		containerName: containerName,
		startTime:     time.Now(),
		restartCount:  0,
		status:        "Running",
		swaggerPath:   swaggerPath,
		apiPath:       apiPath,
	}

	sm.logger.Info("Started Swagger UI for %s on port %d (Container: %s)", serviceName, swaggerPort, containerID)

	// Give the container a moment to start up
	time.Sleep(500 * time.Millisecond)

	// Check if container is still running after startup
	if !sm.isContainerRunning(containerID) {
		sm.logger.Error("Swagger UI container for %s died immediately after startup", serviceName)
		sm.services[serviceName].status = "Failed"
		if sm.statusCallback != nil {
			sm.statusCallback.UpdateServiceStatusMessage(serviceName, "Swagger UI failed to start")
		}
	} else {
		// Clear status message when successfully started
		if sm.statusCallback != nil {
			sm.statusCallback.UpdateServiceStatusMessage(serviceName, "")
		}
	}

	return nil
}

// StopService stops the Swagger UI container for the given service
func (sm *SwaggerUIManager) StopService(serviceName string) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	return sm.stopService(serviceName)
}

// stopService stops a service (internal method, assumes lock is held)
func (sm *SwaggerUIManager) stopService(serviceName string) error {
	service, exists := sm.services[serviceName]
	if !exists {
		return nil
	}

	// Stop and remove Docker container
	if service.containerID != "" {
		if err := sm.stopContainer(service.containerID); err != nil {
			sm.logger.Warn("Failed to stop Swagger UI container for %s: %v", serviceName, err)
		}
	}

	// Release the allocated port
	utils.ReleasePort(service.swaggerPort)

	service.status = "Stopped"
	delete(sm.services, serviceName)

	sm.logger.Info("Stopped Swagger UI for %s", serviceName)
	return nil
}

// GetServiceInfo returns information about a Swagger UI service
func (sm *SwaggerUIManager) GetServiceInfo(serviceName string) *SwaggerUIService {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	service, exists := sm.services[serviceName]
	if !exists {
		return nil
	}

	// Check if container is still running
	if service.containerID != "" {
		if !sm.isContainerRunning(service.containerID) {
			service.status = "Failed"
		}
	}

	return service
}

// GetServiceURL returns the URL for accessing the Swagger UI
func (sm *SwaggerUIManager) GetServiceURL(serviceName string) string {
	service := sm.GetServiceInfo(serviceName)
	if service == nil {
		sm.logger.Debug("GetServiceURL: No Swagger UI service found for %s", serviceName)
		return ""
	}
	if service.status != "Running" {
		sm.logger.Debug("GetServiceURL: Swagger UI service for %s is not running (status: %s)", serviceName, service.status)
		return ""
	}

	url := fmt.Sprintf("http://localhost:%d", service.swaggerPort)
	sm.logger.Debug("GetServiceURL: returning %s for service %s", url, serviceName)
	return url
}

// IsEnabled returns whether Swagger UI management is enabled
func (sm *SwaggerUIManager) IsEnabled() bool {
	return sm.enabled
}

// SetStatusCallback sets the callback for sending status updates
func (sm *SwaggerUIManager) SetStatusCallback(callback common.StatusCallback) {
	sm.statusCallback = callback
}

// isDockerAvailable checks if Docker is available and running
func (sm *SwaggerUIManager) isDockerAvailable() bool {
	cmd := exec.Command("docker", "version")
	err := cmd.Run()
	return err == nil
}

// startSwaggerContainer starts a Docker container with Swagger UI
func (sm *SwaggerUIManager) startSwaggerContainer(serviceName string, targetPort, swaggerPort int, swaggerPath, apiPath string) (string, string, error) {
	containerName := fmt.Sprintf("kpf-swagger-%s", strings.ReplaceAll(serviceName, "_", "-"))

	// Kill any existing container using the same port (like the working bash code)
	sm.stopContainerByPort(swaggerPort)

	// Stop any existing container with the same name
	sm.stopContainerByName(containerName)

	// Build URL like the working bash code: http://localhost:${lport}/${api_path}/${swagger_path}
	swaggerURL := fmt.Sprintf("http://localhost:%d/%s/%s", targetPort, apiPath, swaggerPath)

	// Docker run arguments (simplified to match working bash code)
	args := []string{
		"run",
		"--rm",
		"-d",
		"-p", fmt.Sprintf("%d:8080", swaggerPort),
		"-e", fmt.Sprintf("URL=%s", swaggerURL),
		"swaggerapi/swagger-ui",
	}

	sm.logger.Info("Starting Docker container with command: docker %s", strings.Join(args, " "))
	cmd := exec.Command("docker", args...)
	output, err := cmd.Output()
	if err != nil {
		sm.logger.Error("Docker container startup failed: %v", err)
		return "", "", fmt.Errorf("failed to start Docker container: %w", err)
	}
	sm.logger.Info("Docker container started successfully")

	containerID := strings.TrimSpace(string(output))
	return containerID, containerName, nil
}

// stopContainer stops a Docker container by ID
func (sm *SwaggerUIManager) stopContainer(containerID string) error {
	cmd := exec.Command("docker", "stop", containerID)
	return cmd.Run()
}

// stopContainerByName stops a Docker container by name
func (sm *SwaggerUIManager) stopContainerByName(containerName string) error {
	cmd := exec.Command("docker", "stop", containerName)
	_ = cmd.Run()
	// Ignore errors - container might not exist
	return nil
}

// stopContainerByPort stops any Docker container using the specified port
func (sm *SwaggerUIManager) stopContainerByPort(port int) error {
	cmd := exec.Command("docker", "ps", "--filter", fmt.Sprintf("publish=%d", port), "-q")
	output, err := cmd.Output()
	if err != nil {
		return nil // Ignore errors
	}

	containerID := strings.TrimSpace(string(output))
	if containerID != "" {
		stopCmd := exec.Command("docker", "rm", "-f", containerID)
		_ = stopCmd.Run()
	}
	return nil
}

// isPortReachable checks if a port is reachable (like nc -z in bash)
func (sm *SwaggerUIManager) isPortReachable(port int) bool {
	cmd := exec.Command("nc", "-z", "localhost", fmt.Sprintf("%d", port))
	err := cmd.Run()
	return err == nil
}

// isContainerRunning checks if a Docker container is running
func (sm *SwaggerUIManager) isContainerRunning(containerID string) bool {
	cmd := exec.Command("docker", "ps", "-q", "--filter", fmt.Sprintf("id=%s", containerID))
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	return strings.TrimSpace(string(output)) != ""
}

// MonitorServices monitors all Swagger UI services and restarts failed ones
func (sm *SwaggerUIManager) MonitorServices(services map[string]config.ServiceStatus, configs map[string]config.Service) {
	if !sm.enabled {
		return
	}

	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	// Start Swagger UI for new REST services
	restServicesFound := 0
	runningRestServices := 0
	for serviceName, serviceStatus := range services {
		if serviceConfig, exists := configs[serviceName]; exists {
			if serviceConfig.Type == "rest" {
				restServicesFound++
				sm.logger.Info("Found REST service %s with status: %s", serviceName, serviceStatus.Status)
				if serviceStatus.Status == "Running" {
					runningRestServices++
					if _, uiExists := sm.services[serviceName]; !uiExists {
						sm.logger.Info("Starting Swagger UI for REST service: %s", serviceName)
						go func(name string, status config.ServiceStatus, config config.Service) {
							if err := sm.StartService(name, status, config); err != nil {
								sm.logger.Error("Failed to start Swagger UI for %s: %v", name, err)
							}
						}(serviceName, serviceStatus, serviceConfig)
					}
				}
			}
		}
	}
	if restServicesFound > 0 {
		sm.logger.Info("MonitorServices: Found %d REST services, %d running", restServicesFound, runningRestServices)
	}

	// Stop Swagger UI for services that are no longer running
	for serviceName := range sm.services {
		serviceStatus, exists := services[serviceName]
		if !exists || serviceStatus.Status != "Running" {
			go func(name string) {
				if err := sm.StopService(name); err != nil {
					sm.logger.Error("Failed to stop Swagger UI for %s: %v", name, err)
				}
			}(serviceName)
		}
	}
}
