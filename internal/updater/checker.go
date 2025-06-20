package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/victorkazakov/kportforward/internal/utils"
)

// Checker handles checking for updates
type Checker struct {
	config *UpdateConfig
	logger *utils.Logger
	client *http.Client
}

// NewChecker creates a new update checker
func NewChecker(config *UpdateConfig, logger *utils.Logger) *Checker {
	return &Checker{
		config: config,
		logger: logger,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CheckForUpdates checks if a new version is available
func (c *Checker) CheckForUpdates() (*UpdateInfo, error) {
	c.logger.Info("Checking for updates...")

	// Check if we should skip based on last check time
	if c.shouldSkipCheck() {
		c.logger.Debug("Skipping update check - too recent")
		return &UpdateInfo{Available: false}, nil
	}

	// Get latest release from GitHub
	release, err := c.getLatestRelease()
	if err != nil {
		c.logger.Error("Failed to fetch latest release: %v", err)
		return nil, err
	}

	// Compare versions
	updateInfo := c.compareVersions(release)

	// Update last check time
	if err := c.updateLastCheckTime(); err != nil {
		c.logger.Warn("Failed to update last check time: %v", err)
	}

	if updateInfo.Available {
		c.logger.Info("Update available: %s -> %s", updateInfo.CurrentVersion, updateInfo.LatestVersion)
	} else {
		c.logger.Info("Already up to date: %s", updateInfo.CurrentVersion)
	}

	return updateInfo, nil
}

// getLatestRelease fetches the latest release from GitHub API
func (c *Checker) getLatestRelease() (*Release, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest",
		c.config.RepoOwner, c.config.RepoName)

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch release data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var release Release
	if err := json.Unmarshal(body, &release); err != nil {
		return nil, fmt.Errorf("failed to parse release data: %w", err)
	}

	return &release, nil
}

// compareVersions compares current version with latest release
func (c *Checker) compareVersions(release *Release) *UpdateInfo {
	updateInfo := &UpdateInfo{
		CurrentVersion: c.config.CurrentVersion,
		LatestVersion:  release.TagName,
		ReleaseNotes:   release.Body,
		PublishedAt:    release.PublishedAt,
	}

	// Simple version comparison (assumes semantic versioning)
	if c.isNewerVersion(release.TagName, c.config.CurrentVersion) {
		updateInfo.Available = true

		// Find appropriate asset for current platform
		asset := c.findAssetForPlatform(release.Assets)
		if asset != nil {
			updateInfo.DownloadURL = asset.BrowserDownloadURL
			updateInfo.AssetSize = asset.Size
		}
	}

	return updateInfo
}

// isNewerVersion checks if version A is newer than version B using semantic versioning rules
func (c *Checker) isNewerVersion(versionA, versionB string) bool {
	// Remove 'v' prefix if present
	versionA = strings.TrimPrefix(versionA, "v")
	versionB = strings.TrimPrefix(versionB, "v")

	// Handle special versions
	if versionB == "dev" || versionB == "none" {
		return true
	}

	// Parse versions into segments
	segmentsA := strings.Split(versionA, ".")
	segmentsB := strings.Split(versionB, ".")

	// Compare major.minor.patch segments
	for i := 0; i < 3; i++ {
		// If we ran out of segments, pad with zeros
		var numA, numB int
		if i < len(segmentsA) {
			fmt.Sscanf(segmentsA[i], "%d", &numA)
		}
		if i < len(segmentsB) {
			fmt.Sscanf(segmentsB[i], "%d", &numB)
		}

		if numA > numB {
			return true
		}
		if numA < numB {
			return false
		}
		// If equal, continue to next segment
	}

	// All segments are equal
	return false
}

// findAssetForPlatform finds the appropriate asset for the current platform
func (c *Checker) findAssetForPlatform(assets []Asset) *Asset {
	// Determine platform-specific binary name
	var targetName string
	switch runtime.GOOS {
	case "windows":
		targetName = fmt.Sprintf("kportforward-windows-%s.exe", runtime.GOARCH)
	case "darwin":
		targetName = fmt.Sprintf("kportforward-darwin-%s", runtime.GOARCH)
	case "linux":
		targetName = fmt.Sprintf("kportforward-linux-%s", runtime.GOARCH)
	default:
		c.logger.Warn("Unsupported platform: %s/%s", runtime.GOOS, runtime.GOARCH)
		return nil
	}

	// Find matching asset
	for _, asset := range assets {
		if asset.Name == targetName {
			return &asset
		}
	}

	c.logger.Warn("No asset found for platform %s", targetName)
	return nil
}

// shouldSkipCheck determines if we should skip the update check
func (c *Checker) shouldSkipCheck() bool {
	lastCheckTime, err := c.getLastCheckTime()
	if err != nil {
		// If we can't read the last check time, proceed with check
		return false
	}

	return time.Since(lastCheckTime) < c.config.CheckInterval
}

// getLastCheckTime reads the last check time from file
func (c *Checker) getLastCheckTime() (time.Time, error) {
	if c.config.LastCheckFile == "" {
		return time.Time{}, fmt.Errorf("last check file not configured")
	}

	data, err := os.ReadFile(c.config.LastCheckFile)
	if err != nil {
		return time.Time{}, err
	}

	return time.Parse(time.RFC3339, string(data))
}

// updateLastCheckTime writes the current time to the last check file
func (c *Checker) updateLastCheckTime() error {
	if c.config.LastCheckFile == "" {
		return nil
	}

	// Ensure directory exists
	dir := filepath.Dir(c.config.LastCheckFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write current time
	now := time.Now().Format(time.RFC3339)
	return os.WriteFile(c.config.LastCheckFile, []byte(now), 0644)
}

// ForceCheck forces an update check regardless of last check time
func (c *Checker) ForceCheck() (*UpdateInfo, error) {
	c.logger.Info("Forcing update check...")

	// Temporarily clear last check time to force check
	originalInterval := c.config.CheckInterval
	c.config.CheckInterval = 0

	updateInfo, err := c.CheckForUpdates()

	// Restore original interval
	c.config.CheckInterval = originalInterval

	return updateInfo, err
}
