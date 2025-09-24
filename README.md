# PromFire - Prometheus Benchmarking Tool

A tool to benchmark your Prometheus setup by replicating existing metrics data with modified labels to simulate increased load.

## Features

- **Data Discovery**: Automatically discovers all metrics in your Prometheus instance
- **Smart Filtering**: Excludes system metrics and allows custom exclusion patterns
- **Label Replication**: Creates new time series by adding configurable labels
- **Rate Control**: Configurable rate limiting to match your current ingestion rate
- **Remote Write**: Uses Prometheus remote write protocol for efficient data ingestion
- **Dry Run Mode**: Test your configuration without actually writing data

## Project Structure

```
promfire/
├── cmd/promfire/           # Main application entry point
├── internal/               # Private application packages
│   ├── config/            # Configuration management
│   ├── benchmarker/       # Core benchmarking logic
│   └── writer/            # Prometheus remote write client
├── examples/              # Example configurations
├── docs/                  # Documentation
├── bin/                   # Built binaries
└── Makefile              # Build automation
```

## Quick Start

### Build and Test

```bash
# Build the application
make build

# Test with dry run (safe)
make test

```

### Basic Usage

```bash
# Run with default config
./bin/promfire

# Specify custom config file
./bin/promfire -config /path/to/config.yaml

# Dry run to see what would be replicated
./bin/promfire -dry-run

# Check version
./bin/promfire -version
```

### Configuration

Copy and modify `config.yaml` to match your setup:

```yaml
prometheus:
  query_url: "http://localhost:9090"
  remote_write_url: "http://localhost:9090/api/v1/write"

benchmark:
  replication_factor: 2
  query_range_hours: 24
  query_step_seconds: 60
  samples_per_second: 1000
  batch_size: 100

replication_labels:
  - name: "benchmark_instance"
    values: ["bench-1", "bench-2", "bench-3"]
  - name: "benchmark_region"
    values: ["us-east", "us-west", "eu-central"]
```

## How It Works

1. **Discovery**: Queries Prometheus for all available metric names
2. **Filtering**: Excludes system metrics and applies custom exclusion patterns
3. **Data Retrieval**: Fetches historical data for each metric using range queries
4. **Replication**: Creates new time series by combining original labels with replication labels
5. **Ingestion**: Sends replicated data back to Prometheus using remote write protocol

## Build

```bash
go mod tidy
go build -o promfire
```

## Examples

### Double Your Metric Load
Set `replication_factor: 2` to create one additional copy of each time series.

### Simulate Multi-Region Deployment
Configure replication labels to add region and instance identifiers:

```yaml
replication_labels:
  - name: "region"
    values: ["us-east", "us-west", "eu-central"]
  - name: "instance_type"
    values: ["production", "staging"]
```

### Rate-Limited Testing
Set `samples_per_second` to match your target ingestion rate to avoid overwhelming your Prometheus instance.

## Safety Features

- **Dry Run Mode**: Always test your configuration first
- **Rate Limiting**: Built-in rate limiting to prevent overwhelming your system
- **Batch Processing**: Efficient batching of remote write requests
- **Graceful Shutdown**: Handles interrupt signals cleanly
- **Metric Filtering**: Automatically excludes system metrics

## Monitoring

The tool logs its progress and provides metrics on:
- Number of metrics discovered and filtered
- Replication progress per metric
- Sample ingestion rate
- Error rates and failed operations
