/*
Package config provides centralized configuration for the PubSub system.

This package contains all constants and configuration structures
shared between the producer, tracker, and monitor components.
*/
package config

import "time"

// Default Kafka Configuration
const (
	// DefaultKafkaBroker is the default Kafka broker address.
	DefaultKafkaBroker = "localhost:9092"
	// DefaultConsumerGroup is the default consumer group.
	DefaultConsumerGroup = "order-tracker-group"
	// DefaultTopic is the default Kafka topic.
	DefaultTopic = "orders"
)

// Log Files
const (
	// TrackerLogFile is the name of the structured log file.
	TrackerLogFile = "logs/tracker.log"
	// TrackerEventsFile is the name of the event audit file.
	TrackerEventsFile = "logs/tracker.events"
)

// Common timeouts and intervals
const (
	// FlushTimeoutMs is the default flush timeout for messages (in ms).
	FlushTimeoutMs = 15000
)

// Producer constants
const (
	// ProducerMessageInterval defines the interval between sending two messages.
	ProducerMessageInterval = 2 * time.Second
	// ProducerFlushTimeout defines the maximum wait time for sending messages on shutdown.
	ProducerFlushTimeout = 5 * time.Second
	// ProducerDeliveryChannelSize is the buffer size for delivery reports.
	ProducerDeliveryChannelSize = 10000
	// ProducerDefaultTaxRate is the default tax rate.
	ProducerDefaultTaxRate = 0.20
	// ProducerDefaultShippingFee is the default shipping fee.
	ProducerDefaultShippingFee = 2.50
	// ProducerDefaultCurrency is the default currency.
	ProducerDefaultCurrency = "EUR"
	// ProducerDefaultPayment is the default payment method.
	ProducerDefaultPayment = "credit_card"
	// ProducerDefaultWarehouse is the default warehouse.
	ProducerDefaultWarehouse = "PARIS-01"
)

// Tracker (consumer) constants
const (
	// TrackerMetricsInterval is the metrics calculation interval.
	TrackerMetricsInterval = 30 * time.Second
	// TrackerConsumerReadTimeout is the wait time for reading a Kafka message.
	TrackerConsumerReadTimeout = 1 * time.Second
	// TrackerMaxConsecutiveErrors is the maximum number of consecutive errors tolerated before alerting.
	TrackerMaxConsecutiveErrors = 3
	// TrackerServiceName is the service name for logs.
	TrackerServiceName = "order-tracker"
)

// Log Monitor constants
const (
	// MonitorMaxRecentLogs is the maximum number of recent logs to keep in memory.
	MonitorMaxRecentLogs = 20
	// MonitorMaxRecentEvents is the maximum number of recent events to keep in memory.
	MonitorMaxRecentEvents = 20
	// MonitorMaxHistorySize is the history size for charts.
	MonitorMaxHistorySize = 50
	// MonitorLogChannelBuffer is the buffer size for the log channel.
	MonitorLogChannelBuffer = 100
	// MonitorEventChannelBuffer is the buffer size for the event channel.
	MonitorEventChannelBuffer = 100

	// Success Rate Thresholds (%)

	// MonitorSuccessRateExcellent defines the threshold for an excellent success rate.
	MonitorSuccessRateExcellent = 95.0
	// MonitorSuccessRateGood defines the threshold for a good success rate.
	MonitorSuccessRateGood = 80.0

	// Throughput Thresholds (messages per second)

	// MonitorThroughputNormal defines the threshold for normal throughput.
	MonitorThroughputNormal = 0.3
	// MonitorThroughputLow defines the threshold for low throughput.
	MonitorThroughputLow = 0.1

	// Error Timeout Thresholds

	// MonitorErrorTimeoutCritical is the delay before an error is considered critical.
	MonitorErrorTimeoutCritical = 1 * time.Minute
	// MonitorErrorTimeoutWarning is the delay before an error is considered a warning.
	MonitorErrorTimeoutWarning = 5 * time.Minute

	// Throughput Quality Score Thresholds

	// MonitorQualityThroughputHigh is the threshold for a high throughput score.
	MonitorQualityThroughputHigh = 0.5
	// MonitorQualityThroughputMedium is the threshold for a medium throughput score.
	MonitorQualityThroughputMedium = 0.3
	// MonitorQualityThroughputLow is the threshold for a low throughput score.
	MonitorQualityThroughputLow = 0.1

	// Global Quality Score Thresholds

	// MonitorQualityScoreExcellent is the threshold for an excellent global quality score.
	MonitorQualityScoreExcellent = 90.0
	// MonitorQualityScoreGood is the threshold for a good global quality score.
	MonitorQualityScoreGood = 70.0
	// MonitorQualityScoreMedium is the threshold for a medium global quality score.
	MonitorQualityScoreMedium = 50.0

	// Time Intervals

	// MonitorFileCheckInterval is the interval for checking file existence.
	MonitorFileCheckInterval = 1 * time.Second
	// MonitorFilePollInterval is the interval for reading new content in files.
	MonitorFilePollInterval = 200 * time.Millisecond
	// MonitorUIUpdateInterval is the UI refresh interval.
	MonitorUIUpdateInterval = 500 * time.Millisecond

	// Display Limits

	// MonitorMaxLogRowLength is the maximum length of a displayed log row.
	MonitorMaxLogRowLength = 75
	// MonitorMaxEventRowLength is the maximum length of a displayed event row.
	MonitorMaxEventRowLength = 75
	// MonitorTruncateSuffix is the suffix added when text is truncated.
	MonitorTruncateSuffix = "..."
)
