package config

import (
	"time"
)

// Config represents the main configuration structure
type Config struct {
	PortForwards       map[string]Service `yaml:"portForwards"`
	MonitoringInterval time.Duration      `yaml:"monitoringInterval"`
	UIOptions          UIConfig           `yaml:"uiOptions"`
}

// Service represents a single port-forward service configuration
type Service struct {
	Target      string `yaml:"target"`
	TargetPort  int    `yaml:"targetPort"`
	LocalPort   int    `yaml:"localPort"`
	Namespace   string `yaml:"namespace"`
	Type        string `yaml:"type"`
	SwaggerPath string `yaml:"swaggerPath,omitempty"`
	APIPath     string `yaml:"apiPath,omitempty"`
}

// UIConfig represents UI-specific configuration options
type UIConfig struct {
	RefreshRate time.Duration `yaml:"refreshRate"`
	Theme       string        `yaml:"theme"`
}

// ServiceStatus represents the runtime status of a service
type ServiceStatus struct {
	Name          string
	Status        string
	LocalPort     int // Actual port being used (may differ from config if reassigned)
	PID           int // Process ID of kubectl port-forward
	StartTime     time.Time
	RestartCount  int
	LastError     string
	StatusMessage string // Transient status message (e.g., "Starting gRPC UI...")
	InCooldown    bool
	CooldownUntil time.Time
}
