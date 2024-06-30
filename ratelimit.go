package remilia

import (
	"sync"
	"time"
)

type Clock interface {
	Now() time.Time
	Sleep(d time.Duration)
}

type realClock struct{}

func (realClock) Now() time.Time {
	return time.Now()
}

func (realClock) Sleep(d time.Duration) {
	time.Sleep(d)
}

type Limiter interface {
	Take() (bool, time.Duration)
}

var (
	defaultClock               = realClock{}
	defaultCapacity            = int64(100)
	defaultFillInterval        = time.Second
	defaultFillQuantum         = int64(10)
	defaultInitiallyAvailToken = int64(100)
)

type RateLimitation struct {
	clock Clock
	mu    sync.Mutex

	capacity       int64
	fillQuantum    int64
	initAvailToken int64
	lastestTime    time.Time
	fillInterval   time.Duration
}

func NewBucket(optFns ...RateLimitionOptionFunc) (*RateLimitation, error) {
	bucket := &RateLimitation{
		clock:          defaultClock,
		capacity:       defaultCapacity,
		fillInterval:   defaultFillInterval,
		fillQuantum:    defaultFillQuantum,
		initAvailToken: defaultInitiallyAvailToken,
	}

	for _, optFn := range optFns {
		if err := optFn(bucket); err != nil {
			return nil, err
		}
	}

	bucket.lastestTime = bucket.clock.Now()
	if bucket.initAvailToken > bucket.capacity {
		bucket.initAvailToken = bucket.capacity
	}

	return bucket, nil
}

func (b *RateLimitation) Take(count int64) time.Duration {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := b.clock.Now()
	elapsed := now.Sub(b.lastestTime)
	intervals := int64(elapsed / b.fillInterval)
	newTokens := intervals * b.fillQuantum

	b.lastestTime = now
	avail := b.initAvailToken + newTokens
	if avail >= b.capacity {
		avail = b.capacity
	}

	avail = avail - count
	if avail >= 0 {
		b.initAvailToken = avail
		return 0
	} else {
		endTime := b.lastestTime.Add(time.Duration(-avail/b.fillQuantum) * (b.fillInterval))
		b.initAvailToken = 0
		b.lastestTime = endTime

		return endTime.Sub(now)
	}
}

func (b *RateLimitation) Wrap(op func() error) ExecutableFunc {
	return func() error {
		wait := b.Take(1)
		if wait > 0 {
			b.clock.Sleep(wait)
		}
		return op()
	}
}

type RateLimitionOptionFunc func(*RateLimitation) error

func withLimitationClock(clock Clock) RateLimitionOptionFunc {
	return func(b *RateLimitation) error {
		b.clock = clock
		return nil
	}
}

func withLimitationCapacity(capacity int64) RateLimitionOptionFunc {
	return func(b *RateLimitation) error {
		b.capacity = capacity
		return nil
	}
}

func withLimitationFillInterval(fillInterval time.Duration) RateLimitionOptionFunc {
	return func(b *RateLimitation) error {
		b.fillInterval = fillInterval
		return nil
	}
}

func withLimitationFillQuantum(fillQuantum int64) RateLimitionOptionFunc {
	return func(b *RateLimitation) error {
		b.fillQuantum = fillQuantum
		return nil
	}
}

func withLimitationInitiallyAvailToken(initiallyAvailToken int64) RateLimitionOptionFunc {
	return func(b *RateLimitation) error {
		b.initAvailToken = initiallyAvailToken
		return nil
	}
}
