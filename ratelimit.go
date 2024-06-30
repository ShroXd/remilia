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

type Bucket struct {
	clock Clock
	mu    sync.Mutex

	capacity        int64
	fillQuantum     int64
	availableTokens int64
	lastestTime     time.Time
	fillInterval    time.Duration
}

// TODO: fuck these params' type, we should use time instead of fucking int64
func NewBucket(clock Clock, capacity int64, fillInterval time.Duration, fillQuantum int64, availableTokens int64) *Bucket {
	// TODO: check params

	bucket := &Bucket{
		clock:           clock,
		capacity:        capacity,
		lastestTime:     clock.Now(),
		fillInterval:    fillInterval,
		fillQuantum:     fillQuantum,
		availableTokens: availableTokens,
	}

	return bucket
}

// TODO: use tick instead of time
func (b *Bucket) Take(count int64) time.Duration {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := b.clock.Now()
	elapsed := now.Sub(b.lastestTime)
	intervals := int64(elapsed / b.fillInterval)
	newTokens := intervals * b.fillQuantum

	b.lastestTime = now
	avail := b.availableTokens + newTokens
	if avail >= b.capacity {
		avail = b.capacity
	}

	avail = avail - count
	if avail >= 0 {
		b.availableTokens = avail
		return 0
	} else {
		endTime := b.lastestTime.Add(time.Duration(-avail/b.fillQuantum) * (b.fillInterval))
		b.availableTokens = 0
		b.lastestTime = endTime

		return endTime.Sub(now)
	}
}

func (b *Bucket) Wrap(op func() error) ExecutableFunc {
	return func() error {
		wait := b.Take(1)
		if wait > 0 {
			b.clock.Sleep(wait)
		}
		return op()
	}
}
