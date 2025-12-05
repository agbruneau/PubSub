package monitor

import (
	"testing"
	"time"

	"github.com/agbruneau/PubSub/pkg/models"
)

func TestNewMonitor(t *testing.T) {
	m := New()

	if m == nil {
		t.Fatal("New() returned nil")
	}
	if m.Metrics == nil {
		t.Fatal("Metrics is nil")
	}
	if m.Metrics.RecentLogs == nil {
		t.Fatal("RecentLogs is nil")
	}
	if m.Metrics.RecentEvents == nil {
		t.Fatal("RecentEvents is nil")
	}
}

func TestProcessLog(t *testing.T) {
	m := New()

	// Process an info log
	entry := models.LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     models.LogLevelINFO,
		Message:   "Test message",
	}
	m.ProcessLog(entry)

	if len(m.Metrics.RecentLogs) != 1 {
		t.Errorf("Expected 1 log, got %d", len(m.Metrics.RecentLogs))
	}
	if m.Metrics.ErrorCount != 0 {
		t.Errorf("Expected 0 errors, got %d", m.Metrics.ErrorCount)
	}
}

func TestProcessLogError(t *testing.T) {
	m := New()

	// Process an error log
	entry := models.LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     models.LogLevelERROR,
		Message:   "Error message",
	}
	m.ProcessLog(entry)

	if m.Metrics.ErrorCount != 1 {
		t.Errorf("Expected 1 error, got %d", m.Metrics.ErrorCount)
	}
	if m.Metrics.LastErrorTime.IsZero() {
		t.Error("LastErrorTime should be set")
	}
}

func TestProcessLogMetrics(t *testing.T) {
	m := New()

	// Process a metrics log
	entry := models.LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     models.LogLevelINFO,
		Message:   "Métriques système périodiques",
		Metadata: map[string]interface{}{
			"messages_received":    float64(100),
			"messages_processed":   float64(95),
			"messages_failed":      float64(5),
			"messages_per_second":  "2.5",
			"success_rate_percent": "95.0",
		},
	}
	m.ProcessLog(entry)

	if m.Metrics.MessagesReceived != 100 {
		t.Errorf("Expected 100 messages received, got %d", m.Metrics.MessagesReceived)
	}
	if m.Metrics.MessagesProcessed != 95 {
		t.Errorf("Expected 95 messages processed, got %d", m.Metrics.MessagesProcessed)
	}
	if m.Metrics.MessagesFailed != 5 {
		t.Errorf("Expected 5 messages failed, got %d", m.Metrics.MessagesFailed)
	}
	if m.Metrics.CurrentMessagesPerSec != 2.5 {
		t.Errorf("Expected 2.5 msg/s, got %f", m.Metrics.CurrentMessagesPerSec)
	}
	if m.Metrics.CurrentSuccessRate != 95.0 {
		t.Errorf("Expected 95%% success rate, got %f", m.Metrics.CurrentSuccessRate)
	}
}

func TestProcessEvent(t *testing.T) {
	m := New()
	m.Metrics.StartTime = time.Now().Add(-10 * time.Second)

	// Process a successful event
	entry := models.EventEntry{
		Timestamp:    time.Now().Format(time.RFC3339),
		EventType:    "order_received",
		Deserialized: true,
		KafkaOffset:  123,
	}
	m.ProcessEvent(entry)

	if len(m.Metrics.RecentEvents) != 1 {
		t.Errorf("Expected 1 event, got %d", len(m.Metrics.RecentEvents))
	}
	if m.Metrics.MessagesReceived != 1 {
		t.Errorf("Expected 1 message received, got %d", m.Metrics.MessagesReceived)
	}
	if m.Metrics.MessagesProcessed != 1 {
		t.Errorf("Expected 1 message processed, got %d", m.Metrics.MessagesProcessed)
	}
}

func TestProcessEventFailed(t *testing.T) {
	m := New()
	m.Metrics.StartTime = time.Now().Add(-10 * time.Second)

	// Process a failed event
	entry := models.EventEntry{
		Timestamp:    time.Now().Format(time.RFC3339),
		EventType:    "deserialization_failed",
		Deserialized: false,
		KafkaOffset:  456,
	}
	m.ProcessEvent(entry)

	if m.Metrics.MessagesFailed != 1 {
		t.Errorf("Expected 1 failed message, got %d", m.Metrics.MessagesFailed)
	}
	if m.Metrics.ErrorCount != 1 {
		t.Errorf("Expected 1 error, got %d", m.Metrics.ErrorCount)
	}
}

func TestGetHealthStatus(t *testing.T) {
	tests := []struct {
		rate     float64
		expected HealthStatus
	}{
		{98.0, HealthGood},
		{90.0, HealthWarning},
		{50.0, HealthCritical},
	}

	for _, tt := range tests {
		status, _, _ := GetHealthStatus(tt.rate)
		if status != tt.expected {
			t.Errorf("Rate %.1f: expected %v, got %v", tt.rate, tt.expected, status)
		}
	}
}

func TestGetThroughputStatus(t *testing.T) {
	tests := []struct {
		mps      float64
		expected HealthStatus
	}{
		{0.5, HealthGood},
		{0.2, HealthWarning},
		{0.0, HealthCritical},
	}

	for _, tt := range tests {
		status, _, _ := GetThroughputStatus(tt.mps)
		if status != tt.expected {
			t.Errorf("MPS %.1f: expected %v, got %v", tt.mps, tt.expected, status)
		}
	}
}

