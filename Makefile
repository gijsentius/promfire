.PHONY: build clean test install deps fmt vet lint run

# Application name
APP_NAME := promfire
BUILD_DIR := bin
CMD_DIR := cmd/promfire

# Build the application
build: deps
	mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(APP_NAME) ./$(CMD_DIR)

# Install dependencies
deps:
	go mod tidy

# Format code
fmt:
	go fmt ./...

# Vet code
vet:
	go vet ./...

# Run basic checks
lint: fmt vet

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)

# Test with dry run
test: build
	./$(BUILD_DIR)/$(APP_NAME) -dry-run -config examples/config-test-rate.yaml

# Run with default config
run: build
	./$(BUILD_DIR)/$(APP_NAME)

# Run with light config example
run-light: build
	./$(BUILD_DIR)/$(APP_NAME) -config examples/config-light.yaml

# Run with heavy config example
run-heavy: build
	./$(BUILD_DIR)/$(APP_NAME) -config examples/config-heavy.yaml

# Install to /usr/local/bin
install: build
	sudo cp $(BUILD_DIR)/$(APP_NAME) /usr/local/bin/

# Show help
help:
	@echo "Available targets:"
	@echo "  build      - Build the promfire binary"
	@echo "  deps       - Download and tidy Go dependencies"
	@echo "  fmt        - Format Go code"
	@echo "  vet        - Run go vet"
	@echo "  lint       - Run fmt and vet"
	@echo "  clean      - Remove build artifacts"
	@echo "  test       - Run with dry-run mode"
	@echo "  run        - Run with default config"
	@echo "  run-light  - Run with light load config"
	@echo "  run-heavy  - Run with heavy load config"
	@echo "  install    - Install to /usr/local/bin"
	@echo "  help       - Show this help"
