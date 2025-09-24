package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

// Config represents the application configuration
type Config struct {
	Prometheus     Prometheus         `yaml:"prometheus"`
	Benchmark      Benchmark          `yaml:"benchmark"`
	Replication    []ReplicationLabel `yaml:"replication_labels"`
	ExcludeMetrics []string           `yaml:"exclude_metrics"`
	LogLevel       string             `yaml:"log_level,omitempty"`
}

// Prometheus contains Prometheus connection settings
type Prometheus struct {
	QueryURL       string `yaml:"query_url"`
	RemoteWriteURL string `yaml:"remote_write_url"`
}

// Benchmark contains benchmarking parameters
type Benchmark struct {
	ReplicationFactor int `yaml:"replication_factor"`
	QueryRangeHours   int `yaml:"query_range_hours"`
	QueryStepSeconds  int `yaml:"query_step_seconds"`
	SamplesPerSecond  int `yaml:"samples_per_second"`
	BatchSize         int `yaml:"batch_size"`
}

// ReplicationLabel contains label replication configuration
type ReplicationLabel struct {
	Name   string   `yaml:"name"`
	Values []string `yaml:"values"`
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	// Set defaults
	config.setDefaults()

	return &config, nil
}

// setDefaults sets default values for unspecified configuration
func (c *Config) setDefaults() {
	if c.Benchmark.ReplicationFactor == 0 {
		c.Benchmark.ReplicationFactor = 2
	}
	if c.Benchmark.QueryRangeHours == 0 {
		c.Benchmark.QueryRangeHours = 24
	}
	if c.Benchmark.QueryStepSeconds == 0 {
		c.Benchmark.QueryStepSeconds = 60
	}
	if c.Benchmark.SamplesPerSecond == 0 {
		c.Benchmark.SamplesPerSecond = 1000
	}
	if c.Benchmark.BatchSize == 0 {
		c.Benchmark.BatchSize = 100
	}
	if c.Prometheus.QueryURL == "" {
		c.Prometheus.QueryURL = "http://localhost:9090"
	}
	if c.Prometheus.RemoteWriteURL == "" {
		c.Prometheus.RemoteWriteURL = "http://localhost:9090/api/v1/write"
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Benchmark.ReplicationFactor < 1 {
		return fmt.Errorf("replication_factor must be at least 1")
	}
	if c.Benchmark.QueryRangeHours < 1 {
		return fmt.Errorf("query_range_hours must be at least 1")
	}
	if c.Benchmark.QueryStepSeconds < 1 {
		return fmt.Errorf("query_step_seconds must be at least 1")
	}
	if c.Benchmark.SamplesPerSecond < 1 {
		return fmt.Errorf("samples_per_second must be at least 1")
	}
	if c.Benchmark.BatchSize < 1 {
		return fmt.Errorf("batch_size must be at least 1")
	}
	return nil
}
