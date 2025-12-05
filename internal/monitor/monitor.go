/*
Package monitor fournit un moniteur de logs en temps réel TUI pour le système PubSub.

Ce paquet implémente la surveillance de fichiers, l'analyse de logs, la visualisation de métriques,
et une interface utilisateur terminal interactive utilisant les widgets termui.
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

// HealthStatus définit les niveaux de santé pour les indicateurs du tableau de bord.
type HealthStatus int

const (
	HealthGood     HealthStatus = iota // Indique une condition saine, typiquement affichée en vert.
	HealthWarning                      // Indique un avertissement, typiquement affiché en jaune.
	HealthCritical                     // Indique un état critique, typiquement affiché en rouge.
)

// Alias locaux pour la lisibilité
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

// Metrics agrège et gère l'état de toutes les métriques collectées par le moniteur.
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

// Monitor encapsule toutes les fonctionnalités de surveillance
type Monitor struct {
	Metrics *Metrics
}

// New crée une nouvelle instance de Monitor
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

// WaitForFile attend que le fichier spécifié existe et retourne un descripteur ouvert.
func WaitForFile(filename string) *os.File {
	for {
		file, err := os.Open(filename)
		if err == nil {
			return file
		}
		time.Sleep(FileCheckInterval)
	}
}

// waitForFileRecreation attend qu'un fichier supprimé soit recréé.
func waitForFileRecreation(filename string) *os.File {
	for {
		time.Sleep(FileCheckInterval)
		file, err := os.Open(filename)
		if err == nil {
			return file
		}
	}
}

// parseAndSendLogEntry analyse une ligne JSON et l'envoie au canal approprié.
func parseAndSendLogEntry(line string, logChan chan<- models.LogEntry) {
	var entry models.LogEntry
	if err := json.Unmarshal([]byte(line), &entry); err == nil {
		select {
		case logChan <- entry:
		default:
			// Canal plein, ignorer
		}
	}
}

// parseAndSendEventEntry analyse une ligne JSON et l'envoie au canal approprié.
func parseAndSendEventEntry(line string, eventChan chan<- models.EventEntry) {
	var entry models.EventEntry
	if err := json.Unmarshal([]byte(line), &entry); err == nil {
		select {
		case eventChan <- entry:
		default:
			// Canal plein, ignorer
		}
	}
}

// readNewLines lit les nouvelles lignes du fichier et les envoie aux canaux.
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

// MonitorFile surveille un fichier en continu, similaire à `tail -f`.
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

// ProcessLog traite une entrée de journal provenant de tracker.log.
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

// ProcessEvent traite une entrée d'événement provenant de tracker.events.
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

// StatusThreshold définit un seuil pour l'évaluation de l'état.
type StatusThreshold struct {
	MinValue float64
	Status   HealthStatus
	Text     string
	Color    ui.Color
}

// evaluateStatus évalue une valeur par rapport à des seuils ordonnés.
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

// GetHealthStatus évalue le taux de succès et retourne un état de santé.
func GetHealthStatus(successRate float64) (HealthStatus, string, ui.Color) {
	return evaluateStatus(successRate, healthThresholds)
}

// GetThroughputStatus évalue le débit de messages et retourne un état de santé.
func GetThroughputStatus(mps float64) (HealthStatus, string, ui.Color) {
	return evaluateStatus(mps, throughputThresholds)
}

// GetErrorStatus évalue les erreurs et retourne un état de santé.
func GetErrorStatus(errorCount int64, lastErrorTime time.Time) (HealthStatus, string, ui.Color) {
	if errorCount == 0 {
		return HealthGood, "● AUCUNE", ui.ColorGreen
	}

	timeSinceError := time.Since(lastErrorTime)
	if timeSinceError > ErrorTimeoutWarning {
		return HealthGood, "● AUCUNE", ui.ColorGreen
	} else if timeSinceError > ErrorTimeoutCritical {
		return HealthWarning, "● RÉCENTE", ui.ColorYellow
	}
	return HealthCritical, "● ACTIVE", ui.ColorRed
}

// CalculateQualityScore calcule un score de qualité global (0-100).
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

// CreateMetricsTable initialise le widget de tableau des métriques.
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

// CreateHealthDashboard initialise le widget de tableau de bord de santé.
func CreateHealthDashboard() *widgets.Table {
	table := widgets.NewTable()
	table.Rows = [][]string{
		{"Indicateur", "État"},
		{"Santé globale", "●"},
		{"Taux de succès", "●"},
		{"Débit", "●"},
		{"Erreurs", "●"},
		{"Temps d'activité", "-"},
		{"Qualité", "-"},
	}
	table.TextStyle = ui.NewStyle(ui.ColorWhite)
	table.RowStyles[0] = ui.NewStyle(ui.ColorYellow, ui.ColorClear, ui.ModifierBold)
	table.SetRect(50, 0, 110, 9)
	table.ColumnWidths = []int{25, 35}
	return table
}

// CreateLogList initialise le widget de liste des logs.
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

// CreateEventList initialise le widget de liste des événements.
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

// CreateMessagesPerSecondChart initialise le widget de graphique de débit.
func CreateMessagesPerSecondChart() *widgets.Plot {
	plot := widgets.NewPlot()
	plot.Title = "Débit de messages (msg/s)"
	plot.Data = [][]float64{{}}
	plot.SetRect(0, 19, 80, 29)
	plot.AxesColor = ui.ColorWhite
	plot.LineColors[0] = ui.ColorGreen
	plot.Marker = widgets.MarkerDot
	return plot
}

// CreateSuccessRateChart initialise le widget de graphique de taux de succès.
func CreateSuccessRateChart() *widgets.Plot {
	plot := widgets.NewPlot()
	plot.Title = "Taux de succès (%)"
	plot.Data = [][]float64{{}}
	plot.SetRect(80, 19, 160, 29)
	plot.AxesColor = ui.ColorWhite
	plot.LineColors[0] = ui.ColorBlue
	plot.Marker = widgets.MarkerDot
	return plot
}

// UpdateMetricsTable met à jour le tableau des métriques.
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

// getGlobalHealthStatus détermine la santé globale à partir des états individuels.
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
		return HealthGood, "● EXCELLENT", ui.ColorGreen
	}
}

// getQualityText retourne le texte et la couleur pour un score de qualité.
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

// formatUptime formate le temps d'activité en chaîne lisible.
func formatUptime(uptime time.Duration) string {
	if uptime.Hours() >= 1 {
		return fmt.Sprintf("%.1fh", uptime.Hours())
	} else if uptime.Minutes() >= 1 {
		return fmt.Sprintf("%.0fm", uptime.Minutes())
	}
	return fmt.Sprintf("%.0fs", uptime.Seconds())
}

// UpdateHealthDashboard met à jour le tableau de bord de santé.
func UpdateHealthDashboard(dashboard *widgets.Table, m *Metrics) {
	successStatus, successText, successColor := GetHealthStatus(m.CurrentSuccessRate)
	throughputStatus, throughputText, throughputColor := GetThroughputStatus(m.CurrentMessagesPerSec)
	errorStatus, errorText, errorColor := GetErrorStatus(m.ErrorCount, m.LastErrorTime)

	_, globalText, globalColor := getGlobalHealthStatus(successStatus, throughputStatus, errorStatus)

	qualityScore := CalculateQualityScore(m.CurrentSuccessRate, m.CurrentMessagesPerSec, m.ErrorCount, m.Uptime)
	qualityText, qualityColor := getQualityText(qualityScore)
	uptimeStr := formatUptime(m.Uptime)

	dashboard.Rows = [][]string{
		{"Indicateur", "État"},
		{"Santé globale", globalText},
		{"Taux de succès", successText},
		{"Débit", throughputText},
		{"Erreurs", errorText},
		{"Temps d'activité", uptimeStr},
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

// formatLogRow formate une entrée de log pour l'affichage.
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

// UpdateLogList met à jour la liste des logs récents.
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

// formatEventRow formate une entrée d'événement pour l'affichage.
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

// UpdateEventList met à jour la liste des événements récents.
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

// UpdateCharts met à jour les graphiques de débit et de taux de succès.
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

// UpdateUI rafraîchit tous les widgets UI avec les dernières métriques.
func (m *Monitor) UpdateUI(table *widgets.Table, healthDashboard *widgets.Table, logList *widgets.List, eventList *widgets.List, mpsChart *widgets.Plot, srChart *widgets.Plot) {
	m.Metrics.mu.RLock()
	defer m.Metrics.mu.RUnlock()

	UpdateMetricsTable(table, m.Metrics)
	UpdateHealthDashboard(healthDashboard, m.Metrics)
	UpdateLogList(logList, m.Metrics.RecentLogs)
	UpdateEventList(eventList, m.Metrics.RecentEvents)
	UpdateCharts(mpsChart, srChart, m.Metrics.MessagesPerSecond, m.Metrics.SuccessRateHistory)
}
