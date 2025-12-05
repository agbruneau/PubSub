/*
Point d'entrée du moniteur pour le système PubSub de démonstration Kafka.

Ceci est le point d'entrée principal pour le binaire du moniteur de logs TUI.
Construction: go build -o monitor.exe ./cmd/monitor
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

// main est la fonction principale qui initialise et lance le moniteur TUI.
// Elle configure l'interface utilisateur, lance la surveillance des fichiers de logs en arrière-plan,
// et gère la boucle d'événements pour l'affichage et les interactions utilisateur.
func main() {
	if err := ui.Init(); err != nil {
		fmt.Printf("Erreur lors de l'initialisation de l'UI: %v\n", err)
		os.Exit(1)
	}
	defer ui.Close()

	// Créer une instance du moniteur
	mon := monitor.New()

	// Canaux pour les logs et les événements
	logChan := make(chan models.LogEntry, config.MonitorLogChannelBuffer)
	eventChan := make(chan models.EventEntry, config.MonitorEventChannelBuffer)

	// Démarrer la surveillance des fichiers
	go monitor.MonitorFile(config.TrackerLogFile, logChan, nil)
	go monitor.MonitorFile(config.TrackerEventsFile, nil, eventChan)

	// Traiter les logs et les événements
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

	// Créer les widgets
	metricsTable := monitor.CreateMetricsTable()
	healthDashboard := monitor.CreateHealthDashboard()
	logList := monitor.CreateLogList()
	eventList := monitor.CreateEventList()
	mpsChart := monitor.CreateMessagesPerSecondChart()
	srChart := monitor.CreateSuccessRateChart()

	// Gérer le redimensionnement et les événements UI
	uiEvents := ui.PollEvents()
	ticker := time.NewTicker(config.MonitorUIUpdateInterval)
	defer ticker.Stop()

	mon.Metrics.StartTime = time.Now()

	// Configuration initiale de la mise en page (layout)
	// Nous définissons des rectangles statiques pour commencer
	termWidth, termHeight := ui.TerminalDimensions()
	// La hauteur de la grille est divisée en 3 sections:
	// 1. Haut: Métriques et Santé (hauteur 9)
	// 2. Milieu: Logs et Événements (hauteur 10)
	// 3. Bas: Graphiques (reste de la hauteur)

	// Largeur divisée par 2 pour la plupart des éléments
	midWidth := termWidth / 2

	// Section 1
	metricsTable.SetRect(0, 0, 50, 9)
	healthDashboard.SetRect(50, 0, termWidth, 9)

	// Section 2
	logList.SetRect(0, 9, midWidth, 19)
	eventList.SetRect(midWidth, 9, termWidth, 19)

	// Section 3
	mpsChart.SetRect(0, 19, midWidth, termHeight)
	srChart.SetRect(midWidth, 19, termWidth, termHeight)

	ui.Render(metricsTable, healthDashboard, logList, eventList, mpsChart, srChart)

	for {
		select {
		case e := <-uiEvents:
			switch e.ID {
			case "q", "<C-c>":
				return
			case "<Resize>":
				payload := e.Payload.(ui.Resize)
				termWidth = payload.Width
				termHeight = payload.Height
				midWidth = termWidth / 2

				metricsTable.SetRect(0, 0, 50, 9)
				healthDashboard.SetRect(50, 0, termWidth, 9)
				logList.SetRect(0, 9, midWidth, 19)
				eventList.SetRect(midWidth, 9, termWidth, 19)
				mpsChart.SetRect(0, 19, midWidth, termHeight)
				srChart.SetRect(midWidth, 19, termWidth, termHeight)

				ui.Clear()
				ui.Render(metricsTable, healthDashboard, logList, eventList, mpsChart, srChart)
			}
		case <-ticker.C:
			mon.Metrics.Uptime = time.Since(mon.Metrics.StartTime)
			mon.UpdateUI(metricsTable, healthDashboard, logList, eventList, mpsChart, srChart)
			ui.Render(metricsTable, healthDashboard, logList, eventList, mpsChart, srChart)
		}
	}
}
