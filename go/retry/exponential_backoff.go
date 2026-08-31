package retry

import (
	"context"
	"math/rand/v2"
	"sync"
	"time"

	k4k3ruSDKAppError "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
)

// ExponentialBackoffConfig contains exponential backoff settings.
type ExponentialBackoffConfig struct {
	InitialInterval time.Duration
	MaxInterval     time.Duration
	Multiplier      float64
	JitterRate      float64
}

// ExponentialBackoff provides context-aware exponential retry delays.
type ExponentialBackoff struct {
	mu            sync.Mutex
	config        ExponentialBackoffConfig
	nextInterval  time.Duration
	randomFloat64 func() float64
}

// NewExponentialBackoff creates an exponential backoff.
//
// Parameters:
//   - config: exponential interval and jitter settings.
//
// Returns:
//   - Configured exponential backoff.
//   - Configuration validation error.
//
// Version:
//   - 2026-08-31: Added.
func NewExponentialBackoff(config ExponentialBackoffConfig) (*ExponentialBackoff, error) {
	return newExponentialBackoff(config, rand.Float64)
}

func newExponentialBackoff(config ExponentialBackoffConfig, randomFloat64 func() float64) (*ExponentialBackoff, error) {
	if config.InitialInterval <= 0 {
		return nil, k4k3ruSDKAppError.Tracef("failed to create exponential backoff: %w: initial_interval=out_of_range min_value=1", k4k3ruSDKAppError.InvalidParameter())
	}
	if config.MaxInterval < config.InitialInterval {
		return nil, k4k3ruSDKAppError.Tracef("failed to create exponential backoff: %w: max_interval=out_of_range min_value=%d", k4k3ruSDKAppError.InvalidParameter(), config.InitialInterval)
	}
	if config.Multiplier <= 1 {
		return nil, k4k3ruSDKAppError.Tracef("failed to create exponential backoff: %w: multiplier=out_of_range min_value_exclusive=1", k4k3ruSDKAppError.InvalidParameter())
	}
	if config.JitterRate < 0 || config.JitterRate >= 1 {
		return nil, k4k3ruSDKAppError.Tracef("failed to create exponential backoff: %w: jitter_rate=out_of_range min_value=0 max_value_exclusive=1", k4k3ruSDKAppError.InvalidParameter())
	}
	if randomFloat64 == nil {
		return nil, k4k3ruSDKAppError.Tracef("failed to create exponential backoff: %w: random_source=null", k4k3ruSDKAppError.InvalidParameter())
	}
	return &ExponentialBackoff{
		config:        config,
		nextInterval:  config.InitialInterval,
		randomFloat64: randomFloat64,
	}, nil
}

// Wait waits for the next jittered backoff interval.
//
// Parameters:
//   - ctx: cancellation context.
//
// Returns:
//   - Context error when canceled before the interval elapses.
//
// Version:
//   - 2026-08-31: Added.
func (b *ExponentialBackoff) Wait(ctx context.Context) error {
	if ctx == nil {
		return k4k3ruSDKAppError.Tracef("failed to wait for exponential backoff: %w: context=null", k4k3ruSDKAppError.InvalidParameter())
	}
	if b == nil {
		return k4k3ruSDKAppError.Tracef("failed to wait for exponential backoff: %w: exponential_backoff=null", k4k3ruSDKAppError.InvalidParameter())
	}
	delay := b.nextDelay()
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// Reset resets the next interval to the configured initial interval.
//
// Version:
//   - 2026-08-31: Added.
func (b *ExponentialBackoff) Reset() {
	if b == nil {
		return
	}
	b.mu.Lock()
	b.nextInterval = b.config.InitialInterval
	b.mu.Unlock()
}

func (b *ExponentialBackoff) nextDelay() time.Duration {
	b.mu.Lock()
	defer b.mu.Unlock()

	interval := b.nextInterval
	next := float64(interval) * b.config.Multiplier
	if next >= float64(b.config.MaxInterval) {
		b.nextInterval = b.config.MaxInterval
	} else {
		b.nextInterval = time.Duration(next)
	}

	factor := 1 + (2*b.randomFloat64()-1)*b.config.JitterRate
	delay := time.Duration(float64(interval) * factor)
	if delay > b.config.MaxInterval {
		return b.config.MaxInterval
	}
	if delay < time.Nanosecond {
		return time.Nanosecond
	}
	return delay
}
