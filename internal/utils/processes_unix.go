//go:build !windows

package utils

import (
	//"context"
	"bufio"
	"fmt"
	"io"
	"os/exec"
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
