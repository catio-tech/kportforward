package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"gopkg.in/yaml.v3"
)

// LoadConfig loads and merges configuration from embedded defaults and user config
func LoadConfig() (*Config, error) {
	// Load default config: try remote → cached → embedded fallback
	defaultYAML, err := loadDefaultsWithRemote()
	if err != nil {
		return nil, fmt.Errorf("failed to load default config: %w", err)
	}

	config := &Config{}
	if err := yaml.Unmarshal(defaultYAML, config); err != nil {
		return nil, fmt.Errorf("failed to parse default config: %w", err)
	}

	// Try to load user config and merge if it exists
	userConfigPath, err := getUserConfigPath()
	if err != nil {
		return config, nil // Return default config if we can't determine user config path
	}

	if _, err := os.Stat(userConfigPath); os.IsNotExist(err) {
		return config, nil // Return default config if user config doesn't exist
	}

	userConfig, err := loadUserConfig(userConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load user config: %w", err)
	}

	// Merge user config into default config
	mergedConfig := mergeConfigs(config, userConfig)
	return mergedConfig, nil
}

// LoadMultiEnvConfig loads the multi-environment configuration used by --multi-env.
// It starts from the embedded environments.yaml and, if the user has a
// ~/.config/kportforward/environments.yaml, that file fully replaces the
// embedded default (the environment list is explicit, not merged per-service).
func LoadMultiEnvConfig() (*MultiEnvConfig, error) {
	cfg := &MultiEnvConfig{}
	if err := yaml.Unmarshal(DefaultEnvironmentsYAML, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse embedded environments config: %w", err)
	}

	if userPath, err := getUserMultiEnvConfigPath(); err == nil {
		if data, err := os.ReadFile(userPath); err == nil {
			userCfg := &MultiEnvConfig{}
			if err := yaml.Unmarshal(data, userCfg); err != nil {
				return nil, fmt.Errorf("failed to parse user environments config: %w", err)
			}
			cfg = userCfg
		}
	}

	if len(cfg.Environments) == 0 {
		return nil, fmt.Errorf("multi-env config has no environments")
	}
	if err := validateMultiEnvConfig(cfg); err != nil {
		return nil, err
	}
	// Fall back to the single-env defaults for unset UI/monitoring settings.
	if cfg.MonitoringInterval == 0 {
		cfg.MonitoringInterval = time.Second
	}
	if cfg.UIOptions.RefreshRate == 0 {
		cfg.UIOptions.RefreshRate = 100 * time.Millisecond
	}
	if cfg.UIOptions.Theme == "" {
		cfg.UIOptions.Theme = "dark"
	}
	return cfg, nil
}

// validateMultiEnvConfig rejects a multi-env config that cannot work: an
// environment without a context, or a service whose local port is outside the
// valid TCP range (1-65535). Catching this at load gives a clear error instead
// of a misleading "port already in use" when the OS later refuses to bind it.
func validateMultiEnvConfig(mec *MultiEnvConfig) error {
	for _, env := range mec.Environments {
		if env.Context == "" {
			return fmt.Errorf("multi-env: environment %q has no context", env.Name)
		}
		for name, svc := range env.Services {
			if svc.LocalPort < 1 || svc.LocalPort > 65535 {
				return fmt.Errorf("multi-env: environment %q service %q localPort %d is outside the valid range 1-65535",
					env.Name, name, svc.LocalPort)
			}
		}
	}
	return nil
}

// FlattenEnvironments collapses a MultiEnvConfig into a single Config whose
// PortForwards contains every environment's services, each keyed as
// "<service>-<env>" and pinned to that environment's kubectl context. This lets
// the rest of the manager treat multi-env exactly like a large single-env run,
// while every forward still targets the correct cluster.
func FlattenEnvironments(mec *MultiEnvConfig) *Config {
	out := &Config{
		PortForwards:       make(map[string]Service),
		MonitoringInterval: mec.MonitoringInterval,
		UIOptions:          mec.UIOptions,
	}
	for _, env := range mec.Environments {
		for name, service := range env.Services {
			if service.Disabled {
				continue
			}
			service.Context = env.Context
			out.PortForwards[name+"-"+env.Name] = service
		}
	}
	return out
}

// getUserMultiEnvConfigPath returns the user override path for the multi-env config.
func getUserMultiEnvConfigPath() (string, error) {
	base, err := getUserConfigPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(base), "environments.yaml"), nil
}

// getUserConfigPath returns the appropriate config path for the current platform
func getUserConfigPath() (string, error) {
	var configDir string

	switch runtime.GOOS {
	case "windows":
		configDir = os.Getenv("APPDATA")
		if configDir == "" {
			return "", fmt.Errorf("APPDATA environment variable not set")
		}
	default: // Unix-like systems (macOS, Linux)
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get user home directory: %w", err)
		}
		configDir = filepath.Join(homeDir, ".config")
	}

	return filepath.Join(configDir, "kportforward", "config.yaml"), nil
}

// loadUserConfig loads configuration from the user's config file
func loadUserConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	config := &Config{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return config, nil
}

// mergeConfigs merges user configuration into default configuration
// User config takes precedence for individual services and settings
func mergeConfigs(defaultConfig, userConfig *Config) *Config {
	merged := &Config{
		PortForwards:       make(map[string]Service),
		MonitoringInterval: defaultConfig.MonitoringInterval,
		UIOptions:          defaultConfig.UIOptions,
	}

	// Start with default port forwards
	for name, service := range defaultConfig.PortForwards {
		merged.PortForwards[name] = service
	}

	// Override with user port forwards (additive)
	if userConfig.PortForwards != nil {
		for name, service := range userConfig.PortForwards {
			merged.PortForwards[name] = service
		}
	}

	// Override monitoring interval if specified by user
	if userConfig.MonitoringInterval != 0 {
		merged.MonitoringInterval = userConfig.MonitoringInterval
	}

	// Override UI options if specified by user
	if userConfig.UIOptions.RefreshRate != 0 {
		merged.UIOptions.RefreshRate = userConfig.UIOptions.RefreshRate
	}
	if userConfig.UIOptions.Theme != "" {
		merged.UIOptions.Theme = userConfig.UIOptions.Theme
	}

	for name, service := range merged.PortForwards {
		if service.Disabled {
			delete(merged.PortForwards, name)
		}
	}

	return merged
}

// CreateUserConfigDir creates the user config directory if it doesn't exist
func CreateUserConfigDir() error {
	configPath, err := getUserConfigPath()
	if err != nil {
		return err
	}

	configDir := filepath.Dir(configPath)
	return os.MkdirAll(configDir, 0755)
}
