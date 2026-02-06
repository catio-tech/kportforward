package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/victorkazakov/kportforward/internal/config"
	"github.com/victorkazakov/kportforward/internal/portforward"
	"github.com/victorkazakov/kportforward/internal/ui"
	"github.com/victorkazakov/kportforward/internal/ui_handlers"
	"github.com/victorkazakov/kportforward/internal/updater"
	"github.com/victorkazakov/kportforward/internal/utils"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"

	// CLI flags
	enableGRPCUI         bool
	enableSwaggerUI      bool
	logFile              string
	pprofAddr            string
	memStatsInterval     time.Duration
	heapSnapshotDir      string
	heapSnapshotInterval time.Duration

	// Global root command
	rootCmd = &cobra.Command{
		Use:   "kportforward",
		Short: "A modern Kubernetes port-forward manager",
		Long: `kportforward is a cross-platform tool for managing multiple Kubernetes port-forwards
with a modern terminal UI, automatic recovery, and built-in update system.

Examples:
  # Basic usage
  kportforward

  # With UI integrations
  kportforward --grpcui --swaggerui
  
  # Write logs to file
  kportforward --log-file ./kportforward.log
  
  # Production setup with logging
  kportforward --grpcui --swaggerui --log-file /var/log/kportforward.log

  # Performance profiling
  kportforward profile --cpuprofile=cpu.prof --duration=30s`,
		Run: runPortForward,
	}
)

func main() {

	// Add CLI flags
	rootCmd.Flags().BoolVar(&enableGRPCUI, "grpcui", false, "Enable gRPC UI for RPC services")
	rootCmd.Flags().BoolVar(&enableSwaggerUI, "swaggerui", false, "Enable Swagger UI for REST services")
	rootCmd.Flags().StringVar(&logFile, "log-file", "", "Write logs to file (default: logs are discarded to avoid interfering with TUI)")
	rootCmd.Flags().StringVar(&pprofAddr, "pprof", "", "Start pprof HTTP server (e.g. localhost:6060)")
	rootCmd.Flags().DurationVar(&memStatsInterval, "mem-stats-interval", 0, "Log memory stats every interval (0 to disable)")
	rootCmd.Flags().StringVar(&heapSnapshotDir, "heap-snapshot-dir", "", "Directory to write periodic heap snapshots")
	rootCmd.Flags().DurationVar(&heapSnapshotInterval, "heap-snapshot-interval", 0, "Interval for heap snapshots (0 to disable)")

	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("kportforward %s\n", version)
			fmt.Printf("commit: %s\n", commit)
			fmt.Printf("built: %s\n", date)
		},
	})

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// initializeLogger creates a logger with the appropriate output destination
func initializeLogger(logFile string) (*utils.Logger, error) {
	if logFile == "" {
		// When no log file is specified, discard logs to avoid interfering with TUI
		// The TUI provides visual status updates, so logging to stdout would corrupt the display
		return utils.NewLoggerWithOutput(utils.LevelInfo, io.Discard), nil
	}

	// Create logger with file output
	logger, err := utils.NewLoggerWithFile(utils.LevelInfo, logFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create file logger: %w", err)
	}

	return logger, nil
}

