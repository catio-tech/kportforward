//go:build windows

package utils

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// ProcessInfo represents information about a running process
type ProcessInfo struct {
	PID     int
	Command string
	Args    []string
}

// StartKubectlPortForward starts a kubectl port-forward process with Windows-specific settings
func StartKubectlPortForward(namespace, target string, localPort, targetPort int, logger *Logger, serviceName string) (*exec.Cmd, error) {
	return StartKubectlPortForwardWithTimeout(namespace, target, localPort, targetPort, 30*time.Second, logger, serviceName)
}

// StartKubectlPortForwardWithTimeout starts a kubectl port-forward process with a timeout on Windows
func StartKubectlPortForwardWithTimeout(namespace, target string, localPort, targetPort int, timeout time.Duration, logger *Logger, serviceName string) (*exec.Cmd, error) {
	args := []string{
		"port-forward",
		"-n", namespace,
		target,
		fmt.Sprintf("%d:%d", localPort, targetPort),
		"--request-timeout=" + fmt.Sprintf("%.0fs", timeout.Seconds()),
	}

	cmd := exec.Command("kubectl", args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get kubectl stdout: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get kubectl stderr: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start kubectl port-forward: %w", err)
	}

	go streamKubectlOutput(stdout, logger, serviceName, false)
	go streamKubectlOutput(stderr, logger, serviceName, true)

	go func() {
		err := cmd.Wait()
		if err != nil && logger != nil {
			logger.Debug("kubectl port-forward exited for %s: %v", serviceName, err)
		}
	}()

	return cmd, nil
}

// IsProcessRunning checks if a process is running on Windows
func IsProcessRunning(pid int) bool {
	if pid <= 0 {
		return false
	}

	cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid), "/FO", "CSV")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	// If the process exists, tasklist output is longer than just the header
	return len(string(output)) > 100
}

func isTaskkillProcessMissing(err error) bool {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode() == 128
	}
	return false
}

func runTaskkill(args ...string) error {
	cmd := exec.Command("taskkill", args...)
	if err := cmd.Run(); err != nil {
		if isTaskkillProcessMissing(err) {
			return nil
		}
		return err
	}
	return nil
}

// KillProcess terminates a process with the given PID
func KillProcess(pid int) error {
	if pid <= 0 {
		return fmt.Errorf("invalid PID: %d", pid)
	}

	// Try to kill the process tree first
	if err := KillProcessGroup(pid); err == nil {
		return nil
	}

	// Fallback: kill just the PID
	return runTaskkill("/F", "/PID", strconv.Itoa(pid))
}

// KillProcessGroup terminates a process tree
func KillProcessGroup(pid int) error {
	if pid <= 0 {
		return fmt.Errorf("invalid PID: %d", pid)
	}

	return runTaskkill("/F", "/T", "/PID", strconv.Itoa(pid))
}

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

// KillProcessOnPort finds and kills any process listening on the given TCP port.
// This is used to clean up zombie kubectl processes that survived a previous shutdown.
func KillProcessOnPort(port int) error {
	// Use netstat to find the PID holding the port
	cmd := exec.Command("cmd", "/C", fmt.Sprintf("netstat -ano | findstr \"LISTENING\" | findstr \":%d \"", port))
	output, err := cmd.Output()
	if err != nil || len(output) == 0 {
		return nil // No process found on this port — nothing to kill
	}

	// Parse PIDs from netstat output (may have multiple lines for IPv4/IPv6)
	killed := make(map[int]bool)
	lines := splitLines(string(output))
	for _, line := range lines {
		fields := splitFields(line)
		if len(fields) < 5 {
			continue
		}
		pid, err := strconv.Atoi(fields[len(fields)-1])
		if err != nil || pid <= 0 || killed[pid] {
			continue
		}
		killed[pid] = true
		_ = runTaskkill("/F", "/PID", strconv.Itoa(pid))
	}

	if len(killed) > 0 {
		// Brief wait for OS to release the port
		time.Sleep(500 * time.Millisecond)
	}
	return nil
}

// splitLines splits a string into non-empty lines
func splitLines(s string) []string {
	var lines []string
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

// splitFields splits a line into whitespace-separated fields
func splitFields(s string) []string {
	var fields []string
	for _, f := range strings.Split(s, " ") {
		f = strings.TrimSpace(f)
		if f != "" {
			fields = append(fields, f)
		}
	}
	return fields
}

func streamKubectlOutput(r io.Reader, logger *Logger, serviceName string, isErr bool) {
	if logger == nil {
		return
	}
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if isErr {
			logger.Warn("kubectl[%s] %s", serviceName, line)
		} else {
			logger.Debug("kubectl[%s] %s", serviceName, line)
		}
	}
	if err := scanner.Err(); err != nil {
		logger.Debug("kubectl[%s] output read error: %v", serviceName, err)
	}
}
