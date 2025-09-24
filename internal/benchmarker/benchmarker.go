package benchmarker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"

	"golang.org/x/time/rate"
	"promfire/internal/config"
	"promfire/internal/logger"
	"promfire/internal/writer"
)

// Benchmarker handles the main benchmarking logic
type Benchmarker struct {
	config         *config.Config
	dryRun         bool
	client         *http.Client
	excludeRegexes []*regexp.Regexp
	remoteWriter   *writer.RemoteWriter
}

// PrometheusResponse represents a response from Prometheus API
type PrometheusResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Values [][]any           `json:"values"`
		} `json:"result"`
	} `json:"data"`
}

// NewBenchmarker creates a new Benchmarker instance
func NewBenchmarker(cfg *config.Config, dryRun bool) (*Benchmarker, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Compile exclude regex patterns
	var excludeRegexes []*regexp.Regexp
	for _, pattern := range cfg.ExcludeMetrics {
		regex, err := regexp.Compile(pattern)
		if err != nil {
			logger.Warn("Invalid exclude pattern", map[string]any{
				"pattern": pattern,
				"error":   err.Error(),
			})
			continue
		}
		excludeRegexes = append(excludeRegexes, regex)
	}

	var remoteWriter *writer.RemoteWriter
	if !dryRun {
		remoteWriter = writer.NewRemoteWriter(cfg.Prometheus.RemoteWriteURL, cfg.Benchmark.BatchSize)
		if remoteWriter == nil {
			return nil, fmt.Errorf("failed to create remote writer")
		}
		logger.Info("Remote writer initialized", map[string]any{
			"remote_write_url": cfg.Prometheus.RemoteWriteURL,
			"batch_size":       cfg.Benchmark.BatchSize,
		})
	}

	return &Benchmarker{
		config:         cfg,
		dryRun:         dryRun,
		client:         client,
		excludeRegexes: excludeRegexes,
		remoteWriter:   remoteWriter,
	}, nil
}

// Run executes the benchmarking process
func (b *Benchmarker) Run(ctx context.Context) error {
	logger.Info("Starting benchmark process")

	// Step 1: Discover all metrics
	metrics, err := b.discoverMetrics(ctx)
	if err != nil {
		return fmt.Errorf("discovering metrics: %w", err)
	}

	logger.Info("Metric discovery completed", map[string]interface{}{
		"total_metrics": len(metrics),
	})

	// Step 2: Filter metrics
	filteredMetrics := b.filterMetrics(metrics)
	logger.Info("Metric filtering completed", map[string]interface{}{
		"filtered_metrics": len(filteredMetrics),
		"excluded_metrics": len(metrics) - len(filteredMetrics),
	})

	// Step 3: Query and replicate each metric
	return b.processMetrics(ctx, filteredMetrics)
}

