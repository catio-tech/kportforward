package portforward

import (
	"io"
	"net"
	"testing"

	"github.com/victorkazakov/kportforward/internal/config"
	"github.com/victorkazakov/kportforward/internal/utils"
)

// TestResolvePortStrictFailsOnConflict verifies that in multi-env (strict) mode a
// port conflict is a hard error (no silent reassignment), while the default mode
// still reassigns to a free port.
func TestResolvePortStrictFailsOnConflict(t *testing.T) {
	// Occupy a port so it is unavailable.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to occupy a port: %v", err)
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port

	logger := utils.NewLoggerWithOutput(utils.LevelInfo, io.Discard)
	svc := config.Service{Target: "service/x", TargetPort: 80, LocalPort: port, Namespace: "ns"}

	// strict (multi-env): must fail loudly instead of reassigning.
	smStrict := NewServiceManager("x-prod", svc, logger)
	smStrict.strictPort = true
	if _, err := smStrict.resolvePort(); err == nil {
		t.Fatal("expected strict resolvePort to fail on a port conflict, got nil")
	}

	// default (single-env): must reassign to a different, free port.
	smLoose := NewServiceManager("x", svc, logger)
	got, err := smLoose.resolvePort()
	if err != nil {
		t.Fatalf("non-strict resolvePort should reassign, got error: %v", err)
	}
	if got == port {
		t.Fatalf("non-strict resolvePort should have moved off the busy port %d", port)
	}
}
