package config

import (
	"time"
)

// Config represents the main configuration structure
type Config struct {
	PortForwards       map[string]Service `yaml:"portForwards"`
	MonitoringInterval time.Duration      `yaml:"monitoringInterval"`
	UIOptions          UIConfig           `yaml:"uiOptions"`
	Collector          CollectorConfig    `yaml:"collector,omitempty"`
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
	Disabled    bool   `yaml:"disabled,omitempty"`
}

// UIConfig represents UI-specific configuration options
type UIConfig struct {
	RefreshRate time.Duration `yaml:"refreshRate"`
	Theme       string        `yaml:"theme"`
}

// ServiceStatus represents the runtime status of a service
type ServiceStatus struct {
	Name          string
	Status        string // Possible values: "Starting", "Connecting", "Running", "Degraded", "Failed", "Suspended", "Reconnecting", "Stopped"
	LocalPort     int    // Actual port being used (may differ from config if reassigned)
	PID           int    // Process ID of kubectl port-forward
	StartTime     time.Time
	RestartCount  int
	LastError     string
	StatusMessage string // Transient status message (e.g., "Starting gRPC UI...")
	InCooldown    bool
	CooldownUntil time.Time
	GlobalStatus  string `json:"globalStatus,omitempty"` // Global access status: "healthy", "auth_failure", "network_failure"
}

// CollectorConfig represents the configuration for the data collector
type CollectorConfig struct {
	Enabled bool `yaml:"enabled"`
	Tenants []string `yaml:"tenants"`
	Services ServiceEndpoints `yaml:"services"`
	Output OutputConfig `yaml:"output"`
	Idempotency IdempotencyConfig `yaml:"idempotency"`
}

// ServiceEndpoints contains the URLs/hosts for services to collect from
type ServiceEndpoints struct {
	Environment ServiceEndpoint `yaml:"environment"`
	ArchitectureInventory ServiceEndpoint `yaml:"architecture_inventory"`
	Recommendations ServiceEndpoint `yaml:"recommendations"`
	Requirements ServiceEndpoint `yaml:"requirements"`
}

// ServiceEndpoint represents a single service endpoint configuration
type ServiceEndpoint struct {
	URL string `yaml:"url,omitempty"` // For REST services
	Host string `yaml:"host,omitempty"` // For gRPC services
}

// OutputConfig configures where and how to emit events
type OutputConfig struct {
	Format      string `yaml:"format"`      // "json" or "text"
	Destination string `yaml:"destination"` // "stdout" or file path (used by `collect` command)
	LogFile     string `yaml:"log_file"`    // file path used when collector runs embedded inside kportforward
}

// IdempotencyConfig configures state tracking for idempotency
type IdempotencyConfig struct {
	StateFile string `yaml:"state_file"`
}
