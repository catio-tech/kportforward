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
	Disabled    bool   `yaml:"disabled,omitempty"`
	// Context pins this forward to a specific kubectl context (used by multi-env
	// mode). Empty means "use the current kubectl context" — the default,
	// single-environment behavior.
	Context string `yaml:"context,omitempty"`
}

// MultiEnvConfig is the configuration used in --multi-env mode. It is loaded
// from a separate file (environments.yaml) and is not related to the
// single-environment Config above.
type MultiEnvConfig struct {
	Environments       []Environment `yaml:"environments"`
	MonitoringInterval time.Duration `yaml:"monitoringInterval"`
	UIOptions          UIConfig      `yaml:"uiOptions"`
}

// Environment is one entry in the multi-env config: a named environment pinned
// to a kubectl context, with its own set of services (and their own ports).
type Environment struct {
	Name     string             `yaml:"name"`
	Context  string             `yaml:"context"`
	Services map[string]Service `yaml:"services"`
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
