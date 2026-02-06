package config

import (
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// ConfigCache provides cached configuration loading
type ConfigCache struct {
	config   *Config
	loadTime time.Time
	mutex    sync.RWMutex
	ttl      time.Duration
}

// OptimizedConfigLoader provides optimized configuration loading with caching
type OptimizedConfigLoader struct {
	cache           *ConfigCache
	parsedDefault   *Config
	parseOnce       sync.Once
	userConfigPath  string
	userConfigCache *Config
	userConfigTime  time.Time
	userConfigMutex sync.RWMutex
}

// NewOptimizedConfigLoader creates a new optimized config loader
func NewOptimizedConfigLoader() *OptimizedConfigLoader {
	return &OptimizedConfigLoader{
		cache: &ConfigCache{
			ttl: 30 * time.Second, // Cache for 30 seconds
		},
	}
}

// LoadConfigOptimized loads configuration with caching and optimization
func (ocl *OptimizedConfigLoader) LoadConfigOptimized() (*Config, error) {
	ocl.cache.mutex.RLock()
	if ocl.cache.config != nil && time.Since(ocl.cache.loadTime) < ocl.cache.ttl {
		config := ocl.cache.config
		ocl.cache.mutex.RUnlock()
		return config, nil
	}
	ocl.cache.mutex.RUnlock()

	// Need to load/reload config
	ocl.cache.mutex.Lock()
	defer ocl.cache.mutex.Unlock()

	// Double-check pattern
	if ocl.cache.config != nil && time.Since(ocl.cache.loadTime) < ocl.cache.ttl {
		return ocl.cache.config, nil
	}

	// Parse default config once
	defaultConfig, err := ocl.getDefaultConfigOptimized()
	if err != nil {
		return nil, err
	}

	// Try to load and merge user config
	userConfig, err := ocl.getUserConfigOptimized()
	if err != nil {
		// Return default config if user config fails
		ocl.cache.config = defaultConfig
		ocl.cache.loadTime = time.Now()
		return defaultConfig, nil
	}

	// Merge configs
	merged := ocl.mergeConfigsOptimized(defaultConfig, userConfig)

	ocl.cache.config = merged
	ocl.cache.loadTime = time.Now()

	return merged, nil
}

// getDefaultConfigOptimized parses the default config once and caches it
func (ocl *OptimizedConfigLoader) getDefaultConfigOptimized() (*Config, error) {
	var err error
	ocl.parseOnce.Do(func() {
		ocl.parsedDefault = &Config{}
		err = yaml.Unmarshal(DefaultConfigYAML, ocl.parsedDefault)
	})

	if err != nil {
		return nil, err
	}

	// Return a copy to prevent mutations
	return ocl.copyConfig(ocl.parsedDefault), nil
}

// getUserConfigOptimized loads user config with caching
func (ocl *OptimizedConfigLoader) getUserConfigOptimized() (*Config, error) {
	userConfigPath, err := getUserConfigPath()
	if err != nil {
		return nil, err
	}

	// Check if we need to reload user config
	ocl.userConfigMutex.RLock()
	if ocl.userConfigPath == userConfigPath &&
		ocl.userConfigCache != nil &&
		time.Since(ocl.userConfigTime) < 10*time.Second {
		config := ocl.userConfigCache
		ocl.userConfigMutex.RUnlock()
		return ocl.copyConfig(config), nil
	}
	ocl.userConfigMutex.RUnlock()

	// Load user config
	userConfig, err := loadUserConfig(userConfigPath)
	if err != nil {
		return nil, err
	}

	// Cache the result
	ocl.userConfigMutex.Lock()
	ocl.userConfigPath = userConfigPath
	ocl.userConfigCache = userConfig
	ocl.userConfigTime = time.Now()
	ocl.userConfigMutex.Unlock()

	return ocl.copyConfig(userConfig), nil
}

// mergeConfigsOptimized merges configs with optimized memory allocation
func (ocl *OptimizedConfigLoader) mergeConfigsOptimized(defaultConfig, userConfig *Config) *Config {
	// Pre-allocate merged config
	totalServices := len(defaultConfig.PortForwards)
	if userConfig.PortForwards != nil {
		totalServices += len(userConfig.PortForwards)
	}

	merged := &Config{
		PortForwards:       make(map[string]Service, totalServices),
		MonitoringInterval: defaultConfig.MonitoringInterval,
		UIOptions:          defaultConfig.UIOptions,
	}

	// Copy default port forwards
	for name, service := range defaultConfig.PortForwards {
		merged.PortForwards[name] = service
	}

	// Add/override with user port forwards
	if userConfig.PortForwards != nil {
		for name, service := range userConfig.PortForwards {
			merged.PortForwards[name] = service
		}
	}

	// Override settings if specified by user
	if userConfig.MonitoringInterval != 0 {
		merged.MonitoringInterval = userConfig.MonitoringInterval
	}

	if userConfig.UIOptions.RefreshRate != 0 {
		merged.UIOptions.RefreshRate = userConfig.UIOptions.RefreshRate
	}
	if userConfig.UIOptions.Theme != "" {
		merged.UIOptions.Theme = userConfig.UIOptions.Theme
	}

	for name, service := range merged.PortForwards {
		if service.Disabled {
			delete(merged.PortForwards, name)
		}
	}

	return merged
}

// copyConfig creates a deep copy of the config to prevent mutations
func (ocl *OptimizedConfigLoader) copyConfig(original *Config) *Config {
	if original == nil {
		return nil
	}

	copy := &Config{
		PortForwards:       make(map[string]Service, len(original.PortForwards)),
		MonitoringInterval: original.MonitoringInterval,
		UIOptions:          original.UIOptions,
	}

	for name, service := range original.PortForwards {
		copy.PortForwards[name] = service
	}

	return copy
}

// InvalidateCache clears the configuration cache
func (ocl *OptimizedConfigLoader) InvalidateCache() {
	ocl.cache.mutex.Lock()
	defer ocl.cache.mutex.Unlock()

	ocl.cache.config = nil
	ocl.cache.loadTime = time.Time{}
}

// Global optimized loader instance
var optimizedLoader = NewOptimizedConfigLoader()

// LoadConfigFast is a drop-in replacement for LoadConfig with optimizations
func LoadConfigFast() (*Config, error) {
	return optimizedLoader.LoadConfigOptimized()
}

// ConfigStats provides statistics about configuration loading
type ConfigStats struct {
	LoadCount       int64
	CacheHits       int64
	CacheMisses     int64
	LastLoadTime    time.Duration
	AverageLoadTime time.Duration
}

// OptimizedConfigWithStats provides config loading with performance statistics
type OptimizedConfigWithStats struct {
	*OptimizedConfigLoader
	stats      ConfigStats
	statsMutex sync.RWMutex
}

// NewOptimizedConfigWithStats creates a config loader with statistics
func NewOptimizedConfigWithStats() *OptimizedConfigWithStats {
	return &OptimizedConfigWithStats{
		OptimizedConfigLoader: NewOptimizedConfigLoader(),
	}
}

// LoadConfigWithStats loads config and tracks performance statistics
func (ocs *OptimizedConfigWithStats) LoadConfigWithStats() (*Config, ConfigStats, error) {
	startTime := time.Now()

	// Check cache first
	ocs.cache.mutex.RLock()
	cacheHit := ocs.cache.config != nil && time.Since(ocs.cache.loadTime) < ocs.cache.ttl
	ocs.cache.mutex.RUnlock()

	config, err := ocs.LoadConfigOptimized()
	loadTime := time.Since(startTime)

	// Update statistics
	ocs.statsMutex.Lock()
	ocs.stats.LoadCount++
	if cacheHit {
		ocs.stats.CacheHits++
	} else {
		ocs.stats.CacheMisses++
	}
	ocs.stats.LastLoadTime = loadTime
	ocs.stats.AverageLoadTime = time.Duration(
		(int64(ocs.stats.AverageLoadTime) + int64(loadTime)) / 2,
	)
	stats := ocs.stats
	ocs.statsMutex.Unlock()

	return config, stats, err
}

// GetStats returns current performance statistics
func (ocs *OptimizedConfigWithStats) GetStats() ConfigStats {
	ocs.statsMutex.RLock()
	defer ocs.statsMutex.RUnlock()
	return ocs.stats
}
