package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	// Verify default values
	if cfg.App.Env != "development" {
		t.Errorf("Expected env 'development', got %s", cfg.App.Env)
	}
	if cfg.Kafka.Broker != DefaultKafkaBroker {
		t.Errorf("Expected broker '%s', got %s", DefaultKafkaBroker, cfg.Kafka.Broker)
	}
	if cfg.Kafka.Topic != DefaultTopic {
		t.Errorf("Expected topic '%s', got %s", DefaultTopic, cfg.Kafka.Topic)
	}
	if cfg.Retry.MaxAttempts != 3 {
		t.Errorf("Expected 3 max attempts, got %d", cfg.Retry.MaxAttempts)
	}
	if !cfg.DLQ.Enabled {
		t.Error("Expected DLQ to be enabled by default")
	}
}

func TestInvalidEnvVars(t *testing.T) {
	// Set invalid environment variables
	os.Setenv("PRODUCER_INTERVAL_MS", "invalid")
	os.Setenv("RETRY_MAX_ATTEMPTS", "not-a-number")
	defer func() {
		os.Unsetenv("PRODUCER_INTERVAL_MS")
		os.Unsetenv("RETRY_MAX_ATTEMPTS")
	}()

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should fall back to defaults (or previously set values)
	// Default Producer.IntervalMs is 2000 (from DefaultConfig)
	if cfg.Producer.IntervalMs != 2000 {
		t.Errorf("Expected default interval 2000, got %d", cfg.Producer.IntervalMs)
	}
	// Default Retry.MaxAttempts is 3
	if cfg.Retry.MaxAttempts != 3 {
		t.Errorf("Expected default max attempts 3, got %d", cfg.Retry.MaxAttempts)
	}
}

func TestLoadWithDefaults(t *testing.T) {
	// Load with non-existent file should return defaults
	cfg, err := Load("nonexistent.yaml")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if cfg.Kafka.Broker != DefaultKafkaBroker {
		t.Errorf("Expected default broker, got %s", cfg.Kafka.Broker)
	}
}

