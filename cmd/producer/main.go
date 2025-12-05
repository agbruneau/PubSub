/*
Point d'entrée du producteur pour le système PubSub de démonstration Kafka.

Ceci est le point d'entrée principal pour le binaire du producteur.
Construction: go build -o producer.exe ./cmd/producer
*/
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/agbruneau/PubSub/internal/producer"
)

// main est la fonction principale qui initialise et lance le service producteur.
// Elle charge la configuration, initialise la connexion Kafka, et démarre la boucle de production.
// Elle écoute également les signaux système (SIGINT, SIGTERM) pour un arrêt gracieux.
func main() {
	// Charger la configuration
	config := producer.NewConfig()

	// Créer et initialiser le producteur
	prod := producer.New(config)
	if err := prod.Initialize(); err != nil {
		fmt.Printf("Erreur fatale lors de l'initialisation: %v\n", err)
		os.Exit(1)
	}
	defer prod.Close()

	fmt.Println("🟢 Le producteur est démarré et prêt à envoyer des messages...")
	fmt.Printf("📤 Publication vers le sujet '%s'\n", config.Topic)

	// Gérer les signaux d'arrêt
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	// Démarrer la boucle de production
	prod.Run(sigchan)
}
