package remilia

import (
	"context"
	"math"
	"math/rand"
	"time"
)

type Random interface {
	Int63n(n int64) int64
}

type defaultRandom struct{}

func (r *defaultRandom) Int63n(n int64) int64 {
	return rand.Int63n(n)
}

type Backoff interface {
	Reset()
	Next() time.Duration
	GetMaxAttempt() uint8
	GetCurrentAttempt() uint8
}

type ExponentialBackoff struct {
	minDelay      time.Duration
	maxDelay      time.Duration
	multiplier    float64
	attempt       uint8
	maxAttempt    uint8
	linearAttempt uint8

	random  Random
	backoff JitterBackoff
}

var (
	DefaultMinDelay      = 100 * time.Millisecond
	DefaultMaxDelay      = 10 * time.Second
	DefaultMultiplier    = 2.0
	DefaultMaxAttempt    = uint8(10)
	DefaultLinearAttempt = uint8(5)
	DefaultRandom        = &defaultRandom{}
)

func NewExponentialBackoff(optFns ...ExponentialBackoffOptionFunc) *ExponentialBackoff {
	eb := &ExponentialBackoff{
		minDelay:      DefaultMinDelay,
		maxDelay:      DefaultMaxDelay,
		multiplier:    DefaultMultiplier,
		attempt:       0,
		maxAttempt:    DefaultMaxAttempt,
		linearAttempt: DefaultLinearAttempt,
		random:        DefaultRandom,
	}

	// TODO: return the error from option func
	for _, optFn := range optFns {
		optFn(eb)
	}

	eb.backoff = FullJitterBuilder(eb.minDelay, eb.maxDelay, eb.multiplier, eb.random)
	eb.Reset()

	return eb
}

type ExponentialBackoffOptionFunc OptionFunc[*ExponentialBackoff]

func WithMinDelay(d time.Duration) ExponentialBackoffOptionFunc {
	return func(eb *ExponentialBackoff) error {
		eb.minDelay = d
		return nil
	}
}

func WithMaxDelay(d time.Duration) ExponentialBackoffOptionFunc {
	return func(eb *ExponentialBackoff) error {
		eb.maxDelay = d
		return nil
	}
}

func WithMultiplier(m float64) ExponentialBackoffOptionFunc {
	return func(eb *ExponentialBackoff) error {
		eb.multiplier = m
		return nil
	}
}

func WithRandomImp(r Random) ExponentialBackoffOptionFunc {
	return func(eb *ExponentialBackoff) error {
		eb.random = r
		return nil
	}
}

func WithMaxAttempt(a uint8) ExponentialBackoffOptionFunc {
	return func(eb *ExponentialBackoff) error {
		eb.maxAttempt = a
		return nil
	}
}

func WithLinearAttempt(a uint8) ExponentialBackoffOptionFunc {
	return func(eb *ExponentialBackoff) error {
		eb.linearAttempt = a
		return nil
	}
}

func (eb *ExponentialBackoff) Reset() {
	eb.attempt = 0
}

func (eb *ExponentialBackoff) Next() time.Duration {
	eb.attempt++
	delay := eb.backoff(eb.attempt)

	return delay
}

func (eb *ExponentialBackoff) GetMaxAttempt() uint8 {
	return eb.maxAttempt
}

func (eb *ExponentialBackoff) GetCurrentAttempt() uint8 {
	return eb.attempt
}

type JitterBackoff func(attempt uint8) time.Duration

func FullJitterBuilder(minDelay time.Duration, capacity time.Duration, multiplier float64, random Random) JitterBackoff {
	return func(attempt uint8) time.Duration {
		cap := float64(capacity)
		att := float64(attempt)
		base := float64(minDelay)

		// TODO: switch to linear backoff after a certain number of attempts
		temp := math.Min(cap, base*math.Pow(att, multiplier))
		diff := int64(temp) - int64(base)
		if diff <= 0 {
			diff = 1
		}
		sleep := random.Int63n(diff) + int64(base)

		return time.Duration(sleep)
	}
}

type ExponentialBackoffFactory struct {
	opts []ExponentialBackoffOptionFunc
}

func NewExponentialBackoffFactory(opts ...ExponentialBackoffOptionFunc) *ExponentialBackoffFactory {
	return &ExponentialBackoffFactory{
		opts: opts,
	}
}

func (f *ExponentialBackoffFactory) New() *ExponentialBackoff {
	return NewExponentialBackoff(f.opts...)
}

func (f *ExponentialBackoffFactory) Reset(eb *ExponentialBackoff) {
	eb.Reset()
}

type RetryableFunc func() error

func Retry(ctx context.Context, op RetryableFunc, eb Backoff) error {
	var lastErr error
	maxAttempts := eb.GetMaxAttempt()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			lastErr = op()
			if lastErr == nil {
				return nil
			}

			delay := eb.Next()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		if eb.GetCurrentAttempt() >= maxAttempts {
			break
		}
	}

	return lastErr
}
