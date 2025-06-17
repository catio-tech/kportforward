# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Overview

kportforward is a modern, cross-platform Go application that automates managing and monitoring multiple Kubernetes port-forwards. It features a rich terminal UI, automatic recovery, embedded configuration, and built-in update system. The tool reads configuration from embedded defaults (which can be overridden by user config), starts defined port-forwards using `kubectl port-forward`, and continuously monitors their status with automatic restart capabilities.

## Development Commands

```bash
# Build the application
go build -o bin/kportforward ./cmd/kportforward

# Build for all platforms
./scripts/build.sh

# Run tests
go test ./...

# Run performance benchmarks
go test -bench=. -benchmem ./...

# Performance profiling
./bin/kportforward profile --cpuprofile=cpu.prof --memprofile=mem.prof --duration=60s

# Analyze performance profiles
go tool pprof cpu.prof
go tool pprof mem.prof

# Run with verbose logging for debugging
./bin/kportforward --help

# Install git hooks for automatic formatting
./scripts/install-hooks.sh

# Create a release
./scripts/release.sh v1.0.0
```

## Key Components

### Go Package Structure
- `cmd/kportforward/main.go`: Main application entry point with CLI setup
- `cmd/kportforward/profile.go`: Performance profiling command with CPU/memory analysis
- `internal/config/`: Configuration system with embedded defaults and user merging
  - `config.go`: Configuration loading and merging logic
  - `config_optimized.go`: High-performance configuration loading with caching
  - `config_bench_test.go`: Performance benchmarks for configuration operations
  - `embedded.go`: Embedded default configuration using `//go:embed`
  - `types.go`: Configuration data structures
- `internal/portforward/`: Port-forward management and monitoring
  - `manager.go`: Service manager with UI handler integration
  - `manager_bench_test.go`: Performance benchmarks for manager operations
  - `service.go`: Individual service management
- `internal/ui/`: Modern terminal UI using Bubble Tea framework
  - `tui.go`: Main TUI application and event handling
  - `model.go`: UI state management and updates
  - `styles.go`: Terminal styling and layout
- `internal/ui_handlers/`: gRPC UI and Swagger UI automation
  - `grpc.go`: gRPC UI process management
  - `swagger.go`: Swagger UI Docker container management
  - Platform-specific implementations (`*_unix.go`, `*_windows.go`)
- `internal/updater/`: Auto-update system with GitHub releases integration
- `internal/utils/`: Cross-platform utilities for ports, processes, and logging
  - `ports_optimized.go`: High-performance port management with caching and pooling
  - `ports_bench_test.go`: Performance benchmarks for port operations

### Build and Deployment
- `scripts/build.sh`: Cross-platform build script (darwin/amd64, darwin/arm64, linux/amd64, windows/amd64)
- `scripts/release.sh`: Automated release creation with GitHub CLI
- `scripts/install-hooks.sh`: Git pre-commit hooks for automatic Go formatting
- `.github/workflows/build.yml`: CI/CD for automated builds and tests on push/PR
- `.github/workflows/release.yml`: Automated release workflow for tagged versions

## Usage Commands

```bash
# Display help information
./bin/kportforward --help

# Basic usage with embedded configuration
./bin/kportforward

# With gRPC UI support for RPC services
./bin/kportforward --grpcui

# With Swagger UI support for REST services
./bin/kportforward --swaggerui

# With both gRPC UI and Swagger UI support
./bin/kportforward --grpcui --swaggerui

# With log file output
./bin/kportforward --log-file /path/to/logfile.log

# Performance profiling
./bin/kportforward profile --cpuprofile=cpu.prof --memprofile=mem.prof --duration=30s

# Check version information
./bin/kportforward version
```

## Dependencies

### Build Dependencies
- Go 1.21+
- Git (for version information in builds)

### Runtime Dependencies
- `kubectl`: Kubernetes CLI for managing clusters
  ```bash
  brew install kubectl
  ```

### Optional Dependencies
- `grpcui`: For gRPC web interfaces (when using `--grpcui`)
  ```bash
  go install github.com/fullstorydev/grpcui/cmd/grpcui@latest
  ```

- `docker`: Required for Swagger UI (when using `--swaggerui`)
  ```bash
  # Install Docker Desktop from https://www.docker.com/
  ```

