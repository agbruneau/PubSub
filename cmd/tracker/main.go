/*
Point d'entrée du tracker pour le système PubSub de démonstration Kafka.

Ceci est le point d'entrée principal pour le binaire du tracker (consommateur).
Construction: go build -o tracker.exe ./cmd/tracker
*/
package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/agbruneau/PubSub/internal/tracker"
)

// main est la fonction principale qui initialise et lance le service tracker.
// Elle charge la configuration, initialise la connexion Kafka et les loggers,
// et démarre la consommation des messages. Elle gère également l'arrêt gracieux via signaux.
func main() {
	// Charger la configuration
	config := tracker.NewConfig()

	// Créer et initialiser le tracker
	trk := tracker.New(config)
	if err := trk.Initialize(); err != nil {
		log.Fatalf("Erreur fatale lors de l'initialisation: %v", err)
	}
	defer trk.Close()

	fmt.Println("🟢 Le consommateur est en cours d'exécution...")
	fmt.Printf("📝 Logs d'observabilité système dans %s\n", config.LogFile)
	fmt.Printf("📋 Journalisation complète des messages dans %s\n", config.EventsFile)

	// Gérer les signaux d'arrêt
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	// Démarrer le tracker dans une goroutine
	done := make(chan struct{})
	go func() {
		trk.Run()
		close(done)
	}()

	// Attendre un signal d'arrêt
	<-sigchan
	fmt.Println("\n⚠️ Signal d'arrêt reçu...")
	trk.Stop()
	<-done

	fmt.Println("🔴 Consommateur arrêté.")
}
