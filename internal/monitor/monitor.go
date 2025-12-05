/*
Package monitor provides a real-time TUI log monitor for the PubSub system.

This package implements file monitoring, log analysis, metrics visualization,
and an interactive terminal user interface using termui widgets.
*/
package monitor

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/agbruneau/PubSub/internal/config"
	"github.com/agbruneau/PubSub/pkg/models"
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

// HealthStatus defines health levels for dashboard indicators.
type HealthStatus int

const (
	// HealthGood indicates a healthy condition, typically displayed in green.
	HealthGood HealthStatus = iota
	// HealthWarning indicates a warning, typically displayed in yellow.
	HealthWarning
	// HealthCritical indicates a critical state, typically displayed in red.
	HealthCritical
)

// Local aliases for readability
const (
	MaxRecentLogs           = config.MonitorMaxRecentLogs
	MaxRecentEvents         = config.MonitorMaxRecentEvents
	MaxHistorySize          = config.MonitorMaxHistorySize
	LogChannelBuffer        = config.MonitorLogChannelBuffer
	EventChannelBuffer      = config.MonitorEventChannelBuffer
	SuccessRateExcellent    = config.MonitorSuccessRateExcellent
	SuccessRateGood         = config.MonitorSuccessRateGood
	ThroughputNormal        = config.MonitorThroughputNormal
	ThroughputLow           = config.MonitorThroughputLow
	ErrorTimeoutCritical    = config.MonitorErrorTimeoutCritical
	ErrorTimeoutWarning     = config.MonitorErrorTimeoutWarning
	QualityThroughputHigh   = config.MonitorQualityThroughputHigh
	QualityThroughputMedium = config.MonitorQualityThroughputMedium
	QualityThroughputLow    = config.MonitorQualityThroughputLow
	QualityScoreExcellent   = config.MonitorQualityScoreExcellent
	QualityScoreGood        = config.MonitorQualityScoreGood
	QualityScoreMedium      = config.MonitorQualityScoreMedium
	FileCheckInterval       = config.MonitorFileCheckInterval
	FilePollInterval        = config.MonitorFilePollInterval
	UIUpdateInterval        = config.MonitorUIUpdateInterval
	MaxLogRowLength         = config.MonitorMaxLogRowLength
	MaxEventRowLength       = config.MonitorMaxEventRowLength
	TruncateSuffix          = config.MonitorTruncateSuffix
)

// Metrics aggregates and manages the state of all metrics collected by the monitor.
type Metrics struct {
	mu                    sync.RWMutex
	StartTime             time.Time           // Monitor start time.
	MessagesReceived      int64               // Total number of messages received.
	MessagesProcessed     int64               // Total number of messages processed successfully.
	MessagesFailed        int64               // Total number of failed messages.
	MessagesPerSecond     []float64           // Message throughput history.
	SuccessRateHistory    []float64           // Success rate history.
	RecentLogs            []models.LogEntry   // List of recent logs.
	RecentEvents          []models.EventEntry // List of recent events.
	LastUpdateTime        time.Time           // Last metrics update time.
	Uptime                time.Duration       // Uptime duration.
	CurrentMessagesPerSec float64             // Current throughput.
	CurrentSuccessRate    float64             // Current success rate.
	ErrorCount            int64               // Total number of errors.
	LastErrorTime         time.Time           // Time of the last error.
}

// Monitor encapsulates all monitoring functionalities.
type Monitor struct {
	Metrics *Metrics // The monitored metrics.
}

// New creates a new Monitor instance.
//
// Returns:
//   - *Monitor: A new initialized Monitor instance.
func New() *Monitor {
	return &Monitor{
		Metrics: &Metrics{
			StartTime:          time.Now(),
			RecentLogs:         make([]models.LogEntry, 0, MaxRecentLogs),
			RecentEvents:       make([]models.EventEntry, 0, MaxRecentEvents),
			MessagesPerSecond:  make([]float64, 0, MaxHistorySize),
			SuccessRateHistory: make([]float64, 0, MaxHistorySize),
			LastErrorTime:      time.Time{},
		},
	}
}