### Development Dependencies
- GitHub CLI (`gh`) for releases: `brew install gh`

## Architecture

The application uses modern Go patterns and frameworks:

### Core Design Patterns
- **Embedded Configuration**: Default services embedded at compile-time using `//go:embed`
- **Additive User Config**: User configuration at `~/.config/kportforward/config.yaml` merges with defaults
- **High-Performance Caching**: TTL-based caching with optimized data structures for 4,200x faster config loading
- **Object Pooling**: Memory optimization with sync.Pool for reduced garbage collection
- **Interface-Based UI Handlers**: `UIHandler` interface allows pluggable UI management systems
- **Channel-Based Communication**: Status updates flow through channels to the TUI
- **Context-Aware Shutdown**: Graceful shutdown using `context.Context`
- **Cross-Platform Process Management**: Platform-specific implementations using build tags
- **Performance Monitoring**: Built-in profiling and benchmarking capabilities

### Key Libraries
- **Bubble Tea**: Modern TUI framework for reactive terminal interfaces
- **Lipgloss**: Terminal styling and layout
- **Cobra**: CLI framework for commands and flags
- **YAML v3**: Configuration parsing and merging

### UI Handler System
- **gRPC UI**: Spawns and manages `grpcui` processes for RPC services with intelligent connection testing
- **Swagger UI**: Manages Docker containers running Swagger UI for REST services with intelligent connection testing
- **Automatic Lifecycle**: UI handlers start/stop automatically based on service status
- **Health Monitoring**: Continuous monitoring with restart capabilities for both processes and containers
- **Connection Testing**: Pre-flight TCP checks ensure services are accessible before starting UI components
- **Retry Logic**: Failed UI starts are retried automatically through monitoring loops
- **Smart URL Generation**: Only displays clickable URLs for services that are actually accessible
- **Container Health**: Docker container readiness verification for Swagger UI services

## Configuration

### Embedded Default Configuration
The application includes 18 pre-configured services embedded at compile-time. These cover common Kubernetes services and can be found in `internal/config/default.yaml`.

### User Configuration Override
Users can create `~/.config/kportforward/config.yaml` to add services or override defaults:

```yaml
portForwards:
  my-service:
    target: "service/my-service"
    targetPort: 8080
    localPort: 9080
    namespace: "default"
    type: "rest"
    swaggerPath: "docs/swagger"
    apiPath: "api/v1"
monitoringInterval: 5s
uiOptions:
  refreshRate: 1s
  theme: "dark"
```

### Configuration Fields
- `target`: Kubernetes resource (e.g., `service/name`, `deployment/name`)
- `targetPort`: Port on the target resource
- `localPort`: Local machine port for forwarding
- `namespace`: Kubernetes namespace
- `type`: Service type (`web`, `rest`, `rpc`) for UI automation
- `swaggerPath`: Path to Swagger documentation (REST services)
- `apiPath`: Base API path (REST services)

## Key Features

### Core Functionality
- **Cross-Platform**: Works on macOS, Linux, and Windows
- **Modern Terminal UI**: Interactive interface with real-time updates and keyboard navigation
- **Automatic Recovery**: Monitors and restarts failed port-forwards with exponential backoff
- **Embedded Configuration**: 18 pre-configured services with user override capability
- **Auto-Updates**: Daily update checks with in-UI notifications

### Advanced Features
- **UI Integration**: Automated gRPC UI and Swagger UI for API services
- **Context Awareness**: Detects Kubernetes context changes and restarts services
- **High-Performance Port Management**: Optimized port conflict resolution (600x faster) with intelligent caching
- **Performance Profiling**: Built-in CPU and memory profiling with `profile` command
- **Log File Support**: Configurable log output to files with `--log-file` flag
- **Optimized Algorithms**: Smart caching, object pooling, and concurrent processing
- **Interactive Sorting**: Sort services by name, status, type, port, or uptime
- **Detail Views**: Expandable service details with error information
- **Graceful Shutdown**: Clean process termination with proper cleanup

## Development Workflow

### Adding New Features
1. **Write Tests First**: Add tests to appropriate `*_test.go` files
2. **Implement Feature**: Follow existing patterns and interfaces
3. **Format Code**: Git hooks automatically run `gofmt -s -w .`
4. **Run Tests**: `go test ./...` must pass
5. **Build and Test**: `go build` and manual testing

