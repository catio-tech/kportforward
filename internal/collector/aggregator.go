package collector

import (
	"context"
	"fmt"

	"github.com/victorkazakov/kportforward/internal/utils"
)

// Aggregator handles metric aggregation logic
type Aggregator struct {
	clients *ServiceClients
	logger  *utils.Logger
}

// NewAggregator creates a new aggregator instance
func NewAggregator(clients *ServiceClients, logger *utils.Logger) *Aggregator {
	return &Aggregator{
		clients: clients,
		logger:  logger,
	}
}

// CollectWorkspaceMetrics collects all metrics for a single workspace.
// Individual service failures are logged as warnings but do not abort collection —
// the event is emitted with zero for any metric that could not be retrieved.
func (a *Aggregator) CollectWorkspaceMetrics(ctx context.Context, tenantID, workspaceID string) (*UsageMetrics, error) {
	metrics := &UsageMetrics{}

	// Collect architecture inventory metadata (component and relationship counts)
	if archMeta, err := a.clients.GetArchInventoryMetadata(ctx, tenantID, workspaceID); err != nil {
		a.logger.Warn("Failed to get architecture inventory metadata for workspace %s: %v", workspaceID, err)
	} else if archMeta != nil {
		metrics.ComponentCount = archMeta.ComponentCount
		metrics.RelationshipCount = archMeta.RelationshipCount
	}

	// Collect recommendation count
	if recCount, err := a.clients.GetRecommendationCount(ctx, tenantID, workspaceID); err != nil {
		a.logger.Warn("Failed to get recommendation count for workspace %s: %v", workspaceID, err)
	} else {
		metrics.RecommendationCount = recCount
	}

	// Collect requirements count
	if reqCount, err := a.clients.GetRequirementsCount(ctx, tenantID, workspaceID); err != nil {
		a.logger.Warn("Failed to get requirements count for workspace %s: %v", workspaceID, err)
	} else {
		metrics.RequirementsCount = reqCount
	}

	// Validate metrics (sanity check)
	if err := a.validateMetrics(metrics); err != nil {
		a.logger.Warn("Metric validation warning for workspace %s: %v", workspaceID, err)
	}

	return metrics, nil
}

// CollectTenantMetrics aggregates metrics across all workspaces for a tenant
func (a *Aggregator) CollectTenantMetrics(ctx context.Context, tenantID string, workspaceMetrics map[string]*UsageMetrics) (*UsageMetrics, error) {
	tenantMetrics := &UsageMetrics{
		WorkspaceCount: len(workspaceMetrics),
	}

	// Sum up all workspace metrics
	for _, wm := range workspaceMetrics {
		tenantMetrics.ComponentCount += wm.ComponentCount
		tenantMetrics.RelationshipCount += wm.RelationshipCount
		tenantMetrics.RecommendationCount += wm.RecommendationCount
		tenantMetrics.RequirementsCount += wm.RequirementsCount
	}

	return tenantMetrics, nil
}

// validateMetrics performs sanity checks on collected metrics
func (a *Aggregator) validateMetrics(metrics *UsageMetrics) error {
	// Check for negative values (should never happen)
	if metrics.ComponentCount < 0 ||
		metrics.RelationshipCount < 0 ||
		metrics.RecommendationCount < 0 ||
		metrics.RequirementsCount < 0 {
		return fmt.Errorf("detected negative metric values")
	}

	// Check for unreasonably high values (potential data issue)
	if metrics.ComponentCount > 1000000 ||
		metrics.RelationshipCount > 10000000 ||
		metrics.RecommendationCount > 100000 ||
		metrics.RequirementsCount > 100000 {
		return fmt.Errorf("detected unusually high metric values")
	}

	return nil
}
