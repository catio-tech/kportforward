package utils

import (
	"strings"
	"testing"
	"time"
)

func TestBuildKubectlPortForwardArgs_NoContext(t *testing.T) {
	args := BuildKubectlPortForwardArgs("catio-data-extraction", "service/extractor-config-service-rpc", 50102, 50051, 30*time.Second, "")

	if args[0] != "port-forward" {
		t.Fatalf("expected first arg to be port-forward, got %q (%v)", args[0], args)
	}
	joined := strings.Join(args, " ")
	if strings.Contains(joined, "--context") {
		t.Fatalf("expected no --context when kubeContext is empty, got %v", args)
	}
	if !strings.Contains(joined, "50102:50051") {
		t.Fatalf("expected port mapping 50102:50051 in %v", args)
	}
	if !strings.Contains(joined, "-n catio-data-extraction") {
		t.Fatalf("expected namespace flag in %v", args)
	}
}

func TestBuildKubectlPortForwardArgs_WithContext(t *testing.T) {
	ctx := "arn:aws:eks:us-west-2:090135924592:cluster/catio-cluster"
	args := BuildKubectlPortForwardArgs("catio-data-extraction", "service/extractor-config-service-rpc", 70102, 50051, 30*time.Second, ctx)

	// --context must come before the port-forward subcommand.
	if args[0] != "--context" || args[1] != ctx {
		t.Fatalf("expected args to start with --context %q, got %v", ctx, args)
	}
	if args[2] != "port-forward" {
		t.Fatalf("expected port-forward after context, got %v", args)
	}
	if !strings.Contains(strings.Join(args, " "), "70102:50051") {
		t.Fatalf("expected port mapping 70102:50051 in %v", args)
	}
}
