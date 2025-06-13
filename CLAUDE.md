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
- `.github/workflows/`: CI/CD automation for build, test, and release

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
- **gRPC UI**: Spawns and manages `grpcui` processes for RPC services
- **Swagger UI**: Manages Docker containers running Swagger UI for REST services
- **Automatic Lifecycle**: UI handlers start/stop automatically based on service status
- **Health Monitoring**: Continuous monitoring with restart capabilities

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
```bash
# Create new release (requires GitHub CLI)
./scripts/release.sh v1.1.0

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
- **Swagger UI failures**: Ensure Docker Desktop is running
- **Port conflicts**: Application automatically resolves these
- **Context issues**: Verify with `kubectl config current-context`

### Debugging
- **Verbose Logging**: Check logger initialization in `main.go`
- **UI Handler Logs**: gRPC UI logs in `/tmp/kpf_grpcui_*.log`
- **Process Issues**: Use platform-specific process utilities in `utils/`
- **Configuration Issues**: Verify embedded config loading in `config/`
- **Performance Issues**: Use `kportforward profile` for CPU/memory analysis
- **Benchmark Failures**: Run `go test -bench=. -benchmem ./...` to verify optimizations

### Development Tips
- **Use Git Hooks**: Run `./scripts/install-hooks.sh` for automatic formatting
- **Test Early**: Write tests before implementing features
- **Follow Interfaces**: Use `UIHandler` pattern for new UI integrations
- **Cross-Platform**: Test on different operating systems when possible
- **Error Handling**: Always handle errors gracefully with proper logging