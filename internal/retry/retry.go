/*
Package retry fournit une logique de relance avec un backoff exponentiel.
*/
package retry

import (
	"context"
	"errors"
	"math"
	"math/rand"
	"time"
)

// Config contient la configuration pour le mécanisme de relance.
type Config struct {
	MaxAttempts  int           // Nombre maximum de tentatives (incluant la première).
	InitialDelay time.Duration // Délai initial avant la première relance.
	MaxDelay     time.Duration // Délai maximum entre les relances.
	Multiplier   float64       // Multiplicateur pour le backoff exponentiel.
}

// DefaultConfig retourne une configuration de relance par défaut.
//
// Retourne:
//   - Config: La structure de configuration initialisée avec des valeurs standards.
func DefaultConfig() Config {
	return Config{
		MaxAttempts:  3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     5 * time.Second,
		Multiplier:   2.0,
	}
}

// PermanentError enveloppe une erreur pour indiquer qu'elle ne doit pas être retentée.
type PermanentError struct {
	Err error
}

// Error retourne le message d'erreur de l'erreur enveloppée.
//
// Retourne:
//   - string: Le message d'erreur.
func (e *PermanentError) Error() string {
	return e.Err.Error()
}

// Unwrap retourne l'erreur sous-jacente.
//
// Retourne:
//   - error: L'erreur originale.
func (e *PermanentError) Unwrap() error {
	return e.Err
}

// Permanent enveloppe une erreur pour indiquer qu'elle ne doit pas être retentée.
//
// Paramètres:
//   - err: L'erreur à envelopper.
//
// Retourne:
//   - error: Une erreur de type PermanentError, ou nil si err est nil.
func Permanent(err error) error {
	if err == nil {
		return nil
	}
	return &PermanentError{Err: err}
}

// IsPermanent vérifie si une erreur est marquée comme permanente.
//
// Paramètres:
//   - err: L'erreur à vérifier.
//
// Retourne:
//   - bool: Vrai si l'erreur est de type PermanentError, faux sinon.
func IsPermanent(err error) bool {
	var permanentErr *PermanentError
	return errors.As(err, &permanentErr)
}

// Result contient le résultat d'une opération de relance.
type Result struct {
	Attempts int           // Nombre de tentatives effectuées.
	Duration time.Duration // Durée totale de toutes les tentatives.
	Err      error         // Erreur finale (nil si succès).
}

// Do exécute la fonction donnée avec une logique de relance.
// La fonction est relancée jusqu'à ce qu'elle réussisse, retourne une erreur permanente,
// ou que le nombre maximum de tentatives soit atteint.
//
// Paramètres:
//   - ctx: Le contexte pour l'annulation et les délais.
//   - cfg: La configuration de relance.
//   - fn: La fonction à exécuter, retournant une erreur.
//
// Retourne:
//   - Result: Le résultat contenant le nombre de tentatives, la durée et l'erreur finale.
func Do(ctx context.Context, cfg Config, fn func() error) Result {
	start := time.Now()
	var lastErr error

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		// Vérification de l'annulation du contexte
		select {
		case <-ctx.Done():
			return Result{
				Attempts: attempt,
				Duration: time.Since(start),
				Err:      ctx.Err(),
			}
		default:
		}

		// Exécution de la fonction
		err := fn()
		if err == nil {
			return Result{
				Attempts: attempt,
				Duration: time.Since(start),
				Err:      nil,
			}
		}

		lastErr = err

		// Ne pas relancer les erreurs permanentes
		if IsPermanent(err) {
			return Result{
				Attempts: attempt,
				Duration: time.Since(start),
				Err:      err,
			}
		}

		// Ne pas dormir après la dernière tentative
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

// calculateDelay calcule le délai pour une tentative donnée en utilisant un backoff exponentiel avec jitter.
//
// Paramètres:
//   - attempt: Le numéro de la tentative actuelle.
//   - cfg: La configuration de relance.
//
// Retourne:
//   - time.Duration: La durée à attendre avant la prochaine tentative.
func calculateDelay(attempt int, cfg Config) time.Duration {
	// Calcul du délai exponentiel
	delay := float64(cfg.InitialDelay) * math.Pow(cfg.Multiplier, float64(attempt-1))

	// Plafonnement au délai maximum
	if delay > float64(cfg.MaxDelay) {
		delay = float64(cfg.MaxDelay)
	}

	// Ajout du jitter (±25%)
	jitter := delay * 0.25 * (rand.Float64()*2 - 1)
	delay += jitter

	return time.Duration(delay)
}

// DoWithCallback est comme Do mais appelle le callback fourni après chaque tentative échouée.
// Ceci est utile pour la journalisation ou les métriques.
//
// Paramètres:
//   - ctx: Le contexte pour l'annulation.
//   - cfg: La configuration de relance.
//   - fn: La fonction à exécuter.
//   - onRetry: La fonction de rappel exécutée après un échec. Reçoit le numéro de tentative, l'erreur et le prochain délai.
//
// Retourne:
//   - Result: Le résultat de l'opération.
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