// WaitForFile waits for the specified file to exist and returns an open file descriptor.
// This function blocks until the file is accessible.
//
// Parameters:
//   - filename: The path of the file to wait for.
//
// Returns:
//   - *os.File: The open file descriptor.
func WaitForFile(filename string) *os.File {
	for {
		file, err := os.Open(filename)
		if err == nil {
			return file
		}
		time.Sleep(FileCheckInterval)
	}
}

// waitForFileRecreation waits for a deleted file to be recreated.
//
// Parameters:
//   - filename: The path of the file to wait for.
//
// Returns:
//   - *os.File: The open file descriptor.
func waitForFileRecreation(filename string) *os.File {
	for {
		time.Sleep(FileCheckInterval)
		file, err := os.Open(filename)
		if err == nil {
			return file
		}
	}
}

// parseAndSendLogEntry parses a JSON line and sends it to the appropriate channel.
//
// Parameters:
//   - line: The JSON text line to parse.
//   - logChan: The channel to send the parsed log entry to.
func parseAndSendLogEntry(line string, logChan chan<- models.LogEntry) {
	var entry models.LogEntry
	if err := json.Unmarshal([]byte(line), &entry); err == nil {
		select {
		case logChan <- entry:
		default:
			// Channel full, ignore
		}
	}
}

// parseAndSendEventEntry parses a JSON line and sends it to the appropriate channel.
//
// Parameters:
//   - line: The JSON text line to parse.
//   - eventChan: The channel to send the parsed event entry to.
func parseAndSendEventEntry(line string, eventChan chan<- models.EventEntry) {
	var entry models.EventEntry
	if err := json.Unmarshal([]byte(line), &entry); err == nil {
		select {
		case eventChan <- entry:
		default:
			// Channel full, ignore
		}
	}
}

// readNewLines reads new lines from the file and sends them to the channels.
//
// Parameters:
//   - file: The file descriptor.
//   - filename: The file name (to identify the log type).
//   - currentPos: The current reading position in the file.
//   - logChan: The channel for logs.
//   - eventChan: The channel for events.
//
// Returns:
//   - int64: The new reading position.
func readNewLines(file *os.File, filename string, currentPos int64, logChan chan<- models.LogEntry, eventChan chan<- models.EventEntry) int64 {
	_, err := file.Seek(currentPos, 0)
	if err != nil {
		return currentPos
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		if filename == config.TrackerLogFile {
			parseAndSendLogEntry(line, logChan)
		} else if filename == config.TrackerEventsFile {
			parseAndSendEventEntry(line, eventChan)
		}
	}

	if err := scanner.Err(); err != nil {
		return currentPos
	}

	newPos, err := file.Seek(0, os.SEEK_CUR)
	if err != nil {
		return currentPos
	}
	return newPos
}

// MonitorFile continuously monitors a file, similar to `tail -f`.
//
// Parameters:
//   - filename: The path of the file to monitor.
//   - logChan: The channel to send logs to.
//   - eventChan: The channel to send events to.
func MonitorFile(filename string, logChan chan<- models.LogEntry, eventChan chan<- models.EventEntry) {
	file := WaitForFile(filename)
	var currentPos int64

	for {
		stat, err := os.Stat(filename)
		if err != nil {
			file.Close()
			file = waitForFileRecreation(filename)
			currentPos = 0
			continue
		}

		if stat.Size() < currentPos {
			file.Close()
			file = WaitForFile(filename)
			currentPos = 0
		}

		if currentPos < stat.Size() {
			newPos := readNewLines(file, filename, currentPos, logChan, eventChan)
			file.Close()
			file = WaitForFile(filename)
			currentPos = newPos
		} else {
			time.Sleep(FilePollInterval)
		}
	}
}

