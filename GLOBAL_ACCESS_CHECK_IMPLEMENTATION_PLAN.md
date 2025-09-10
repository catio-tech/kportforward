# Global Access Check Implementation Plan

## Overview

This document outlines the implementation plan for adding a global access check system to kportforward to prevent performance degradation and resource buildup when kubectl credentials expire (especially AWS SSO tokens).

## Problem Statement

When AWS SSO tokens expire after system sleep/wake cycles, all 23+ port-forward services fail simultaneously and enter retry loops, causing:
- Accumulation of kubectl processes
- Goroutine leaks from failed restart attempts
- System performance degradation
- Resource waste from continuous failed connections

## Solution Architecture

### Global Access Check System
- Single lightweight kubectl connectivity test
- Exponential backoff with auth-specific cooldowns
- Service suspension/resumption instead of process killing
- Clear separation between auth failures vs network issues

## Implementation Phases

### Phase 1: Core Global Access Check (High Priority)

#### 1.1 Add Global State to Manager
**File**: `internal/portforward/manager.go`
**Estimated Time**: 1 hour

```go
type Manager struct {
    // ... existing fields ...
    
    // Global access state
    globalAccessHealthy    bool
    globalAccessLastCheck  time.Time
    globalAccessFailCount  int
    globalAccessCooldown   time.Time
    globalAccessMutex      sync.RWMutex
}
```

#### 1.2 Implement Global Access Check Function
**File**: `internal/portforward/manager.go`
**Estimated Time**: 2 hours

- Add `checkGlobalAccess()` method with 10s timeout
- Use `kubectl get nodes` as lightweight connectivity test
- Parse stderr for auth vs network failures
- Add comprehensive error classification

#### 1.3 Integrate with Monitoring Loop
**File**: `internal/portforward/manager.go`
**Estimated Time**: 1.5 hours

- Modify `monitorServices()` to check global access first
- Add `checkAndUpdateGlobalAccess()` with smart cooldown logic
- Implement exponential backoff:
  - Auth failures: 5m, 10m, 30m
  - Network issues: 30s, 1m, 2m

**Testing**: Run with expired AWS SSO token, verify cooldown behavior

### Phase 2: Service Suspension System (Medium Priority)

#### 2.1 Add Service Suspension Logic
**File**: `internal/portforward/manager.go`
**Estimated Time**: 2 hours

- Implement `suspendAllServices()` method
- Add `resumeServicesIfNeeded()` method
- Add "Suspended" status type to service states

#### 2.2 Update Service Status Types
**File**: `internal/config/types.go`
**Estimated Time**: 0.5 hours

```go
type ServiceStatus struct {
    // ... existing fields ...
    GlobalStatus string `json:"globalStatus,omitempty"` // "healthy", "auth_failure", "network_failure"
}
```

#### 2.3 Enhanced Service Manager Integration
**File**: `internal/portforward/service.go`
**Estimated Time**: 1.5 hours

- Add `isAuthError()` helper function
- Modify `Start()` method with enhanced error classification
- Prevent individual service cooldowns during global failures

**Testing**: Simulate network failure, verify services suspend/resume correctly

### Phase 3: Process Management Improvements (Medium Priority)

#### 3.1 Add Kubectl Command Timeouts
**File**: `internal/utils/processes_unix.go`
**Estimated Time**: 1 hour

- Create `StartKubectlPortForwardWithTimeout()` function
- Add 30-second default timeout for kubectl commands
- Ensure proper context cancellation

#### 3.2 Enhanced Process Cleanup
**File**: `internal/utils/processes.go`
**Estimated Time**: 1.5 hours

- Modify `KillProcess()` to kill entire process groups
- Add `KillProcessGroup()` helper function
- Ensure no orphaned kubectl processes remain

**Testing**: Start/stop services rapidly, verify no kubectl processes leak

### Phase 4: UI Integration (Low Priority)

#### 4.1 Add Global Status to TUI
**File**: `internal/ui/model.go`
**Estimated Time**: 1 hour

- Add global status field to TUI model
- Update status channel to include global state
- Add `GetGlobalAccessStatus()` method to manager

#### 4.2 Enhanced Header Display
**File**: `internal/ui/tui.go`
**Estimated Time**: 0.5 hours

```go
func (m *Model) renderHeader() string {
    globalStatus := "✅ Connected"
    if !m.globalAccessHealthy {
        globalStatus := "❌ Access Failed"
    }
    return fmt.Sprintf("Kubectl Status: %s | Services: %d", globalStatus, len(m.services))
}
```

#### 4.3 Service Status Indicators
**File**: `internal/ui/styles.go`
**Estimated Time**: 0.5 hours

