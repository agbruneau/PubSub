/*
Monitor entry point for the Kafka Demo PubSub system.

This is the main entry point for the log monitor TUI binary.
Build: go build -o monitor.exe ./cmd/monitor
*/
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/agbruneau/PubSub/internal/config"
	"github.com/agbruneau/PubSub/internal/monitor"
	"github.com/agbruneau/PubSub/pkg/models"
	ui "github.com/gizak/termui/v3"
)

func main() {
	if err := ui.Init(); err != nil {
		fmt.Printf("Error initializing UI: %v\n", err)
		os.Exit(1)
	}
	defer ui.Close()

	// Create monitor instance
	mon := monitor.New()

	// Channels for logs and events
	logChan := make(chan models.LogEntry, monitor.LogChannelBuffer)
	eventChan := make(chan models.EventEntry, monitor.EventChannelBuffer)

	// Start file monitoring
	go monitor.MonitorFile(config.TrackerLogFile, logChan, nil)
	go monitor.MonitorFile(config.TrackerEventsFile, nil, eventChan)

	// Process logs and events
	go func() {
		for {
			select {
			case log := <-logChan:
				mon.ProcessLog(log)
			case event := <-eventChan:
				mon.ProcessEvent(event)
			}
		}
	}()

	// Create widgets
	metricsTable := monitor.CreateMetricsTable()
	healthDashboard := monitor.CreateHealthDashboard()
	logList := monitor.CreateLogList()
	eventList := monitor.CreateEventList()
	mpsChart := monitor.CreateMessagesPerSecondChart()
	srChart := monitor.CreateSuccessRateChart()

	// Handle resize
	uiEvents := ui.PollEvents()
	ticker := time.NewTicker(monitor.UIUpdateInterval)
	defer ticker.Stop()

	mon.Metrics.StartTime = time.Now()

	for {
		select {
		case e := <-uiEvents:
			switch e.ID {
			case "q", "<C-c>":
				return
			case "<Resize>":
				metricsTable.SetRect(0, 0, 50, 9)
				healthDashboard.SetRect(50, 0, 110, 9)
				logList.SetRect(0, 9, 80, 19)
				eventList.SetRect(80, 9, 160, 19)
				mpsChart.SetRect(0, 19, 80, 29)
				srChart.SetRect(80, 19, 160, 29)
				ui.Clear()
			}
		case <-ticker.C:
			mon.Metrics.Uptime = time.Since(mon.Metrics.StartTime)
			mon.UpdateUI(metricsTable, healthDashboard, logList, eventList, mpsChart, srChart)
			ui.Render(metricsTable, healthDashboard, logList, eventList, mpsChart, srChart)
		}
	}
}