### Code Quality
- **Git Hooks**: Pre-commit hooks ensure Go formatting
- **Interface Design**: Use interfaces for testability and modularity
- **Error Handling**: Comprehensive error handling with proper logging
- **Cross-Platform**: Use build tags for platform-specific code

### Testing Strategy
- **Unit Tests**: Core logic tested with mocks and fakes
- **Performance Benchmarks**: Comprehensive benchmark suite measuring critical operations
- **Integration Tests**: UI handler interfaces tested with mock implementations
- **CI Testing**: GitHub Actions run tests on multiple platforms
- **Manual Testing**: Real Kubernetes cluster testing for validation
- **Performance Testing**: CPU and memory profiling for large service counts (100+ services)

## Testing

### Running Tests
```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test ./... -v

# Run tests for specific package
go test ./internal/config -v

# Run tests with coverage
go test ./... -cover

# Run performance benchmarks
go test -bench=. -benchmem ./...

# Run specific benchmarks
go test -bench=BenchmarkLoadConfig -benchmem ./internal/config
go test -bench=BenchmarkPortOperations -benchmem ./internal/utils
```

### Test Coverage
- **Config Package**: Configuration loading, validation, merging, performance benchmarks
- **Utils Package**: Port management, logging, cross-platform utilities, optimized algorithms
- **Portforward Package**: Manager lifecycle, UI handler integration, concurrent operations
- **UI Handlers Package**: gRPC UI and Swagger UI functionality
- **Performance Package**: Benchmarking, profiling, optimization validation

## Build and Release

### Local Development
```bash
# Build for current platform
go build -o bin/kportforward ./cmd/kportforward

# Build for all platforms
./scripts/build.sh
```

### Release Process

**IMPORTANT**: Always check existing releases before creating a new one to determine the correct version number.

```bash
# 1. Check existing releases first
gh release list
# OR
git tag --sort=-version:refname | head -5

# 2. Determine next version number (semantic versioning)
# - Patch: v1.2.1 (bug fixes)
# - Minor: v1.3.0 (new features)
# - Major: v2.0.0 (breaking changes)

# 3. Create new release with correct version
./scripts/release.sh v1.2.1

# 4. Push tag and create GitHub release
git push origin v1.2.1
gh release create v1.2.1 --title "Title" --notes "Release notes" dist/*

# GitHub Actions automatically:
# - Builds for all platforms
# - Runs tests
# - Creates GitHub release
# - Uploads binaries
```

## Troubleshooting

### Common Issues
- **Build Failures**: Check Go version (requires 1.21+)
- **Missing kubectl**: Install with `brew install kubectl`
- **gRPC UI not working**: Install with `go install github.com/fullstorydev/grpcui/cmd/grpcui@latest`
- **gRPC UI links not appearing**: Service must be running and accessible; gRPC UI only shows URLs for connected services
- **gRPC UI "site can't be reached"**: Fixed in latest version - URLs only appear when services are actually accessible
- **Swagger UI failures**: Ensure Docker Desktop is running
- **Swagger UI links not appearing**: Service must be running and accessible; Swagger UI only shows URLs for connected services
- **Swagger UI "site can't be reached"**: Fixed in latest version - URLs only appear when containers are actually running
- **Port conflicts**: Application automatically resolves these
- **Context issues**: Verify with `kubectl config current-context`

### Debugging
- **Verbose Logging**: Check logger initialization in `main.go`
- **Log File Debugging**: Use `--log-file /tmp/debug.log` to capture detailed logs
- **gRPC UI Debug**: Look for "TCP connection test" and "Starting gRPC UI" messages in logs
- **Swagger UI Debug**: Look for "TCP connection test" and "Starting Swagger UI" messages in logs
- **UI Handler Logs**: gRPC UI logs in `/tmp/kpf_grpcui_*.log`
- **Container Issues**: Check Docker container status: `docker ps | grep kpf-swagger`
- **Connection Issues**: Check if port-forwards are working: `kubectl port-forward -n <namespace> <service> <port>`
- **Service Accessibility**: Verify services are accessible and support their respective protocols
- **Process Issues**: Use platform-specific process utilities in `utils/`
- **Configuration Issues**: Verify embedded config loading in `config/`
- **Performance Issues**: Use `kportforward profile` for CPU/memory analysis
- **Benchmark Failures**: Run `go test -bench=. -benchmem ./...` to verify optimizations

