package retry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestDoWithCallback vérifie que DoWithCallback appelle le callback.
func TestDoWithCallback(t *testing.T) {
	ctx := context.Background()
	cfg := DefaultConfig()
	cfg.MaxAttempts = 3
	cfg.InitialDelay = 1 * time.Millisecond // Délai court pour le test

	var callbackCount int
	var callbackErr error
	var callbackDelay time.Duration

	fn := func() error {
		return errors.New("fail")
	}

	onRetry := func(attempt int, err error, nextDelay time.Duration) {
		callbackCount++
		callbackErr = err
		callbackDelay = nextDelay
	}

	result := DoWithCallback(ctx, cfg, fn, onRetry)

	assert.Error(t, result.Err)
	assert.Equal(t, 3, result.Attempts)
	assert.Equal(t, 2, callbackCount, "Le callback doit être appelé 2 fois (entre les 3 tentatives)")
	assert.Equal(t, "fail", callbackErr.Error())
	assert.Greater(t, int64(callbackDelay), int64(0))
}

// TestPermanentErrorMethod vérifie la méthode Error() de PermanentError.
func TestPermanentErrorMethod(t *testing.T) {
	err := errors.New("base error")
	permErr := Permanent(err)

	assert.Equal(t, "base error", permErr.Error())
	assert.Equal(t, err, errors.Unwrap(permErr))
}

// TestPermanentNil vérifie que Permanent(nil) retourne nil.
func TestPermanentNil(t *testing.T) {
	assert.Nil(t, Permanent(nil))
}
