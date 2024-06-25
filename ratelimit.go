package remilia

import (
	"sync/atomic"
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

	capacity        int64
	lastestTime     int64
	fillInterval    int64
	fillQuantum     int64
	availableTokens int64
}

func NewBucket(clock Clock, capacity int64, fillInterval int64, fillQuantum int64, availableTokens int64) *Bucket {
	// TODO: check params

	bucket := &Bucket{
		clock:           clock,
		capacity:        capacity,
		lastestTime:     clock.Now().UnixNano(),
		fillInterval:    fillInterval,
		fillQuantum:     fillQuantum,
		availableTokens: availableTokens,
	}

	return bucket
}

func (b *Bucket) Take(count int64) (bool, time.Duration) {
	// TODO: check count

	now := b.clock.Now().UnixNano()
	newTokens := (now - atomic.LoadInt64(&b.lastestTime)) / b.fillInterval * b.fillQuantum

	atomic.StoreInt64(&b.lastestTime, now)
	atomic.AddInt64(&b.availableTokens, newTokens)
	if atomic.LoadInt64(&b.availableTokens) >= b.capacity {
		atomic.StoreInt64(&b.availableTokens, b.capacity)
	}

	avail := atomic.LoadInt64(&b.availableTokens) - count
	if avail >= 0 {
		// atomic.StoreInt64(&b.availableTokens, avail)
		return true, 0
	} else {
		endTime := atomic.LoadInt64(&b.lastestTime) + (-avail/b.fillQuantum)*b.fillInterval
		atomic.StoreInt64(&b.availableTokens, 0)
		atomic.StoreInt64(&b.lastestTime, endTime)

		return false, time.Duration(endTime-now) * time.Nanosecond
	}
}
