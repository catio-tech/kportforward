package ui_handlers

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/victorkazakov/kportforward/internal/config"
	"github.com/victorkazakov/kportforward/internal/utils"
)

// SwaggerUIManager manages Swagger UI containers for REST services
type SwaggerUIManager struct {
	services map[string]*SwaggerUIService
	logger   *utils.Logger
	mutex    sync.RWMutex
	enabled  bool
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

	// Only start for REST services that are running
	if serviceConfig.Type != "rest" || serviceStatus.Status != "Running" {
		return nil
	}

	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	// Check if already running
	if service, exists := sm.services[serviceName]; exists && service.status == "Running" {
		return nil
	}

	// Find available port for Swagger UI (thread-safe)
	swaggerPort, err := utils.FindAvailablePortSafe(8080)
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

	// Start Docker container
	containerID, containerName, err := sm.startSwaggerContainer(serviceName, serviceStatus.LocalPort, swaggerPort, swaggerPath, apiPath)
	if err != nil {
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

	sm.logger.Info("Started Swagger UI for %s on port %d", serviceName, swaggerPort)
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
	if service == nil || service.status != "Running" {
		return ""
	}

	return fmt.Sprintf("http://localhost:%d", service.swaggerPort)
}

// IsEnabled returns whether Swagger UI management is enabled
func (sm *SwaggerUIManager) IsEnabled() bool {
	return sm.enabled
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

	// Stop any existing container with the same name
	sm.stopContainerByName(containerName)

	// Docker run arguments
	args := []string{
		"run",
		"-d",   // Detached mode
		"--rm", // Remove container when it stops
		"--name", containerName,
		"-p", fmt.Sprintf("%d:8080", swaggerPort),
		"-e", fmt.Sprintf("SWAGGER_JSON=http://host.docker.internal:%d/%s", targetPort, swaggerPath),
		"swaggerapi/swagger-ui",
	}

	// Add host networking for Docker Desktop
	if sm.isDockerDesktop() {
		// Docker Desktop automatically provides host.docker.internal
	} else {
		// For Linux Docker, add host networking
		args = append([]string{"run", "-d", "--rm", "--name", containerName, "--network=host"}, args[4:]...)
		// Update the environment variable for Linux
		for i, arg := range args {
			if strings.HasPrefix(arg, "SWAGGER_JSON=") {
				args[i] = fmt.Sprintf("SWAGGER_JSON=http://localhost:%d/%s", targetPort, swaggerPath)
				break
			}
		}
	}

	cmd := exec.Command("docker", args...)
	output, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("failed to start Docker container: %w", err)
	}

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

// isContainerRunning checks if a Docker container is running
func (sm *SwaggerUIManager) isContainerRunning(containerID string) bool {
	cmd := exec.Command("docker", "ps", "-q", "--filter", fmt.Sprintf("id=%s", containerID))
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	return strings.TrimSpace(string(output)) != ""
}

// isDockerDesktop checks if we're running Docker Desktop (vs Docker on Linux)
func (sm *SwaggerUIManager) isDockerDesktop() bool {
	cmd := exec.Command("docker", "version", "--format", "{{.Server.Os}}")
	_, err := cmd.Output()
	if err != nil {
		return true // Assume Docker Desktop if we can't determine
	}

	// Docker Desktop reports as "linux" but has different networking
	// We'll use a heuristic: check if host.docker.internal resolves
	checkCmd := exec.Command("ping", "-c", "1", "host.docker.internal")
	return checkCmd.Run() == nil
}

// MonitorServices monitors all Swagger UI services and restarts failed ones
func (sm *SwaggerUIManager) MonitorServices(services map[string]config.ServiceStatus, configs map[string]config.Service) {
	if !sm.enabled {
		return
	}

	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	// Start Swagger UI for new REST services
	for serviceName, serviceStatus := range services {
		if serviceConfig, exists := configs[serviceName]; exists {
			if serviceConfig.Type == "rest" && serviceStatus.Status == "Running" {
				if _, uiExists := sm.services[serviceName]; !uiExists {
					go func(name string, status config.ServiceStatus, config config.Service) {
						if err := sm.StartService(name, status, config); err != nil {
							sm.logger.Error("Failed to start Swagger UI for %s: %v", name, err)
						}
					}(serviceName, serviceStatus, serviceConfig)
				}
			}
		}
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