// ProcessLog processes a log entry from tracker.log and updates metrics.
//
// Parameters:
//   - entry: The log entry to process.
func (m *Monitor) ProcessLog(entry models.LogEntry) {
	m.Metrics.mu.Lock()
	defer m.Metrics.mu.Unlock()

	m.Metrics.RecentLogs = append(m.Metrics.RecentLogs, entry)
	if len(m.Metrics.RecentLogs) > MaxRecentLogs {
		m.Metrics.RecentLogs = m.Metrics.RecentLogs[1:]
	}

	if entry.Level == models.LogLevelERROR {
		m.Metrics.ErrorCount++
		m.Metrics.LastErrorTime = time.Now()
	}

	if entry.Message == "Métriques système périodiques" && entry.Metadata != nil {
		if msgsReceived, ok := entry.Metadata["messages_received"].(float64); ok {
			m.Metrics.MessagesReceived = int64(msgsReceived)
		}
		if msgsProcessed, ok := entry.Metadata["messages_processed"].(float64); ok {
			m.Metrics.MessagesProcessed = int64(msgsProcessed)
		}
		if msgsFailed, ok := entry.Metadata["messages_failed"].(float64); ok {
			m.Metrics.MessagesFailed = int64(msgsFailed)
		}
		if mpsStr, ok := entry.Metadata["messages_per_second"].(string); ok {
			if mps, err := strconv.ParseFloat(mpsStr, 64); err == nil {
				m.Metrics.MessagesPerSecond = append(m.Metrics.MessagesPerSecond, mps)
				if len(m.Metrics.MessagesPerSecond) > MaxHistorySize {
					m.Metrics.MessagesPerSecond = m.Metrics.MessagesPerSecond[1:]
				}
				m.Metrics.CurrentMessagesPerSec = mps
			}
		}
		if srStr, ok := entry.Metadata["success_rate_percent"].(string); ok {
			if sr, err := strconv.ParseFloat(srStr, 64); err == nil {
				m.Metrics.SuccessRateHistory = append(m.Metrics.SuccessRateHistory, sr)
				if len(m.Metrics.SuccessRateHistory) > MaxHistorySize {
					m.Metrics.SuccessRateHistory = m.Metrics.SuccessRateHistory[1:]
				}
				m.Metrics.CurrentSuccessRate = sr
			}
		}
	}

	m.Metrics.LastUpdateTime = time.Now()
}

// ProcessEvent processes an event entry from tracker.events and updates metrics.
//
// Parameters:
//   - entry: The event entry to process.
func (m *Monitor) ProcessEvent(entry models.EventEntry) {
	m.Metrics.mu.Lock()
	defer m.Metrics.mu.Unlock()

	m.Metrics.RecentEvents = append(m.Metrics.RecentEvents, entry)
	if len(m.Metrics.RecentEvents) > MaxRecentEvents {
		m.Metrics.RecentEvents = m.Metrics.RecentEvents[1:]
	}

	if entry.Deserialized {
		m.Metrics.MessagesProcessed++
	} else {
		m.Metrics.MessagesFailed++
		m.Metrics.ErrorCount++
		m.Metrics.LastErrorTime = time.Now()
	}
	m.Metrics.MessagesReceived++

	uptime := time.Since(m.Metrics.StartTime)
	if uptime.Seconds() > 0 {
		m.Metrics.CurrentMessagesPerSec = float64(m.Metrics.MessagesReceived) / uptime.Seconds()
	}
	if m.Metrics.MessagesReceived > 0 {
		m.Metrics.CurrentSuccessRate = float64(m.Metrics.MessagesProcessed) / float64(m.Metrics.MessagesReceived) * 100
	}

	m.Metrics.LastUpdateTime = time.Now()
}

// StatusThreshold defines a threshold for status evaluation.
type StatusThreshold struct {
	MinValue float64      // The minimum value for this threshold.
	Status   HealthStatus // The associated health status.
	Text     string       // The text to display.
	Color    ui.Color     // The color to use.
}

// evaluateStatus evaluates a value against ordered thresholds.
//
// Parameters:
//   - value: The value to evaluate.
//   - thresholds: The list of thresholds.
//
// Returns:
//   - HealthStatus: The health status.
//   - string: The associated text.
//   - ui.Color: The associated color.
func evaluateStatus(value float64, thresholds []StatusThreshold) (HealthStatus, string, ui.Color) {
	for _, t := range thresholds {
		if value >= t.MinValue {
			return t.Status, t.Text, t.Color
		}
	}
	if len(thresholds) > 0 {
		last := thresholds[len(thresholds)-1]
		return last.Status, last.Text, last.Color
	}
	return HealthCritical, "● INCONNU", ui.ColorRed
}

