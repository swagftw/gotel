// Package config provides configuration management for GoTel using Viper
package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for GoTel
type Config struct {
	// Core settings
	OtelEndpoint string `mapstructure:"otel_endpoint"`
	LogLevel     string `mapstructure:"log_level"`
	EnableDebug  bool   `mapstructure:"debug"`
	Environment  string `mapstructure:"environment"`

	// Feature flags
	EnableAsyncMetrics bool `mapstructure:"enable_async_metrics"`
	EnableMetrics      bool `mapstructure:"metrics_enabled"`
	EnableHealthCheck  bool `mapstructure:"health_checks_enabled"`

	// Performance settings
	MetricBufferSize int           `mapstructure:"metric_buffer_size"`
	SendInterval     time.Duration `mapstructure:"send_interval"`
	MinSendInterval  time.Duration `mapstructure:"min_send_interval"`

	// HTTP Client settings
	HTTPTimeout           time.Duration `mapstructure:"http_timeout"`
	RetryCount            int           `mapstructure:"retry_count"`
	RetryWaitTime         time.Duration `mapstructure:"retry_wait_time"`
	RetryMaxWaitTime      time.Duration `mapstructure:"retry_max_wait_time"`
	MaxIdleConnections    int           `mapstructure:"max_idle_connections"`
	MaxConnectionsPerHost int           `mapstructure:"max_connections_per_host"`
	IdleConnectionTimeout time.Duration `mapstructure:"idle_connection_timeout"`
	KeepAlive             time.Duration `mapstructure:"keep_alive"`

	// Application identification
	AppName    string `mapstructure:"app_name"`
	AppVersion string `mapstructure:"app_version"`
	Instance   string `mapstructure:"instance"`

	// OTEL specific settings
	OtelHeaders map[string]string `mapstructure:"otel_headers"`
	Insecure    bool              `mapstructure:"insecure"`
}

// Default returns a configuration with sensible defaults
func Default() *Config {
	return &Config{
		// Core settings
		OtelEndpoint: "localhost:4318",
		LogLevel:     "info",
		EnableDebug:  false,
		Environment:  "development",

		// Feature flags
		EnableAsyncMetrics: true,
		EnableMetrics:      true,
		EnableHealthCheck:  true,

		// Performance settings
		MetricBufferSize: 100000,
		SendInterval:     30 * time.Second,
		MinSendInterval:  10 * time.Millisecond, // Faster default for counter accuracy

		// HTTP Client settings
		HTTPTimeout:           30 * time.Second,
		RetryCount:            3,
		RetryWaitTime:         1 * time.Second,
		RetryMaxWaitTime:      10 * time.Second,
		MaxIdleConnections:    100,
		MaxConnectionsPerHost: 10,
		IdleConnectionTimeout: 90 * time.Second,
		KeepAlive:             30 * time.Second,

		// Application identification
		AppName:    "gotel-app",
		AppVersion: "1.0.0",
		Instance:   "default",

		// OTEL specific settings
		OtelHeaders: make(map[string]string),
		Insecure:    true, // Default to insecure for local development
	}
}

// FromEnv creates configuration from environment variables using Viper
func FromEnv() *Config {
	return FromEnvWithPrefix("GOTEL")
}

// FromEnvWithPrefix creates configuration from environment variables with a custom prefix
func FromEnvWithPrefix(prefix string) *Config {
	v := viper.New()

	// Set defaults
	cfg := Default()
	setDefaults(v, cfg)

	// Configure Viper
	v.SetEnvPrefix(prefix)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Also check for common environment variables without prefix
	v.BindEnv("otel_endpoint", "OTEL_ENDPOINT", "OTEL_EXPORTER_OTLP_ENDPOINT")
	v.BindEnv("log_level", "LOG_LEVEL")
	v.BindEnv("debug", "DEBUG")
	v.BindEnv("environment", "ENVIRONMENT")
	v.BindEnv("insecure", "OTEL_EXPORTER_OTLP_INSECURE")

	// Unmarshal into config struct
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		// If unmarshaling fails, return defaults with a warning
		fmt.Printf("Warning: Failed to unmarshal configuration: %v. Using defaults.\n", err)
		return cfg
	}

	return &config
}

// FromFile loads configuration from a file
func FromFile(filename string) (*Config, error) {
	v := viper.New()

	// Set defaults
	cfg := Default()
	setDefaults(v, cfg)

	// Configure file reading
	v.SetConfigFile(filename)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Read configuration file
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Unmarshal into config struct
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	return &config, nil
}

// setDefaults sets default values in Viper
func setDefaults(v *viper.Viper, cfg *Config) {
	v.SetDefault("otel_endpoint", cfg.OtelEndpoint)
	v.SetDefault("log_level", cfg.LogLevel)
	v.SetDefault("debug", cfg.EnableDebug)
	v.SetDefault("environment", cfg.Environment)

	v.SetDefault("enable_async_metrics", cfg.EnableAsyncMetrics)
	v.SetDefault("metrics_enabled", cfg.EnableMetrics)
	v.SetDefault("health_checks_enabled", cfg.EnableHealthCheck)

	v.SetDefault("metric_buffer_size", cfg.MetricBufferSize)
	v.SetDefault("send_interval", cfg.SendInterval)
	v.SetDefault("min_send_interval", cfg.MinSendInterval)

	v.SetDefault("http_timeout", cfg.HTTPTimeout)
	v.SetDefault("retry_count", cfg.RetryCount)
	v.SetDefault("retry_wait_time", cfg.RetryWaitTime)
	v.SetDefault("retry_max_wait_time", cfg.RetryMaxWaitTime)
	v.SetDefault("max_idle_connections", cfg.MaxIdleConnections)
	v.SetDefault("max_connections_per_host", cfg.MaxConnectionsPerHost)
	v.SetDefault("idle_connection_timeout", cfg.IdleConnectionTimeout)
	v.SetDefault("keep_alive", cfg.KeepAlive)

	v.SetDefault("app_name", cfg.AppName)
	v.SetDefault("app_version", cfg.AppVersion)
	v.SetDefault("instance", cfg.Instance)

	v.SetDefault("insecure", cfg.Insecure)
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.OtelEndpoint == "" {
		return fmt.Errorf("otel_endpoint is required")
	}

	if c.MetricBufferSize <= 0 {
		return fmt.Errorf("metric_buffer_size must be positive")
	}

	if c.RetryCount < 0 {
		return fmt.Errorf("retry_count cannot be negative")
	}

	if c.MaxIdleConnections <= 0 {
		return fmt.Errorf("max_idle_connections must be positive")
	}

	if c.MaxConnectionsPerHost <= 0 {
		return fmt.Errorf("max_connections_per_host must be positive")
	}

	validLogLevels := map[string]bool{
		"debug": true, "info": true, "warn": true, "error": true,
	}
	if !validLogLevels[strings.ToLower(c.LogLevel)] {
		return fmt.Errorf("invalid log_level: %s (must be debug, info, warn, or error)", c.LogLevel)
	}

	return nil
}

// GetLabels returns common labels based on configuration
func (c *Config) GetLabels() map[string]string {
	return map[string]string{
		"app":         c.AppName,
		"version":     c.AppVersion,
		"instance":    c.Instance,
		"environment": c.Environment,
	}
}
