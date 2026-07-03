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
	if !ok || ps.LocalPort < 70000 || ps.LocalPort >= 80000 {
		t.Fatalf("prod extractor-config-service expected in 70xxx, got %+v (ok=%v)", ps.LocalPort, ok)
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