var (
	healthThresholds = []StatusThreshold{
		{SuccessRateExcellent, HealthGood, "● EXCELLENT", ui.ColorGreen},
		{SuccessRateGood, HealthWarning, "● BON", ui.ColorYellow},
		{0, HealthCritical, "● CRITIQUE", ui.ColorRed},
	}

	throughputThresholds = []StatusThreshold{
		{ThroughputNormal, HealthGood, "● NORMAL", ui.ColorGreen},
		{ThroughputLow, HealthWarning, "● FAIBLE", ui.ColorYellow},
		{0, HealthCritical, "● ARRÊTÉ", ui.ColorRed},
	}
)

// GetHealthStatus evaluates the success rate and returns a health status.
//
// Parameters:
//   - successRate: The success rate in percentage.
//
// Returns:
//   - HealthStatus: The health status.
//   - string: The status text.
//   - ui.Color: The status color.
func GetHealthStatus(successRate float64) (HealthStatus, string, ui.Color) {
	return evaluateStatus(successRate, healthThresholds)
}

// GetThroughputStatus evaluates the message throughput and returns a health status.
//
// Parameters:
//   - mps: The throughput in messages per second.
//
// Returns:
//   - HealthStatus: The health status.
//   - string: The status text.
//   - ui.Color: The status color.
func GetThroughputStatus(mps float64) (HealthStatus, string, ui.Color) {
	return evaluateStatus(mps, throughputThresholds)
}

// GetErrorStatus evaluates errors and returns a health status.
//
// Parameters:
//   - errorCount: The total number of errors.
//   - lastErrorTime: The time of the last error.
//
// Returns:
//   - HealthStatus: The health status.
//   - string: The status text.
//   - ui.Color: The status color.
func GetErrorStatus(errorCount int64, lastErrorTime time.Time) (HealthStatus, string, ui.Color) {
	if errorCount == 0 {
		return HealthGood, "● AUCUN", ui.ColorGreen
	}

	timeSinceError := time.Since(lastErrorTime)
	if timeSinceError > ErrorTimeoutWarning {
		return HealthGood, "● AUCUN", ui.ColorGreen
	} else if timeSinceError > ErrorTimeoutCritical {
		return HealthWarning, "● RÉCENT", ui.ColorYellow
	}
	return HealthCritical, "● ACTIF", ui.ColorRed
}

// CalculateQualityScore calculates a global quality score (0-100).
//
// Parameters:
//   - successRate: The success rate in percentage.
//   - mps: The throughput in messages per second.
//   - errorCount: The number of errors.
//   - uptime: The uptime duration.
//
// Returns:
//   - float64: The calculated quality score.
func CalculateQualityScore(successRate, mps float64, errorCount int64, uptime time.Duration) float64 {
	successScore := (successRate / 100.0) * 50.0

	throughputScore := 0.0
	if mps >= QualityThroughputHigh {
		throughputScore = 30.0
	} else if mps >= QualityThroughputMedium {
		throughputScore = 25.0
	} else if mps >= QualityThroughputLow {
		throughputScore = 15.0
	} else if mps > 0 {
		throughputScore = 10.0
	}

	errorScore := 20.0
	if errorCount > 0 {
		errorPenalty := float64(errorCount) * 2.0
		if errorPenalty > 20.0 {
			errorPenalty = 20.0
		}
		errorScore = 20.0 - errorPenalty
		if errorScore < 0 {
			errorScore = 0
		}
	}

	return successScore + throughputScore + errorScore
}

// CreateMetricsTable initializes the metrics table widget.
//
// Returns:
//   - *widgets.Table: The initialized table widget.
func CreateMetricsTable() *widgets.Table {
	table := widgets.NewTable()
	table.Rows = [][]string{
		{"Métrique", "Valeur"},
		{"Messages reçus", "0"},
		{"Messages traités", "0"},
		{"Messages échoués", "0"},
		{"Débit (msg/s)", "0.00"},
		{"Taux de succès", "0.00%"},
		{"Dernière màj", "-"},
	}
	table.TextStyle = ui.NewStyle(ui.ColorWhite)
	table.RowStyles[0] = ui.NewStyle(ui.ColorYellow, ui.ColorClear, ui.ModifierBold)
	table.SetRect(0, 0, 50, 9)
	table.ColumnWidths = []int{30, 20}
	return table
}

