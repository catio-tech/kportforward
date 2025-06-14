# kportforward

A modern, cross-platform Kubernetes port-forward manager with a rich terminal UI, automatic recovery, and built-in update system.

[![Build Status](https://github.com/catio-tech/kportforward/workflows/Build%20and%20Test/badge.svg)](https://github.com/catio-tech/kportforward/actions)
[![Release](https://img.shields.io/github/v/release/catio-tech/kportforward)](https://github.com/catio-tech/kportforward/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/catio-tech/kportforward)](https://go.dev/)

## ✨ Features

- **🎨 Modern Terminal UI**: Interactive interface with real-time updates and keyboard navigation
- **🔄 Automatic Recovery**: Monitors and automatically restarts failed port-forwards
- **🌐 Cross-Platform**: Works on macOS, Linux, and Windows
- **📊 Smart Monitoring**: Health checks with exponential backoff for frequently failing services
- **🆙 Auto-Updates**: Daily update checks with in-UI notifications
- **🎯 UI Integration**: Automated gRPC UI and Swagger UI for API services
- **⚙️ Embedded Config**: Pre-configured services with user override support
- **🚀 High Performance**: Optimized for managing 100+ concurrent port-forwards with 4,200x faster config loading
- **📊 Performance Profiling**: Built-in CPU and memory profiling tools for performance analysis

## 📥 Installation

### Quick Install

Download the latest release for your platform:

```bash
# macOS (Intel)
curl -L https://github.com/catio-tech/kportforward/releases/latest/download/kportforward-darwin-amd64 -o kportforward
chmod +x kportforward
sudo mv kportforward /usr/local/bin/

# macOS (Apple Silicon)
curl -L https://github.com/catio-tech/kportforward/releases/latest/download/kportforward-darwin-arm64 -o kportforward
chmod +x kportforward
sudo mv kportforward /usr/local/bin/

# Linux
curl -L https://github.com/catio-tech/kportforward/releases/latest/download/kportforward-linux-amd64 -o kportforward
chmod +x kportforward
sudo mv kportforward /usr/local/bin/

# Windows (PowerShell)
Invoke-WebRequest -Uri "https://github.com/catio-tech/kportforward/releases/latest/download/kportforward-windows-amd64.exe" -OutFile "kportforward.exe"
```

### Manual Installation

1. Go to the [Releases page](https://github.com/catio-tech/kportforward/releases)
2. Download the appropriate binary for your platform
3. Make it executable and place it in your PATH

## 🚀 Quick Start

1. **Start the application**:
   ```bash
   kportforward
   ```

2. **Use the interactive interface**:
   - `↑↓` or `j/k` - Navigate services
   - `Enter` - View service details
   - `n/s/t/p/u` - Sort by Name/Status/Type/Port/Uptime
   - `r` - Reverse sort order
   - `q` - Quit

3. **With UI integrations**:
   ```bash
   # Enable gRPC UI for RPC services
   kportforward --grpcui
   
   # Enable Swagger UI for REST services  
   kportforward --swaggerui
   
   # Enable both
   kportforward --grpcui --swaggerui
   ```

4. **With log file output**:
   ```bash
   # Write logs to a file instead of stdout
   kportforward --log-file /path/to/logfile.log
   
   # Combine with UI features
   kportforward --grpcui --swaggerui --log-file /var/log/kportforward.log
   ```

## ⚙️ Configuration

kportforward uses embedded configuration for immediate functionality, with support for user customizations.

### User Configuration

Create `~/.config/kportforward/config.yaml` (Unix) or `%APPDATA%/kportforward/config.yaml` (Windows):

```yaml
# Add your own services (merged with embedded config)
portForwards:
  my-service:
    target: "service/my-service"
    targetPort: 80
    localPort: 8080
    namespace: "default"
    type: "web"

# Override default settings
monitoringInterval: 2s
uiOptions:
  refreshRate: 500ms
  theme: "dark"
```

### Service Types

- **`rest`**: REST APIs (enables Swagger UI with `--swaggerui`)
- **`rpc`**: gRPC services (enables gRPC UI with `--grpcui`)  
- **`web`**: Web applications
- **`other`**: Other services

## 🎯 UI Integrations

### gRPC UI
Automatically launches web interfaces for gRPC services:
```bash
kportforward --grpcui
```
- Requires: `go install github.com/fullstorydev/grpcui/cmd/grpcui@latest`
- Accessible at: `http://localhost:<auto-assigned-port>`

### Swagger UI
Automatically launches Swagger UI for REST APIs:
```bash
kportforward --swaggerui
```
- Requires: Docker Desktop
- Accessible at: `http://localhost:<auto-assigned-port>`

## 🛠️ Development

### Prerequisites

- Go 1.21+
- kubectl (configured for your cluster)

### Building

```bash
# Build for current platform
go build -o bin/kportforward ./cmd/kportforward

# Build for all platforms
./scripts/build.sh

# Create a release
./scripts/release.sh v1.0.0
```

### Performance Testing

```bash
# Run benchmarks
go test -bench=. -benchmem ./...

# Profile CPU usage
./kportforward profile --cpuprofile=cpu.prof --duration=30s

# Profile memory usage
./kportforward profile --memprofile=mem.prof --duration=30s

# Analyze profiles
go tool pprof cpu.prof
go tool pprof mem.prof
```

### Testing

```bash
# Run tests
go test ./...

# Run benchmarks
go test -bench=. -benchmem ./...

# Run with verbose logging
go run ./cmd/kportforward -v
```

### Git Hooks

Install pre-commit hooks to automatically format Go code:

```bash
# Install git hooks
./scripts/install-hooks.sh
```

The pre-commit hook will:
- Automatically format Go code with `gofmt -s -w` before each commit
- Add formatted files back to staging
- Abort the commit if files were formatted (so you can review changes)

To bypass the hook for a specific commit (not recommended):
```bash
git commit --no-verify
```

## 📱 Terminal UI

```
┌─ kportforward v1.0.0 ─ Context: my-cluster ─ Services (18/18 running) ──┐
│                                                                           │
│ Services (18/18 running)  [↑↓] Navigate [Enter] Details [q] Quit         │
│                                                                           │
│ Name                 Status    URL                      Type    Uptime    │
│ ─────────────────────────────────────────────────────────────────────── │
│ ● flyte-console      Running   http://localhost:8088    web     2h3m      │
│ ● flyte-admin-rpc    Running   http://localhost:8089    rpc     2h3m      │
│ ● api-gateway        Running   http://localhost:8080    rest    1h45m     │
│ ● process-monitor    Failed    -                        rpc     0s        │
│ ...                                                                       │
│                                                                           │
│ Last Error: process-monitor: connection refused                           │
│ [n/s/t/p/u] Sort by Name/Status/Type/Port/Uptime  [r] Reverse           │
└───────────────────────────────────────────────────────────────────────────┘
```

## 🔧 Troubleshooting

### Common Issues

**Port conflicts**: kportforward automatically finds available ports when configured ports are in use.

**gRPC UI not starting**:
- Install grpcui: `go install github.com/fullstorydev/grpcui/cmd/grpcui@latest`
- Check logs in `/tmp/kpf_grpcui_*.log`

**Swagger UI not starting**:
- Ensure Docker is running
- Check that REST services expose Swagger documentation

**Services frequently restarting**:
- Services enter cooldown mode with exponential backoff
- Check Kubernetes context: `kubectl config current-context`
- Verify service exists: `kubectl get svc -n <namespace>`

### Debug Mode

```bash
# Check application version
kportforward version

# View help and available commands
kportforward --help

# Performance profiling
kportforward profile --cpuprofile=cpu.prof --memprofile=mem.prof --duration=60s

# Debug with log file
kportforward --log-file /tmp/debug.log

# Check service status manually
kubectl port-forward -n <namespace> <service> <port>:<port>

# Verify kubectl context
kubectl config current-context
```

## 📊 Performance

kportforward is optimized for high performance with enterprise-scale deployments:

### Performance Benchmarks
- **Config Loading**: 4,200x faster with caching (126ms → 30ns)
- **Port Conflict Resolution**: 600x faster with object pooling
- **Port Availability Checks**: 1,280x faster with caching
- **Application Startup**: 10x faster overall
- **Memory Usage**: 93% reduction in allocations

### Performance Features
- **Intelligent Caching**: TTL-based caching for expensive operations
- **Object Pooling**: Reduced garbage collection pressure
- **Optimized Data Structures**: Pre-allocated maps and concurrent-safe operations
- **Smart Algorithms**: Optimized port finding and conflict resolution

### Performance Monitoring
```bash
# Run performance profiling
kportforward profile --duration=60s --cpuprofile=cpu.prof --memprofile=mem.prof

# Analyze CPU usage
go tool pprof cpu.prof

# Analyze memory usage
go tool pprof mem.prof

# Run benchmarks during development
go test -bench=. -benchmem ./...
```

For detailed performance analysis, see [PERFORMANCE_REPORT.md](PERFORMANCE_REPORT.md).

## 💡 Examples

### Basic Usage
```bash
# Start with embedded services
kportforward

# Use arrow keys to navigate, Enter for details, q to quit
```

### Custom Configuration
Create `~/.config/kportforward/config.yaml`:
```yaml
portForwards:
  # Add your microservice
  user-service:
    target: "service/user-service"
    targetPort: 8080
    localPort: 3001
    namespace: "production"
    type: "rest"
    swaggerPath: "docs/swagger.json"
    
  # Add your gRPC service  
  notification-service:
    target: "deployment/notification-service"
    targetPort: 9090
    localPort: 3002
    namespace: "production"
    type: "rpc"

# Override defaults
monitoringInterval: 3s
uiOptions:
  refreshRate: 1s
```

Then run with UI integrations:
```bash
kportforward --grpcui --swaggerui
```

### Advanced Workflows
```bash
# Different contexts with logging
kubectl config use-context staging
kportforward --grpcui --log-file /var/log/staging.log

kubectl config use-context production  
kportforward --swaggerui --log-file /var/log/production.log

# Check what's running
kportforward version  # Shows current version and update status
```

### Development Workflow
```bash
# Developer working on microservices with logging
kportforward --grpcui --swaggerui --log-file ./dev.log

# Production deployment with log rotation
kportforward --log-file /var/log/kportforward.log

# Access services:
# - Embedded services available at configured ports
# - gRPC UI: http://localhost:9090 (auto-assigned)
# - Swagger UI: http://localhost:8080 (auto-assigned)
# - Your APIs accessible through UI tools
# - All activity logged to specified file
```

## 📋 Documentation

- **[README.md](README.md)**: This file - project overview and usage
- **[CLAUDE.md](CLAUDE.md)**: Developer guide and architecture documentation  
- **[IMPLEMENTATION_LOG.md](IMPLEMENTATION_LOG.md)**: Development progress and technical decisions
- **[PERFORMANCE_REPORT.md](PERFORMANCE_REPORT.md)**: Detailed performance analysis and benchmarks

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/amazing-feature`
3. Commit your changes: `git commit -m 'Add amazing feature'`
4. Push to the branch: `git push origin feature/amazing-feature`
5. Open a Pull Request

### Performance Considerations

When contributing, please:
- Run benchmarks: `go test -bench=. -benchmem ./...`
- Profile changes with large service counts (100+ services)
- Test memory usage: `go tool pprof mem.prof`
- Ensure optimizations don't break existing functionality

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) for the excellent TUI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss) for terminal styling
- [Cobra](https://github.com/spf13/cobra) for CLI framework