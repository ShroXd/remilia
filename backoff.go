package remilia

import (
	"context"
	"math"
	"math/rand"
	"time"
)

type randomWrapper interface {
	Int63n(n int64) int64
}

type defaultRandom struct{}

func (r *defaultRandom) Int63n(n int64) int64 {
	return rand.Int63n(n)
}

type backoff interface {
	Reset()
	Next() time.Duration
	GetMaxAttempt() uint8
	GetCurrentAttempt() uint8
}

type exponentialBackoff struct {
	minDelay      time.Duration
	maxDelay      time.Duration
	multiplier    float64
	attempt       uint8
	maxAttempt    uint8
	linearAttempt uint8

	random  randomWrapper
	backoff jitterBackoff
}

var (
	defaultMinDelay      = 100 * time.Millisecond
	defaultMaxDelay      = 10 * time.Second
	defaultMultiplier    = 2.0
	defaultMaxAttempt    = uint8(10)
	defaultLinearAttempt = uint8(5)
	defaultRandomStruct  = &defaultRandom{}
)

func newExponentialBackoff(optFns ...exponentialBackoffOptionFunc) *exponentialBackoff {
	eb := &exponentialBackoff{
		minDelay:      defaultMinDelay,
		maxDelay:      defaultMaxDelay,
		multiplier:    defaultMultiplier,
		attempt:       0,
		maxAttempt:    defaultMaxAttempt,
		linearAttempt: defaultLinearAttempt,
		random:        defaultRandomStruct,
	}

	// TODO: return the error from option func
	for _, optFn := range optFns {
		optFn(eb)
	}

	eb.backoff = fullJitterBuilder(eb.minDelay, eb.maxDelay, eb.multiplier, eb.random)
	eb.Reset()

	return eb
}

type exponentialBackoffOptionFunc optionFunc[*exponentialBackoff]

func withMinDelay(d time.Duration) exponentialBackoffOptionFunc {
	return func(eb *exponentialBackoff) error {
		eb.minDelay = d
		return nil
	}
}

func withMaxDelay(d time.Duration) exponentialBackoffOptionFunc {
	return func(eb *exponentialBackoff) error {
		eb.maxDelay = d
		return nil
	}
}

func withMultiplier(m float64) exponentialBackoffOptionFunc {
	return func(eb *exponentialBackoff) error {
		eb.multiplier = m
		return nil
	}
}

func withRandomImp(r randomWrapper) exponentialBackoffOptionFunc {
	return func(eb *exponentialBackoff) error {
		eb.random = r
		return nil
	}
}

func withMaxAttempt(a uint8) exponentialBackoffOptionFunc {
	return func(eb *exponentialBackoff) error {
		eb.maxAttempt = a
		return nil
	}
}

func withLinearAttempt(a uint8) exponentialBackoffOptionFunc {
	return func(eb *exponentialBackoff) error {
		eb.linearAttempt = a
		return nil
	}
}

func (eb *exponentialBackoff) Reset() {
	eb.attempt = 0
}

func (eb *exponentialBackoff) Next() time.Duration {
	eb.attempt++
	delay := eb.backoff(eb.attempt)

	return delay
}

func (eb *exponentialBackoff) GetMaxAttempt() uint8 {
	return eb.maxAttempt
}

func (eb *exponentialBackoff) GetCurrentAttempt() uint8 {
	return eb.attempt
}

type jitterBackoff func(attempt uint8) time.Duration

func fullJitterBuilder(minDelay time.Duration, capacity time.Duration, multiplier float64, random randomWrapper) jitterBackoff {
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

type exponentialBackoffFactory struct {
	opts []exponentialBackoffOptionFunc
}

func newExponentialBackoffFactory(opts ...exponentialBackoffOptionFunc) *exponentialBackoffFactory {
	return &exponentialBackoffFactory{
		opts: opts,
	}
}

func (f *exponentialBackoffFactory) New() *exponentialBackoff {
	return newExponentialBackoff(f.opts...)
}

func (f *exponentialBackoffFactory) Reset(eb *exponentialBackoff) {
	eb.Reset()
}

type retryableFunc func() error

func retry(ctx context.Context, op retryableFunc, eb backoff) error {
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
