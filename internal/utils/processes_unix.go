//go:build !windows

package utils

import (
	"context"
	"fmt"
	"os/exec"
	"syscall"
	"time"
)

// StartKubectlPortForward starts a kubectl port-forward process with Unix-specific settings
func StartKubectlPortForward(namespace, target string, localPort, targetPort int) (*exec.Cmd, error) {
	return StartKubectlPortForwardWithTimeout(namespace, target, localPort, targetPort, 30*time.Second)
}

// StartKubectlPortForwardWithTimeout starts a kubectl port-forward process with a timeout
func StartKubectlPortForwardWithTimeout(namespace, target string, localPort, targetPort int, timeout time.Duration) (*exec.Cmd, error) {
	args := []string{
		"port-forward",
		"-n", namespace,
		target,
		fmt.Sprintf("%d:%d", localPort, targetPort),
		"--request-timeout=" + fmt.Sprintf("%.0fs", timeout.Seconds()),
	}

	// Create context with timeout for command execution
	ctx, cancel := context.WithTimeout(context.Background(), timeout+5*time.Second)

	cmd := exec.CommandContext(ctx, "kubectl", args...)

	// Set up process group for proper cleanup on Unix systems
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	// Store the cancel function for later use if needed
	// Note: This is a simplified approach - in production you might want to store this differently
	go func() {
		// Cancel context if the command takes too long to start
		time.Sleep(timeout + 10*time.Second)
		cancel()
	}()

	err := cmd.Start()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to start kubectl port-forward: %w", err)
	}

	return cmd, nil
}
