package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type ConfigManager struct {
	config *Config
}

// durationValue is a custom flag type that accepts both duration strings and plain numbers (seconds)
type durationValue struct {
	value *time.Duration
}

func (d *durationValue) String() string {
	if d.value == nil {
		return "0s"
	}
	return d.value.String()
}

func (d *durationValue) Set(s string) error {
	// Try parsing as duration first (e.g., "30s", "1m")
	if duration, err := time.ParseDuration(s); err == nil {
		*d.value = duration
		return nil
	}

	// Try parsing as plain number (seconds)
	if seconds, err := strconv.Atoi(s); err == nil {
		*d.value = time.Duration(seconds) * time.Second
		return nil
	}

	return fmt.Errorf("invalid duration format: %s (use either duration like '30s' or seconds like '30')", s)
}

// newDurationValue creates a new duration flag value
func newDurationValue(val time.Duration, p *time.Duration) *durationValue {
	*p = val
	return &durationValue{value: p}
}

// NewConfigManager creates a new configuration manager
func NewConfigManager() *ConfigManager {
	return &ConfigManager{
		config: &Config{},
	}
}

// LoadConfig loads configuration from command line flags and environment variables
func (cm *ConfigManager) LoadConfig() (*Config, error) {
	// Set default values
	cm.setDefaults()

	// Parse environment variables first
	cm.parseEnvironmentVariables()

	// Parse command line flags (these override environment variables)
	cm.parseCommandLineFlags()

	// Validate configuration
	if err := cm.validateConfig(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return cm.config, nil
}

// setDefaults sets default configuration values
func (cm *ConfigManager) setDefaults() {
	cm.config.BaseURL = ""
	// cm.config.APIEndpoint = "ListPhysicalDevices"
	cm.config.PollInterval = 5 * time.Second
	cm.config.RequestTimeout = 1 * time.Second
	cm.config.ShowTimestamp = true
	cm.config.ColorOutput = true
	cm.config.Username = "admin"
	cm.config.Password = "admin"
}

// parseEnvironmentVariables reads configuration from environment variables
func (cm *ConfigManager) parseEnvironmentVariables() {
	if base_url := os.Getenv("PT_BASE_URL"); base_url != "" {
		cm.config.BaseURL = base_url
	}

	if interval := os.Getenv("PT_POLL_INTERVAL"); interval != "" {
		// Try parsing as duration first (e.g., "30s", "1m")
		if duration, err := time.ParseDuration(interval); err == nil {
			cm.config.PollInterval = duration
		} else if seconds, err := strconv.Atoi(interval); err == nil {
			// Try parsing as plain number (seconds)
			cm.config.PollInterval = time.Duration(seconds) * time.Second
		}
	}

	if timeout := os.Getenv("PT_REQUEST_TIMEOUT"); timeout != "" {
		if timeout, err := strconv.Atoi(timeout); err == nil {
			cm.config.RequestTimeout = time.Duration(timeout) * time.Second
		}

		// if duration, err := time.Duration(timeout); err == nil {
		// 	cm.config.RequestTimeout = duration
		// }
	}

	if noColor := os.Getenv("PT_NO_COLOR"); noColor != "" {
		if value, err := strconv.ParseBool(noColor); err == nil {
			cm.config.ColorOutput = !value
		}
	}

	if noTimestamp := os.Getenv("NO_TIMESTAMP"); noTimestamp != "" {
		if value, err := strconv.ParseBool(noTimestamp); err == nil {
			cm.config.ShowTimestamp = !value
		}
	}

	if username := os.Getenv("PT_API_USERNAME"); username != "" {
		cm.config.Username = username
	}

	if password := os.Getenv("PT_API_PASSWORD"); password != "" {
		cm.config.Password = password
	}
}

// parseCommandLineFlags parses command line arguments
func (cm *ConfigManager) parseCommandLineFlags() {
	var (
		base_url = flag.String("base_url", cm.config.BaseURL, "Base URL (REQUIRED) (https://<mgmt>/api/v2/)") // noColor  = flag.Bool("no-color", !cm.config.ColorOutput, "Disable colored output")
		username = flag.String("username", cm.config.Username, "API username for authentication")
		password = flag.String("password", cm.config.Password, "API password for authentication")
		showHelp = flag.Bool("help", false, "Show help message")
	)

	// Custom duration flag that accepts both duration strings and plain numbers
	interval := newDurationValue(cm.config.PollInterval, &cm.config.PollInterval)
	flag.Var(interval, "interval", "Poll interval (e.g., 30, 60, or 30s, 1m)")

	flag.Usage = cm.printUsage
	flag.Parse()

	if *showHelp {
		cm.printUsage()
		os.Exit(0)
	}

	// Apply command line flag values
	cm.config.BaseURL = *base_url
	// cm.config.ColorOutput = !*noColor
	cm.config.Username = *username
	cm.config.Password = *password
	// Note: PollInterval is automatically set by the custom flag
}

// validateConfig validates the configuration values
func (cm *ConfigManager) validateConfig() error {
	if cm.config.BaseURL == "" {
		return fmt.Errorf("base URL is required. Set it via -base_url flag or PT_BASE_URL environment variable")
	}

	if !strings.HasSuffix(cm.config.BaseURL, "/") {
		cm.config.BaseURL += "/"
	}

	if cm.config.PollInterval < 1*time.Second {
		return fmt.Errorf("poll interval must be at least 1 second")
	}

	// if cm.config.RequestTimeout < 1*time.Second {
	// 	return fmt.Errorf("request timeout must be at least 1 second")
	// }

	// if cm.config.RequestTimeout > cm.config.PollInterval {
	// 	cm.config.RequestTimeout = cm.config.PollInterval / 2
	// }

	return nil
}

// printUsage prints the usage information
func (cm *ConfigManager) printUsage() {
	fmt.Fprintf(os.Stderr, `Go API Monitor - Physical Devices Monitor

Usage: %s [OPTIONS]

This application periodically polls the Physical Devices API PT NGFW.

OPTIONS:
`, os.Args[0])

	flag.PrintDefaults()

	fmt.Fprintf(os.Stderr, `
ENVIRONMENT VARIABLES:
  PT_BASE_URL          API BASE URL (REQUIRED) (example: https://pt-mgmt/api/v2/)
  PT_POLL_INTERVAL     Poll interval in seconds or duration (e.g., "30", "60", "30s", "1m") (default: 5)
  PT_API_USERNAME      API username for authentication (default: admin)
  PT_API_PASSWORD      API password for authentication (default: admin)

EXAMPLES:
  # Basic usage with required base URL
  %s -base_url https://my-api.com/api/v2/

  # Use custom endpoint and interval (seconds)
  %s -base_url https://my-api.com/api/v2/ -interval 60

  # Use custom endpoint and interval (duration)
  %s -base_url https://my-api.com/api/v2/ -interval 1m30s

  # Set configuration via environment variables
  export PT_BASE_URL="https://my-api.com/api/v2/"
  export PT_POLL_INTERVAL="60"
  %s

KEYBOARD SHORTCUTS:
  Ctrl+C    Exit the application

`, os.Args[0], os.Args[0], os.Args[0], os.Args[0])
}

// GetConfig returns the current configuration
func (cm *ConfigManager) GetConfig() *Config {
	return cm.config
}

// PrintConfig prints the current configuration (for debugging)
func (cm *ConfigManager) PrintConfig() {
	fmt.Printf("Configuration:\n")
	fmt.Printf("  Base URL:         %s\n", cm.config.BaseURL)
	fmt.Printf("  Poll Interval:    %v\n", cm.config.PollInterval)
	fmt.Printf("  Username:         %s\n", cm.config.Username)
	fmt.Println()
}
