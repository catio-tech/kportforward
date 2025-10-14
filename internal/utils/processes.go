package utils

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
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

	// On Unix systems, we can send signal 0 to check if process exists
	if runtime.GOOS != "windows" {
		err = process.Signal(syscall.Signal(0))
		return err == nil
	}

	// On Windows, FindProcess doesn't guarantee the process is running
	// We need to use a different approach
	return isProcessRunningWindows(pid)
}

// isProcessRunningWindows checks if a process is running on Windows
func isProcessRunningWindows(pid int) bool {
	cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid), "/FO", "CSV")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	// If the process exists, tasklist will return more than just the header line
	lines := len(string(output))
	return lines > 100 // Simple heuristic - header is much shorter than full output
}

// KillProcess terminates a process with the given PID
func KillProcess(pid int) error {
	if pid <= 0 {
		return fmt.Errorf("invalid PID: %d", pid)
	}

	// First try to kill the process group to ensure all child processes are terminated
	if err := KillProcessGroup(pid); err != nil {
		// If process group kill fails, fall back to individual process kill
		process, err := os.FindProcess(pid)
		if err != nil {
			return fmt.Errorf("failed to find process %d: %w", pid, err)
		}

		if runtime.GOOS == "windows" {
			// On Windows, use taskkill for more reliable termination
			cmd := exec.Command("taskkill", "/F", "/PID", strconv.Itoa(pid))
			return cmd.Run()
		}

		// On Unix systems, send SIGTERM first, then SIGKILL if needed
		if runtime.GOOS != "windows" {
			err = process.Signal(syscall.SIGTERM)
			if err != nil {
				// If SIGTERM fails, try SIGKILL
				return process.Signal(syscall.SIGKILL)
			}
		} else {
			// On Windows, just kill the process
			return process.Kill()
		}
	}

	return nil
}

// KillProcessGroup terminates a process group to ensure all child processes are killed
// Platform-specific implementations are in processes_unix.go and processes_windows.go

// StartKubectlPortForward is implemented in platform-specific files

// GetProcessInfo retrieves information about a running process
func GetProcessInfo(pid int) (*ProcessInfo, error) {
	if !IsProcessRunning(pid) {
		return nil, fmt.Errorf("process %d is not running", pid)
	}

	// This is a simplified implementation
	// In a production system, you might want to parse /proc/{pid}/cmdline on Linux
	// or use WMI queries on Windows for more detailed information
	return &ProcessInfo{
		PID:     pid,
		Command: "kubectl",
		Args:    []string{"port-forward"},
	}, nil
}
