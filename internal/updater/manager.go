package updater

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/victorkazakov/kportforward/internal/utils"
)

// Manager coordinates update checking and application
type Manager struct {
	checker *Checker
	config  *UpdateConfig
	logger  *utils.Logger
	ctx     context.Context
	cancel  context.CancelFunc

	// Channels for communication
	updateChan  chan *UpdateInfo
	checkTicker *time.Ticker

	// State
	lastUpdateInfo *UpdateInfo
}

// NewManager creates a new update manager
func NewManager(repoOwner, repoName, currentVersion string, logger *utils.Logger) *Manager {
	ctx, cancel := context.WithCancel(context.Background())

	// Get user cache directory for storing last check time
	cacheDir, err := getUserCacheDir()
	if err != nil {
		logger.Warn("Failed to get cache directory: %v", err)
		cacheDir = "."
	}

	config := &UpdateConfig{
		RepoOwner:      repoOwner,
		RepoName:       repoName,
		CurrentVersion: currentVersion,
		CheckInterval:  24 * time.Hour, // Daily checks
		LastCheckFile:  filepath.Join(cacheDir, "kportforward", "last_update_check"),
		UpdateChannel:  "stable",
	}

	checker := NewChecker(config, logger)

	return &Manager{
		checker:    checker,
		config:     config,
		logger:     logger,
		ctx:        ctx,
		cancel:     cancel,
		updateChan: make(chan *UpdateInfo, 1),
	}
}

// Start begins the update checking process
func (m *Manager) Start() error {
	m.logger.Info("Starting update manager")

	// Start periodic checking (which will do an initial check after a short delay)
	m.checkTicker = time.NewTicker(m.config.CheckInterval)
	go m.periodicCheck()

	// Check for updates after a short delay to not block startup
	go func() {
		time.Sleep(2 * time.Second) // Wait 2 seconds before first update check
		updateInfo, err := m.checker.CheckForUpdates()
		if err != nil {
			m.logger.Error("Initial update check failed: %v", err)
			return
		}

		m.lastUpdateInfo = updateInfo
		if updateInfo.Available {
			select {
			case m.updateChan <- updateInfo:
			case <-m.ctx.Done():
			}
		}
	}()

	return nil
}

// Stop gracefully stops the update manager
func (m *Manager) Stop() error {
	m.logger.Info("Stopping update manager")

	m.cancel()
	if m.checkTicker != nil {
		m.checkTicker.Stop()
	}
	close(m.updateChan)

	return nil
}

// GetUpdateChannel returns the channel for update notifications
func (m *Manager) GetUpdateChannel() <-chan *UpdateInfo {
	return m.updateChan
}

// ForceCheck manually triggers an update check
func (m *Manager) ForceCheck() (*UpdateInfo, error) {
	m.logger.Info("Manual update check requested")

	updateInfo, err := m.checker.ForceCheck()
	if err != nil {
		return nil, err
	}

	m.lastUpdateInfo = updateInfo
	return updateInfo, nil
}

// GetLastUpdateInfo returns the last known update information
func (m *Manager) GetLastUpdateInfo() *UpdateInfo {
	return m.lastUpdateInfo
}

// IsUpdateAvailable returns true if an update is available
func (m *Manager) IsUpdateAvailable() bool {
	return m.lastUpdateInfo != nil && m.lastUpdateInfo.Available
}

// periodicCheck runs the periodic update checking loop
func (m *Manager) periodicCheck() {
	defer m.checkTicker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return

		case <-m.checkTicker.C:
			updateInfo, err := m.checker.CheckForUpdates()
			if err != nil {
				m.logger.Error("Periodic update check failed: %v", err)
				continue
			}

			// Only notify if this is a new update we haven't seen before
			if updateInfo.Available && (m.lastUpdateInfo == nil ||
				m.lastUpdateInfo.LatestVersion != updateInfo.LatestVersion) {

				m.lastUpdateInfo = updateInfo
				select {
				case m.updateChan <- updateInfo:
				case <-m.ctx.Done():
					return
				}
			}
		}
	}
}

// PrepareUpdate downloads and prepares the update (but doesn't apply it)
func (m *Manager) PrepareUpdate(updateInfo *UpdateInfo) error {
	if updateInfo.DownloadURL == "" {
		return fmt.Errorf("no download URL available")
	}

	m.logger.Info("Preparing update %s", updateInfo.LatestVersion)

	// TODO: Implement download and verification
	// For now, just log that we would download
	m.logger.Info("Would download from: %s", updateInfo.DownloadURL)
	m.logger.Info("Asset size: %d bytes", updateInfo.AssetSize)

	return nil
}

// getUserCacheDir returns the appropriate cache directory for the current platform
func getUserCacheDir() (string, error) {
	switch runtime.GOOS {
	case "windows":
		cacheDir := os.Getenv("LOCALAPPDATA")
		if cacheDir == "" {
			cacheDir = os.Getenv("TEMP")
		}
		if cacheDir == "" {
			return "", fmt.Errorf("could not determine cache directory")
		}
		return cacheDir, nil

	case "darwin":
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(homeDir, "Library", "Caches"), nil

	default: // Linux and other Unix-like systems
		cacheDir := os.Getenv("XDG_CACHE_HOME")
		if cacheDir != "" {
			return cacheDir, nil
		}

		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(homeDir, ".cache"), nil
	}
}
