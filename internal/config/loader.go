package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// AppConfig is the main configuration structure for the application.
type AppConfig struct {
	App      AppSettings    `yaml:"app"`
	Kafka    KafkaConfig    `yaml:"kafka"`
	Producer ProducerConfig `yaml:"producer"`
	Tracker  TrackerConfig  `yaml:"tracker"`
	Monitor  MonitorConfig  `yaml:"monitor"`
	Retry    RetryConfig    `yaml:"retry"`
	DLQ      DLQConfig      `yaml:"dlq"`
}

// AppSettings contains general application settings.
type AppSettings struct {
	Env      string `yaml:"env"`
	LogLevel string `yaml:"log_level"`
}

// KafkaConfig contains Kafka connection settings.
type KafkaConfig struct {
	Broker        string `yaml:"broker"`
	Topic         string `yaml:"topic"`
	ConsumerGroup string `yaml:"consumer_group"`
}

// ProducerConfig contains producer-specific settings.
type ProducerConfig struct {
	IntervalMs     int `yaml:"interval_ms"`
	FlushTimeoutMs int `yaml:"flush_timeout_ms"`
}

// TrackerConfig contains tracker-specific settings.
type TrackerConfig struct {
	LogFile                string `yaml:"log_file"`
	EventsFile             string `yaml:"events_file"`
	MetricsIntervalSeconds int    `yaml:"metrics_interval_seconds"`
	ReadTimeoutMs          int    `yaml:"read_timeout_ms"`
	MaxConsecutiveErrors   int    `yaml:"max_consecutive_errors"`
}

// MonitorConfig contains monitor-specific settings.
type MonitorConfig struct {
	MaxRecentLogs   int `yaml:"max_recent_logs"`
	MaxRecentEvents int `yaml:"max_recent_events"`
	UIUpdateMs      int `yaml:"ui_update_ms"`
}

// RetryConfig contains retry pattern settings.
type RetryConfig struct {
	MaxAttempts    int     `yaml:"max_attempts"`
	InitialDelayMs int     `yaml:"initial_delay_ms"`
	MaxDelayMs     int     `yaml:"max_delay_ms"`
	Multiplier     float64 `yaml:"multiplier"`
}

// DLQConfig contains Dead Letter Queue settings.
type DLQConfig struct {
	Enabled bool   `yaml:"enabled"`
	Topic   string `yaml:"topic"`
}

// DefaultConfig returns a configuration with default values.
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

// Load loads configuration from a YAML file, falling back to defaults.
// Environment variables override YAML values.
func Load(configPath string) (*AppConfig, error) {
	cfg := DefaultConfig()

	// Try to load from YAML file
	if configPath != "" {
		if err := loadFromYAML(configPath, cfg); err != nil {
			// File not found is okay, we use defaults
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

// loadFromEnv overrides configuration with environment variables.
func loadFromEnv(cfg *AppConfig) {
	// App settings
	if v := os.Getenv("APP_ENV"); v != "" {
		cfg.App.Env = v
	}
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		cfg.App.LogLevel = v
	}

	// Kafka settings
	if v := os.Getenv("KAFKA_BROKER"); v != "" {
		cfg.Kafka.Broker = v
	}
	if v := os.Getenv("KAFKA_TOPIC"); v != "" {
		cfg.Kafka.Topic = v
	}
	if v := os.Getenv("KAFKA_CONSUMER_GROUP"); v != "" {
		cfg.Kafka.ConsumerGroup = v
	}

	// Producer settings
	if v := os.Getenv("PRODUCER_INTERVAL_MS"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			cfg.Producer.IntervalMs = i
		}
	}

	// Tracker settings
	if v := os.Getenv("TRACKER_LOG_FILE"); v != "" {
		cfg.Tracker.LogFile = v
	}
	if v := os.Getenv("TRACKER_EVENTS_FILE"); v != "" {
		cfg.Tracker.EventsFile = v
	}

	// Retry settings
	if v := os.Getenv("RETRY_MAX_ATTEMPTS"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			cfg.Retry.MaxAttempts = i
		}
	}

	// DLQ settings
	if v := os.Getenv("DLQ_ENABLED"); v != "" {
		cfg.DLQ.Enabled = v == "true" || v == "1"
	}
	if v := os.Getenv("DLQ_TOPIC"); v != "" {
		cfg.DLQ.Topic = v
	}
}

// GetProducerInterval returns the producer interval as a duration.
func (c *AppConfig) GetProducerInterval() time.Duration {
	return time.Duration(c.Producer.IntervalMs) * time.Millisecond
}

// GetFlushTimeout returns the flush timeout as a duration.
func (c *AppConfig) GetFlushTimeout() time.Duration {
	return time.Duration(c.Producer.FlushTimeoutMs) * time.Millisecond
}

// GetMetricsInterval returns the metrics interval as a duration.
func (c *AppConfig) GetMetricsInterval() time.Duration {
	return time.Duration(c.Tracker.MetricsIntervalSeconds) * time.Second
}

// GetReadTimeout returns the read timeout as a duration.
func (c *AppConfig) GetReadTimeout() time.Duration {
	return time.Duration(c.Tracker.ReadTimeoutMs) * time.Millisecond
}

// GetInitialRetryDelay returns the initial retry delay as a duration.
func (c *AppConfig) GetInitialRetryDelay() time.Duration {
	return time.Duration(c.Retry.InitialDelayMs) * time.Millisecond
}

// GetMaxRetryDelay returns the max retry delay as a duration.
func (c *AppConfig) GetMaxRetryDelay() time.Duration {
	return time.Duration(c.Retry.MaxDelayMs) * time.Millisecond
}
