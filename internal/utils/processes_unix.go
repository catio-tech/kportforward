//go:build !windows

package utils

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// StartKubectlPortForward starts a kubectl port-forward process with Unix-specific settings
func StartKubectlPortForward(namespace, target string, localPort, targetPort int, logger *Logger, serviceName string) (*exec.Cmd, error) {
	return StartKubectlPortForwardWithTimeout(namespace, target, localPort, targetPort, 30*time.Second, logger, serviceName)
}

// StartKubectlPortForwardWithTimeout starts a kubectl port-forward process with a timeout
func StartKubectlPortForwardWithTimeout(namespace, target string, localPort, targetPort int, timeout time.Duration, logger *Logger, serviceName string) (*exec.Cmd, error) {
	args := []string{
		"port-forward",
		"-n", namespace,
		target,
		fmt.Sprintf("%d:%d", localPort, targetPort),
		"--request-timeout=" + fmt.Sprintf("%.0fs", timeout.Seconds()),
	}

	cmd := exec.Command("kubectl", args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

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

// KillProcessOnPort finds and kills any process listening on the given TCP port.
// This is used to clean up zombie kubectl processes that survived a previous shutdown.
func KillProcessOnPort(port int) error {
	// Try lsof first (available on macOS and most Linux distros)
	cmd := exec.Command("lsof", "-ti", fmt.Sprintf("tcp:%d", port))
	output, err := cmd.Output()
	if err != nil || len(output) == 0 {
		return nil // No process found — nothing to kill
	}

	killed := false
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		pid, err := strconv.Atoi(strings.TrimSpace(line))
		if err != nil || pid <= 0 {
			continue
		}
		_ = syscall.Kill(pid, syscall.SIGKILL)
		killed = true
	}

	if killed {
		time.Sleep(500 * time.Millisecond)
	}
	return nil
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
