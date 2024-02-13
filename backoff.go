package remilia

import (
	"math"
	"math/rand"
	"sync/atomic"
	"time"
)

type Random interface {
	Int63n(n int64) int64
}

type defaultRandom struct{}

func (r *defaultRandom) Int63n(n int64) int64 {
	return rand.Int63n(n)
}

type ExponentialBackoff struct {
	minDelay   time.Duration
	maxDelay   time.Duration
	multiplier float64
	attempt    int32

	random  Random
	backoff JitterBackoff
}

var (
	DefaultMinDelay   = 100 * time.Millisecond
	DefaultMaxDelay   = 10 * time.Second
	DefaultMultiplier = 2.0
	DefaultRandom     = &defaultRandom{}
)

func NewExponentialBackoff(optFns ...ExponentialBackoffOptionFn) *ExponentialBackoff {
	eb := &ExponentialBackoff{
		minDelay:   DefaultMinDelay,
		maxDelay:   DefaultMaxDelay,
		multiplier: DefaultMultiplier,
		attempt:    0,
		random:     DefaultRandom,
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

func (eb *ExponentialBackoff) Reset() {
	eb.attempt = 0
}

func (eb *ExponentialBackoff) Next() time.Duration {
	atomic.AddInt32(&eb.attempt, 1)
	delay := eb.backoff(eb.attempt)

	return delay
}

type JitterBackoff func(attempt int32) time.Duration

func FullJitterBuilder(minDelay time.Duration, capacity time.Duration, multiplier float64, random Random) JitterBackoff {
	return func(attempt int32) time.Duration {
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
