package remilia

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewExponentialBackoff(t *testing.T) {
	t.Run("Successfully build with default value", func(t *testing.T) {
		eb := NewExponentialBackoff()

		assert.Equal(t, DefaultMinDelay, eb.minDelay, "minDelay should be equal to DefaultMinDelay")
		assert.Equal(t, DefaultMaxDelay, eb.maxDelay, "maxDelay should be equal to DefaultMaxDelay")
		assert.Equal(t, DefaultMultiplier, eb.multiplier, "multiplier should be equal to DefaultMultiplier")
		assert.Equal(t, int32(0), eb.attempt, "attempt should be equal to 0")
		assert.Equal(t, DefaultRandom, eb.random, "random should be equal to DefaultRandom")
	})

	t.Run("Successfully build with valid options", func(t *testing.T) {
		eb := NewExponentialBackoff(
			MinDelay(10*time.Millisecond),
			MaxDelay(1*time.Second),
			Multiplier(3.0),
			RandomImp(&defaultRandom{}),
		)

		assert.Equal(t, 10*time.Millisecond, eb.minDelay, "minDelay should be equal to 10*time.Millisecond")
		assert.Equal(t, 1*time.Second, eb.maxDelay, "maxDelay should be equal to 1*time.Second")
		assert.Equal(t, 3.0, eb.multiplier, "multiplier should be equal to 3.0")
		assert.Equal(t, int32(0), eb.attempt, "attempt should be equal to 0")
		assert.Equal(t, DefaultRandom, eb.random, "random should be equal to DefaultRandom")
	})
}

func TestExponentialBackoff(t *testing.T) {
	t.Run("Successfully reset", func(t *testing.T) {
		eb := NewExponentialBackoff()

		eb.attempt = 10
		eb.Reset()

		assert.Equal(t, int32(0), eb.attempt, "attempt should be equal to 0")
	})

	t.Run("Successfully next", func(t *testing.T) {
		eb := NewExponentialBackoff()

		backoff := eb.Next()
		assert.Equal(t, DefaultMinDelay, backoff, "backoff should be equal to DefaultMinDelay")
		assert.Equal(t, int32(1), eb.attempt, "attempt should be equal to 1")
	})
}

func TestReset(t *testing.T) {
	eb := NewExponentialBackoff()

	eb.attempt = 10
	eb.Reset()

	assert.Equal(t, int32(0), eb.attempt, "attempt should be equal to 0")
}

type MockRandom struct {
	returnValues []int64
	callCount    int
}

func (m *MockRandom) Int63n(n int64) int64 {
	if m.callCount >= len(m.returnValues) {
		return 0
	}

	val := m.returnValues[m.callCount]
	m.callCount++
	return val % n
}

func TestFullJitterBuilder(t *testing.T) {
	minDelay := 1 * time.Second
	capacity := 10 * time.Second
	multiplier := 2.0
	random := &MockRandom{
		returnValues: []int64{1, 2, 3},
	}

	backoffFunc := FullJitterBuilder(minDelay, capacity, multiplier, random)

	testCases := []struct {
		attempt  int32
		expected time.Duration
	}{
		{0, 1 * time.Second},
		{1, 2 * time.Second},
		{2, 4 * time.Second},
		{3, 8 * time.Second},
	}

	for _, tc := range testCases {
		backoff := backoffFunc(tc.attempt)
		assert.InEpsilon(t, tc.expected, backoff, float64(time.Millisecond), "backoff should be equal to expected")
	}
}
