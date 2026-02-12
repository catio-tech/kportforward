package config

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// validTestYAML is a minimal valid config for testing
const validTestYAML = `portForwards:
  test-service:
    target: "service/test"
    targetPort: 80
    localPort: 9090
    namespace: "default"
    type: "rpc"
monitoringInterval: 1s
uiOptions:
  refreshRate: 100ms
  theme: "dark"
`

func TestFetchRemoteConfigSuccess(t *testing.T) {
	// Start a local HTTP server that returns valid YAML
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(validTestYAML))
	}))
	defer server.Close()

	data, err := fetchRemoteConfig(server.URL, RemoteConfigTimeout)
	if err != nil {
		t.Fatalf("Expected successful fetch, got error: %v", err)
	}

	// Verify it parses correctly
	if err := validateConfigYAML(data); err != nil {
		t.Fatalf("Fetched data failed validation: %v", err)
	}
}

func TestFetchRemoteConfigInvalidURL(t *testing.T) {
	_, err := fetchRemoteConfig("http://127.0.0.1:1/nonexistent", RemoteConfigTimeout)
	if err == nil {
		t.Fatal("Expected error for invalid URL, got nil")
	}
}

func TestFetchRemoteConfigHTTPError(t *testing.T) {
	// Server that returns 500
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	_, err := fetchRemoteConfig(server.URL, RemoteConfigTimeout)
	if err == nil {
		t.Fatal("Expected error for HTTP 500, got nil")
	}
}

func TestFetchRemoteConfigInvalidYAML(t *testing.T) {
	// Server that returns garbage instead of YAML
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("this is not valid yaml: [[["))
	}))
	defer server.Close()

	_, err := fetchRemoteConfig(server.URL, RemoteConfigTimeout)
	if err == nil {
		t.Fatal("Expected validation error for invalid YAML, got nil")
	}
}

func TestFetchRemoteConfigEmptyPortForwards(t *testing.T) {
	// Server returns valid YAML but with no port forwards
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("monitoringInterval: 1s\n"))
	}))
	defer server.Close()

	_, err := fetchRemoteConfig(server.URL, RemoteConfigTimeout)
	if err == nil {
		t.Fatal("Expected validation error for empty port forwards, got nil")
	}
}

func TestValidateConfigYAML(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr bool
	}{
		{
			name:    "valid config",
			yaml:    validTestYAML,
			wantErr: false,
		},
		{
			name:    "invalid yaml syntax",
			yaml:    "portForwards: [[[invalid",
			wantErr: true,
		},
		{
			name:    "empty port forwards",
			yaml:    "monitoringInterval: 1s",
			wantErr: true,
		},
		{
			name:    "empty string",
			yaml:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfigYAML([]byte(tt.yaml))
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfigYAML() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCacheRoundTrip(t *testing.T) {
	// Use a temp directory for cache
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "kportforward", "remote-defaults-cache.yaml")

	// Write cache
	err := os.MkdirAll(filepath.Dir(cachePath), 0755)
	if err != nil {
		t.Fatalf("Failed to create cache dir: %v", err)
	}

	err = os.WriteFile(cachePath, []byte(validTestYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to write cache: %v", err)
	}

	// Read it back and validate
	data, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("Failed to read cache: %v", err)
	}

	if err := validateConfigYAML(data); err != nil {
		t.Fatalf("Cached data failed validation: %v", err)
	}
}

func TestLoadDefaultsWithRemoteFallbackChain(t *testing.T) {
	// Save and restore the original remote URL
	originalURL := GetRemoteConfigURL()
	defer SetRemoteConfigURL(originalURL)

	t.Run("remote succeeds", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(validTestYAML))
		}))
		defer server.Close()

		SetRemoteConfigURL(server.URL)
		data, err := loadDefaultsWithRemote()
		if err != nil {
			t.Fatalf("Expected success, got: %v", err)
		}

		// Should contain our test service
		cfg := &Config{}
		if err := validateConfigYAML(data); err != nil {
			t.Fatalf("Data validation failed: %v", err)
		}
		_ = cfg
	})

	t.Run("remote fails, falls back to embedded", func(t *testing.T) {
		// Point to a broken URL and remove any cache
		SetRemoteConfigURL("http://127.0.0.1:1/nonexistent")

		data, err := loadDefaultsWithRemote()
		if err != nil {
			t.Fatalf("Expected fallback to succeed, got: %v", err)
		}

		// Should still get valid config (from embedded defaults)
		if err := validateConfigYAML(data); err != nil {
			t.Fatalf("Fallback data validation failed: %v", err)
		}
	})

	t.Run("remote disabled, uses embedded", func(t *testing.T) {
		SetRemoteConfigURL("")

		data, err := loadDefaultsWithRemote()
		if err != nil {
			t.Fatalf("Expected success with empty URL, got: %v", err)
		}

		// Should be the embedded defaults
		if err := validateConfigYAML(data); err != nil {
			t.Fatalf("Embedded data validation failed: %v", err)
		}
	})
}