- Add visual indicators for suspended services
- Different colors for auth vs network failures
- Clear messaging about global vs individual issues

**Testing**: Verify UI clearly shows global status and service suspension

### Phase 5: Testing & Performance Validation

#### 5.1 Unit Tests
**Files**: `*_test.go`
**Estimated Time**: 3 hours

- Test global access check logic
- Test service suspension/resumption
- Test error classification functions
- Mock kubectl failures for testing

#### 5.2 Integration Testing
**Estimated Time**: 2 hours

- Test with expired AWS SSO tokens
- Verify resource cleanup under failure conditions
- Performance testing with 30+ services
- Sleep/wake cycle simulation

#### 5.3 Performance Benchmarks
**File**: `internal/portforward/manager_bench_test.go`
**Estimated Time**: 1 hour

- Benchmark global access check performance
- Memory usage validation
- Process count monitoring

## Implementation Guidelines

### Error Handling Patterns
```go
// Auth error detection
func isAuthError(err error) bool {
    errStr := strings.ToLower(err.Error())
    return strings.Contains(errStr, "unauthorized") ||
           strings.Contains(errStr, "authentication") ||
           strings.Contains(errStr, "token") ||
           strings.Contains(errStr, "credential")
}

// Network error detection  
func isNetworkError(err error) bool {
    errStr := strings.ToLower(err.Error())
    return strings.Contains(errStr, "connection refused") ||
           strings.Contains(errStr, "timeout") ||
           strings.Contains(errStr, "network")
}
```

### Logging Strategy
- Info level: Global access recovery, service resumption
- Warn level: Global access failures, service suspension
- Error level: Unexpected errors, process cleanup failures
- Debug level: Access check details, cooldown logic

### Configuration Options
Consider adding to config:
```yaml
globalAccessCheck:
  enabled: true
  checkInterval: 30s
  authFailureCooldown: [5m, 10m, 30m]
  networkFailureCooldown: [30s, 1m, 2m]
  maxConsecutiveFailures: 3
```

## Testing Scenarios

### 1. AWS SSO Token Expiration
- Start kportforward with valid tokens
- Wait for token expiration or force expiration
- Verify services suspend instead of restart loop
- Refresh tokens and verify automatic resumption

### 2. Network Connectivity Issues
- Simulate network outage
- Verify shorter cooldown periods
- Test recovery behavior

### 3. Mixed Failure Scenarios
- Some services fail individually while global access works
- Verify individual vs global failure handling

### 4. Performance Under Load
- Run with 30+ services
- Monitor resource usage during failures
- Validate no process/goroutine leaks

## Success Criteria

1. **Resource Management**: No kubectl process accumulation during auth failures
2. **Performance**: System remains responsive during credential expiration
3. **Recovery**: Automatic service resumption when access recovers
4. **User Experience**: Clear indication of global vs individual service issues
5. **Reliability**: No service restarts during global access failures

## Rollout Strategy

### Development
1. Implement Phase 1 with feature flag
2. Test with expired credentials
3. Validate resource cleanup

### Testing
1. Unit tests for all new functions
2. Integration tests with real kubectl failures
3. Performance benchmarks

### Deployment
1. Deploy with global access check disabled initially
2. Enable for subset of users
3. Monitor performance metrics
4. Full rollout after validation

## Risk Mitigation

### Backwards Compatibility
- All changes are additive
- Existing behavior preserved when feature disabled
- Graceful degradation if global check fails

### Failure Scenarios
- Global access check timeout → fall back to individual service monitoring
- Suspension logic failure → services continue normal operation
- Resume logic failure → manual restart still works

## Maintenance Considerations

### Monitoring
- Add metrics for global access check success/failure rates
- Monitor cooldown effectiveness
- Track resource usage improvements

### Documentation Updates
- Update CLAUDE.md with new troubleshooting steps
- Add global access check to README
- Document new configuration options

## Estimated Total Implementation Time

- **Phase 1 (Core)**: 4.5 hours
- **Phase 2 (Suspension)**: 4 hours  
- **Phase 3 (Process Mgmt)**: 2.5 hours
- **Phase 4 (UI)**: 2 hours
- **Phase 5 (Testing)**: 6 hours

**Total**: ~19 hours over 3-4 development sessions

## Dependencies

- No external dependencies required
- Uses existing kubectl binary
- Compatible with current Bubble Tea UI framework
- Works with existing configuration system

## Future Enhancements

1. **Configurable Access Checks**: Support different kubectl commands for access validation
2. **Metrics Integration**: Export Prometheus metrics for global access status
3. **Multi-Cluster Support**: Per-cluster global access checking
4. **Smart Recovery**: Predictive token refresh before expiration
5. **Health Dashboard**: Web UI for cluster access status across environments