// CreateHealthDashboard initializes the health dashboard widget.
//
// Returns:
//   - *widgets.Table: The initialized table widget.
func CreateHealthDashboard() *widgets.Table {
	table := widgets.NewTable()
	table.Rows = [][]string{
		{"Indicateur", "Statut"},
		{"Santé Globale", "●"},
		{"Taux Succès", "●"},
		{"Débit", "●"},
		{"Erreurs", "●"},
		{"Uptime", "-"},
		{"Qualité", "-"},
	}
	table.TextStyle = ui.NewStyle(ui.ColorWhite)
	table.RowStyles[0] = ui.NewStyle(ui.ColorYellow, ui.ColorClear, ui.ModifierBold)
	table.SetRect(50, 0, 110, 9)
	table.ColumnWidths = []int{25, 35}
	return table
}

// CreateLogList initializes the log list widget.
//
// Returns:
//   - *widgets.List: The initialized list widget.
func CreateLogList() *widgets.List {
	list := widgets.NewList()
	list.Title = "Logs Récents (tracker.log)"
	list.Rows = []string{"En attente de logs..."}
	list.TextStyle = ui.NewStyle(ui.ColorWhite)
	list.SelectedRowStyle = ui.NewStyle(ui.ColorBlack, ui.ColorWhite)
	list.WrapText = true
	list.SetRect(0, 9, 80, 19)
	return list
}

// CreateEventList initializes the event list widget.
//
// Returns:
//   - *widgets.List: The initialized list widget.
func CreateEventList() *widgets.List {
	list := widgets.NewList()
	list.Title = "Événements Récents (tracker.events)"
	list.Rows = []string{"En attente d'événements..."}
	list.TextStyle = ui.NewStyle(ui.ColorWhite)
	list.SelectedRowStyle = ui.NewStyle(ui.ColorBlack, ui.ColorWhite)
	list.WrapText = true
	list.SetRect(80, 9, 160, 19)
	return list
}

// CreateMessagesPerSecondChart initializes the throughput chart widget.
//
// Returns:
//   - *widgets.Plot: The initialized plot widget.
func CreateMessagesPerSecondChart() *widgets.Plot {
	plot := widgets.NewPlot()
	plot.Title = "Débit Messages (msg/s)"
	plot.Data = [][]float64{{}}
	plot.SetRect(0, 19, 80, 29)
	plot.AxesColor = ui.ColorWhite
	plot.LineColors[0] = ui.ColorGreen
	plot.Marker = widgets.MarkerDot
	return plot
}

// CreateSuccessRateChart initializes the success rate chart widget.
//
// Returns:
//   - *widgets.Plot: The initialized plot widget.
func CreateSuccessRateChart() *widgets.Plot {
	plot := widgets.NewPlot()
	plot.Title = "Taux de Succès (%)"
	plot.Data = [][]float64{{}}
	plot.SetRect(80, 19, 160, 29)
	plot.AxesColor = ui.ColorWhite
	plot.LineColors[0] = ui.ColorBlue
	plot.Marker = widgets.MarkerDot
	return plot
}

// UpdateMetricsTable updates the metrics table.
//
// Parameters:
//   - table: The table widget to update.
//   - m: The current metrics.
func UpdateMetricsTable(table *widgets.Table, m *Metrics) {
	table.Rows = [][]string{
		{"Métrique", "Valeur"},
		{"Messages reçus", fmt.Sprintf("%d", m.MessagesReceived)},
		{"Messages traités", fmt.Sprintf("%d", m.MessagesProcessed)},
		{"Messages échoués", fmt.Sprintf("%d", m.MessagesFailed)},
		{"Débit (msg/s)", fmt.Sprintf("%.2f", m.CurrentMessagesPerSec)},
		{"Taux de succès", fmt.Sprintf("%.2f%%", m.CurrentSuccessRate)},
		{"Dernière màj", m.LastUpdateTime.Format("15:04:05")},
	}
}

