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

func NewExponentialBackoff(optFns ...ExponentialBackoffOptionFn) *ExponentialBackoff {
	eb := &ExponentialBackoff{
		minDelay:      DefaultMinDelay,
		maxDelay:      DefaultMaxDelay,
		multiplier:    DefaultMultiplier,
		attempt:       0,
		maxAttempt:    DefaultMaxAttempt,
		linearAttempt: DefaultLinearAttempt,
		random:        DefaultRandom,
	}

	for _, optFn := range optFns {
		optFn(eb)
	}

	eb.backoff = FullJitterBuilder(eb.minDelay, eb.maxDelay, eb.multiplier, eb.random)
	eb.Reset()

	return eb
}

type ExponentialBackoffOptionFn func(*ExponentialBackoff)

func MinDelay(d time.Duration) ExponentialBackoffOptionFn {
	return func(eb *ExponentialBackoff) {
		eb.minDelay = d
	}
}

func MaxDelay(d time.Duration) ExponentialBackoffOptionFn {
	return func(eb *ExponentialBackoff) {
		eb.maxDelay = d
	}
}

func Multiplier(m float64) ExponentialBackoffOptionFn {
	return func(eb *ExponentialBackoff) {
		eb.multiplier = m
	}
}

func RandomImp(r Random) ExponentialBackoffOptionFn {
	return func(eb *ExponentialBackoff) {
		eb.random = r
	}
}

func MaxAttempt(a uint8) ExponentialBackoffOptionFn {
	return func(eb *ExponentialBackoff) {
		eb.maxAttempt = a
	}
}

func LinearAttempt(a uint8) ExponentialBackoffOptionFn {
	return func(eb *ExponentialBackoff) {
		eb.linearAttempt = a
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
	opts []ExponentialBackoffOptionFn
}

func NewExponentialBackoffFactory(opts ...ExponentialBackoffOptionFn) *ExponentialBackoffFactory {
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

type RetryableFunc[T any] func() (T, error)

func Retry[T any](ctx context.Context, op RetryableFunc[T], eb Backoff) (T, error) {
	var lastErr error
	var result T
	maxAttempts := eb.GetMaxAttempt()

	for {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
			result, lastErr = op()
			if lastErr == nil {
				return result, nil
			}

			delay := eb.Next()
			select {
			case <-ctx.Done():
				return result, ctx.Err()
			case <-time.After(delay):
			}
		}

		if eb.GetCurrentAttempt() >= maxAttempts {
			break
		}
	}

	return result, lastErr
}
