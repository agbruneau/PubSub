/*
Package monitor provides a real-time log monitoring TUI for the PubSub system.

This package implements file watching, log parsing, metrics visualization,
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
	HealthGood     HealthStatus = iota // Indicates a healthy condition, typically displayed in green.
	HealthWarning                      // Indicates a warning, typically displayed in yellow.
	HealthCritical                     // Indicates a critical state, typically displayed in red.
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
	StartTime             time.Time
	MessagesReceived      int64
	MessagesProcessed     int64
	MessagesFailed        int64
	MessagesPerSecond     []float64
	SuccessRateHistory    []float64
	RecentLogs            []models.LogEntry
	RecentEvents          []models.EventEntry
	LastUpdateTime        time.Time
	Uptime                time.Duration
	CurrentMessagesPerSec float64
	CurrentSuccessRate    float64
	ErrorCount            int64
	LastErrorTime         time.Time
}

// Monitor encapsulates all monitoring functionality
type Monitor struct {
	Metrics *Metrics
}

// New creates a new Monitor instance
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

// WaitForFile waits for the specified file to exist and returns an open handle.
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

// readNewLines reads new lines from the file and sends them to channels.
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

// MonitorFile monitors a file continuously, similar to `tail -f`.
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

// ProcessLog processes a log entry from tracker.log.
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

	if entry.Message == "Periodic system metrics" && entry.Metadata != nil {
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

// ProcessEvent processes an event entry from tracker.events.
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
	MinValue float64
	Status   HealthStatus
	Text     string
	Color    ui.Color
}

// evaluateStatus evaluates a value against ordered thresholds.
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
	return HealthCritical, "● UNKNOWN", ui.ColorRed
}

var (
	healthThresholds = []StatusThreshold{
		{SuccessRateExcellent, HealthGood, "● EXCELLENT", ui.ColorGreen},
		{SuccessRateGood, HealthWarning, "● GOOD", ui.ColorYellow},
		{0, HealthCritical, "● CRITICAL", ui.ColorRed},
	}

	throughputThresholds = []StatusThreshold{
		{ThroughputNormal, HealthGood, "● NORMAL", ui.ColorGreen},
		{ThroughputLow, HealthWarning, "● LOW", ui.ColorYellow},
		{0, HealthCritical, "● STOPPED", ui.ColorRed},
	}
)

// GetHealthStatus evaluates the success rate and returns a health status.
func GetHealthStatus(successRate float64) (HealthStatus, string, ui.Color) {
	return evaluateStatus(successRate, healthThresholds)
}

// GetThroughputStatus evaluates message throughput and returns a health status.
func GetThroughputStatus(mps float64) (HealthStatus, string, ui.Color) {
	return evaluateStatus(mps, throughputThresholds)
}

// GetErrorStatus evaluates errors and returns a health status.
func GetErrorStatus(errorCount int64, lastErrorTime time.Time) (HealthStatus, string, ui.Color) {
	if errorCount == 0 {
		return HealthGood, "● NONE", ui.ColorGreen
	}

	timeSinceError := time.Since(lastErrorTime)
	if timeSinceError > ErrorTimeoutWarning {
		return HealthGood, "● NONE", ui.ColorGreen
	} else if timeSinceError > ErrorTimeoutCritical {
		return HealthWarning, "● RECENT", ui.ColorYellow
	}
	return HealthCritical, "● ACTIVE", ui.ColorRed
}

// CalculateQualityScore calculates a global quality score (0-100).
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
func CreateMetricsTable() *widgets.Table {
	table := widgets.NewTable()
	table.Rows = [][]string{
		{"Metric", "Value"},
		{"Messages received", "0"},
		{"Messages processed", "0"},
		{"Messages failed", "0"},
		{"Throughput (msg/s)", "0.00"},
		{"Success rate", "0.00%"},
		{"Last update", "-"},
	}
	table.TextStyle = ui.NewStyle(ui.ColorWhite)
	table.RowStyles[0] = ui.NewStyle(ui.ColorYellow, ui.ColorClear, ui.ModifierBold)
	table.SetRect(0, 0, 50, 9)
	table.ColumnWidths = []int{30, 20}
	return table
}

// CreateHealthDashboard initializes the health dashboard widget.
func CreateHealthDashboard() *widgets.Table {
	table := widgets.NewTable()
	table.Rows = [][]string{
		{"Indicator", "Status"},
		{"Global health", "●"},
		{"Success rate", "●"},
		{"Throughput", "●"},
		{"Errors", "●"},
		{"Uptime", "-"},
		{"Quality", "-"},
	}
	table.TextStyle = ui.NewStyle(ui.ColorWhite)
	table.RowStyles[0] = ui.NewStyle(ui.ColorYellow, ui.ColorClear, ui.ModifierBold)
	table.SetRect(50, 0, 110, 9)
	table.ColumnWidths = []int{25, 35}
	return table
}

// CreateLogList initializes the log list widget.
func CreateLogList() *widgets.List {
	list := widgets.NewList()
	list.Title = "Recent Logs (tracker.log)"
	list.Rows = []string{"Waiting for logs..."}
	list.TextStyle = ui.NewStyle(ui.ColorWhite)
	list.SelectedRowStyle = ui.NewStyle(ui.ColorBlack, ui.ColorWhite)
	list.WrapText = true
	list.SetRect(0, 9, 80, 19)
	return list
}

// CreateEventList initializes the event list widget.
func CreateEventList() *widgets.List {
	list := widgets.NewList()
	list.Title = "Recent Events (tracker.events)"
	list.Rows = []string{"Waiting for events..."}
	list.TextStyle = ui.NewStyle(ui.ColorWhite)
	list.SelectedRowStyle = ui.NewStyle(ui.ColorBlack, ui.ColorWhite)
	list.WrapText = true
	list.SetRect(80, 9, 160, 19)
	return list
}

// CreateMessagesPerSecondChart initializes the throughput chart widget.
func CreateMessagesPerSecondChart() *widgets.Plot {
	plot := widgets.NewPlot()
	plot.Title = "Message Throughput (msg/s)"
	plot.Data = [][]float64{{}}
	plot.SetRect(0, 19, 80, 29)
	plot.AxesColor = ui.ColorWhite
	plot.LineColors[0] = ui.ColorGreen
	plot.Marker = widgets.MarkerDot
	return plot
}

// CreateSuccessRateChart initializes the success rate chart widget.
func CreateSuccessRateChart() *widgets.Plot {
	plot := widgets.NewPlot()
	plot.Title = "Success Rate (%)"
	plot.Data = [][]float64{{}}
	plot.SetRect(80, 19, 160, 29)
	plot.AxesColor = ui.ColorWhite
	plot.LineColors[0] = ui.ColorBlue
	plot.Marker = widgets.MarkerDot
	return plot
}

// UpdateMetricsTable updates the metrics table.
func UpdateMetricsTable(table *widgets.Table, m *Metrics) {
	table.Rows = [][]string{
		{"Metric", "Value"},
		{"Messages received", fmt.Sprintf("%d", m.MessagesReceived)},
		{"Messages processed", fmt.Sprintf("%d", m.MessagesProcessed)},
		{"Messages failed", fmt.Sprintf("%d", m.MessagesFailed)},
		{"Throughput (msg/s)", fmt.Sprintf("%.2f", m.CurrentMessagesPerSec)},
		{"Success rate", fmt.Sprintf("%.2f%%", m.CurrentSuccessRate)},
		{"Last update", m.LastUpdateTime.Format("15:04:05")},
	}
}

// getGlobalHealthStatus determines global health from individual statuses.
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
		return globalStatus, "● WARNING", ui.ColorYellow
	case HealthCritical:
		return globalStatus, "● CRITICAL", ui.ColorRed
	default:
		return HealthGood, "● EXCELLENT", ui.ColorGreen
	}
}

// getQualityText returns text and color for a quality score.
func getQualityText(qualityScore float64) (string, ui.Color) {
	if qualityScore >= QualityScoreExcellent {
		return fmt.Sprintf("EXCELLENT (%.0f)", qualityScore), ui.ColorGreen
	} else if qualityScore >= QualityScoreGood {
		return fmt.Sprintf("GOOD (%.0f)", qualityScore), ui.ColorYellow
	} else if qualityScore >= QualityScoreMedium {
		return fmt.Sprintf("MEDIUM (%.0f)", qualityScore), ui.ColorYellow
	}
	return fmt.Sprintf("LOW (%.0f)", qualityScore), ui.ColorRed
}

// formatUptime formats uptime as a readable string.
func formatUptime(uptime time.Duration) string {
	if uptime.Hours() >= 1 {
		return fmt.Sprintf("%.1fh", uptime.Hours())
	} else if uptime.Minutes() >= 1 {
		return fmt.Sprintf("%.0fm", uptime.Minutes())
	}
	return fmt.Sprintf("%.0fs", uptime.Seconds())
}

// UpdateHealthDashboard updates the health dashboard.
func UpdateHealthDashboard(dashboard *widgets.Table, m *Metrics) {
	successStatus, successText, successColor := GetHealthStatus(m.CurrentSuccessRate)
	throughputStatus, throughputText, throughputColor := GetThroughputStatus(m.CurrentMessagesPerSec)
	errorStatus, errorText, errorColor := GetErrorStatus(m.ErrorCount, m.LastErrorTime)

	_, globalText, globalColor := getGlobalHealthStatus(successStatus, throughputStatus, errorStatus)

	qualityScore := CalculateQualityScore(m.CurrentSuccessRate, m.CurrentMessagesPerSec, m.ErrorCount, m.Uptime)
	qualityText, qualityColor := getQualityText(qualityScore)
	uptimeStr := formatUptime(m.Uptime)

	dashboard.Rows = [][]string{
		{"Indicator", "Status"},
		{"Global health", globalText},
		{"Success rate", successText},
		{"Throughput", throughputText},
		{"Errors", errorText},
		{"Uptime", uptimeStr},
		{"Quality", qualityText},
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

// UpdateLogList updates the recent logs list.
func UpdateLogList(list *widgets.List, logs []models.LogEntry) {
	rows := make([]string, 0, len(logs))
	for i := len(logs) - 1; i >= 0; i-- {
		rows = append(rows, formatLogRow(logs[i]))
	}
	if len(rows) == 0 {
		rows = []string{"Waiting for logs..."}
	}
	list.Rows = rows
}

// formatEventRow formats an event entry for display.
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

// UpdateEventList updates the recent events list.
func UpdateEventList(list *widgets.List, events []models.EventEntry) {
	rows := make([]string, 0, len(events))
	for i := len(events) - 1; i >= 0; i-- {
		rows = append(rows, formatEventRow(events[i]))
	}
	if len(rows) == 0 {
		rows = []string{"Waiting for events..."}
	}
	list.Rows = rows
}

// UpdateCharts updates the throughput and success rate charts.
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
func (m *Monitor) UpdateUI(table *widgets.Table, healthDashboard *widgets.Table, logList *widgets.List, eventList *widgets.List, mpsChart *widgets.Plot, srChart *widgets.Plot) {
	m.Metrics.mu.RLock()
	defer m.Metrics.mu.RUnlock()

	UpdateMetricsTable(table, m.Metrics)
	UpdateHealthDashboard(healthDashboard, m.Metrics)
	UpdateLogList(logList, m.Metrics.RecentLogs)
	UpdateEventList(eventList, m.Metrics.RecentEvents)
	UpdateCharts(mpsChart, srChart, m.Metrics.MessagesPerSecond, m.Metrics.SuccessRateHistory)
}
