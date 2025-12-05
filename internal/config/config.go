/*
Package config fournit la configuration centralisée pour le système PubSub.

Ce paquet contient toutes les constantes et structures de configuration
partagées entre les composants producer, tracker et monitor.
*/
package config

import "time"

// Configuration par défaut de Kafka
const (
	DefaultKafkaBroker   = "localhost:9092"
	DefaultConsumerGroup = "order-tracker-group"
	DefaultTopic         = "orders"
)

// Fichiers de logs
const (
	TrackerLogFile    = "tracker.log"
	TrackerEventsFile = "tracker.events"
)

// Délais et intervalles communs
const (
	FlushTimeoutMs = 15000
)

// Constantes pour le producteur
const (
	ProducerMessageInterval     = 2 * time.Second
	ProducerFlushTimeout        = 5 * time.Second
	ProducerDeliveryChannelSize = 10000
	ProducerDefaultTaxRate      = 0.20
	ProducerDefaultShippingFee  = 2.50
	ProducerDefaultCurrency     = "EUR"
	ProducerDefaultPayment      = "credit_card"
	ProducerDefaultWarehouse    = "PARIS-01"
)

// Constantes pour le tracker (consommateur)
const (
	TrackerMetricsInterval      = 30 * time.Second
	TrackerConsumerReadTimeout  = 1 * time.Second
	TrackerMaxConsecutiveErrors = 3
	TrackerServiceName          = "order-tracker"
)

// Constantes pour le moniteur de logs
const (
	MonitorMaxRecentLogs      = 20
	MonitorMaxRecentEvents    = 20
	MonitorMaxHistorySize     = 50
	MonitorLogChannelBuffer   = 100
	MonitorEventChannelBuffer = 100

	// Seuils de taux de succès (%)
	MonitorSuccessRateExcellent = 95.0
	MonitorSuccessRateGood      = 80.0

	// Seuils de débit (messages par seconde)
	MonitorThroughputNormal = 0.3
	MonitorThroughputLow    = 0.1

	// Seuils de temps d'erreur
	MonitorErrorTimeoutCritical = 1 * time.Minute
	MonitorErrorTimeoutWarning  = 5 * time.Minute

	// Seuils de score de qualité pour le débit
	MonitorQualityThroughputHigh   = 0.5
	MonitorQualityThroughputMedium = 0.3
	MonitorQualityThroughputLow    = 0.1

	// Seuils de score de qualité global
	MonitorQualityScoreExcellent = 90.0
	MonitorQualityScoreGood      = 70.0
	MonitorQualityScoreMedium    = 50.0

	// Intervalles de temps
	MonitorFileCheckInterval = 1 * time.Second
	MonitorFilePollInterval  = 200 * time.Millisecond
	MonitorUIUpdateInterval  = 500 * time.Millisecond

	// Limites d'affichage
	MonitorMaxLogRowLength   = 75
	MonitorMaxEventRowLength = 75
	MonitorTruncateSuffix    = "..."
)
