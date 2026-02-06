package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v3"
)

// LoadConfig loads and merges configuration from embedded defaults and user config
func LoadConfig() (*Config, error) {
	// Start with embedded default config
	config := &Config{}
	if err := yaml.Unmarshal(DefaultConfigYAML, config); err != nil {
		return nil, fmt.Errorf("failed to parse embedded config: %w", err)
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
