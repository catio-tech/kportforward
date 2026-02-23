package collector

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// StateManager handles idempotency state tracking for the collector
type StateManager struct {
	stateFile string
	state     *CollectorState
}

// NewStateManager creates a new state manager
func NewStateManager(stateFile string) *StateManager {
	// Expand ~ in file path
	if strings.HasPrefix(stateFile, "~/") {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			stateFile = filepath.Join(homeDir, stateFile[2:])
		}
	}

	return &StateManager{
		stateFile: stateFile,
		state: &CollectorState{
			TenantCheckpoints: make(map[string]map[string]time.Time),
		},
	}
}

// Load loads the state from disk
func (sm *StateManager) Load() error {
	data, err := os.ReadFile(sm.stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			// State file doesn't exist yet, that's okay
			return nil
		}
		return fmt.Errorf("failed to read state file: %w", err)
	}

	if err := json.Unmarshal(data, sm.state); err != nil {
		return fmt.Errorf("failed to parse state file: %w", err)
	}

	return nil
}

// Save atomically saves the state to disk
func (sm *StateManager) Save() error {
	// Ensure directory exists
	dir := filepath.Dir(sm.stateFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	// Marshal state to JSON
	data, err := json.MarshalIndent(sm.state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Write to temp file first for atomicity
	tempFile := sm.stateFile + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp state file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempFile, sm.stateFile); err != nil {
		os.Remove(tempFile) // Clean up temp file
		return fmt.Errorf("failed to rename state file: %w", err)
	}

	return nil
}

// ShouldCollect checks if the given time bucket should be collected
func (sm *StateManager) ShouldCollect(bucketStart, bucketEnd time.Time) bool {
	// If we've already collected this exact bucket, skip it
	if sm.state.LastCollection.TimeBucketStart.Equal(bucketStart) &&
		sm.state.LastCollection.TimeBucketEnd.Equal(bucketEnd) &&
		!sm.state.LastCollection.CompletedAt.IsZero() {
		return false
	}
	return true
}

// MarkCollectionComplete updates the state to mark a collection as complete
func (sm *StateManager) MarkCollectionComplete(bucketStart, bucketEnd time.Time) {
	sm.state.LastCollection = LastCollectionInfo{
		TimeBucketStart: bucketStart,
		TimeBucketEnd:   bucketEnd,
		CompletedAt:     time.Now(),
	}
}

// MarkWorkspaceCollected marks a specific workspace as collected
func (sm *StateManager) MarkWorkspaceCollected(tenantID, workspaceID string) {
	if sm.state.TenantCheckpoints == nil {
		sm.state.TenantCheckpoints = make(map[string]map[string]time.Time)
	}
	if sm.state.TenantCheckpoints[tenantID] == nil {
		sm.state.TenantCheckpoints[tenantID] = make(map[string]time.Time)
	}
	sm.state.TenantCheckpoints[tenantID][workspaceID] = time.Now()
}

// GetState returns the current state (for inspection)
func (sm *StateManager) GetState() *CollectorState {
	return sm.state
}
