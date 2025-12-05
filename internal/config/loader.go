package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// AppConfig est la structure principale de configuration pour l'application.
type AppConfig struct {
	App      AppSettings    `yaml:"app"`
	Kafka    KafkaConfig    `yaml:"kafka"`
	Producer ProducerConfig `yaml:"producer"`
	Tracker  TrackerConfig  `yaml:"tracker"`
	Monitor  MonitorConfig  `yaml:"monitor"`
	Retry    RetryConfig    `yaml:"retry"`
	DLQ      DLQConfig      `yaml:"dlq"`
}

// AppSettings contient les paramètres généraux de l'application.
type AppSettings struct {
	Env      string `yaml:"env"`
	LogLevel string `yaml:"log_level"`
}

// KafkaConfig contient les paramètres de connexion Kafka.
type KafkaConfig struct {
	Broker        string `yaml:"broker"`
	Topic         string `yaml:"topic"`
	ConsumerGroup string `yaml:"consumer_group"`
}

// ProducerConfig contient les paramètres spécifiques au producteur.
type ProducerConfig struct {
	IntervalMs     int `yaml:"interval_ms"`
	FlushTimeoutMs int `yaml:"flush_timeout_ms"`
}

// TrackerConfig contient les paramètres spécifiques au tracker.
type TrackerConfig struct {
	LogFile                string `yaml:"log_file"`
	EventsFile             string `yaml:"events_file"`
	MetricsIntervalSeconds int    `yaml:"metrics_interval_seconds"`
	ReadTimeoutMs          int    `yaml:"read_timeout_ms"`
	MaxConsecutiveErrors   int    `yaml:"max_consecutive_errors"`
}

// MonitorConfig contient les paramètres spécifiques au moniteur.
type MonitorConfig struct {
	MaxRecentLogs   int `yaml:"max_recent_logs"`
	MaxRecentEvents int `yaml:"max_recent_events"`
	UIUpdateMs      int `yaml:"ui_update_ms"`
}

// RetryConfig contient les paramètres du modèle de relance (retry).
type RetryConfig struct {
	MaxAttempts    int     `yaml:"max_attempts"`
	InitialDelayMs int     `yaml:"initial_delay_ms"`
	MaxDelayMs     int     `yaml:"max_delay_ms"`
	Multiplier     float64 `yaml:"multiplier"`
}

// DLQConfig contient les paramètres de la file d'attente des lettres mortes (DLQ).
type DLQConfig struct {
	Enabled bool   `yaml:"enabled"`
	Topic   string `yaml:"topic"`
}

// DefaultConfig retourne une configuration avec des valeurs par défaut.
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

// Load charge la configuration depuis un fichier YAML, en utilisant les valeurs par défaut si nécessaire.
// Les variables d'environnement surchargent les valeurs du fichier YAML.
func Load(configPath string) (*AppConfig, error) {
	cfg := DefaultConfig()

	// Essayer de charger depuis le fichier YAML
	if configPath != "" {
		if err := loadFromYAML(configPath, cfg); err != nil {
			// Le fichier non trouvé est acceptable, on utilise les défauts
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("erreur lors du chargement du fichier de config: %w", err)
			}
		}
	}

	// Surcharger avec les variables d'environnement
	loadFromEnv(cfg)

	return cfg, nil
}

// loadFromYAML charge la configuration depuis un fichier YAML.
func loadFromYAML(path string, cfg *AppConfig) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("erreur lors de l'analyse YAML: %w", err)
	}

	return nil
}

// loadFromEnv surcharge la configuration avec les variables d'environnement.
func loadFromEnv(cfg *AppConfig) {
	// Paramètres App
	if v := os.Getenv("APP_ENV"); v != "" {
		cfg.App.Env = v
	}
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		cfg.App.LogLevel = v
	}

	// Paramètres Kafka
	if v := os.Getenv("KAFKA_BROKER"); v != "" {
		cfg.Kafka.Broker = v
	}
	if v := os.Getenv("KAFKA_TOPIC"); v != "" {
		cfg.Kafka.Topic = v
	}
	if v := os.Getenv("KAFKA_CONSUMER_GROUP"); v != "" {
		cfg.Kafka.ConsumerGroup = v
	}

	// Paramètres Producer
	if v := os.Getenv("PRODUCER_INTERVAL_MS"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			cfg.Producer.IntervalMs = i
		}
	}

	// Paramètres Tracker
	if v := os.Getenv("TRACKER_LOG_FILE"); v != "" {
		cfg.Tracker.LogFile = v
	}
	if v := os.Getenv("TRACKER_EVENTS_FILE"); v != "" {
		cfg.Tracker.EventsFile = v
	}

	// Paramètres Retry
	if v := os.Getenv("RETRY_MAX_ATTEMPTS"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			cfg.Retry.MaxAttempts = i
		}
	}

	// Paramètres DLQ
	if v := os.Getenv("DLQ_ENABLED"); v != "" {
		cfg.DLQ.Enabled = v == "true" || v == "1"
	}
	if v := os.Getenv("DLQ_TOPIC"); v != "" {
		cfg.DLQ.Topic = v
	}
}

// GetProducerInterval retourne l'intervalle du producteur sous forme de durée.
func (c *AppConfig) GetProducerInterval() time.Duration {
	return time.Duration(c.Producer.IntervalMs) * time.Millisecond
}

// GetFlushTimeout retourne le délai d'expiration du flush sous forme de durée.
func (c *AppConfig) GetFlushTimeout() time.Duration {
	return time.Duration(c.Producer.FlushTimeoutMs) * time.Millisecond
}

// GetMetricsInterval retourne l'intervalle des métriques sous forme de durée.
func (c *AppConfig) GetMetricsInterval() time.Duration {
	return time.Duration(c.Tracker.MetricsIntervalSeconds) * time.Second
}

// GetReadTimeout retourne le délai de lecture sous forme de durée.
func (c *AppConfig) GetReadTimeout() time.Duration {
	return time.Duration(c.Tracker.ReadTimeoutMs) * time.Millisecond
}

// GetInitialRetryDelay retourne le délai initial de relance sous forme de durée.
func (c *AppConfig) GetInitialRetryDelay() time.Duration {
	return time.Duration(c.Retry.InitialDelayMs) * time.Millisecond
}

// GetMaxRetryDelay retourne le délai maximum de relance sous forme de durée.
func (c *AppConfig) GetMaxRetryDelay() time.Duration {
	return time.Duration(c.Retry.MaxDelayMs) * time.Millisecond
}