func runPortForward(cmd *cobra.Command, args []string) {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize logger
	logger, err := initializeLogger(logFile)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	logger.Info("Starting kportforward with %d services", len(cfg.PortForwards))

	// Optional pprof server for live profiling
	if pprofAddr != "" {
		go func() {
			logger.Info("Starting pprof server on %s", pprofAddr)
			if err := http.ListenAndServe(pprofAddr, nil); err != nil {
				logger.Warn("pprof server stopped: %v", err)
			}
		}()
	}

	// Optional periodic memory stats logging
	var memStatsStop chan struct{}
	if memStatsInterval > 0 {
		memStatsStop = make(chan struct{})
		go func() {
			ticker := time.NewTicker(memStatsInterval)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					logMemStats(logger)
				case <-memStatsStop:
					return
				}
			}
		}()
	}

	// Optional periodic heap snapshots
	var heapSnapStop chan struct{}
	if heapSnapshotInterval > 0 && heapSnapshotDir != "" {
		heapSnapStop = make(chan struct{})
		go func() {
			ticker := time.NewTicker(heapSnapshotInterval)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					writeHeapSnapshot(heapSnapshotDir, logger)
				case <-heapSnapStop:
					return
				}
			}
		}()
	}

	// Initialize UI handlers
	var grpcUIManager *ui_handlers.GRPCUIManager
	var swaggerUIManager *ui_handlers.SwaggerUIManager

	if enableGRPCUI {
		grpcUIManager = ui_handlers.NewGRPCUIManager(logger)
		if err := grpcUIManager.Enable(); err != nil {
			logger.Warn("Failed to enable gRPC UI: %v", err)
			grpcUIManager = nil
		}
	}

	if enableSwaggerUI {
		swaggerUIManager = ui_handlers.NewSwaggerUIManager(logger)
		if err := swaggerUIManager.Enable(); err != nil {
			logger.Warn("Failed to enable Swagger UI: %v", err)
			swaggerUIManager = nil
		}
	}

	// Create port forward manager
	manager := portforward.NewManager(cfg, logger)

	// Set UI handlers on the manager
	manager.SetUIHandlers(grpcUIManager, swaggerUIManager)

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start port forwarding
	if err := manager.Start(); err != nil {
		logger.Error("Failed to start port forwarding: %v", err)
		os.Exit(1)
	}

	// Initialize and start update manager
	// Repository information for update checks - ensure this matches your GitHub repository
	repoOwner := "catio-tech"
	repoName := "kportforward"
	updateManager := updater.NewManager(repoOwner, repoName, version, logger)
	if err := updateManager.Start(); err != nil {
		logger.Error("Failed to start update manager: %v", err)
		// Don't exit - updates are not critical
	}

	// Initialize and start TUI
	tui := ui.NewTUI(manager.GetStatusChannel(), cfg.PortForwards, manager, manager.GetContextChannel())
	if err := tui.Start(); err != nil {
		logger.Error("Failed to start TUI: %v", err)
		os.Exit(1)
	}

	// Update TUI with initial context and UI handler status
	tui.UpdateKubernetesContext(manager.GetKubernetesContext())
	tui.UpdateUIHandlerStatus(grpcUIManager != nil, swaggerUIManager != nil)

	// Listen for update notifications
	go func() {
		updateChan := updateManager.GetUpdateChannel()
		for updateInfo := range updateChan {
			tui.NotifyUpdateAvailable(updateInfo)
		}
	}()

	// Wait for shutdown signal or TUI quit
	select {
	case <-sigChan:
		logger.Info("Received shutdown signal, stopping services...")
	case <-tui.GetQuitChannel():
		logger.Info("TUI quit, stopping services...")
	}

	// Create a timeout context for graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Channel to track shutdown completion
	shutdownDone := make(chan bool, 1)

	go func() {
		// Graceful shutdown in proper order

		// 1. Stop TUI first to prevent new UI interactions
		if err := tui.Stop(); err != nil {
			logger.Error("Error stopping TUI: %v", err)
		}

		// 2. Stop update manager
		if err := updateManager.Stop(); err != nil {
			logger.Error("Error stopping update manager: %v", err)
		}

		// 3. Stop UI handlers
		if grpcUIManager != nil {
			if err := grpcUIManager.Disable(); err != nil {
				logger.Error("Error stopping gRPC UI manager: %v", err)
			}
		}

		if swaggerUIManager != nil {
			if err := swaggerUIManager.Disable(); err != nil {
				logger.Error("Error stopping Swagger UI manager: %v", err)
			}
		}

		// 4. Stop port-forward manager last
		if err := manager.Stop(); err != nil {
			logger.Error("Error during shutdown: %v", err)
		}

		shutdownDone <- true
	}()

	// Wait for shutdown to complete or timeout
	select {
	case <-shutdownDone:
		logger.Info("Shutdown complete")
	case <-shutdownCtx.Done():
		logger.Error("Shutdown timed out, forcing exit")
		os.Exit(1)
	}

	if memStatsStop != nil {
		close(memStatsStop)
	}
	if heapSnapStop != nil {
		close(heapSnapStop)
	}

	// Close log file if it was opened
	if err := logger.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "Error closing log file: %v\n", err)
	}
}

func displayStatus(status map[string]config.ServiceStatus, kubeContext string) {
	fmt.Printf("\n=== kportforward Status (Context: %s) ===\n", kubeContext)
	fmt.Printf("%-25s %-10s %-8s %-8s %-10s %s\n",
		"Service", "Status", "Local", "PID", "Uptime", "Error")
	fmt.Println(string(make([]byte, 80, 80)))

	for name, svc := range status {
		uptime := ""
		if !svc.StartTime.IsZero() {
			uptime = utils.FormatUptime(svc.StartTime.Sub(svc.StartTime))
		}

		errorMsg := svc.LastError
		if len(errorMsg) > 30 {
			errorMsg = errorMsg[:27] + "..."
		}

		fmt.Printf("%-25s %-10s %-8d %-8d %-10s %s\n",
			name, svc.Status, svc.LocalPort, svc.PID, uptime, errorMsg)
	}
}

func logMemStats(logger *utils.Logger) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	logger.Info(
		"MemStats: alloc=%dMB heap=%dMB sys=%dMB num_gc=%d goroutines=%d",
		bToMb(m.Alloc),
		bToMb(m.HeapAlloc),
		bToMb(m.Sys),
		m.NumGC,
		runtime.NumGoroutine(),
	)
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}

func writeHeapSnapshot(dir string, logger *utils.Logger) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		logger.Warn("Heap snapshot mkdir failed: %v", err)
		return
	}

	ts := time.Now().Format("20060102-150405")
	path := filepath.Join(dir, fmt.Sprintf("heap-%s.pb.gz", ts))

	f, err := os.Create(path)
	if err != nil {
		logger.Warn("Heap snapshot create failed: %v", err)
		return
	}
	defer f.Close()

	// Force GC so the snapshot is meaningful
	runtime.GC()

	if err := pprof.WriteHeapProfile(f); err != nil {
		logger.Warn("Heap snapshot write failed: %v", err)
		return
	}

	logger.Info("Wrote heap snapshot: %s", path)
}
