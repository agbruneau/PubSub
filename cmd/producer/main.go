//go:build kafka
// +build kafka

/*
Producer entry point for the Kafka Demo PubSub system.

This is the main entry point for the producer binary.
Build: go build -o producer.exe ./cmd/producer
*/
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/agbruneau/PubSub/internal/producer"
)

func main() {
	// Load configuration
	config := producer.NewConfig()

	// Create and initialize the producer
	prod := producer.New(config)
	if err := prod.Initialize(); err != nil {
		fmt.Printf("Fatal error during initialization: %v\n", err)
		os.Exit(1)
	}
	defer prod.Close()

	fmt.Println("🟢 Producer is started and ready to send messages...")
	fmt.Printf("📤 Publishing to topic '%s'\n", config.Topic)

	// Handle stop signals
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	// Start the production loop
	prod.Run(sigchan)
}
