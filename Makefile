# ==============================================================================
# MAKEFILE - Kafka Demo PubSub Project
# ==============================================================================
#
# This Makefile provides convenient commands for building, testing, and managing
# the Kafka Demo PubSub project.
#
# Usage:
#   make help       - Display help
#   make build      - Build all binaries
#   make test       - Run tests
#   make run        - Start the complete environment
#   make stop       - Stop the environment
#   make clean      - Clean generated files
#
# ==============================================================================

# Variables
BINARY_DIR = bin
BINARY_PRODUCER = $(BINARY_DIR)/producer
BINARY_TRACKER = $(BINARY_DIR)/tracker
BINARY_MONITOR = $(BINARY_DIR)/monitor
GO = go
DOCKER_COMPOSE = docker compose

# OS detection
ifeq ($(OS),Windows_NT)
	BINARY_EXT = .exe
	RM = del /Q
	RMDIR = rmdir /S /Q
	MKDIR = mkdir
else
	BINARY_EXT =
	RM = rm -f
	RMDIR = rm -rf
	MKDIR = mkdir -p
endif

# Default targets
.PHONY: all build test clean run stop help deps lint

all: build

# ==============================================================================
# BUILD
# ==============================================================================

## build: Build all binaries
build: build-producer build-tracker build-monitor

## build-producer: Build the producer
build-producer:
	@echo "🔨 Building producer..."
	$(MKDIR) $(BINARY_DIR) 2>nul || true
	$(GO) build -tags kafka -o $(BINARY_PRODUCER)$(BINARY_EXT) ./cmd/producer

## build-tracker: Build the tracker (consumer)
build-tracker:
	@echo "🔨 Building tracker..."
	$(MKDIR) $(BINARY_DIR) 2>nul || true
	$(GO) build -tags kafka -o $(BINARY_TRACKER)$(BINARY_EXT) ./cmd/tracker

## build-monitor: Build the log monitor
build-monitor:
	@echo "🔨 Building log monitor..."
	$(MKDIR) $(BINARY_DIR) 2>nul || true
	$(GO) build -o $(BINARY_MONITOR)$(BINARY_EXT) ./cmd/monitor

# ==============================================================================
# DOCKER
# ==============================================================================

## docker-build: Build all Docker images
docker-build:
	@echo "🐳 Building Docker images..."
	docker build --target producer -t pubsub-producer .
	docker build --target tracker -t pubsub-tracker .
	docker build --target monitor -t pubsub-monitor .

## docker-build-producer: Build producer Docker image
docker-build-producer:
	@echo "🐳 Building producer Docker image..."
	docker build --target producer -t pubsub-producer .

## docker-build-tracker: Build tracker Docker image
docker-build-tracker:
	@echo "🐳 Building tracker Docker image..."
	docker build --target tracker -t pubsub-tracker .

## docker-up: Start all services (Kafka + apps)
docker-up:
	@echo "🚀 Starting all services..."
	docker compose --profile full up -d

## docker-up-kafka: Start Kafka only
docker-up-kafka:
	@echo "🚀 Starting Kafka..."
	docker compose up -d

## docker-down: Stop all services
docker-down:
	@echo "🛑 Stopping all services..."
	docker compose --profile full down

## docker-logs: Show logs for all services
docker-logs:
	docker compose --profile full logs -f

# ==============================================================================
# TESTS
# ==============================================================================

## test: Run all tests
test:
	@echo "🧪 Running tests..."
	$(GO) test -v ./...

## test-cover: Run tests with coverage
test-cover:
	@echo "🧪 Running tests with coverage..."
	$(GO) test -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "📊 Coverage report generated: coverage.html"

## test-models: Run only models tests
test-models:
	@echo "🧪 Running models tests..."
	$(GO) test -v ./pkg/models/...

## test-internal: Run internal package tests
test-internal:
	@echo "🧪 Running internal tests..."
	$(GO) test -tags kafka -v ./internal/...

# ==============================================================================
# DEPENDENCIES
# ==============================================================================

## deps: Download dependencies
deps:
	@echo "📦 Downloading dependencies..."
	$(GO) mod download
	$(GO) mod tidy

## deps-upgrade: Upgrade dependencies
deps-upgrade:
	@echo "⬆️  Upgrading dependencies..."
	$(GO) get -u ./...
	$(GO) mod tidy

# ==============================================================================
# CODE QUALITY
# ==============================================================================

## lint: Analyze code with go vet
lint:
	@echo "🔍 Analyzing code..."
	$(GO) vet ./...
	@echo "✅ Analysis complete"

## fmt: Format code
fmt:
	@echo "🎨 Formatting code..."
	$(GO) fmt ./...

# ==============================================================================
# DOCKER & EXECUTION
# ==============================================================================

## docker-up: Start Docker containers
docker-up:
	@echo "🐳 Starting Docker containers..."
	$(DOCKER_COMPOSE) up -d

## docker-down: Stop Docker containers
docker-down:
	@echo "🐳 Stopping Docker containers..."
	$(DOCKER_COMPOSE) down

## docker-logs: Display Kafka logs
docker-logs:
	$(DOCKER_COMPOSE) logs -f kafka

## run: Start the complete environment (Linux/macOS)
run:
	@echo "🚀 Starting environment..."
	./start.sh

