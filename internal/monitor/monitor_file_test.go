package monitor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/agbruneau/PubSub/internal/config"
	"github.com/agbruneau/PubSub/pkg/models"
)

func TestReadNewLines(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	// Create a log file
	f, err := os.Create(logFile)
	if err != nil {
		t.Fatalf("Failed to create test log file: %v", err)
	}
	defer f.Close()

	// Write initial content
	f.WriteString("{\"level\":\"INFO\",\"message\":\"msg1\"}\n")
	f.Sync()

	// Prepare channels
	logChan := make(chan models.LogEntry, 10)
	eventChan := make(chan models.EventEntry, 10)

	// Read first batch
	pos := readNewLines(f, config.TrackerLogFile, 0, logChan, eventChan)

	if pos == 0 {
		t.Error("Expected position to advance")
	}

	select {
	case l := <-logChan:
		if l.Message != "msg1" {
			t.Errorf("Expected 'msg1', got '%s'", l.Message)
		}
	default:
		t.Error("Expected log entry")
	}

	// Write more content
	f.WriteString("{\"level\":\"INFO\",\"message\":\"msg2\"}\n")
	f.Sync()

	// Read second batch from new position
	newPos := readNewLines(f, config.TrackerLogFile, pos, logChan, eventChan)
	if newPos <= pos {
		t.Error("Expected position to advance again")
	}

	select {
	case l := <-logChan:
		if l.Message != "msg2" {
			t.Errorf("Expected 'msg2', got '%s'", l.Message)
		}
	default:
		t.Error("Expected second log entry")
	}
}
