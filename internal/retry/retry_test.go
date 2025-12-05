package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDoSuccess(t *testing.T) {
	cfg := Config{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
	}

	callCount := 0
	result := Do(context.Background(), cfg, func() error {
		callCount++
		return nil
	})

	if result.Err != nil {
		t.Errorf("Expected no error, got %v", result.Err)
	}
	if result.Attempts != 1 {
		t.Errorf("Expected 1 attempt, got %d", result.Attempts)
	}
	if callCount != 1 {
		t.Errorf("Expected function to be called once, called %d times", callCount)
	}
}

func TestDoRetryThenSuccess(t *testing.T) {
	cfg := Config{
		MaxAttempts:  5,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
	}

	callCount := 0
	result := Do(context.Background(), cfg, func() error {
		callCount++
		if callCount < 3 {
			return errors.New("temporary error")
		}
		return nil
	})

	if result.Err != nil {
		t.Errorf("Expected no error, got %v", result.Err)
	}
	if result.Attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", result.Attempts)
	}
}

func TestDoMaxAttemptsExhausted(t *testing.T) {
	cfg := Config{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
	}

	expectedErr := errors.New("persistent error")
	callCount := 0
	result := Do(context.Background(), cfg, func() error {
		callCount++
		return expectedErr
	})

	if result.Err == nil {
		t.Error("Expected error, got nil")
	}
	if result.Attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", result.Attempts)
	}
	if callCount != 3 {
		t.Errorf("Expected function to be called 3 times, called %d times", callCount)
	}
}

func TestDoPermanentError(t *testing.T) {
	cfg := Config{
		MaxAttempts:  5,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
	}

	callCount := 0
	result := Do(context.Background(), cfg, func() error {
		callCount++
		return Permanent(errors.New("permanent error"))
	})

	if result.Err == nil {
		t.Error("Expected error, got nil")
	}
	if result.Attempts != 1 {
		t.Errorf("Expected 1 attempt (permanent error), got %d", result.Attempts)
	}
	if callCount != 1 {
		t.Errorf("Expected function to be called once, called %d times", callCount)
	}
}

func TestDoContextCancellation(t *testing.T) {
	cfg := Config{
		MaxAttempts:  10,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     1 * time.Second,
		Multiplier:   2.0,
	}

	ctx, cancel := context.WithCancel(context.Background())
	callCount := 0

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	result := Do(ctx, cfg, func() error {
		callCount++
		return errors.New("error")
	})

	if !errors.Is(result.Err, context.Canceled) {
		t.Errorf("Expected context.Canceled error, got %v", result.Err)
	}
}

func TestIsPermanent(t *testing.T) {
	regularErr := errors.New("regular error")
	permanentErr := Permanent(errors.New("permanent error"))

	if IsPermanent(regularErr) {
		t.Error("Regular error should not be permanent")
	}
	if !IsPermanent(permanentErr) {
		t.Error("Permanent error should be detected as permanent")
	}
	if IsPermanent(nil) {
		t.Error("Nil should not be permanent")
	}
}

func TestCalculateDelay(t *testing.T) {
	cfg := Config{
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     1 * time.Second,
		Multiplier:   2.0,
	}

	// Test that delays increase exponentially (approximately, due to jitter)
	delay1 := calculateDelay(1, cfg)
	delay2 := calculateDelay(2, cfg)
	delay3 := calculateDelay(3, cfg)

	// Delays should be roughly increasing (accounting for jitter)
	if delay1 > 200*time.Millisecond {
		t.Errorf("First delay too large: %v", delay1)
	}
	if delay2 < 100*time.Millisecond || delay2 > 400*time.Millisecond {
		t.Errorf("Second delay out of range: %v", delay2)
	}
	if delay3 < 200*time.Millisecond || delay3 > 800*time.Millisecond {
		t.Errorf("Third delay out of range: %v", delay3)
	}
}
