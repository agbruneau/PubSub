package monitor

import (
	"testing"
	"time"

	"github.com/agbruneau/PubSub/pkg/models"
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"github.com/stretchr/testify/assert"
)

// TestUpdateUI vérifie que UpdateUI ne panique pas et met à jour les widgets.
func TestUpdateUI(t *testing.T) {
	// Setup
	m := New()
	m.Metrics.MessagesReceived = 100
	m.Metrics.MessagesProcessed = 90
	m.Metrics.MessagesFailed = 10
	m.Metrics.CurrentMessagesPerSec = 5.5
	m.Metrics.CurrentSuccessRate = 90.0
	m.Metrics.RecentLogs = append(m.Metrics.RecentLogs, models.LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     models.LogLevelINFO,
		Message:   "Test log",
	})
	m.Metrics.RecentEvents = append(m.Metrics.RecentEvents, models.EventEntry{
		Timestamp:    time.Now().Format(time.RFC3339),
		EventType:    "Test event",
		Deserialized: true,
	})
	m.Metrics.MessagesPerSecond = []float64{1.0, 5.5}
	m.Metrics.SuccessRateHistory = []float64{100.0, 90.0}

	table := CreateMetricsTable()
	healthDashboard := CreateHealthDashboard()
	logList := CreateLogList()
	eventList := CreateEventList()
	mpsChart := CreateMessagesPerSecondChart()
	srChart := CreateSuccessRateChart()

	// Exécuter
	m.UpdateUI(table, healthDashboard, logList, eventList, mpsChart, srChart)

	// Vérifier
	assert.Equal(t, "100", table.Rows[1][1]) // Messages reçus
	assert.Equal(t, "90", table.Rows[2][1])  // Messages traités
	assert.Equal(t, "10", table.Rows[3][1])  // Messages échoués

	// Vérifier listes
	assert.NotEmpty(t, logList.Rows)
	assert.NotEmpty(t, eventList.Rows)
	assert.NotEqual(t, "En attente de logs...", logList.Rows[0])

	// Vérifier charts
	assert.Equal(t, 2, len(mpsChart.Data[0]))
	assert.Equal(t, 2, len(srChart.Data[0]))
}

// TestGetGlobalHealthStatus vérifie la logique de santé globale.
func TestGetGlobalHealthStatus(t *testing.T) {
	tests := []struct {
		success    HealthStatus
		throughput HealthStatus
		errorSt    HealthStatus
		expected   HealthStatus
		color      ui.Color
	}{
		{HealthGood, HealthGood, HealthGood, HealthGood, ui.ColorGreen},
		{HealthWarning, HealthGood, HealthGood, HealthWarning, ui.ColorYellow},
		{HealthGood, HealthCritical, HealthGood, HealthCritical, ui.ColorRed},
		{HealthGood, HealthGood, HealthCritical, HealthCritical, ui.ColorRed},
		{HealthWarning, HealthWarning, HealthWarning, HealthWarning, ui.ColorYellow},
	}

	for _, tt := range tests {
		status, _, color := getGlobalHealthStatus(tt.success, tt.throughput, tt.errorSt)
		assert.Equal(t, tt.expected, status)
		assert.Equal(t, tt.color, color)
	}
}

// TestGetQualityText vérifie les textes de qualité.
func TestGetQualityText(t *testing.T) {
	tests := []struct {
		score    float64
		expected string
		color    ui.Color
	}{
		{95.0, "EXCELLENT (95)", ui.ColorGreen},
		{85.0, "BON (85)", ui.ColorYellow},
		{65.0, "MOYEN (65)", ui.ColorYellow},
		{40.0, "FAIBLE (40)", ui.ColorRed},
	}

	for _, tt := range tests {
		text, color := getQualityText(tt.score)
		assert.Equal(t, tt.expected, text)
		assert.Equal(t, tt.color, color)
	}
}

// TestFormatLogRow vérifie le formatage des logs.
func TestFormatLogRow(t *testing.T) {
	entry := models.LogEntry{
		Timestamp: "2024-01-01T12:00:00Z",
		Level:     models.LogLevelERROR,
		Message:   "Erreur critique",
	}

	row := formatLogRow(entry)
	assert.Contains(t, row, "🔴")
	assert.Contains(t, row, "12:00:00")
	assert.Contains(t, row, "Erreur critique")
}

// TestFormatEventRow vérifie le formatage des événements.
func TestFormatEventRow(t *testing.T) {
	entry := models.EventEntry{
		Timestamp:    "2024-01-01T12:00:00Z",
		EventType:    "order_created",
		Deserialized: true,
		KafkaOffset:  123,
	}

	row := formatEventRow(entry)
	assert.Contains(t, row, "✅")
	assert.Contains(t, row, "12:00:00")
	assert.Contains(t, row, "Offset: 123")
	assert.Contains(t, row, "order_created")

	// Test échec
	entry.Deserialized = false
	row = formatEventRow(entry)
	assert.Contains(t, row, "❌")
}

// TestUpdateMetricsTable vérifie la mise à jour de la table de métriques.
func TestUpdateMetricsTable(t *testing.T) {
	table := widgets.NewTable()
	metrics := &Metrics{
		MessagesReceived:      100,
		MessagesProcessed:     95,
		MessagesFailed:        5,
		CurrentMessagesPerSec: 10.5,
		CurrentSuccessRate:    95.0,
		LastUpdateTime:        time.Now(),
	}

	UpdateMetricsTable(table, metrics)

	assert.Equal(t, "100", table.Rows[1][1])
	assert.Equal(t, "10.50", table.Rows[4][1])
	assert.Equal(t, "95.00%", table.Rows[5][1])
}

// TestUpdateHealthDashboard vérifie la mise à jour du dashboard.
func TestUpdateHealthDashboard(t *testing.T) {
	dashboard := widgets.NewTable()
	metrics := &Metrics{
		CurrentSuccessRate:    100.0,
		CurrentMessagesPerSec: 5.0,
		ErrorCount:            0,
		Uptime:                1 * time.Hour,
	}

	UpdateHealthDashboard(dashboard, metrics)

	// Vérifier que les styles sont appliqués
	assert.NotEmpty(t, dashboard.RowStyles)
	assert.Equal(t, "● EXCELLENT", dashboard.Rows[1][1])
}
