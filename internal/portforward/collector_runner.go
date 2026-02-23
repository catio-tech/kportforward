package portforward

import (
	"context"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/victorkazakov/kportforward/internal/collector"
	"github.com/victorkazakov/kportforward/internal/utils"
)

// startBackgroundCollector starts a goroutine that waits for the environment
// service port to be reachable, then runs the data collector once. After the
// first attempt it polls hourly — the idempotency state file prevents duplicate
// collections within the same daily time bucket.
//
// Output is written to a log file (not stdout) because the TUI occupies stdout.
func (m *Manager) startBackgroundCollector() {
	if !m.config.Collector.Enabled {
		m.logger.Info("Collector: disabled in config, skipping background collection")
		return
	}
	if len(m.config.Collector.Tenants) == 0 {
		m.logger.Info("Collector: no tenants configured, skipping background collection")
		return
	}

	envAddr := collectorTCPAddress(m.config.Collector.Services.Environment.URL)
	if envAddr == "" {
		m.logger.Warn("Collector: no environment service address configured, skipping background collection")
		return
	}

	m.logger.Info("Collector: background collection enabled, polling %s every 10s", envAddr)
	go m.collectorLoop(envAddr)
}

func (m *Manager) collectorLoop(envAddr string) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			if isTCPReachable(envAddr) {
				m.logger.Info("Collector: environment service reachable, starting collection")
				m.runCollection()
				// Slow down after first attempt — idempotency handles daily dedup
				ticker.Reset(1 * time.Hour)
			} else {
				m.logger.Info("Collector: waiting for environment service at %s", envAddr)
			}
		}
	}
}

func (m *Manager) runCollection() {
	logFile := expandLogFilePath(m.config.Collector.Output.LogFile)

	if err := os.MkdirAll(filepath.Dir(logFile), 0755); err != nil {
		m.logger.Error("Collector: failed to create log directory: %v", err)
		return
	}

	logger, err := utils.NewLoggerJSONWithFile(utils.LevelInfo, logFile)
	if err != nil {
		m.logger.Error("Collector: failed to open log file %s: %v", logFile, err)
		return
	}
	defer logger.Close()

	col, err := collector.NewCollector(m.config, logger, "embedded")
	if err != nil {
		m.logger.Error("Collector: failed to initialise: %v", err)
		return
	}
	defer col.Close()

	ctx, cancel := context.WithTimeout(m.ctx, 2*time.Minute)
	defer cancel()

	if err := col.Run(ctx, 24*time.Hour, false); err != nil {
		m.logger.Error("Collector: run failed: %v", err)
		return
	}

	m.logger.Info("Collector: events written to %s", logFile)
}

// collectorTCPAddress extracts a host:port string from a URL or host:port.
// "localhost" is replaced with "127.0.0.1" to avoid Windows resolving it to
// ::1 (IPv6) first, which kubectl port-forward does not bind to.
func collectorTCPAddress(raw string) string {
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		u, err := url.Parse(raw)
		if err != nil {
			return ""
		}
		port := u.Port()
		if port == "" {
			if u.Scheme == "https" {
				port = "443"
			} else {
				port = "80"
			}
		}
		host := u.Hostname()
		if host == "localhost" {
			host = "127.0.0.1"
		}
		return host + ":" + port
	}
	addr := raw
	if strings.HasPrefix(addr, "localhost:") {
		addr = "127.0.0.1:" + addr[len("localhost:"):]
	}
	return addr
}

// isTCPReachable reports whether a TCP connection can be made to addr.
func isTCPReachable(addr string) bool {
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// expandLogFilePath expands a leading ~/ to the user home directory.
func expandLogFilePath(path string) string {
	if path == "" {
		path = "~/.config/kportforward/usage-metrics.log"
	}
	if strings.HasPrefix(path, "~/") {
		if homeDir, err := os.UserHomeDir(); err == nil {
			return filepath.Join(homeDir, path[2:])
		}
	}
	return path
}