// getGlobalHealthStatus determines the global health from individual statuses.
//
// Parameters:
//   - successStatus: The success rate status.
//   - throughputStatus: The throughput status.
//   - errorStatus: The error status.
//
// Returns:
//   - HealthStatus: The global status.
//   - string: The status text.
//   - ui.Color: The status color.
func getGlobalHealthStatus(successStatus, throughputStatus, errorStatus HealthStatus) (HealthStatus, string, ui.Color) {
	globalStatus := successStatus
	if throughputStatus > globalStatus {
		globalStatus = throughputStatus
	}
	if errorStatus > globalStatus {
		globalStatus = errorStatus
	}

	switch globalStatus {
	case HealthWarning:
		return globalStatus, "● ATTENTION", ui.ColorYellow
	case HealthCritical:
		return globalStatus, "● CRITIQUE", ui.ColorRed
	default:
		return globalStatus, "● EXCELLENT", ui.ColorGreen
	}
}

// getQualityText returns the text and color for a quality score.
//
// Parameters:
//   - qualityScore: The quality score (0-100).
//
// Returns:
//   - string: The qualitative text.
//   - ui.Color: The associated color.
func getQualityText(qualityScore float64) (string, ui.Color) {
	if qualityScore >= QualityScoreExcellent {
		return fmt.Sprintf("EXCELLENT (%.0f)", qualityScore), ui.ColorGreen
	} else if qualityScore >= QualityScoreGood {
		return fmt.Sprintf("BON (%.0f)", qualityScore), ui.ColorYellow
	} else if qualityScore >= QualityScoreMedium {
		return fmt.Sprintf("MOYEN (%.0f)", qualityScore), ui.ColorYellow
	}
	return fmt.Sprintf("FAIBLE (%.0f)", qualityScore), ui.ColorRed
}

// formatUptime formats the uptime duration into a readable string.
//
// Parameters:
//   - uptime: The uptime duration.
//
// Returns:
//   - string: The formatted string (e.g., "2.5h", "45m", "30s").
func formatUptime(uptime time.Duration) string {
	if uptime.Hours() >= 1 {
		return fmt.Sprintf("%.1fh", uptime.Hours())
	} else if uptime.Minutes() >= 1 {
		return fmt.Sprintf("%.0fm", uptime.Minutes())
	}
	return fmt.Sprintf("%.0fs", uptime.Seconds())
}

// UpdateHealthDashboard updates the health dashboard.
//
// Parameters:
//   - dashboard: The table widget to update.
//   - m: The current metrics.
func UpdateHealthDashboard(dashboard *widgets.Table, m *Metrics) {
	successStatus, successText, successColor := GetHealthStatus(m.CurrentSuccessRate)
	throughputStatus, throughputText, throughputColor := GetThroughputStatus(m.CurrentMessagesPerSec)
	errorStatus, errorText, errorColor := GetErrorStatus(m.ErrorCount, m.LastErrorTime)

	_, globalText, globalColor := getGlobalHealthStatus(successStatus, throughputStatus, errorStatus)

	qualityScore := CalculateQualityScore(m.CurrentSuccessRate, m.CurrentMessagesPerSec, m.ErrorCount, m.Uptime)
	qualityText, qualityColor := getQualityText(qualityScore)
	uptimeStr := formatUptime(m.Uptime)

	dashboard.Rows = [][]string{
		{"Indicateur", "Statut"},
		{"Santé Globale", globalText},
		{"Taux Succès", successText},
		{"Débit", throughputText},
		{"Erreurs", errorText},
		{"Uptime", uptimeStr},
		{"Qualité", qualityText},
	}

	dashboard.RowStyles = make(map[int]ui.Style)
	dashboard.RowStyles[0] = ui.NewStyle(ui.ColorYellow, ui.ColorClear, ui.ModifierBold)
	dashboard.RowStyles[1] = ui.NewStyle(globalColor, ui.ColorClear, ui.ModifierBold)
	dashboard.RowStyles[2] = ui.NewStyle(successColor, ui.ColorClear)
	dashboard.RowStyles[3] = ui.NewStyle(throughputColor, ui.ColorClear)
	dashboard.RowStyles[4] = ui.NewStyle(errorColor, ui.ColorClear)
	dashboard.RowStyles[5] = ui.NewStyle(ui.ColorCyan, ui.ColorClear)
	dashboard.RowStyles[6] = ui.NewStyle(qualityColor, ui.ColorClear, ui.ModifierBold)
}

