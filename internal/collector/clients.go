package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	commonpb "github.com/catio-tech/protos/pb/go_proto/common"
	archInventorypb "github.com/catio-tech/protos/pb/go_proto/service-interfaces/architectureInventoryService"
	recommendationspb "github.com/catio-tech/protos/pb/go_proto/service-interfaces/recommendationsService"
	requirementspb "github.com/catio-tech/protos/pb/go_proto/service-interfaces/requirementsService"

	"github.com/victorkazakov/kportforward/internal/config"
)

// ServiceClients holds all the service clients needed for data collection
type ServiceClients struct {
	httpClient *http.Client
	grpcConns  map[string]*grpc.ClientConn
	endpoints  config.ServiceEndpoints
}

// NewServiceClients creates a new service clients instance
func NewServiceClients(endpoints config.ServiceEndpoints) *ServiceClients {
	return &ServiceClients{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		grpcConns: make(map[string]*grpc.ClientConn),
		endpoints: endpoints,
	}
}

// Close closes all gRPC connections
func (sc *ServiceClients) Close() error {
	for _, conn := range sc.grpcConns {
		if err := conn.Close(); err != nil {
			return err
		}
	}
	return nil
}

// getGRPCConn gets or creates a gRPC connection to the specified host.
// Connections are established lazily on first RPC call; use the RPC context to control timeouts.
func (sc *ServiceClients) getGRPCConn(host string) (*grpc.ClientConn, error) {
	if conn, exists := sc.grpcConns[host]; exists {
		return conn, nil
	}

	conn, err := grpc.NewClient(host,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC client for %s: %w", host, err)
	}

	sc.grpcConns[host] = conn
	return conn, nil
}

// GetWorkspaces fetches all workspaces for a given tenant from the environment service
func (sc *ServiceClients) GetWorkspaces(ctx context.Context, tenantID string) ([]Environment, error) {
	url := fmt.Sprintf("%s/api/environment/%s", sc.endpoints.Environment.URL, tenantID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := sc.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call environment service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("environment service returned status %d: %s", resp.StatusCode, string(body))
	}

	var environments []Environment
	if err := json.NewDecoder(resp.Body).Decode(&environments); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return environments, nil
}

// GetArchInventoryMetadata fetches component and relationship counts from architecture-inventory service
func (sc *ServiceClients) GetArchInventoryMetadata(ctx context.Context, tenantID, workspaceID string) (*ArchInventoryMetadata, error) {
	conn, err := sc.getGRPCConn(sc.endpoints.ArchitectureInventory.Host)
	if err != nil {
		return nil, err
	}

	client := archInventorypb.NewMetadataClient(conn)
	resp, err := client.GetMetadata(ctx, &archInventorypb.GetMetadataRequest{
		TenantId:      tenantID,
		EnvironmentId: &workspaceID,
	})
	if err != nil {
		return nil, fmt.Errorf("GetMetadata RPC failed: %w", err)
	}

	return &ArchInventoryMetadata{
		ComponentCount:    int(resp.ComponentCount),
		RelationshipCount: int(resp.RelationshipCount),
	}, nil
}

// GetRecommendationCount fetches the total count of recommendations from recommendations service
func (sc *ServiceClients) GetRecommendationCount(ctx context.Context, tenantID, workspaceID string) (int, error) {
	conn, err := sc.getGRPCConn(sc.endpoints.Recommendations.Host)
	if err != nil {
		return 0, err
	}

	pageSize := int32(1)
	client := recommendationspb.NewRecommendationsClient(conn)
	resp, err := client.ListRecommendations(ctx, &recommendationspb.ListRecommendationsRequest{
		TenantId:      tenantID,
		EnvironmentId: workspaceID,
		Pagination: &commonpb.PaginationOptions{
			PageSize: &pageSize,
		},
	})
	if err != nil {
		return 0, fmt.Errorf("ListRecommendations RPC failed: %w", err)
	}

	return int(resp.Pagination.GetTotalCount()), nil
}

// GetRequirementsCount fetches the total count of requirements from requirements service
func (sc *ServiceClients) GetRequirementsCount(ctx context.Context, tenantID, workspaceID string) (int, error) {
	conn, err := sc.getGRPCConn(sc.endpoints.Requirements.Host)
	if err != nil {
		return 0, err
	}

	pageSize := int32(1)
	client := requirementspb.NewRequirementsClient(conn)
	resp, err := client.ListRequirementsAvailable(ctx, &requirementspb.ListRequirementsAvailableRequest{
		TenantId:    tenantID,
		WorkspaceId: &workspaceID,
		Pagination: &commonpb.PaginationOptions{
			PageSize: &pageSize,
		},
	})
	if err != nil {
		return 0, fmt.Errorf("ListRequirementsAvailable RPC failed: %w", err)
	}

	return int(resp.Pagination.GetTotalCount()), nil
}
