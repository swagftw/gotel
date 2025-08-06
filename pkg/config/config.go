// Package config provides configuration management for GoTel using Viper
package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all configuration for GoTel
type Config struct {
	// OTEL settings
	OtelEndpoint string `mapstructure:"otel_endpoint"`

	// Application identification
	ServiceName    string `mapstructure:"otel_service_name"`
	ServiceVersion string `mapstructure:"otel_service_version"`
	Environment    string `mapstructure:"env"`

	// Timing settings
	SendInterval int `mapstructure:"otel_send_interval"`

	// Debug and logging
	EnableDebug bool `mapstructure:"otel_debug"`
}

// Default returns a new Config with default values
func Default() *Config {
	return &Config{
		OtelEndpoint:   "localhost:4318",
		ServiceName:    "gotel-app",
		ServiceVersion: "1.0.0",
		Environment:    "local",
		SendInterval:   30,
		EnableDebug:    false,
	}
}

// LoadConfig loads configuration from environment variables using Viper
func LoadConfig() (*Config, error) {
	cfg := Default()

	v := viper.New()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Set defaults in Viper
	setDefaults(v, cfg)

	setupEnvironmentBindings(v)

	// Bind configuration struct to Viper
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// setDefaults sets default values in Viper
func setDefaults(v *viper.Viper, cfg *Config) {
	v.SetDefault("otel_endpoint", cfg.OtelEndpoint)
	v.SetDefault("otel_debug", cfg.EnableDebug)
	v.SetDefault("env", cfg.Environment)
	v.SetDefault("otel_send_interval", cfg.SendInterval)
	v.SetDefault("otel_service_name", cfg.ServiceName)
	v.SetDefault("otel_service_version", cfg.ServiceVersion)
}

// setupEnvironmentBindings configures environment variable bindings
func setupEnvironmentBindings(v *viper.Viper) {
	envBindings := map[string]string{
		"otel_endpoint":        "OTEL_ENDPOINT",
		"otel_debug":           "OTEL_DEBUG",
		"env":                  "ENV",
		"otel_send_interval":   "OTEL_SEND_INTERVAL",
		"otel_service_name":    "OTEL_SERVICE_NAME",
		"otel_service_version": "OTEL_SERVICE_VERSION",
	}

	for key, env := range envBindings {
		v.BindEnv(key, env)
	}
}

// Validate validates the configuration
func (cfg *Config) Validate() error {
	if cfg.OtelEndpoint == "" {
		return fmt.Errorf("otel_endpoint is required")
	}
	if cfg.ServiceName == "" {
		return fmt.Errorf("service_name is required")
	}
	if cfg.ServiceVersion == "" {
		return fmt.Errorf("service_version is required")
	}
	if cfg.SendInterval <= 0 {
		return fmt.Errorf("send_interval must be positive")
	}

	return nil
}
