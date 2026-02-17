//go:build !windows

package utils

import (
	"fmt"
	"os"
	"syscall"
)

// ProcessInfo represents information about a running process
type ProcessInfo struct {
	PID     int
	Command string
	Args    []string
}

// IsProcessRunning checks if a process with the given PID is still running
func IsProcessRunning(pid int) bool {
	if pid <= 0 {
		return false
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// On Unix, send signal 0 to check if the process exists
	return process.Signal(syscall.Signal(0)) == nil
}

// KillProcess terminates a process with the given PID
func KillProcess(pid int) error {
	if pid <= 0 {
		return fmt.Errorf("invalid PID: %d", pid)
	}

	// Try to kill the process group first to terminate all children
	if err := KillProcessGroup(pid); err != nil {
		// Fall back to individual process kill
		process, err := os.FindProcess(pid)
		if err != nil {
			return fmt.Errorf("failed to find process %d: %w", pid, err)
		}

		// Send SIGTERM first, then SIGKILL if needed
		if err := process.Signal(syscall.SIGTERM); err != nil {
			return process.Signal(syscall.SIGKILL)
		}
	}

	return nil
}

// KillProcessGroup terminates a process group to ensure all child processes are killed
func KillProcessGroup(pid int) error {
	if pid <= 0 {
		return fmt.Errorf("invalid PID: %d", pid)
	}

	// Negative PID means kill the entire process group
	if err := syscall.Kill(-pid, syscall.SIGTERM); err != nil {
		return syscall.Kill(-pid, syscall.SIGKILL)
	}

	return nil
}

// StartKubectlPortForward is implemented in platform-specific files

// GetProcessInfo retrieves information about a running process
func GetProcessInfo(pid int) (*ProcessInfo, error) {
	if !IsProcessRunning(pid) {
		return nil, fmt.Errorf("process %d is not running", pid)
	}

	return &ProcessInfo{
		PID:     pid,
		Command: "kubectl",
		Args:    []string{"port-forward"},
	}, nil
}