## stop: Stop the complete environment (Linux/macOS)
stop:
	@echo "🛑 Stopping environment..."
	./stop.sh

## run-producer: Run the producer directly
run-producer: docker-up build-producer
	@echo "📤 Starting producer..."
	$(BINARY_PRODUCER)$(BINARY_EXT)

## run-tracker: Run the tracker directly
run-tracker: docker-up build-tracker
	@echo "📥 Starting tracker..."
	$(BINARY_TRACKER)$(BINARY_EXT)

## run-monitor: Run the log monitor
run-monitor: build-monitor
	@echo "📊 Starting log monitor..."
	$(BINARY_MONITOR)$(BINARY_EXT)

# ==============================================================================
# KAFKA
# ==============================================================================

## kafka-topics: List Kafka topics
kafka-topics:
	docker exec kafka kafka-topics --bootstrap-server localhost:9092 --list

## kafka-create-topic: Create the 'orders' topic
kafka-create-topic:
	docker exec kafka kafka-topics \
		--bootstrap-server localhost:9092 \
		--create \
		--if-not-exists \
		--topic orders \
		--partitions 1 \
		--replication-factor 1

## kafka-consume: Consume messages from the 'orders' topic
kafka-consume:
	docker exec kafka kafka-console-consumer \
		--bootstrap-server localhost:9092 \
		--topic orders \
		--from-beginning

# ==============================================================================
# CLEANUP
# ==============================================================================

## clean: Clean all generated files
clean:
	@echo "🧹 Cleaning generated files..."
	$(RM) $(BINARY_PRODUCER)$(BINARY_EXT) 2>nul || true
	$(RM) $(BINARY_TRACKER)$(BINARY_EXT) 2>nul || true
	$(RM) $(BINARY_MONITOR)$(BINARY_EXT) 2>nul || true
	$(RMDIR) $(BINARY_DIR) 2>nul || true
	$(RM) tracker.log 2>nul || true
	$(RM) tracker.events 2>nul || true
	$(RM) producer.pid 2>nul || true
	$(RM) tracker.pid 2>nul || true
	$(RM) coverage.out 2>nul || true
	$(RM) coverage.html 2>nul || true
	@echo "✅ Cleanup complete"

## clean-logs: Clean only log files
clean-logs:
	@echo "🧹 Cleaning logs..."
	$(RM) tracker.log 2>nul || true
	$(RM) tracker.events 2>nul || true

## clean-old: Clean old root-level Go files (backup first!)
clean-old:
	@echo "🧹 Cleaning old root-level Go files..."
	@echo "⚠️  Make sure you have committed changes before running this!"
	$(RM) cmd_producer.go 2>nul || true
	$(RM) cmd_tracker.go 2>nul || true
	$(RM) cmd_monitor.go 2>nul || true
	$(RM) producer.go 2>nul || true
	$(RM) tracker.go 2>nul || true
	$(RM) log_monitor.go 2>nul || true
	$(RM) order.go 2>nul || true
	$(RM) models.go 2>nul || true
	$(RM) constants.go 2>nul || true
	$(RM) producer_test.go 2>nul || true
	$(RM) tracker_test.go 2>nul || true
	$(RM) order_test.go 2>nul || true
	$(RM) log_monitor_test.go 2>nul || true

# ==============================================================================
# HELP
# ==============================================================================

## help: Display this help
help:
	@echo ""
	@echo "╔══════════════════════════════════════════════════════════════════════╗"
	@echo "║                    KAFKA DEMO - MAKEFILE HELP                        ║"
	@echo "╚══════════════════════════════════════════════════════════════════════╝"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Available targets:"
	@echo ""
	@echo "  BUILD:"
	@echo "    build            Build all binaries to bin/ directory"
	@echo "    build-producer   Build the producer"
	@echo "    build-tracker    Build the tracker"
	@echo "    build-monitor    Build the log monitor"
	@echo ""
	@echo "  TESTS:"
	@echo "    test             Run all tests"
	@echo "    test-cover       Run tests with coverage report"
	@echo "    test-models      Run only models tests"
	@echo "    test-internal    Run internal package tests"
	@echo ""
	@echo "  DEPENDENCIES:"
	@echo "    deps             Download dependencies"
	@echo "    deps-upgrade     Upgrade dependencies"
	@echo ""
	@echo "  QUALITY:"
	@echo "    lint             Analyze code"
	@echo "    fmt              Format code"
	@echo ""
	@echo "  EXECUTION:"
	@echo "    run              Start the complete environment"
	@echo "    stop             Stop the complete environment"
	@echo "    run-producer     Run the producer"
	@echo "    run-tracker      Run the tracker"
	@echo "    run-monitor      Run the monitor"
	@echo ""
	@echo "  DOCKER:"
	@echo "    docker-up        Start Kafka"
	@echo "    docker-down      Stop Kafka"
	@echo "    docker-logs      Display Kafka logs"
	@echo ""
	@echo "  KAFKA:"
	@echo "    kafka-topics       List topics"
	@echo "    kafka-create-topic Create the 'orders' topic"
	@echo "    kafka-consume      Consume messages"
	@echo ""
	@echo "  CLEANUP:"
	@echo "    clean            Clean all generated files"
	@echo "    clean-logs       Clean logs only"
	@echo "    clean-old        Clean old root-level Go files"
	@echo ""