### Testing kportforward Service

When testing kportforward functionality, especially after configuration changes:

#### Quick Service Test
```bash
# Test for 30 seconds with logging (background mode)
timeout 30s ./bin/kportforward --log-file /tmp/test.log || echo "Test completed"

# Check for restart failures (should be 0)
grep -c "Restarting failed service" /tmp/test.log

# View startup logs
head -30 /tmp/test.log

# View any errors
grep -i "error\|failed" /tmp/test.log
```

#### Manual Port-Forward Testing
```bash
# Test individual service connectivity (using services from embedded config)
kubectl port-forward -n <namespace> service/<service-name> <local-port>:<target-port> &
sleep 3
nc -zv localhost <local-port>  # Should succeed
pkill -f "kubectl port-forward"

# Example with default embedded services
kubectl port-forward -n flyte service/flyteconsole 8088:80 &
sleep 3
nc -zv localhost 8088
pkill -f "kubectl port-forward"
```

#### Configuration Validation
```bash
# Verify Kubernetes connectivity
kubectl config current-context
kubectl get nodes

# Check if embedded services exist in your cluster
kubectl get services -A | grep -E "(flyteconsole|flyteadmin)"

# Verify specific service ports match embedded config
kubectl get service <service-name> -n <namespace> -o jsonpath='{.spec.ports[0]}'
```

#### Common Test Scenarios
- **After Config Changes**: Rebuild and run 30-second test to ensure no restart loops
- **Port Conflicts**: Start multiple instances to test port resolution  
- **Grace Period**: Services should not restart within first 5 seconds of startup
- **Health Checks**: TCP connectivity should work for all configured ports

#### Expected Behavior
- Embedded services start successfully if their Kubernetes resources exist in the cluster
- No "Restarting failed service" messages after grace period for available services
- Services maintain "Running" status throughout test duration for functional resources
- Clean shutdown with all services stopped properly
- Services that don't exist in your cluster will show as "Failed" - this is expected behavior

#### gRPC UI Testing
```bash
# Test gRPC UI functionality
./bin/kportforward --grpcui --log-file /tmp/grpc-test.log

# Check gRPC UI startup messages
grep -i "grpc" /tmp/grpc-test.log

# Look for connection testing
grep "TCP connection test" /tmp/grpc-test.log

# Check if gRPC UI processes are running
ps aux | grep grpcui

# Test gRPC UI accessibility manually
# (after identifying gRPC UI port from logs)
curl -I http://localhost:<grpcui-port>
```

#### gRPC UI Expected Behavior
- gRPC UI URLs only appear for RPC services that are running and accessible
- TCP connection tests pass before gRPC UI startup attempts
- gRPC UI processes start only for services with working port-forwards
- No "site can't be reached" errors for displayed gRPC UI links
- gRPC UI logs show successful connection to target services

#### Swagger UI Testing
```bash
# Test Swagger UI functionality
./bin/kportforward --swaggerui --log-file /tmp/swagger-test.log

# Check Swagger UI startup messages
grep -i "swagger" /tmp/swagger-test.log

# Look for connection testing
grep "TCP connection test" /tmp/swagger-test.log

# Check if Swagger UI containers are running
docker ps | grep kpf-swagger

# Test Swagger UI accessibility manually
# (after identifying Swagger UI port from logs)
curl -I http://localhost:<swagger-port>
```

#### Swagger UI Expected Behavior
- Swagger UI URLs only appear for REST services that are running and accessible
- TCP connection tests pass before container startup attempts
- Docker containers start only for services with working port-forwards
- No "site can't be reached" errors for displayed Swagger UI links
- Container logs show successful startup and accessibility

### Development Tips
- **Use Git Hooks**: Run `./scripts/install-hooks.sh` for automatic formatting
- **Test Early**: Write tests before implementing features
- **Follow Interfaces**: Use `UIHandler` pattern for new UI integrations
- **Cross-Platform**: Test on different operating systems when possible
- **Error Handling**: Always handle errors gracefully with proper logging