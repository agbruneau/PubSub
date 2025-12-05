package tracker

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
)

// TestTracker_Run_LogsGenericErrors replicates the bug where non-Kafka errors
// are ignored and not logged.
func TestTracker_Run_LogsGenericErrors(t *testing.T) {
	// 1. Setup
	// Create buffers to capture logs
	var logBuf bytes.Buffer
	var eventBuf bytes.Buffer // Not really used but needed for helper

	tracker := newTestTracker(&eventBuf, &logBuf)

	// Create and inject mock consumer
	mockConsumer := new(MockKafkaConsumer)
	tracker.consumer = mockConsumer

	// Define the generic error we expect to see
	genericErr := errors.New("something went wrong")

	// 2. Expectation
	// The Run loop calls ReadMessage. We make it return the generic error.
	// We use .Maybe() or .Times() because it might be called in a loop.
	// Since the bug is a busy loop, it will be called MANY times.
	// We just need it to answer at least once.
	mockConsumer.On("ReadMessage", mock.AnythingOfType("time.Duration")).Return(nil, genericErr)

	// 3. Execution
	// Run tracker in background
	go tracker.Run()

	// Give it a moment to loop a few times
	time.Sleep(100 * time.Millisecond)

	// Stop the tracker
	tracker.Stop()

	// 4. Verification
	logs := logBuf.String()

	// Check if the error message is present in the logs
	if !strings.Contains(logs, "something went wrong") {
		t.Fatalf("FAIL: Expected log to contain 'something went wrong', but got:\n%s\n(If empty, it means the error was silently swallowed)", logs)
	} else {
		t.Log("SUCCESS: Generic error was logged.")
	}
}