// formatLogRow formats a log entry for display.
//
// Parameters:
//   - log: The log entry.
//
// Returns:
//   - string: The formatted line for the UI.
func formatLogRow(log models.LogEntry) string {
	levelIcon := "🟢"
	if log.Level == models.LogLevelERROR {
		levelIcon = "🔴"
	}

	timeStr := log.Timestamp
	if len(timeStr) > 19 {
		timeStr = timeStr[11:19]
	}

	row := fmt.Sprintf("%s [%s] %s", levelIcon, timeStr, log.Message)
	if len(row) > MaxLogRowLength {
		row = row[:MaxLogRowLength-len(TruncateSuffix)] + TruncateSuffix
	}
	return row
}

// UpdateLogList updates the list of recent logs.
//
// Parameters:
//   - list: The list widget to update.
//   - logs: The list of recent logs.
func UpdateLogList(list *widgets.List, logs []models.LogEntry) {
	rows := make([]string, 0, len(logs))
	for i := len(logs) - 1; i >= 0; i-- {
		rows = append(rows, formatLogRow(logs[i]))
	}
	if len(rows) == 0 {
		rows = []string{"En attente de logs..."}
	}
	list.Rows = rows
}

// formatEventRow formats an event entry for display.
//
// Parameters:
//   - event: The event entry.
//
// Returns:
//   - string: The formatted line for the UI.
func formatEventRow(event models.EventEntry) string {
	status := "❌"
	if event.Deserialized {
		status = "✅"
	}

	timeStr := event.Timestamp
	if len(timeStr) > 19 {
		timeStr = timeStr[11:19]
	}

	row := fmt.Sprintf("%s [%s] Offset: %d | %s", status, timeStr, event.KafkaOffset, event.EventType)
	if len(row) > MaxEventRowLength {
		row = row[:MaxEventRowLength-len(TruncateSuffix)] + TruncateSuffix
	}
	return row
}

// UpdateEventList updates the list of recent events.
//
// Parameters:
//   - list: The list widget to update.
//   - events: The list of recent events.
func UpdateEventList(list *widgets.List, events []models.EventEntry) {
	rows := make([]string, 0, len(events))
	for i := len(events) - 1; i >= 0; i-- {
		rows = append(rows, formatEventRow(events[i]))
	}
	if len(rows) == 0 {
		rows = []string{"En attente d'événements..."}
	}
	list.Rows = rows
}

// UpdateCharts updates the throughput and success rate charts.
//
// Parameters:
//   - mpsChart: The throughput chart widget.
//   - srChart: The success rate chart widget.
//   - mps: Throughput history.
//   - sr: Success rate history.
func UpdateCharts(mpsChart, srChart *widgets.Plot, mps, sr []float64) {
	if len(mps) > 0 {
		mpsChart.Data = [][]float64{mps}
	} else {
		mpsChart.Data = [][]float64{{0}}
	}

	if len(sr) > 0 {
		srChart.Data = [][]float64{sr}
	} else {
		srChart.Data = [][]float64{{0}}
	}
}

// UpdateUI refreshes all UI widgets with the latest metrics.
//
// Parameters:
//   - table: The metrics table.
//   - healthDashboard: The health dashboard.
//   - logList: The log list.
//   - eventList: The event list.
//   - mpsChart: The throughput chart.
//   - srChart: Le graphique de taux de succès.
func (m *Monitor) UpdateUI(table *widgets.Table, healthDashboard *widgets.Table, logList *widgets.List, eventList *widgets.List, mpsChart *widgets.Plot, srChart *widgets.Plot) {
	m.Metrics.mu.RLock()
	defer m.Metrics.mu.RUnlock()

	UpdateMetricsTable(table, m.Metrics)
	UpdateHealthDashboard(healthDashboard, m.Metrics)
	UpdateLogList(logList, m.Metrics.RecentLogs)
	UpdateEventList(eventList, m.Metrics.RecentEvents)
	UpdateCharts(mpsChart, srChart, m.Metrics.MessagesPerSecond, m.Metrics.SuccessRateHistory)
}
