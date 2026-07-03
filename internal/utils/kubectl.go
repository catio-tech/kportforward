package utils

import (
	"fmt"
	"time"
)

// BuildKubectlPortForwardArgs builds the argument list for `kubectl port-forward`.
// When kubeContext is non-empty the forward is pinned to that context via
// `--context`; otherwise the current kubectl context is used. Shared across
// platforms so the (safety-critical) context pinning has a single definition.
func BuildKubectlPortForwardArgs(namespace, target string, localPort, targetPort int, timeout time.Duration, kubeContext string) []string {
	var args []string
	if kubeContext != "" {
		args = append(args, "--context", kubeContext)
	}
	args = append(args,
		"port-forward",
		"-n", namespace,
		target,
		fmt.Sprintf("%d:%d", localPort, targetPort),
		"--request-timeout="+fmt.Sprintf("%.0fs", timeout.Seconds()),
	)
	return args
}
