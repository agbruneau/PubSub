/*
Package retry provides retry logic with exponential backoff.
*/
package retry

import (
	"context"
	"errors"
	"math"
	"math/rand"
	"time"
)

// Config contains retry configuration.
type Config struct {
	MaxAttempts  int           // Maximum number of attempts (including first try)
	InitialDelay time.Duration // Initial delay before first retry
	MaxDelay     time.Duration // Maximum delay between retries
	Multiplier   float64       // Multiplier for exponential backoff
}

// DefaultConfig returns a default retry configuration.
func DefaultConfig() Config {
	return Config{
		MaxAttempts:  3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     5 * time.Second,
		Multiplier:   2.0,
	}
}

// PermanentError wraps an error to indicate it should not be retried.
type PermanentError struct {
	Err error
}

func (e *PermanentError) Error() string {
	return e.Err.Error()
}

func (e *PermanentError) Unwrap() error {
	return e.Err
}

// Permanent wraps an error to indicate it should not be retried.
func Permanent(err error) error {
	if err == nil {
		return nil
	}
	return &PermanentError{Err: err}
}

// IsPermanent checks if an error is marked as permanent.
func IsPermanent(err error) bool {
	var permanentErr *PermanentError
	return errors.As(err, &permanentErr)
}

// Result contains the result of a retry operation.
type Result struct {
	Attempts int           // Number of attempts made
	Duration time.Duration // Total duration of all attempts
	Err      error         // Final error (nil if successful)
}

// Do executes the given function with retry logic.
// The function is retried until it succeeds, returns a permanent error,
// or the maximum number of attempts is reached.
func Do(ctx context.Context, cfg Config, fn func() error) Result {
	start := time.Now()
	var lastErr error

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return Result{
				Attempts: attempt,
				Duration: time.Since(start),
				Err:      ctx.Err(),
			}
		default:
		}

		// Execute the function
		err := fn()
		if err == nil {
			return Result{
				Attempts: attempt,
				Duration: time.Since(start),
				Err:      nil,
			}
		}

		lastErr = err

		// Don't retry permanent errors
		if IsPermanent(err) {
			return Result{
				Attempts: attempt,
				Duration: time.Since(start),
				Err:      err,
			}
		}

		// Don't sleep after the last attempt
		if attempt < cfg.MaxAttempts {
			delay := calculateDelay(attempt, cfg)
			select {
			case <-ctx.Done():
				return Result{
					Attempts: attempt,
					Duration: time.Since(start),
					Err:      ctx.Err(),
				}
			case <-time.After(delay):
			}
		}
	}

	return Result{
		Attempts: cfg.MaxAttempts,
		Duration: time.Since(start),
		Err:      lastErr,
	}
}

// calculateDelay calculates the delay for a given attempt using exponential backoff with jitter.
func calculateDelay(attempt int, cfg Config) time.Duration {
	// Calculate exponential delay
	delay := float64(cfg.InitialDelay) * math.Pow(cfg.Multiplier, float64(attempt-1))

	// Cap at max delay
	if delay > float64(cfg.MaxDelay) {
		delay = float64(cfg.MaxDelay)
	}

	// Add jitter (±25%)
	jitter := delay * 0.25 * (rand.Float64()*2 - 1)
	delay += jitter

	return time.Duration(delay)
}

// DoWithCallback is like Do but calls the provided callback after each failed attempt.
// This is useful for logging or metrics.
func DoWithCallback(ctx context.Context, cfg Config, fn func() error, onRetry func(attempt int, err error, nextDelay time.Duration)) Result {
	start := time.Now()
	var lastErr error

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return Result{
				Attempts: attempt,
				Duration: time.Since(start),
				Err:      ctx.Err(),
			}
		default:
		}

		err := fn()
		if err == nil {
			return Result{
				Attempts: attempt,
				Duration: time.Since(start),
				Err:      nil,
			}
		}

		lastErr = err

		if IsPermanent(err) {
			return Result{
				Attempts: attempt,
				Duration: time.Since(start),
				Err:      err,
			}
		}

		if attempt < cfg.MaxAttempts {
			delay := calculateDelay(attempt, cfg)
			if onRetry != nil {
				onRetry(attempt, err, delay)
			}
			select {
			case <-ctx.Done():
				return Result{
					Attempts: attempt,
					Duration: time.Since(start),
					Err:      ctx.Err(),
				}
			case <-time.After(delay):
			}
		}
	}

	return Result{
		Attempts: cfg.MaxAttempts,
		Duration: time.Since(start),
		Err:      lastErr,
	}
}