// discoverMetrics discovers all available metrics from Prometheus
func (b *Benchmarker) discoverMetrics(ctx context.Context) ([]string, error) {
	queryURL := fmt.Sprintf("%s/api/v1/label/__name__/values", b.config.Prometheus.QueryURL)

	req, err := http.NewRequestWithContext(ctx, "GET", queryURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	var result struct {
		Status string   `json:"status"`
		Data   []string `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("query failed: %s", string(body))
	}

	return result.Data, nil
}

// filterMetrics filters out excluded metrics based on regex patterns
func (b *Benchmarker) filterMetrics(metrics []string) []string {
	var filtered []string
	for _, metric := range metrics {
		excluded := false
		for _, regex := range b.excludeRegexes {
			if regex.MatchString(metric) {
				excluded = true
				break
			}
		}
		if !excluded {
			filtered = append(filtered, metric)
		}
	}
	return filtered
}

// processMetrics processes each metric by querying and replicating data
func (b *Benchmarker) processMetrics(ctx context.Context, metrics []string) error {
	endTime := time.Now()
	startTime := endTime.Add(-time.Duration(b.config.Benchmark.QueryRangeHours) * time.Hour)
	step := time.Duration(b.config.Benchmark.QueryStepSeconds) * time.Second

	// Create rate limiter for samples per second with larger burst capacity
	samplesPerSecond := b.config.Benchmark.SamplesPerSecond
	burstCapacity := samplesPerSecond * 2 // Allow bursts up to 2 seconds worth of samples
	rateLimiter := rate.NewLimiter(rate.Limit(samplesPerSecond), burstCapacity)

	for _, metricName := range metrics {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		logger.Debug("Processing metric", map[string]interface{}{
			"metric_name": metricName,
		})

		if err := b.processMetric(ctx, metricName, startTime, endTime, step, rateLimiter); err != nil {
			logger.Error("Error processing metric", map[string]interface{}{
				"metric_name": metricName,
				"error":       err.Error(),
			})
			continue
		}
	}

	return nil
}

// processMetric processes a single metric
func (b *Benchmarker) processMetric(ctx context.Context, metricName string, startTime, endTime time.Time, step time.Duration, rateLimiter *rate.Limiter) error {
	// Query the metric data
	data, err := b.queryMetricRange(ctx, metricName, startTime, endTime, step)
	if err != nil {
		return fmt.Errorf("querying metric data: %w", err)
	}

	if len(data.Data.Result) == 0 {
		logger.Debug("No data found for metric", map[string]interface{}{
			"metric_name": metricName,
		})
		return nil
	}

	// Replicate data with modified labels
	for _, series := range data.Data.Result {
		if err := b.replicateSeries(ctx, metricName, series, rateLimiter); err != nil {
			logger.Error("Error replicating series", map[string]interface{}{
				"metric_name": metricName,
				"error":       err.Error(),
			})
			continue
		}
	}

	return nil
}

// queryMetricRange queries a metric over a time range
func (b *Benchmarker) queryMetricRange(ctx context.Context, metricName string, startTime, endTime time.Time, step time.Duration) (*PrometheusResponse, error) {
	params := url.Values{}
	params.Set("query", metricName)
	params.Set("start", strconv.FormatInt(startTime.Unix(), 10))
	params.Set("end", strconv.FormatInt(endTime.Unix(), 10))
	params.Set("step", strconv.FormatInt(int64(step.Seconds()), 10))

	queryURL := fmt.Sprintf("%s/api/v1/query_range?%s", b.config.Prometheus.QueryURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", queryURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	var result PrometheusResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("query failed: %s", string(body))
	}

	return &result, nil
}

// replicateSeries replicates a single time series with modified labels
func (b *Benchmarker) replicateSeries(ctx context.Context, metricName string, series struct {
	Metric map[string]string `json:"metric"`
	Values [][]interface{}   `json:"values"`
}, rateLimiter *rate.Limiter) error {

	// Generate label combinations
	labelCombinations := b.generateLabelCombinations()

	for i, labelSet := range labelCombinations {
		if i >= b.config.Benchmark.ReplicationFactor {
			break
		}

		// Create new labels by combining original with replication labels
		newLabels := make(map[string]string)
		for k, v := range series.Metric {
			newLabels[k] = v
		}
		for k, v := range labelSet {
			newLabels[k] = v
		}

		if b.dryRun {
			logger.Info("DRY RUN: Would replicate series", map[string]interface{}{
				"metric_name":  metricName,
				"labels":       newLabels,
				"sample_count": len(series.Values),
			})
			continue
		}

		// Convert and send samples
		if err := b.sendSamples(ctx, newLabels, series.Values, rateLimiter); err != nil {
			return fmt.Errorf("sending samples: %w", err)
		}
	}

	return nil
}

// generateLabelCombinations generates combinations of replication labels
func (b *Benchmarker) generateLabelCombinations() []map[string]string {
	if len(b.config.Replication) == 0 {
		// Generate default combinations if no replication labels configured
		combinations := make([]map[string]string, b.config.Benchmark.ReplicationFactor)
		for i := 0; i < b.config.Benchmark.ReplicationFactor; i++ {
			combinations[i] = map[string]string{
				"benchmark_replica": fmt.Sprintf("replica-%d", i),
			}
		}
		return combinations
	}

	// Generate combinations from configured replication labels
	var combinations []map[string]string

	// Auto-generate values for benchmark_instance if needed
	processedLabels := make([]config.ReplicationLabel, len(b.config.Replication))
	copy(processedLabels, b.config.Replication)

	for i, labelConfig := range processedLabels {
		if labelConfig.Name == "benchmark_instance" && len(labelConfig.Values) == 0 {
			// Auto-generate benchmark_instance values based on replication factor
			autoValues := make([]string, b.config.Benchmark.ReplicationFactor)
			for j := 0; j < b.config.Benchmark.ReplicationFactor; j++ {
				autoValues[j] = fmt.Sprintf("bench-%d", j+1)
			}
			processedLabels[i].Values = autoValues
			logger.Debug("Auto-generated benchmark_instance values", map[string]interface{}{
				"count":  len(autoValues),
				"values": autoValues,
			})
		}
	}

	// Calculate all possible combinations
	totalCombinations := 1
	for _, labelConfig := range processedLabels {
		if len(labelConfig.Values) > 0 {
			totalCombinations *= len(labelConfig.Values)
		}
	}

	// Generate combinations up to replication factor
	maxCombinations := b.config.Benchmark.ReplicationFactor
	if maxCombinations > totalCombinations {
		maxCombinations = totalCombinations
	}

	for i := 0; i < maxCombinations; i++ {
		labelSet := make(map[string]string)

		// Generate combination index for each label
		combIndex := i
		for _, labelConfig := range processedLabels {
			if len(labelConfig.Values) > 0 {
				valueIndex := combIndex % len(labelConfig.Values)
				labelSet[labelConfig.Name] = labelConfig.Values[valueIndex]
				combIndex = combIndex / len(labelConfig.Values)
			}
		}

		combinations = append(combinations, labelSet)
	}

	return combinations
}

// sendSamples sends samples to Prometheus with rate limiting
func (b *Benchmarker) sendSamples(ctx context.Context, labels map[string]string, values [][]interface{}, rateLimiter *rate.Limiter) error {
	if len(values) == 0 {
		return nil
	}

	// If we have more samples than can fit in burst, send in chunks
	burstSize := rateLimiter.Burst()
	totalSamples := len(values)

	for i := 0; i < totalSamples; i += burstSize {
		end := i + burstSize
		if end > totalSamples {
			end = totalSamples
		}

		chunk := values[i:end]
		chunkSize := len(chunk)

		// Wait for rate limiter tokens for this chunk
		if err := rateLimiter.WaitN(ctx, chunkSize); err != nil {
			return fmt.Errorf("rate limiting: %w", err)
		}

		logger.Debug("Sending sample chunk to Prometheus", map[string]interface{}{
			"chunk_size":   chunkSize,
			"chunk_num":    (i / burstSize) + 1,
			"total_chunks": (totalSamples + burstSize - 1) / burstSize,
			"labels":       labels,
		})

		if b.remoteWriter != nil {
			if err := b.remoteWriter.WriteSamples(ctx, labels, chunk); err != nil {
				return fmt.Errorf("writing chunk %d: %w", (i/burstSize)+1, err)
			}
		}
	}

	return nil
}
