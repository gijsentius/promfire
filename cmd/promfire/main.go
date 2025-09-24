package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"promfire/internal/benchmarker"
	"promfire/internal/config"
	"promfire/internal/logger"
)

func main() {
	var (
		configPath = flag.String("config", "config.yaml", "Path to configuration file")
		dryRun     = flag.Bool("dry-run", false, "Print what would be done without executing")
		version    = flag.Bool("version", false, "Print version information")
		logLevel   = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	)
	flag.Parse()

	if *version {
		logger.Init(logger.INFO, "promfire")
		logger.Info("PromFire v1.0.0 - Prometheus Benchmarking Tool")
		return
	}

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		logger.Init(logger.ERROR, "promfire")
		logger.Fatal("Failed to load config", map[string]any{
			"error":      err.Error(),
			"configPath": *configPath,
		})
	}

	// Initialize logger with configured level
	logl := logger.ParseLogLevel(*logLevel)
	logger.Init(logl, "promfire")

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		logger.Fatal("Invalid configuration", map[string]any{
			"error": err.Error(),
		})
	}

	logger.Info("Starting Prometheus benchmark tool", map[string]any{
		"query_url":          cfg.Prometheus.QueryURL,
		"remote_write_url":   cfg.Prometheus.RemoteWriteURL,
		"replication_factor": cfg.Benchmark.ReplicationFactor,
		"dry_run":            *dryRun,
		"log_level":          logl.String(),
	})

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		logger.Info("Received interrupt signal, shutting down...")
		cancel()
	}()

	// Create and run benchmarker
	bench, err := benchmarker.NewBenchmarker(cfg, *dryRun)

	if err != nil {
		logger.Fatal("Failed to create benchmarker", map[string]any{
			"error": err.Error(),
		})
		return
	}

	if err := bench.Run(ctx); err != nil {
		logger.Fatal("Benchmarker failed", map[string]any{
			"error": err.Error(),
		})
	}

	logger.Info("Benchmark completed successfully")
}
