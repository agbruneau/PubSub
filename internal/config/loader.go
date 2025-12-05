package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// AppConfig est la structure principale de configuration pour l'application.
// Elle regroupe les configurations de tous les sous-systèmes.
type AppConfig struct {
	App      AppSettings    `yaml:"app"`      // Configuration générale de l'application.
	Kafka    KafkaConfig    `yaml:"kafka"`    // Configuration de Kafka.
	Producer ProducerConfig `yaml:"producer"` // Configuration du producteur.
	Tracker  TrackerConfig  `yaml:"tracker"`  // Configuration du tracker.
	Monitor  MonitorConfig  `yaml:"monitor"`  // Configuration du moniteur.
	Retry    RetryConfig    `yaml:"retry"`    // Configuration des relances.
	DLQ      DLQConfig      `yaml:"dlq"`      // Configuration de la Dead Letter Queue.
}

// AppSettings contient les paramètres généraux de l'application.
type AppSettings struct {
	Env      string `yaml:"env"`       // L'environnement d'exécution (ex: development, production).
	LogLevel string `yaml:"log_level"` // Le niveau de journalisation.
}

// KafkaConfig contient les paramètres de connexion Kafka.
type KafkaConfig struct {
	Broker        string `yaml:"broker"`         // L'adresse du broker Kafka.
	Topic         string `yaml:"topic"`          // Le sujet Kafka principal.
	ConsumerGroup string `yaml:"consumer_group"` // L'identifiant du groupe de consommateurs.
}

// ProducerConfig contient les paramètres spécifiques au producteur.
type ProducerConfig struct {
	IntervalMs     int `yaml:"interval_ms"`      // Intervalle entre les messages en millisecondes.
	FlushTimeoutMs int `yaml:"flush_timeout_ms"` // Délai d'attente pour l'envoi des messages en millisecondes.
}

// TrackerConfig contient les paramètres spécifiques au tracker.
type TrackerConfig struct {
	LogFile                string `yaml:"log_file"`                 // Chemin du fichier de logs structurés.
	EventsFile             string `yaml:"events_file"`              // Chemin du fichier d'événements.
	MetricsIntervalSeconds int    `yaml:"metrics_interval_seconds"` // Intervalle de calcul des métriques en secondes.
	ReadTimeoutMs          int    `yaml:"read_timeout_ms"`          // Délai de lecture Kafka en millisecondes.
	MaxConsecutiveErrors   int    `yaml:"max_consecutive_errors"`   // Nombre max d'erreurs consécutives.
}

// MonitorConfig contient les paramètres spécifiques au moniteur.
type MonitorConfig struct {
	MaxRecentLogs   int `yaml:"max_recent_logs"`   // Nombre max de logs récents à afficher.
	MaxRecentEvents int `yaml:"max_recent_events"` // Nombre max d'événements récents à afficher.
	UIUpdateMs      int `yaml:"ui_update_ms"`      // Fréquence de mise à jour de l'UI en millisecondes.
}

// RetryConfig contient les paramètres du modèle de relance (retry).
type RetryConfig struct {
	MaxAttempts    int     `yaml:"max_attempts"`     // Nombre maximum de tentatives.
	InitialDelayMs int     `yaml:"initial_delay_ms"` // Délai initial en millisecondes.
	MaxDelayMs     int     `yaml:"max_delay_ms"`     // Délai maximum en millisecondes.
	Multiplier     float64 `yaml:"multiplier"`       // Multiplicateur pour le backoff.
}

// DLQConfig contient les paramètres de la file d'attente des lettres mortes (DLQ).
type DLQConfig struct {
	Enabled bool   `yaml:"enabled"` // Active ou désactive la DLQ.
	Topic   string `yaml:"topic"`   // Le sujet Kafka pour la DLQ.
}

// DefaultConfig retourne une configuration avec des valeurs par défaut.
// Ces valeurs sont utilisées si aucune configuration externe n'est fournie.
//
// Retourne:
//   - *AppConfig: Un pointeur vers la configuration par défaut.
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
//
// Paramètres:
//   - configPath: Le chemin vers le fichier de configuration YAML (optionnel).
//
// Retourne:
//   - *AppConfig: La configuration chargée.
//   - error: Une erreur si le chargement échoue.
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
//
// Paramètres:
//   - path: Le chemin du fichier.
//   - cfg: La structure de configuration à remplir.
//
// Retourne:
//   - error: Une erreur si la lecture ou le parsing échoue.
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
//
// Paramètres:
//   - cfg: La structure de configuration à mettre à jour.
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
//
// Retourne:
//   - time.Duration: L'intervalle.
func (c *AppConfig) GetProducerInterval() time.Duration {
	return time.Duration(c.Producer.IntervalMs) * time.Millisecond
}

// GetFlushTimeout retourne le délai d'expiration du flush sous forme de durée.
//
// Retourne:
//   - time.Duration: Le délai d'expiration.
func (c *AppConfig) GetFlushTimeout() time.Duration {
	return time.Duration(c.Producer.FlushTimeoutMs) * time.Millisecond
}

// GetMetricsInterval retourne l'intervalle des métriques sous forme de durée.
//
// Retourne:
//   - time.Duration: L'intervalle.
func (c *AppConfig) GetMetricsInterval() time.Duration {
	return time.Duration(c.Tracker.MetricsIntervalSeconds) * time.Second
}

// GetReadTimeout retourne le délai de lecture sous forme de durée.
//
// Retourne:
//   - time.Duration: Le délai.
func (c *AppConfig) GetReadTimeout() time.Duration {
	return time.Duration(c.Tracker.ReadTimeoutMs) * time.Millisecond
}

// GetInitialRetryDelay retourne le délai initial de relance sous forme de durée.
//
// Retourne:
//   - time.Duration: Le délai initial.
func (c *AppConfig) GetInitialRetryDelay() time.Duration {
	return time.Duration(c.Retry.InitialDelayMs) * time.Millisecond
}

// GetMaxRetryDelay retourne le délai maximum de relance sous forme de durée.
//
// Retourne:
//   - time.Duration: Le délai maximum.
func (c *AppConfig) GetMaxRetryDelay() time.Duration {
	return time.Duration(c.Retry.MaxDelayMs) * time.Millisecond
}
