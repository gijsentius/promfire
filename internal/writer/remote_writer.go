package writer

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/golang/snappy"
	"github.com/prometheus/prometheus/prompb"
	"promfire/internal/logger"
)

// TimestampCoordinator ensures globally unique, strictly increasing timestamps
type TimestampCoordinator struct {
	mu            sync.Mutex
	lastTimestamp int64
	increment     int64
}

// NewTimestampCoordinator creates a new timestamp coordinator
func NewTimestampCoordinator() *TimestampCoordinator {
	return &TimestampCoordinator{
		lastTimestamp: time.Now().UnixMilli(),
		increment:     1, // 1ms increment between samples
	}
}

// NextTimestamp returns the next unique timestamp in milliseconds
func (tc *TimestampCoordinator) NextTimestamp() int64 {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	now := time.Now().UnixMilli()
	if now > tc.lastTimestamp {
		tc.lastTimestamp = now
	} else {
		tc.lastTimestamp += tc.increment
	}

	return tc.lastTimestamp
}

// RemoteWriter handles writing samples to Prometheus via remote write protocol
type RemoteWriter struct {
	client              *http.Client
	endpoint            string
	batchSize           int
	timestampCoordinator *TimestampCoordinator
}

// NewRemoteWriter creates a new RemoteWriter instance
func NewRemoteWriter(endpoint string, batchSize int) *RemoteWriter {
	return &RemoteWriter{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		endpoint:             endpoint,
		batchSize:            batchSize,
		timestampCoordinator: NewTimestampCoordinator(),
	}
}

// WriteSamples writes samples for a single time series to Prometheus
func (rw *RemoteWriter) WriteSamples(ctx context.Context, labels map[string]string, values [][]interface{}) error {
	// Convert to Prometheus TimeSeries format
	timeSeries, err := rw.convertToTimeSeries(labels, values)
	if err != nil {
		return fmt.Errorf("converting to time series: %w", err)
	}

	// Send in batches
	return rw.sendInBatches(ctx, []*prompb.TimeSeries{timeSeries})
}

// WriteBatch writes multiple time series to Prometheus
func (rw *RemoteWriter) WriteBatch(ctx context.Context, timeSeries []*prompb.TimeSeries) error {
	return rw.sendInBatches(ctx, timeSeries)
}

// convertToTimeSeries converts labels and values to Prometheus TimeSeries format
func (rw *RemoteWriter) convertToTimeSeries(labels map[string]string, values [][]interface{}) (*prompb.TimeSeries, error) {
	// Create label pairs
	var labelPairs []prompb.Label
	for name, value := range labels {
		labelPairs = append(labelPairs, prompb.Label{
			Name:  name,
			Value: value,
		})
	}

	if len(values) == 0 {
		return nil, fmt.Errorf("no values provided")
	}

	// Convert ALL samples, not just the last one
	var samples []prompb.Sample
	for _, value := range values {
		if len(value) != 2 {
			continue // Skip invalid values
		}

		// Parse value (ignore original timestamp)
		valueStr, ok := value[1].(string)
		if !ok {
			continue // Skip non-string values
		}

		valueFloat, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			continue // Skip unparseable values
		}

		// Use coordinated timestamp to ensure strict ordering
		timestamp := rw.timestampCoordinator.NextTimestamp()

		samples = append(samples, prompb.Sample{
			Timestamp: timestamp,
			Value:     valueFloat,
		})
	}

	if len(samples) == 0 {
		return nil, fmt.Errorf("no valid samples found")
	}

	return &prompb.TimeSeries{
		Labels:  labelPairs,
		Samples: samples,
	}, nil
}

// sendInBatches sends time series data in configurable batch sizes
func (rw *RemoteWriter) sendInBatches(ctx context.Context, timeSeries []*prompb.TimeSeries) error {
	for i := 0; i < len(timeSeries); i += rw.batchSize {
		end := i + rw.batchSize
		if end > len(timeSeries) {
			end = len(timeSeries)
		}

		batch := timeSeries[i:end]
		if err := rw.sendBatch(ctx, batch); err != nil {
			return fmt.Errorf("sending batch %d-%d: %w", i, end, err)
		}

		logger.Debug("Batch sent successfully", map[string]interface{}{
			"batch_size": len(batch),
			"batch_id":   fmt.Sprintf("%d-%d", i, end),
		})
	}

	return nil
}

// sendBatch sends a single batch of time series to Prometheus
func (rw *RemoteWriter) sendBatch(ctx context.Context, timeSeries []*prompb.TimeSeries) error {
	// Create write request
	writeRequest := &prompb.WriteRequest{}
	for _, ts := range timeSeries {
		writeRequest.Timeseries = append(writeRequest.Timeseries, *ts)
	}

	// Marshal to protobuf
	data, err := writeRequest.Marshal()
	if err != nil {
		return fmt.Errorf("marshaling write request: %w", err)
	}

	// Compress with snappy
	compressed := snappy.Encode(nil, data)

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", rw.endpoint, bytes.NewReader(compressed))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-protobuf")
	req.Header.Set("Content-Encoding", "snappy")
	req.Header.Set("X-Prometheus-Remote-Write-Version", "0.1.0")

	// Send request
	resp, err := rw.client.Do(req)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("remote write failed with status %d", resp.StatusCode)
	}

	return nil
}
