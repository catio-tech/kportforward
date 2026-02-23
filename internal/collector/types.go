package collector

import (
	"time"
)

// UsageMetrics represents aggregated metrics for a tenant or workspace
type UsageMetrics struct {
	// Workspace count (tenant-level only)
	WorkspaceCount int `json:"workspace_count,omitempty"`

	// MVP Metrics
	ComponentCount      int `json:"component_count"`
	RelationshipCount   int `json:"relationship_count"`
	RecommendationCount int `json:"recommendation_count"`
	RequirementsCount   int `json:"requirements_count"`

	// Future metrics (placeholders for schema extensibility)
	ArchieConversationCount int     `json:"archie_conversation_count,omitempty"`
	ArchieMessageCount      int     `json:"archie_message_count,omitempty"`
	UserCount               int     `json:"user_count,omitempty"`
	ActiveUserCount         int     `json:"active_user_count,omitempty"`
	ViewCount               int     `json:"view_count,omitempty"`
	ModelCostUSD            float64 `json:"model_cost_usd,omitempty"`
	EstimatedCustomerCostUSD float64 `json:"estimated_customer_cost_usd,omitempty"`
	StorageGB               float64 `json:"storage_gb,omitempty"`
	APICallCount            int     `json:"api_call_count,omitempty"`
}

// CollectionEvent represents a single usage metrics event for Splunk ingestion
type CollectionEvent struct {
	EventType   string    `json:"event_type"`
	EventVersion string   `json:"event_version"`
	Timestamp   time.Time `json:"timestamp"`
	TenantID    string    `json:"tenant_id"`
	WorkspaceID *string   `json:"workspace_id"` // Null for tenant-level rollups
	WorkspaceName *string `json:"workspace_name,omitempty"`
	TimeBucketStart time.Time `json:"time_bucket_start"`
	TimeBucketEnd time.Time `json:"time_bucket_end"`
	Metrics UsageMetrics `json:"metrics"`
	Metadata EventMetadata `json:"metadata"`
}

// EventMetadata contains metadata about the collection process
type EventMetadata struct {
	CollectorVersion string `json:"collector_version"`
	CollectionTimestamp time.Time `json:"collection_timestamp"`
	CollectionDurationMs int64 `json:"collection_duration_ms,omitempty"`
}

// CollectorState tracks the last collection run for idempotency
type CollectorState struct {
	LastCollection LastCollectionInfo `json:"last_collection"`
	TenantCheckpoints map[string]map[string]time.Time `json:"tenant_checkpoints"`
}

// LastCollectionInfo stores information about the last successful collection
type LastCollectionInfo struct {
	TimeBucketStart time.Time `json:"time_bucket_start"`
	TimeBucketEnd time.Time `json:"time_bucket_end"`
	CompletedAt time.Time `json:"completed_at"`
}

// Environment represents a workspace/environment from the environment service
type Environment struct {
	ID string `json:"id"`
	Name string `json:"name"`
	TenantID string `json:"tenantId"`
}

// ArchInventoryMetadata represents the response from architecture-inventory GetMetadata
type ArchInventoryMetadata struct {
	ComponentCount int `json:"componentCount"`
	RelationshipCount int `json:"relationshipCount"`
}

// ListRecommendationsResponse represents the response from recommendations service
type ListRecommendationsResponse struct {
	Total int `json:"total"`
	Items []interface{} `json:"items"` // We only care about the total count
}

// ListRequirementsResponse represents the response from requirements service
type ListRequirementsResponse struct {
	Total int `json:"total"`
	Items []interface{} `json:"items"` // We only care about the total count
}

