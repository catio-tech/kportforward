package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/victorkazakov/kportforward/internal/collector"
	"github.com/victorkazakov/kportforward/internal/config"
	"github.com/victorkazakov/kportforward/internal/utils"
)

var (
	// Collector command flags
	collectOnce           bool
	collectSchedule       string
	collectBucketDuration time.Duration
	collectOutputFile     string
	collectFormat         string
	collectTenantID       string
	collectWorkspaceID    string
	collectForce          bool
)

func init() {
	collectCmd := &cobra.Command{
		Use:   "collect",
		Short: "Collect usage metrics from CATIO services",
		Long: `Collect usage metrics from internal CATIO services and emit structured JSON logs.

The collector aggregates metrics such as component counts, relationship counts,
recommendation counts, and requirements counts per tenant and workspace.

Examples:
  # Run once and exit
  kportforward collect --once

  # Run with custom time bucket (daily is default)
  kportforward collect --once --bucket-duration=24h

  # Output to file instead of stdout
  kportforward collect --once --output-file=/var/log/catio/usage-metrics.log

  # Collect for specific tenant only
  kportforward collect --once --tenant-id=org_abc123

  # Force re-collection of already collected time buckets
  kportforward collect --once --force

  # Run with external scheduling (e.g., cron)
  # 0 0 * * * /usr/local/bin/kportforward collect --once`,
		RunE: runCollect,
	}

	// Add flags
	collectCmd.Flags().BoolVar(&collectOnce, "once", false, "Run once and exit (default: false)")
	collectCmd.Flags().StringVar(&collectSchedule, "schedule", "0 0 * * *", "Cron expression for scheduling (not yet implemented)")
	collectCmd.Flags().DurationVar(&collectBucketDuration, "bucket-duration", 24*time.Hour, "Time bucket size for aggregation")
	collectCmd.Flags().StringVar(&collectOutputFile, "output-file", "", "Output file path (default: stdout)")
	collectCmd.Flags().StringVar(&collectFormat, "format", "json", "Output format (json or text)")
	collectCmd.Flags().StringVar(&collectTenantID, "tenant-id", "", "Collect for specific tenant only")
	collectCmd.Flags().StringVar(&collectWorkspaceID, "workspace-id", "", "Collect for specific workspace only (requires --tenant-id)")
	collectCmd.Flags().BoolVar(&collectForce, "force", false, "Force re-collection even if already collected")

	// Add to root command
	rootCmd.AddCommand(collectCmd)
}

func runCollect(cmd *cobra.Command, args []string) error {
	// Disable remote config for collector (use embedded defaults only)
	// This ensures collector config is available from embedded default.yaml
	config.SetRemoteConfigURL("")

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Check if collector is enabled
	if !cfg.Collector.Enabled {
		return fmt.Errorf("collector is disabled in configuration")
	}

	// Override config with command-line flags
	if collectTenantID != "" {
		cfg.Collector.Tenants = []string{collectTenantID}
	}

	if collectOutputFile != "" {
		cfg.Collector.Output.Destination = collectOutputFile
	}

	if collectFormat != "" {
		cfg.Collector.Output.Format = collectFormat
	}

	// Initialize logger based on output configuration
	var logger *utils.Logger
	if cfg.Collector.Output.Format == "json" {
		if cfg.Collector.Output.Destination == "stdout" || cfg.Collector.Output.Destination == "" {
			logger = utils.NewLoggerJSON(utils.LevelInfo, os.Stdout)
		} else {
			// Expand ~ in file path
			outputFile := cfg.Collector.Output.Destination
			if strings.HasPrefix(outputFile, "~/") {
				homeDir, err := os.UserHomeDir()
				if err == nil {
					outputFile = filepath.Join(homeDir, outputFile[2:])
				}
			}

			var err error
			logger, err = utils.NewLoggerJSONWithFile(utils.LevelInfo, outputFile)
			if err != nil {
				return fmt.Errorf("failed to create JSON logger: %w", err)
			}
			defer logger.Close()
		}
	} else {
		// Text mode logger
		if cfg.Collector.Output.Destination == "stdout" || cfg.Collector.Output.Destination == "" {
			logger = utils.NewLogger(utils.LevelInfo)
		} else {
			var err error
			logger, err = utils.NewLoggerWithFile(utils.LevelInfo, cfg.Collector.Output.Destination)
			if err != nil {
				return fmt.Errorf("failed to create text logger: %w", err)
			}
			defer logger.Close()
		}
	}

	logger.Info("Starting kportforward collector version %s", version)

	// Create collector
	col, err := collector.NewCollector(cfg, logger, version)
	if err != nil {
		return fmt.Errorf("failed to create collector: %w", err)
	}
	defer col.Close()

	// Set up context with cancellation for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Info("Received shutdown signal, cancelling collection...")
		cancel()
	}()

	// Run collection
	if collectOnce {
		// Run once and exit
		if err := col.Run(ctx, collectBucketDuration, collectForce); err != nil {
			return fmt.Errorf("collection failed: %w", err)
		}
		logger.Info("Collection completed successfully")
	} else {
		// Scheduled mode not yet implemented
		return fmt.Errorf("scheduled mode not yet implemented - use --once flag or external scheduler (cron)")
	}

	return nil
}