func TestLoadFromYAML(t *testing.T) {
	// Create a temporary YAML file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	yamlContent := `
app:
  env: "production"
  log_level: "debug"
kafka:
  broker: "kafka.example.com:9092"
  topic: "custom-topic"
  consumer_group: "my-group"
producer:
  interval_ms: 5000
retry:
  max_attempts: 5
  initial_delay_ms: 200
dlq:
  enabled: false
  topic: "custom-dlq"
`
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify loaded values
	if cfg.App.Env != "production" {
		t.Errorf("Expected env 'production', got %s", cfg.App.Env)
	}
	if cfg.Kafka.Broker != "kafka.example.com:9092" {
		t.Errorf("Expected broker 'kafka.example.com:9092', got %s", cfg.Kafka.Broker)
	}
	if cfg.Kafka.Topic != "custom-topic" {
		t.Errorf("Expected topic 'custom-topic', got %s", cfg.Kafka.Topic)
	}
	if cfg.Producer.IntervalMs != 5000 {
		t.Errorf("Expected interval 5000ms, got %d", cfg.Producer.IntervalMs)
	}
	if cfg.Retry.MaxAttempts != 5 {
		t.Errorf("Expected 5 max attempts, got %d", cfg.Retry.MaxAttempts)
	}
	if cfg.DLQ.Enabled {
		t.Error("Expected DLQ to be disabled")
	}
}

func TestLoadFromEnv(t *testing.T) {
	// Set environment variables
	os.Setenv("KAFKA_BROKER", "env-broker:9092")
	os.Setenv("KAFKA_TOPIC", "env-topic")
	os.Setenv("RETRY_MAX_ATTEMPTS", "10")
	os.Setenv("DLQ_ENABLED", "false")
	defer func() {
		os.Unsetenv("KAFKA_BROKER")
		os.Unsetenv("KAFKA_TOPIC")
		os.Unsetenv("RETRY_MAX_ATTEMPTS")
		os.Unsetenv("DLQ_ENABLED")
	}()

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if cfg.Kafka.Broker != "env-broker:9092" {
		t.Errorf("Expected broker 'env-broker:9092', got %s", cfg.Kafka.Broker)
	}
	if cfg.Kafka.Topic != "env-topic" {
		t.Errorf("Expected topic 'env-topic', got %s", cfg.Kafka.Topic)
	}
	if cfg.Retry.MaxAttempts != 10 {
		t.Errorf("Expected 10 max attempts, got %d", cfg.Retry.MaxAttempts)
	}
	if cfg.DLQ.Enabled {
		t.Error("Expected DLQ to be disabled via env")
	}
}

func TestEnvOverridesYAML(t *testing.T) {
	// Create a temporary YAML file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	yamlContent := `
kafka:
  broker: "yaml-broker:9092"
  topic: "yaml-topic"
`
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Set environment variable that should override YAML
	os.Setenv("KAFKA_BROKER", "env-broker:9092")
	defer os.Unsetenv("KAFKA_BROKER")

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Env should override YAML
	if cfg.Kafka.Broker != "env-broker:9092" {
		t.Errorf("Expected env broker to override YAML, got %s", cfg.Kafka.Broker)
	}
	// YAML value should be preserved when no env override
	if cfg.Kafka.Topic != "yaml-topic" {
		t.Errorf("Expected topic 'yaml-topic', got %s", cfg.Kafka.Topic)
	}
}

func TestInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yaml")

	// Write invalid YAML
	if err := os.WriteFile(configPath, []byte("invalid: [yaml: content"), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Error("Expected error for invalid YAML")
	}
}

func TestDurationHelpers(t *testing.T) {
	cfg := DefaultConfig()

	// Test GetProducerInterval
	interval := cfg.GetProducerInterval()
	expected := time.Duration(cfg.Producer.IntervalMs) * time.Millisecond
	if interval != expected {
		t.Errorf("GetProducerInterval: expected %v, got %v", expected, interval)
	}

	// Test GetFlushTimeout
	flush := cfg.GetFlushTimeout()
	expectedFlush := time.Duration(cfg.Producer.FlushTimeoutMs) * time.Millisecond
	if flush != expectedFlush {
		t.Errorf("GetFlushTimeout: expected %v, got %v", expectedFlush, flush)
	}

	// Test GetMetricsInterval
	metrics := cfg.GetMetricsInterval()
	expectedMetrics := time.Duration(cfg.Tracker.MetricsIntervalSeconds) * time.Second
	if metrics != expectedMetrics {
		t.Errorf("GetMetricsInterval: expected %v, got %v", expectedMetrics, metrics)
	}

	// Test GetReadTimeout
	read := cfg.GetReadTimeout()
	expectedRead := time.Duration(cfg.Tracker.ReadTimeoutMs) * time.Millisecond
	if read != expectedRead {
		t.Errorf("GetReadTimeout: expected %v, got %v", expectedRead, read)
	}

	// Test GetInitialRetryDelay
	retryDelay := cfg.GetInitialRetryDelay()
	expectedRetryDelay := time.Duration(cfg.Retry.InitialDelayMs) * time.Millisecond
	if retryDelay != expectedRetryDelay {
		t.Errorf("GetInitialRetryDelay: expected %v, got %v", expectedRetryDelay, retryDelay)
	}

	// Test GetMaxRetryDelay
	maxRetry := cfg.GetMaxRetryDelay()
	expectedMaxRetry := time.Duration(cfg.Retry.MaxDelayMs) * time.Millisecond
	if maxRetry != expectedMaxRetry {
		t.Errorf("GetMaxRetryDelay: expected %v, got %v", expectedMaxRetry, maxRetry)
	}
}

func TestAllEnvVariables(t *testing.T) {
	// Test all supported environment variables
	envVars := map[string]string{
		"APP_ENV":              "staging",
		"LOG_LEVEL":            "warn",
		"KAFKA_BROKER":         "test:9092",
		"KAFKA_TOPIC":          "test-topic",
		"KAFKA_CONSUMER_GROUP": "test-group",
		"PRODUCER_INTERVAL_MS": "1000",
		"TRACKER_LOG_FILE":     "custom.log",
		"TRACKER_EVENTS_FILE":  "custom.events",
		"DLQ_TOPIC":            "test-dlq",
	}

	for k, v := range envVars {
		os.Setenv(k, v)
	}
	defer func() {
		for k := range envVars {
			os.Unsetenv(k)
		}
	}()

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if cfg.App.Env != "staging" {
		t.Errorf("APP_ENV: expected 'staging', got %s", cfg.App.Env)
	}
	if cfg.App.LogLevel != "warn" {
		t.Errorf("LOG_LEVEL: expected 'warn', got %s", cfg.App.LogLevel)
	}
	if cfg.Kafka.ConsumerGroup != "test-group" {
		t.Errorf("KAFKA_CONSUMER_GROUP: expected 'test-group', got %s", cfg.Kafka.ConsumerGroup)
	}
	if cfg.Producer.IntervalMs != 1000 {
		t.Errorf("PRODUCER_INTERVAL_MS: expected 1000, got %d", cfg.Producer.IntervalMs)
	}
	if cfg.Tracker.LogFile != "custom.log" {
		t.Errorf("TRACKER_LOG_FILE: expected 'custom.log', got %s", cfg.Tracker.LogFile)
	}
	if cfg.Tracker.EventsFile != "custom.events" {
		t.Errorf("TRACKER_EVENTS_FILE: expected 'custom.events', got %s", cfg.Tracker.EventsFile)
	}
	if cfg.DLQ.Topic != "test-dlq" {
		t.Errorf("DLQ_TOPIC: expected 'test-dlq', got %s", cfg.DLQ.Topic)
	}
}

func TestDLQEnabledVariations(t *testing.T) {
	tests := []struct {
		value    string
		expected bool
	}{
		{"true", true},
		{"1", true},
		{"false", false},
		{"0", false},
		{"yes", false}, // Only "true" and "1" are truthy
	}

	for _, tt := range tests {
		os.Setenv("DLQ_ENABLED", tt.value)
		cfg, _ := Load("")
		os.Unsetenv("DLQ_ENABLED")

		// Note: Default is true, so we need to check if it changed correctly
		// For "true" and "1", it should be true (matches default anyway)
		// For others, it should be false
		if tt.value == "true" || tt.value == "1" {
			if !cfg.DLQ.Enabled {
				t.Errorf("DLQ_ENABLED=%s: expected true, got false", tt.value)
			}
		} else {
			if cfg.DLQ.Enabled {
				t.Errorf("DLQ_ENABLED=%s: expected false, got true", tt.value)
			}
		}
	}
}