func TestGetErrorStatus(t *testing.T) {
	// No errors
	status, _, _ := GetErrorStatus(0, time.Time{})
	if status != HealthGood {
		t.Errorf("No errors: expected HealthGood, got %v", status)
	}

	// Recent error
	status, _, _ = GetErrorStatus(1, time.Now().Add(-30*time.Second))
	if status != HealthCritical {
		t.Errorf("Recent error: expected HealthCritical, got %v", status)
	}

	// Old error
	status, _, _ = GetErrorStatus(1, time.Now().Add(-10*time.Minute))
	if status != HealthGood {
		t.Errorf("Old error: expected HealthGood, got %v", status)
	}
}

func TestCalculateQualityScore(t *testing.T) {
	// Perfect score
	score := CalculateQualityScore(100.0, 1.0, 0, 1*time.Hour)
	if score < 90 {
		t.Errorf("Perfect conditions: expected score >= 90, got %.1f", score)
	}

	// Poor score
	score = CalculateQualityScore(50.0, 0.0, 10, 1*time.Hour)
	if score > 50 {
		t.Errorf("Poor conditions: expected score <= 50, got %.1f", score)
	}
}

func TestMaxRecentLogs(t *testing.T) {
	m := New()

	// Add more than max logs
	for i := 0; i < MaxRecentLogs+10; i++ {
		entry := models.LogEntry{
			Timestamp: time.Now().Format(time.RFC3339),
			Level:     models.LogLevelINFO,
			Message:   "Test",
		}
		m.ProcessLog(entry)
	}

	if len(m.Metrics.RecentLogs) != MaxRecentLogs {
		t.Errorf("Expected %d logs (max), got %d", MaxRecentLogs, len(m.Metrics.RecentLogs))
	}
}

func TestMaxRecentEvents(t *testing.T) {
	m := New()
	m.Metrics.StartTime = time.Now()

	// Add more than max events
	for i := 0; i < MaxRecentEvents+10; i++ {
		entry := models.EventEntry{
			Timestamp:    time.Now().Format(time.RFC3339),
			EventType:    "test",
			Deserialized: true,
		}
		m.ProcessEvent(entry)
	}

	if len(m.Metrics.RecentEvents) != MaxRecentEvents {
		t.Errorf("Expected %d events (max), got %d", MaxRecentEvents, len(m.Metrics.RecentEvents))
	}
}

func TestFormatUptime(t *testing.T) {
	tests := []struct {
		duration time.Duration
		contains string
	}{
		{30 * time.Second, "s"},
		{5 * time.Minute, "m"},
		{2 * time.Hour, "h"},
	}

	for _, tt := range tests {
		result := formatUptime(tt.duration)
		if len(result) == 0 {
			t.Errorf("formatUptime(%v) returned empty string", tt.duration)
		}
	}
}

func TestCreateWidgets(t *testing.T) {
	// Test widget creation doesn't panic
	table := CreateMetricsTable()
	if table == nil {
		t.Error("CreateMetricsTable returned nil")
	}

	dashboard := CreateHealthDashboard()
	if dashboard == nil {
		t.Error("CreateHealthDashboard returned nil")
	}

	logList := CreateLogList()
	if logList == nil {
		t.Error("CreateLogList returned nil")
	}

	eventList := CreateEventList()
	if eventList == nil {
		t.Error("CreateEventList returned nil")
	}

	mpsChart := CreateMessagesPerSecondChart()
	if mpsChart == nil {
		t.Error("CreateMessagesPerSecondChart returned nil")
	}

	srChart := CreateSuccessRateChart()
	if srChart == nil {
		t.Error("CreateSuccessRateChart returned nil")
	}
}

func TestUpdateLists(t *testing.T) {
	// Test with empty data
	logList := CreateLogList()
	UpdateLogList(logList, []models.LogEntry{})
	if len(logList.Rows) != 1 || logList.Rows[0] != "En attente de logs..." {
		t.Error("Empty log list should show waiting message")
	}

	eventList := CreateEventList()
	UpdateEventList(eventList, []models.EventEntry{})
	if len(eventList.Rows) != 1 || eventList.Rows[0] != "En attente d'événements..." {
		t.Error("Empty event list should show waiting message")
	}

	// Test with data
	logs := []models.LogEntry{
		{Timestamp: "2024-01-01T10:00:00Z", Level: models.LogLevelINFO, Message: "Test"},
	}
	UpdateLogList(logList, logs)
	if len(logList.Rows) != 1 {
		t.Errorf("Expected 1 log row, got %d", len(logList.Rows))
	}

	events := []models.EventEntry{
		{Timestamp: "2024-01-01T10:00:00Z", EventType: "test", Deserialized: true, KafkaOffset: 1},
	}
	UpdateEventList(eventList, events)
	if len(eventList.Rows) != 1 {
		t.Errorf("Expected 1 event row, got %d", len(eventList.Rows))
	}
}

func TestUpdateCharts(t *testing.T) {
	mpsChart := CreateMessagesPerSecondChart()
	srChart := CreateSuccessRateChart()

	// Test with empty data
	UpdateCharts(mpsChart, srChart, []float64{}, []float64{})
	if len(mpsChart.Data[0]) != 1 || mpsChart.Data[0][0] != 0 {
		t.Error("Empty MPS chart should have [0]")
	}

	// Test with data
	mps := []float64{1.0, 2.0, 3.0}
	sr := []float64{90.0, 95.0, 100.0}
	UpdateCharts(mpsChart, srChart, mps, sr)
	if len(mpsChart.Data[0]) != 3 {
		t.Errorf("Expected 3 MPS data points, got %d", len(mpsChart.Data[0]))
	}
}
