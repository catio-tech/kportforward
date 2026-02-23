package collector

import (
	"fmt"
	"time"

	"github.com/victorkazakov/kportforward/internal/utils"
)

// Emitter handles emitting collection events to the configured output
type Emitter struct {
	logger  *utils.Logger
	version string
}

// NewEmitter creates a new event emitter
func NewEmitter(logger *utils.Logger, version string) *Emitter {
	return &Emitter{
		logger:  logger,
		version: version,
	}
}

// EmitWorkspaceEvent emits a per-workspace usage metrics event
func (e *Emitter) EmitWorkspaceEvent(
	tenantID, workspaceID, workspaceName string,
	bucketStart, bucketEnd time.Time,
	metrics UsageMetrics,
	collectionStart time.Time,
) error {
	event := CollectionEvent{
		EventType:       "usage_metrics",
		EventVersion:    "1.0",
		Timestamp:       bucketEnd,
		TenantID:        tenantID,
		WorkspaceID:     &workspaceID,
		WorkspaceName:   &workspaceName,
		TimeBucketStart: bucketStart,
		TimeBucketEnd:   bucketEnd,
		Metrics:         metrics,
		Metadata: EventMetadata{
			CollectorVersion:     e.version,
			CollectionTimestamp:  time.Now(),
			CollectionDurationMs: time.Since(collectionStart).Milliseconds(),
		},
	}

	if err := e.logger.EmitStructured(event); err != nil {
		return fmt.Errorf("failed to emit workspace event: %w", err)
	}

	return nil
}

// EmitTenantEvent emits a per-tenant rollup usage metrics event
func (e *Emitter) EmitTenantEvent(
	tenantID string,
	bucketStart, bucketEnd time.Time,
	metrics UsageMetrics,
	collectionStart time.Time,
) error {
	event := CollectionEvent{
		EventType:       "usage_metrics",
		EventVersion:    "1.0",
		Timestamp:       bucketEnd,
		TenantID:        tenantID,
		WorkspaceID:     nil, // Null for tenant rollup
		TimeBucketStart: bucketStart,
		TimeBucketEnd:   bucketEnd,
		Metrics:         metrics,
		Metadata: EventMetadata{
			CollectorVersion:     e.version,
			CollectionTimestamp:  time.Now(),
			CollectionDurationMs: time.Since(collectionStart).Milliseconds(),
		},
	}

	if err := e.logger.EmitStructured(event); err != nil {
		return fmt.Errorf("failed to emit tenant event: %w", err)
	}

	return nil
}
