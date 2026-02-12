package config

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"gopkg.in/yaml.v3"
)

// DefaultRemoteConfigURL is the default URL to fetch the shared default config from
const DefaultRemoteConfigURL = "https://raw.githubusercontent.com/catio-tech/kportforward/refs/heads/main/internal/config/default.yaml"

// RemoteConfigTimeout is the HTTP timeout for fetching remote config
const RemoteConfigTimeout = 5 * time.Second

// remoteConfigURL holds the active remote URL (can be overridden via CLI flag)
var remoteConfigURL = DefaultRemoteConfigURL

// SetRemoteConfigURL sets the remote config URL. Pass "" to disable remote loading.
func SetRemoteConfigURL(url string) {
	remoteConfigURL = url
}

// GetRemoteConfigURL returns the current remote config URL
func GetRemoteConfigURL() string {
	return remoteConfigURL
}

// loadDefaultsWithRemote tries to load default config with the following fallback chain:
//  1. Fetch from remote URL (with timeout)
//  2. Use locally cached copy of last successful remote fetch
//  3. Fall back to embedded default.yaml compiled into the binary
func loadDefaultsWithRemote() ([]byte, error) {
	// If remote URL is disabled, go straight to embedded defaults
	if remoteConfigURL == "" {
		return DefaultConfigYAML, nil
	}

	// Step 1: Try fetching from remote
	data, err := fetchRemoteConfig(remoteConfigURL, RemoteConfigTimeout)
	if err == nil {
		// Cache for offline use (best-effort, don't fail on cache errors)
		_ = cacheRemoteConfig(data)
		return data, nil
	}

	// Step 2: Remote failed — try local cache
	cached, cacheErr := getCachedRemoteConfig()
	if cacheErr == nil {
		return cached, nil
	}

	// Step 3: Both failed — fall back to embedded defaults
	return DefaultConfigYAML, nil
}

// fetchRemoteConfig performs an HTTP GET to retrieve config YAML from the given URL.
func fetchRemoteConfig(url string, timeout time.Duration) ([]byte, error) {
	client := &http.Client{Timeout: timeout}

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch remote config: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("remote config returned HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read remote config body: %w", err)
	}

	// Validate: ensure the fetched data is a parseable config with at least one service
	if err := validateConfigYAML(data); err != nil {
		return nil, fmt.Errorf("remote config validation failed: %w", err)
	}

	return data, nil
}

// validateConfigYAML checks that raw YAML parses into a Config with at least one port forward.
func validateConfigYAML(data []byte) error {
	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}
	if len(cfg.PortForwards) == 0 {
		return fmt.Errorf("config has no port forwards defined")
	}
	return nil
}

// getRemoteCachePath returns the filesystem path for the cached remote config.
// Uses the same base directory as the user config (%APPDATA% on Windows, ~/.config on Unix).
func getRemoteCachePath() (string, error) {
	var configDir string

	switch runtime.GOOS {
	case "windows":
		configDir = os.Getenv("APPDATA")
		if configDir == "" {
			return "", fmt.Errorf("APPDATA environment variable not set")
		}
	default:
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get user home directory: %w", err)
		}
		configDir = filepath.Join(homeDir, ".config")
	}

	return filepath.Join(configDir, "kportforward", "remote-defaults-cache.yaml"), nil
}

// getCachedRemoteConfig reads and validates the locally cached remote config.
func getCachedRemoteConfig() ([]byte, error) {
	cachePath, err := getRemoteCachePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read cached remote config: %w", err)
	}

	if err := validateConfigYAML(data); err != nil {
		return nil, fmt.Errorf("cached remote config is invalid: %w", err)
	}

	return data, nil
}

// cacheRemoteConfig saves remote config data to the local cache file.
func cacheRemoteConfig(data []byte) error {
	cachePath, err := getRemoteCachePath()
	if err != nil {
		return err
	}

	// Ensure the directory exists
	if err := os.MkdirAll(filepath.Dir(cachePath), 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}
