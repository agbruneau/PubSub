//go:build kafka
// +build kafka

/*
Tracker entry point for the Kafka Demo PubSub system.

This is the main entry point for the tracker (consumer) binary.
Build: go build -o tracker.exe ./cmd/tracker
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

func main() {
	// Load configuration
	config := tracker.NewConfig()

	// Create and initialize the tracker
	trk := tracker.New(config)
	if err := trk.Initialize(); err != nil {
		log.Fatalf("Fatal error during initialization: %v", err)
	}
	defer trk.Close()

	fmt.Println("🟢 Consumer is running...")
	fmt.Printf("📝 System observability logs in %s\n", config.LogFile)
	fmt.Printf("📋 Complete message logging in %s\n", config.EventsFile)

	// Handle stop signals
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	// Start the tracker in a goroutine
	done := make(chan struct{})
	go func() {
		trk.Run()
		close(done)
	}()

	// Wait for stop signal
	<-sigchan
	fmt.Println("\n⚠️ Stop signal received...")
	trk.Stop()
	<-done

	fmt.Println("🔴 Consumer stopped.")
}
