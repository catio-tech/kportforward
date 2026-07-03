package config

import (
	"strings"
	"testing"
)

// TestLoadMultiEnvConfigEmbedded verifies the embedded environments.yaml parses
// and contains a dev (50xxx) and prod (70xxx) environment, each pinned to a
// distinct kubectl context.
func TestLoadMultiEnvConfigEmbedded(t *testing.T) {
	mec, err := LoadMultiEnvConfig()
	if err != nil {
		t.Fatalf("LoadMultiEnvConfig failed: %v", err)
	}
	if len(mec.Environments) < 2 {
		t.Fatalf("expected at least dev+prod environments, got %d", len(mec.Environments))
	}

	byName := map[string]Environment{}
	for _, e := range mec.Environments {
		if e.Context == "" {
			t.Fatalf("environment %q has no context", e.Name)
		}
		if len(e.Services) == 0 {
			t.Fatalf("environment %q has no services", e.Name)
		}
		byName[e.Name] = e
	}

	dev, ok := byName["dev"]
	if !ok {
		t.Fatal("missing dev environment")
	}
	prod, ok := byName["prod"]
	if !ok {
		t.Fatal("missing prod environment")
	}
	if dev.Context == prod.Context {
		t.Fatalf("dev and prod must pin different contexts, both = %q", dev.Context)
	}

	// A representative service should sit in the 50xxx band for dev and 70xxx for prod.
	ds, ok := dev.Services["extractor-config-service"]
	if !ok || ds.LocalPort < 50000 || ds.LocalPort >= 60000 {
		t.Fatalf("dev extractor-config-service expected in 50xxx, got %+v (ok=%v)", ds.LocalPort, ok)
	}
	ps, ok := prod.Services["extractor-config-service"]
	if !ok || ps.LocalPort < 60000 || ps.LocalPort >= 70000 {
		t.Fatalf("prod extractor-config-service expected in 60xxx, got %+v (ok=%v)", ps.LocalPort, ok)
	}

	// Every configured port must be a valid TCP port (<= 65535); 70xxx would not be.
	for _, e := range mec.Environments {
		for name, s := range e.Services {
			if s.LocalPort < 1 || s.LocalPort > 65535 {
				t.Errorf("%s/%s localPort %d is outside the valid TCP range", e.Name, name, s.LocalPort)
			}
		}
	}
}

// TestValidateMultiEnvConfigRejectsBadPort ensures an out-of-range port is
// rejected at load with a clear error (rather than surfacing later as a
// misleading "port already in use").
func TestValidateMultiEnvConfigRejectsBadPort(t *testing.T) {
	ok := &MultiEnvConfig{Environments: []Environment{
		{Name: "prod", Context: "ctx", Services: map[string]Service{"a": {LocalPort: 60102}}},
	}}
	if err := validateMultiEnvConfig(ok); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}

	bad := &MultiEnvConfig{Environments: []Environment{
		{Name: "prod", Context: "ctx", Services: map[string]Service{"a": {LocalPort: 70102}}},
	}}
	if err := validateMultiEnvConfig(bad); err == nil {
		t.Fatal("expected out-of-range port 70102 to be rejected, got nil")
	}

	noCtx := &MultiEnvConfig{Environments: []Environment{
		{Name: "prod", Context: "", Services: map[string]Service{"a": {LocalPort: 60102}}},
	}}
	if err := validateMultiEnvConfig(noCtx); err == nil {
		t.Fatal("expected environment without context to be rejected, got nil")
	}
}

// TestMultiEnvPortsMatchAcrossEnvironments verifies the embedded config's port
// convention: every service present in both dev and prod shares the same last
// three digits and differs only by the 50xxx (dev) / 70xxx (prod) prefix.
func TestMultiEnvPortsMatchAcrossEnvironments(t *testing.T) {
	mec, err := LoadMultiEnvConfig()
	if err != nil {
		t.Fatalf("LoadMultiEnvConfig failed: %v", err)
	}
	byName := map[string]Environment{}
	for _, e := range mec.Environments {
		byName[e.Name] = e
	}
	dev, prod := byName["dev"], byName["prod"]

	for name, ds := range dev.Services {
		ps, ok := prod.Services[name]
		if !ok {
			continue // dev-only service (e.g. overwatch) — no prod counterpart
		}
		if ds.LocalPort/1000 != 50 {
			t.Errorf("dev %s port %d is not in the 50xxx band", name, ds.LocalPort)
		}
		if ps.LocalPort/1000 != 60 {
			t.Errorf("prod %s port %d is not in the 60xxx band", name, ps.LocalPort)
		}
		if ds.LocalPort%1000 != ps.LocalPort%1000 {
			t.Errorf("%s: dev(%d) and prod(%d) must share the last three digits", name, ds.LocalPort, ps.LocalPort)
		}
	}
}

// TestFlattenEnvironments verifies services are env-qualified, context-pinned,
// and keep their per-environment ports.
func TestFlattenEnvironments(t *testing.T) {
	mec := &MultiEnvConfig{
		Environments: []Environment{
			{
				Name:    "dev",
				Context: "ctx-dev",
				Services: map[string]Service{
					"svc-a": {Target: "service/a", TargetPort: 80, LocalPort: 50100, Namespace: "ns"},
					"svc-x": {Target: "service/x", TargetPort: 80, LocalPort: 50200, Namespace: "ns", Disabled: true},
				},
			},
			{
				Name:    "prod",
				Context: "ctx-prod",
				Services: map[string]Service{
					"svc-a": {Target: "service/a", TargetPort: 80, LocalPort: 70100, Namespace: "ns"},
				},
			},
		},
	}

	flat := FlattenEnvironments(mec)

	if _, ok := flat.PortForwards["svc-x-dev"]; ok {
		t.Fatal("disabled service should be omitted from flattened config")
	}

	dev, ok := flat.PortForwards["svc-a-dev"]
	if !ok {
		t.Fatal("missing svc-a-dev in flattened config")
	}
	if dev.Context != "ctx-dev" || dev.LocalPort != 50100 {
		t.Fatalf("svc-a-dev wrong pinning: %+v", dev)
	}

	prod, ok := flat.PortForwards["svc-a-prod"]
	if !ok {
		t.Fatal("missing svc-a-prod in flattened config")
	}
	if prod.Context != "ctx-prod" || prod.LocalPort != 70100 {
		t.Fatalf("svc-a-prod wrong pinning: %+v", prod)
	}

	// Same logical service, distinct keys/ports/contexts — the whole point.
	if strings.HasPrefix(dev.Context, prod.Context) || dev.LocalPort == prod.LocalPort {
		t.Fatal("dev and prod instances of svc-a must differ in context and port")
	}
}
