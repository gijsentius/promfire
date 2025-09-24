# PromFire Architecture

## Project Structure

```
promfire/
├── cmd/
│   └── promfire/           # Main application entry point
│       └── main.go
├── internal/               # Private application packages
│   ├── config/            # Configuration management
│   │   └── config.go
│   ├── benchmarker/       # Core benchmarking logic
│   │   └── benchmarker.go
│   └── writer/            # Prometheus remote write client
│       └── remote_writer.go
├── pkg/                   # Public reusable packages (empty for now)
├── examples/              # Example configurations
│   ├── config-light.yaml
│   └── config-heavy.yaml
├── docs/                  # Documentation
│   └── architecture.md
├── bin/                   # Built binaries (created during build)
├── go.mod                 # Go module definition
├── go.sum                 # Go module checksums
├── config.yaml            # Default configuration
├── Makefile              # Build automation
└── README.md             # Project documentation
```

## Package Organization

### `cmd/promfire/`
Contains the main application entry point. Handles CLI arguments, signal handling, and orchestrates the benchmarking process.

### `internal/config/`
Handles configuration loading, validation, and default value setting. Provides a clean API for accessing application settings.

**Key Components:**
- `Config` struct with validation
- YAML configuration loading
- Default value management

### `internal/benchmarker/`
Core benchmarking logic that discovers metrics, queries data, and orchestrates the replication process.

**Key Components:**
- Metric discovery via Prometheus API
- Time series data querying
- Label combination generation
- Data replication orchestration

### `internal/writer/`
Prometheus remote write protocol implementation for efficiently sending replicated data back to Prometheus.

**Key Components:**
- Remote write protocol implementation
- Batch processing
- Snappy compression
- Rate limiting integration

## Data Flow

1. **Configuration Loading** (`internal/config`)
   - Load and validate YAML configuration
   - Set default values

2. **Metric Discovery** (`internal/benchmarker`)
   - Query Prometheus for all metric names
   - Apply exclusion filters

3. **Data Querying** (`internal/benchmarker`)
   - Query historical data for each metric
   - Process time series data

4. **Label Generation** (`internal/benchmarker`)
   - Generate combinations of replication labels
   - Create new label sets for synthetic data

5. **Data Replication** (`internal/writer`)
   - Convert to Prometheus remote write format
   - Apply rate limiting
   - Send via HTTP POST with compression

## Design Principles

### Clean Architecture
- Separation of concerns between packages
- Internal packages prevent external dependencies
- Clear interfaces between components

### Configurability
- YAML-based configuration
- Environment-specific examples
- Validation and defaults

### Performance
- Batch processing for efficiency
- Rate limiting to prevent system overload
- Concurrent processing where appropriate

### Reliability
- Graceful error handling
- Context-based cancellation
- Comprehensive logging

### Testing
- Dry-run mode for safe testing
- Clear logging and feedback
- Configurable parameters for different load scenarios