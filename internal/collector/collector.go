package collector

import (
	"context"
	"fmt"
	"time"

	"github.com/victorkazakov/kportforward/internal/config"
	"github.com/victorkazakov/kportforward/internal/utils"
)

// Collector orchestrates the data collection process
type Collector struct {
	config      *config.CollectorConfig
	clients     *ServiceClients
	aggregator  *Aggregator
	emitter     *Emitter
	state       *StateManager
	logger      *utils.Logger
	version     string
}

// NewCollector creates a new collector instance
func NewCollector(cfg *config.Config, logger *utils.Logger, version string) (*Collector, error) {
	if !cfg.Collector.Enabled {
		return nil, fmt.Errorf("collector is disabled in configuration")
	}

	clients := NewServiceClients(cfg.Collector.Services)
	aggregator := NewAggregator(clients, logger)
	emitter := NewEmitter(logger, version)
	state := NewStateManager(cfg.Collector.Idempotency.StateFile)

	return &Collector{
		config:     &cfg.Collector,
		clients:    clients,
		aggregator: aggregator,
		emitter:    emitter,
		state:      state,
		logger:     logger,
		version:    version,
	}, nil
}

// Run executes a single collection cycle
func (c *Collector) Run(ctx context.Context, bucketDuration time.Duration, force bool) error {
	collectionStart := time.Now()

	// Calculate time bucket (align to bucket boundaries)
	bucketEnd := time.Now().Truncate(bucketDuration)
	bucketStart := bucketEnd.Add(-bucketDuration)

	c.logger.Info("Starting collection for time bucket: %s to %s", bucketStart.Format(time.RFC3339), bucketEnd.Format(time.RFC3339))

	// Load state for idempotency
	if err := c.state.Load(); err != nil {
		c.logger.Warn("Failed to load state (continuing anyway): %v", err)
	}

	// Check if we should collect this bucket
	if !force && !c.state.ShouldCollect(bucketStart, bucketEnd) {
		c.logger.Info("Time bucket already collected, skipping (use --force to override)")
		return nil
	}

	// Iterate through configured tenants
	for _, tenantID := range c.config.Tenants {
		if err := c.collectTenant(ctx, tenantID, bucketStart, bucketEnd); err != nil {
			c.logger.Error("Failed to collect tenant %s: %v", tenantID, err)
			// Continue with other tenants even if one fails
			continue
		}
	}

	// Mark collection as complete
	c.state.MarkCollectionComplete(bucketStart, bucketEnd)
	if err := c.state.Save(); err != nil {
		c.logger.Error("Failed to save state: %v", err)
		// Don't return error - collection succeeded even if state save failed
	}

	c.logger.Info("Collection completed in %v", time.Since(collectionStart))
	return nil
}

// collectTenant collects metrics for a single tenant and all its workspaces
func (c *Collector) collectTenant(ctx context.Context, tenantID string, bucketStart, bucketEnd time.Time) error {
	c.logger.Info("Collecting metrics for tenant: %s", tenantID)

	// Discover workspaces for this tenant
	workspaces, err := c.clients.GetWorkspaces(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("failed to discover workspaces: %w", err)
	}

	c.logger.Info("Found %d workspaces for tenant %s", len(workspaces), tenantID)

	// Collect metrics for each workspace
	workspaceMetrics := make(map[string]*UsageMetrics)
	for _, workspace := range workspaces {
		metrics, err := c.aggregator.CollectWorkspaceMetrics(ctx, tenantID, workspace.ID)
		if err != nil {
			c.logger.Warn("Failed to collect metrics for workspace %s: %v", workspace.ID, err)
			// Continue with other workspaces
			continue
		}

		workspaceMetrics[workspace.ID] = metrics

		// Emit workspace event
		if err := c.emitter.EmitWorkspaceEvent(
			tenantID,
			workspace.ID,
			workspace.Name,
			bucketStart,
			bucketEnd,
			*metrics,
			time.Now(),
		); err != nil {
			c.logger.Error("Failed to emit workspace event: %v", err)
		}

		// Mark workspace as collected
		c.state.MarkWorkspaceCollected(tenantID, workspace.ID)
	}

	// If we collected metrics for at least one workspace, emit tenant rollup
	if len(workspaceMetrics) > 0 {
		tenantMetrics, err := c.aggregator.CollectTenantMetrics(ctx, tenantID, workspaceMetrics)
		if err != nil {
			return fmt.Errorf("failed to aggregate tenant metrics: %w", err)
		}

		// Emit tenant rollup event
		if err := c.emitter.EmitTenantEvent(
			tenantID,
			bucketStart,
			bucketEnd,
			*tenantMetrics,
			time.Now(),
		); err != nil {
			c.logger.Error("Failed to emit tenant event: %v", err)
		}
	}

	return nil
}

// Close closes all resources
func (c *Collector) Close() error {
	return c.clients.Close()
}
