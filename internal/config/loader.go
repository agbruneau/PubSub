package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// AppConfig is the main configuration structure for the application.
// It aggregates configurations for all subsystems.
type AppConfig struct {
	App      AppSettings    `yaml:"app"`      // General application configuration.
	Kafka    KafkaConfig    `yaml:"kafka"`    // Kafka configuration.
	Producer ProducerConfig `yaml:"producer"` // Producer configuration.
	Tracker  TrackerConfig  `yaml:"tracker"`  // Tracker configuration.
	Monitor  MonitorConfig  `yaml:"monitor"`  // Monitor configuration.
	Retry    RetryConfig    `yaml:"retry"`    // Retry configuration.
	DLQ      DLQConfig      `yaml:"dlq"`      // Dead Letter Queue configuration.
}

// AppSettings contains general application settings.
type AppSettings struct {
	Env      string `yaml:"env"`       // Execution environment (e.g., development, production).
	LogLevel string `yaml:"log_level"` // Logging level.
}

// KafkaConfig contains Kafka connection settings.
type KafkaConfig struct {
	Broker        string `yaml:"broker"`         // Kafka broker address.
	Topic         string `yaml:"topic"`          // Main Kafka topic.
	ConsumerGroup string `yaml:"consumer_group"` // Consumer group identifier.
}

// ProducerConfig contains producer-specific settings.
type ProducerConfig struct {
	IntervalMs     int `yaml:"interval_ms"`      // Interval between messages in milliseconds.
	FlushTimeoutMs int `yaml:"flush_timeout_ms"` // Wait timeout for sending messages in milliseconds.
}

// TrackerConfig contains tracker-specific settings.
type TrackerConfig struct {
	LogFile                string `yaml:"log_file"`                 // Path to the structured log file.
	EventsFile             string `yaml:"events_file"`              // Path to the event file.
	MetricsIntervalSeconds int    `yaml:"metrics_interval_seconds"` // Metrics calculation interval in seconds.
	ReadTimeoutMs          int    `yaml:"read_timeout_ms"`          // Kafka read timeout in milliseconds.
	MaxConsecutiveErrors   int    `yaml:"max_consecutive_errors"`   // Max consecutive errors.
}

// MonitorConfig contains monitor-specific settings.
type MonitorConfig struct {
	MaxRecentLogs   int `yaml:"max_recent_logs"`   // Max recent logs to display.
	MaxRecentEvents int `yaml:"max_recent_events"` // Max recent events to display.
	UIUpdateMs      int `yaml:"ui_update_ms"`      // UI update frequency in milliseconds.
}

// RetryConfig contains retry model settings.
type RetryConfig struct {
	MaxAttempts    int     `yaml:"max_attempts"`     // Maximum number of attempts.
	InitialDelayMs int     `yaml:"initial_delay_ms"` // Initial delay in milliseconds.
	MaxDelayMs     int     `yaml:"max_delay_ms"`     // Maximum delay in milliseconds.
	Multiplier     float64 `yaml:"multiplier"`       // Backoff multiplier.
}

// DLQConfig contains Dead Letter Queue (DLQ) settings.
type DLQConfig struct {
	Enabled bool   `yaml:"enabled"` // Enables or disables DLQ.
	Topic   string `yaml:"topic"`   // Kafka topic for DLQ.
}

// DefaultConfig returns a configuration with default values.
// These values are used if no external configuration is provided.
//
// Returns:
//   - *AppConfig: A pointer to the default configuration.
func DefaultConfig() *AppConfig {
	return &AppConfig{
		App: AppSettings{
			Env:      "development",
			LogLevel: "info",
		},
		Kafka: KafkaConfig{
			Broker:        DefaultKafkaBroker,
			Topic:         DefaultTopic,
			ConsumerGroup: DefaultConsumerGroup,
		},
		Producer: ProducerConfig{
			IntervalMs:     int(ProducerMessageInterval / time.Millisecond),
			FlushTimeoutMs: int(ProducerFlushTimeout / time.Millisecond),
		},
		Tracker: TrackerConfig{
			LogFile:                TrackerLogFile,
			EventsFile:             TrackerEventsFile,
			MetricsIntervalSeconds: int(TrackerMetricsInterval / time.Second),
			ReadTimeoutMs:          int(TrackerConsumerReadTimeout / time.Millisecond),
			MaxConsecutiveErrors:   TrackerMaxConsecutiveErrors,
		},
		Monitor: MonitorConfig{
			MaxRecentLogs:   MonitorMaxRecentLogs,
			MaxRecentEvents: MonitorMaxRecentEvents,
			UIUpdateMs:      int(MonitorUIUpdateInterval / time.Millisecond),
		},
		Retry: RetryConfig{
			MaxAttempts:    3,
			InitialDelayMs: 100,
			MaxDelayMs:     5000,
			Multiplier:     2.0,
		},
		DLQ: DLQConfig{
			Enabled: true,
			Topic:   "orders-dlq",
		},
	}
}

