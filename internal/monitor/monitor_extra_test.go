package monitor

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/agbruneau/PubSub/pkg/models"
)

func TestParseAndSendLogEntry(t *testing.T) {
	logChan := make(chan models.LogEntry, 1)

	// Test valid log
	validJSON := `{"level":"INFO","message":"test message","timestamp":"2024-01-01T00:00:00Z"}`
	parseAndSendLogEntry(validJSON, logChan)

	select {
	case entry := <-logChan:
		if entry.Message != "test message" {
			t.Errorf("Expected message 'test message', got '%s'", entry.Message)
		}
	default:
		t.Error("Expected log entry in channel")
	}

	// Test invalid log (should not panic or send)
	invalidJSON := `{invalid json`
	parseAndSendLogEntry(invalidJSON, logChan)

	select {
	case <-logChan:
		t.Error("Did not expect log entry for invalid JSON")
	default:
		// OK
	}
}

func TestParseAndSendEventEntry(t *testing.T) {
	eventChan := make(chan models.EventEntry, 1)

	// Test valid event
	validJSON := `{"event_type":"order_created","timestamp":"2024-01-01T00:00:00Z","deserialized":true}`
	parseAndSendEventEntry(validJSON, eventChan)

	select {
	case entry := <-eventChan:
		if entry.EventType != "order_created" {
			t.Errorf("Expected event type 'order_created', got '%s'", entry.EventType)
		}
	default:
		t.Error("Expected event entry in channel")
	}

	// Test invalid event
	invalidJSON := `{invalid json`
	parseAndSendEventEntry(invalidJSON, eventChan)

	select {
	case <-eventChan:
		t.Error("Did not expect event entry for invalid JSON")
	default:
		// OK
	}
}

func TestWaitForFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "wait_test.log")

	// Start waiting in a goroutine
	done := make(chan struct{})
	go func() {
		f := WaitForFile(filePath)
		f.Close()
		close(done)
	}()

	// Ensure it blocks (wait a bit)
	select {
	case <-done:
		t.Error("WaitForFile returned before file was created")
	case <-time.After(100 * time.Millisecond):
		// OK, still waiting
	}

	// Create file
	f, err := os.Create(filePath)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	f.Close()

	// Ensure it returns
	select {
	case <-done:
		// OK
	case <-time.After(2 * time.Second): // Increase timeout to account for polling interval
		t.Error("WaitForFile did not return after file creation")
	}
}

func TestWaitForFileRecreation(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "recreate_test.log")

	// Start waiting in a goroutine.
	// waitForFileRecreation waits for a file to appear, sleeping first.
	done := make(chan struct{})
	go func() {
		f := waitForFileRecreation(filePath)
		f.Close()
		close(done)
	}()

	// Ensure it blocks initially
	select {
	case <-done:
		t.Error("waitForFileRecreation returned before file was created")
	case <-time.After(100 * time.Millisecond):
		// OK
	}

	// Create file
	f, err := os.Create(filePath)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	f.Close()

	// Ensure it returns after file creation
	select {
	case <-done:
		// OK
	case <-time.After(2 * time.Second):
		t.Error("waitForFileRecreation did not return after file creation")
	}
}
