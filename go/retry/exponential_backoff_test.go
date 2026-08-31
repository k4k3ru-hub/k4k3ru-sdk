package retry

import (
	"context"
	"errors"
	"testing"
	"time"

	k4k3ruSDKAppError "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
)

func TestNewExponentialBackoffRejectsInvalidConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config ExponentialBackoffConfig
	}{
		{name: "initial interval", config: ExponentialBackoffConfig{MaxInterval: time.Second, Multiplier: 2}},
		{name: "max interval", config: ExponentialBackoffConfig{InitialInterval: 2 * time.Second, MaxInterval: time.Second, Multiplier: 2}},
		{name: "multiplier", config: ExponentialBackoffConfig{InitialInterval: time.Second, MaxInterval: time.Second, Multiplier: 1}},
		{name: "negative jitter", config: ExponentialBackoffConfig{InitialInterval: time.Second, MaxInterval: time.Second, Multiplier: 2, JitterRate: -0.1}},
		{name: "excessive jitter", config: ExponentialBackoffConfig{InitialInterval: time.Second, MaxInterval: time.Second, Multiplier: 2, JitterRate: 1}},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := NewExponentialBackoff(test.config)
			if !errors.Is(err, k4k3ruSDKAppError.InvalidParameter()) {
				t.Fatalf("expected invalid parameter error, got %v", err)
			}
		})
	}
}

func TestExponentialBackoffNextDelayGrowsAndCaps(t *testing.T) {
	t.Parallel()

	backoff, err := newExponentialBackoff(ExponentialBackoffConfig{
		InitialInterval: time.Second,
		MaxInterval:     5 * time.Second,
		Multiplier:      2,
	}, func() float64 { return 0.5 })
	if err != nil {
		t.Fatalf("expected backoff: %v", err)
	}
	expected := []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 5 * time.Second, 5 * time.Second}
	for index, expectedDelay := range expected {
		if delay := backoff.nextDelay(); delay != expectedDelay {
			t.Fatalf("expected delay %s at index %d, got %s", expectedDelay, index, delay)
		}
	}
}

func TestExponentialBackoffNextDelayAppliesJitter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		random   float64
		expected time.Duration
	}{
		{name: "lower bound", random: 0, expected: 800 * time.Millisecond},
		{name: "midpoint", random: 0.5, expected: time.Second},
		{name: "upper bound", random: 1, expected: 1200 * time.Millisecond},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			backoff, err := newExponentialBackoff(ExponentialBackoffConfig{
				InitialInterval: time.Second,
				MaxInterval:     2 * time.Second,
				Multiplier:      2,
				JitterRate:      0.2,
			}, func() float64 { return test.random })
			if err != nil {
				t.Fatalf("expected backoff: %v", err)
			}
			if delay := backoff.nextDelay(); delay != test.expected {
				t.Fatalf("expected delay %s, got %s", test.expected, delay)
			}
		})
	}
}

func TestExponentialBackoffResetRestoresInitialInterval(t *testing.T) {
	t.Parallel()

	backoff, err := newExponentialBackoff(ExponentialBackoffConfig{
		InitialInterval: time.Second,
		MaxInterval:     4 * time.Second,
		Multiplier:      2,
	}, func() float64 { return 0.5 })
	if err != nil {
		t.Fatalf("expected backoff: %v", err)
	}
	backoff.nextDelay()
	backoff.nextDelay()
	backoff.Reset()
	if delay := backoff.nextDelay(); delay != time.Second {
		t.Fatalf("expected reset delay %s, got %s", time.Second, delay)
	}
}

func TestExponentialBackoffWaitReturnsContextError(t *testing.T) {
	t.Parallel()

	backoff, err := NewExponentialBackoff(ExponentialBackoffConfig{
		InitialInterval: time.Hour,
		MaxInterval:     time.Hour,
		Multiplier:      2,
	})
	if err != nil {
		t.Fatalf("expected backoff: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := backoff.Wait(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
}
