package collector

import (
	"testing"
	"time"

	"github.com/victorkazakov/kportforward/internal/config"
	"github.com/victorkazakov/kportforward/internal/utils"
)

func TestNewCollector(t *testing.T) {
	cfg := &config.Config{
		Collector: config.CollectorConfig{
			Enabled: true,
			Tenants: []string{"org_test123"},
			Services: config.ServiceEndpoints{
				Environment: config.ServiceEndpoint{
					URL: "http://localhost:50800",
				},
				ArchitectureInventory: config.ServiceEndpoint{
					Host: "localhost:50100",
				},
				Recommendations: config.ServiceEndpoint{
					Host: "localhost:50106",
				},
				Requirements: config.ServiceEndpoint{
					Host: "localhost:50109",
				},
			},
			Output: config.OutputConfig{
				Format:      "json",
				Destination: "stdout",
			},
			Idempotency: config.IdempotencyConfig{
				StateFile: "/tmp/test_state.json",
			},
		},
	}

	logger := utils.NewLogger(utils.LevelInfo)
	collector, err := NewCollector(cfg, logger, "test-version")
	if err != nil {
		t.Fatalf("Failed to create collector: %v", err)
	}

	if collector == nil {
		t.Fatal("Collector is nil")
	}

	if collector.version != "test-version" {
		t.Errorf("Expected version 'test-version', got '%s'", collector.version)
	}

	if len(collector.config.Tenants) != 1 {
		t.Errorf("Expected 1 tenant, got %d", len(collector.config.Tenants))
	}

	if collector.config.Tenants[0] != "org_test123" {
		t.Errorf("Expected tenant 'org_test123', got '%s'", collector.config.Tenants[0])
	}

	collector.Close()
}

func TestStateManager(t *testing.T) {
	stateFile := "/tmp/test_collector_state.json"
	sm := NewStateManager(stateFile)

	// Test initial state
	bucketStart := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	bucketEnd := time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC)

	if !sm.ShouldCollect(bucketStart, bucketEnd) {
		t.Error("Should collect for new time bucket")
	}

	// Mark as collected
	sm.MarkCollectionComplete(bucketStart, bucketEnd)

	if sm.ShouldCollect(bucketStart, bucketEnd) {
		t.Error("Should not collect already collected bucket")
	}

	// Mark workspace collected
	sm.MarkWorkspaceCollected("org_test", "env_test")
	if sm.state.TenantCheckpoints["org_test"]["env_test"].IsZero() {
		t.Error("Workspace checkpoint not set")
	}

	// Test save and load
	if err := sm.Save(); err != nil {
		t.Fatalf("Failed to save state: %v", err)
	}

	sm2 := NewStateManager(stateFile)
	if err := sm2.Load(); err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	if sm2.ShouldCollect(bucketStart, bucketEnd) {
		t.Error("Loaded state should show bucket already collected")
	}
}

func TestEmitter(t *testing.T) {
	// Use io.Discard to avoid actual output
	logger := utils.NewLoggerJSON(utils.LevelInfo, nil)
	logger.SetJSONMode(true)
	emitter := NewEmitter(logger, "1.0.0")

	if emitter == nil {
		t.Fatal("Emitter is nil")
	}

	if emitter.version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", emitter.version)
	}

	// Note: We don't actually test emitting since it requires a valid io.Writer
	// The emitter functionality is tested through integration tests
}

func TestAggregator(t *testing.T) {
	endpoints := config.ServiceEndpoints{
		Environment: config.ServiceEndpoint{
			URL: "http://localhost:50800",
		},
		ArchitectureInventory: config.ServiceEndpoint{
			Host: "localhost:50100",
		},
		Recommendations: config.ServiceEndpoint{
			Host: "localhost:50106",
		},
		Requirements: config.ServiceEndpoint{
			Host: "localhost:50109",
		},
	}

	clients := NewServiceClients(endpoints)
	logger := utils.NewLogger(utils.LevelInfo)
	aggregator := NewAggregator(clients, logger)

	if aggregator == nil {
		t.Fatal("Aggregator is nil")
	}

	// Test tenant metrics aggregation
	workspaceMetrics := map[string]*UsageMetrics{
		"env_1": {
			ComponentCount:      100,
			RelationshipCount:   200,
			RecommendationCount: 10,
			RequirementsCount:   50,
		},
		"env_2": {
			ComponentCount:      150,
			RelationshipCount:   300,
			RecommendationCount: 15,
			RequirementsCount:   75,
		},
	}

	tenantMetrics, err := aggregator.CollectTenantMetrics(nil, "org_test", workspaceMetrics)
	if err != nil {
		t.Fatalf("Failed to collect tenant metrics: %v", err)
	}

	if tenantMetrics.WorkspaceCount != 2 {
		t.Errorf("Expected 2 workspaces, got %d", tenantMetrics.WorkspaceCount)
	}

	if tenantMetrics.ComponentCount != 250 {
		t.Errorf("Expected 250 components, got %d", tenantMetrics.ComponentCount)
	}

	if tenantMetrics.RelationshipCount != 500 {
		t.Errorf("Expected 500 relationships, got %d", tenantMetrics.RelationshipCount)
	}

	if tenantMetrics.RecommendationCount != 25 {
		t.Errorf("Expected 25 recommendations, got %d", tenantMetrics.RecommendationCount)
	}

	if tenantMetrics.RequirementsCount != 125 {
		t.Errorf("Expected 125 requirements, got %d", tenantMetrics.RequirementsCount)
	}
}

func TestMetricsValidation(t *testing.T) {
	endpoints := config.ServiceEndpoints{}
	clients := NewServiceClients(endpoints)
	logger := utils.NewLogger(utils.LevelInfo)
	aggregator := NewAggregator(clients, logger)

	// Test valid metrics
	validMetrics := &UsageMetrics{
		ComponentCount:      100,
		RelationshipCount:   200,
		RecommendationCount: 10,
		RequirementsCount:   50,
	}

	if err := aggregator.validateMetrics(validMetrics); err != nil {
		t.Errorf("Valid metrics should pass validation: %v", err)
	}

	// Test negative values
	negativeMetrics := &UsageMetrics{
		ComponentCount: -1,
	}

	if err := aggregator.validateMetrics(negativeMetrics); err == nil {
		t.Error("Negative metrics should fail validation")
	}

	// Test unreasonably high values
	highMetrics := &UsageMetrics{
		ComponentCount: 10000000,
	}

	if err := aggregator.validateMetrics(highMetrics); err == nil {
		t.Error("Unreasonably high metrics should fail validation")
	}
}