// Load loads the configuration from a YAML file, utilizing default values if necessary.
// Environment variables override values from the YAML file.
//
// Parameters:
//   - configPath: Path to the YAML configuration file (optional).
//
// Returns:
//   - *AppConfig: The loaded configuration.
//   - error: An error if loading fails.
func Load(configPath string) (*AppConfig, error) {
	cfg := DefaultConfig()

	// Try to load from YAML file
	if configPath != "" {
		if err := loadFromYAML(configPath, cfg); err != nil {
			// Not found file is acceptable, use defaults
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("error loading config file: %w", err)
			}
		}
	}

	// Override with environment variables
	loadFromEnv(cfg)

	return cfg, nil
}

// loadFromYAML loads configuration from a YAML file.
//
// Parameters:
//   - path: The file path.
//   - cfg: The configuration structure to fill.
//
// Returns:
//   - error: An error if reading or parsing fails.
func loadFromYAML(path string, cfg *AppConfig) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("error parsing YAML: %w", err)
	}

	return nil
}

// loadFromEnv overrides the configuration with environment variables.
//
// Parameters:
//   - cfg: The configuration structure to update.
func loadFromEnv(cfg *AppConfig) {
	// App Parameters
	if v := os.Getenv("APP_ENV"); v != "" {
		cfg.App.Env = v
	}
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		cfg.App.LogLevel = v
	}

	// Kafka Parameters
	if v := os.Getenv("KAFKA_BROKER"); v != "" {
		cfg.Kafka.Broker = v
	}
	if v := os.Getenv("KAFKA_TOPIC"); v != "" {
		cfg.Kafka.Topic = v
	}
	if v := os.Getenv("KAFKA_CONSUMER_GROUP"); v != "" {
		cfg.Kafka.ConsumerGroup = v
	}

	// Producer Parameters
	if v := os.Getenv("PRODUCER_INTERVAL_MS"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			cfg.Producer.IntervalMs = i
		}
	}

	// Tracker Parameters
	if v := os.Getenv("TRACKER_LOG_FILE"); v != "" {
		cfg.Tracker.LogFile = v
	}
	if v := os.Getenv("TRACKER_EVENTS_FILE"); v != "" {
		cfg.Tracker.EventsFile = v
	}

	// Retry Parameters
	if v := os.Getenv("RETRY_MAX_ATTEMPTS"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			cfg.Retry.MaxAttempts = i
		}
	}

	// DLQ Parameters
	if v := os.Getenv("DLQ_ENABLED"); v != "" {
		cfg.DLQ.Enabled = v == "true" || v == "1"
	}
	if v := os.Getenv("DLQ_TOPIC"); v != "" {
		cfg.DLQ.Topic = v
	}
}

// GetProducerInterval returns the producer interval as a duration.
//
// Returns:
//   - time.Duration: The interval.
func (c *AppConfig) GetProducerInterval() time.Duration {
	return time.Duration(c.Producer.IntervalMs) * time.Millisecond
}

// GetFlushTimeout returns the flush timeout as a duration.
//
// Returns:
//   - time.Duration: The timeout.
func (c *AppConfig) GetFlushTimeout() time.Duration {
	return time.Duration(c.Producer.FlushTimeoutMs) * time.Millisecond
}

// GetMetricsInterval returns the metrics interval as a duration.
//
// Returns:
//   - time.Duration: The interval.
func (c *AppConfig) GetMetricsInterval() time.Duration {
	return time.Duration(c.Tracker.MetricsIntervalSeconds) * time.Second
}

// GetReadTimeout returns the read timeout as a duration.
//
// Returns:
//   - time.Duration: The timeout.
func (c *AppConfig) GetReadTimeout() time.Duration {
	return time.Duration(c.Tracker.ReadTimeoutMs) * time.Millisecond
}

// GetInitialRetryDelay returns the initial retry delay as a duration.
//
// Returns:
//   - time.Duration: The initial delay.
func (c *AppConfig) GetInitialRetryDelay() time.Duration {
	return time.Duration(c.Retry.InitialDelayMs) * time.Millisecond
}

// GetMaxRetryDelay returns the maximum retry delay as a duration.
//
// Returns:
//   - time.Duration: The maximum delay.
func (c *AppConfig) GetMaxRetryDelay() time.Duration {
	return time.Duration(c.Retry.MaxDelayMs) * time.Millisecond
}
