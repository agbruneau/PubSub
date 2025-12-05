/*
Package config provides centralized configuration for the PubSub system.

This package contains all constants and configuration structures
shared between the producer, tracker, and monitor components.
*/
package config

import "time"

// Kafka default configuration
const (
	DefaultKafkaBroker   = "localhost:9092"
	DefaultConsumerGroup = "order-tracker-group"
	DefaultTopic         = "orders"
)

// Log files
const (
	TrackerLogFile    = "tracker.log"
	TrackerEventsFile = "tracker.events"
)

// Common timeouts and intervals
const (
	FlushTimeoutMs = 15000
)

// Producer constants
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

// Tracker constants
const (
	TrackerMetricsInterval      = 30 * time.Second
	TrackerConsumerReadTimeout  = 1 * time.Second
	TrackerMaxConsecutiveErrors = 3
	TrackerServiceName          = "order-tracker"
)

// Monitor constants
const (
	MonitorMaxRecentLogs      = 20
	MonitorMaxRecentEvents    = 20
	MonitorMaxHistorySize     = 50
	MonitorLogChannelBuffer   = 100
	MonitorEventChannelBuffer = 100

	// Success rate thresholds (%)
	MonitorSuccessRateExcellent = 95.0
	MonitorSuccessRateGood      = 80.0

	// Throughput thresholds (messages per second)
	MonitorThroughputNormal = 0.3
	MonitorThroughputLow    = 0.1

	// Error time thresholds
	MonitorErrorTimeoutCritical = 1 * time.Minute
	MonitorErrorTimeoutWarning  = 5 * time.Minute

	// Quality score thresholds for throughput
	MonitorQualityThroughputHigh   = 0.5
	MonitorQualityThroughputMedium = 0.3
	MonitorQualityThroughputLow    = 0.1

	// Quality score thresholds
	MonitorQualityScoreExcellent = 90.0
	MonitorQualityScoreGood      = 70.0
	MonitorQualityScoreMedium    = 50.0

	// Time intervals
	MonitorFileCheckInterval = 1 * time.Second
	MonitorFilePollInterval  = 200 * time.Millisecond
	MonitorUIUpdateInterval  = 500 * time.Millisecond

	// Display limits
	MonitorMaxLogRowLength   = 75
	MonitorMaxEventRowLength = 75
	MonitorTruncateSuffix    = "..."
)
