package remilia

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewExponentialBackoff(t *testing.T) {
	t.Run("Successfully build with default value", func(t *testing.T) {
		eb := newExponentialBackoff()

		assert.Equal(t, defaultMinDelay, eb.minDelay, "minDelay should be equal to DefaultMinDelay")
		assert.Equal(t, defaultMaxDelay, eb.maxDelay, "maxDelay should be equal to DefaultMaxDelay")
		assert.Equal(t, defaultMultiplier, eb.multiplier, "multiplier should be equal to DefaultMultiplier")
		assert.Equal(t, uint8(0), eb.attempt, "attempt should be equal to 0")
		assert.Equal(t, defaultRandomStruct, eb.random, "random should be equal to DefaultRandom")
	})

	t.Run("Successfully build with valid options", func(t *testing.T) {
		eb := newExponentialBackoff(
			WithWorkMinDelay(10*time.Millisecond),
			WithWorkMaxDelay(1*time.Second),
			WithWorkMultiplier(3.0),
			withRandomImp(&defaultRandom{}),
			WithWorkMaxAttempt(3),
			WithWorkLinearAttempt(2),
		)

		assert.Equal(t, 10*time.Millisecond, eb.minDelay, "minDelay should be equal to 10*time.Millisecond")
		assert.Equal(t, 1*time.Second, eb.maxDelay, "maxDelay should be equal to 1*time.Second")
		assert.Equal(t, 3.0, eb.multiplier, "multiplier should be equal to 3.0")
		assert.Equal(t, uint8(0), eb.attempt, "attempt should be equal to 0")
		assert.Equal(t, defaultRandomStruct, eb.random, "random should be equal to DefaultRandom")
		assert.Equal(t, uint8(3), eb.maxAttempt, "maxAttempt should be equal to 3")
		assert.Equal(t, uint8(2), eb.linearAttempt, "linearAttempt should be equal to 2")
	})
}

func TestExponentialBackoff(t *testing.T) {
	t.Run("Successfully reset", func(t *testing.T) {
		eb := newExponentialBackoff()

		eb.attempt = 10
		eb.Reset()

		assert.Equal(t, uint8(0), eb.attempt, "attempt should be equal to 0")
	})

	t.Run("Successfully next", func(t *testing.T) {
		eb := newExponentialBackoff()

		backoff := eb.Next()
		assert.Equal(t, defaultMinDelay, backoff, "backoff should be equal to DefaultMinDelay")
		assert.Equal(t, uint8(1), eb.attempt, "attempt should be equal to 1")
	})

	t.Run("GetMaxAttempt", func(t *testing.T) {
		eb := newExponentialBackoff()
		assert.Equal(t, defaultMaxAttempt, eb.GetMaxAttempt(), "maxAttempt should be equal to DefaultMaxAttempt")
	})

	t.Run("GetCurrentAttempt", func(t *testing.T) {
		eb := newExponentialBackoff()
		assert.Equal(t, uint8(0), eb.GetCurrentAttempt(), "attempt should be equal to 0")
	})
}

func TestReset(t *testing.T) {
	eb := newExponentialBackoff()

	eb.attempt = 10
	eb.Reset()

	assert.Equal(t, uint8(0), eb.attempt, "attempt should be equal to 0")
}

type mockRandom struct {
	returnValues []int64
	callCount    int
}

func (m *mockRandom) Int63n(n int64) int64 {
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
	random := &mockRandom{
		returnValues: []int64{1, 2, 3},
	}

	backoffFunc := fullJitterBuilder(minDelay, capacity, multiplier, random)

	testCases := []struct {
		attempt  uint8
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

type mockExponentialBackoff struct {
	Attempt    uint8
	MaxAttempt uint8
}

func (m *mockExponentialBackoff) Reset() {
	m.Attempt = 0
}

func (m *mockExponentialBackoff) Next() time.Duration {
	m.Attempt++
	return 1 * time.Millisecond
}

func (m *mockExponentialBackoff) GetMaxAttempt() uint8 {
	return m.MaxAttempt
}

func (m *mockExponentialBackoff) GetCurrentAttempt() uint8 {
	return m.Attempt
}

func TestRetry(t *testing.T) {
	t.Run("Successfully run without retry", func(t *testing.T) {
		ctx := context.Background()
		operation := func() error {
			return nil
		}
		eb := &mockExponentialBackoff{
			MaxAttempt: 3,
		}

		err := retry(ctx, operation, eb)

		assert.NoError(t, err, "err should be nil")
		assert.Equal(t, uint8(0), eb.GetCurrentAttempt(), "attempt should be equal to 0")
	})

	t.Run("Successfully run after retries", func(t *testing.T) {
		ctx := context.Background()
		failures := 2
		attempts := 0
		operation := func() error {
			attempts++
			if attempts <= failures {
				return assert.AnError
			}

			return nil
		}

		eb := &mockExponentialBackoff{
			MaxAttempt: 3,
		}

		err := retry(ctx, operation, eb)

		assert.NoError(t, err, "err should be nil")
		assert.Equal(t, uint8(failures), eb.GetCurrentAttempt(), "attempt should be equal to failures")
	})

	t.Run("Failure after all attempts", func(t *testing.T) {
		ctx := context.Background()
		failures := 3
		operation := func() error {
			return errors.New("permanent error")
		}

		eb := &mockExponentialBackoff{
			MaxAttempt: 3,
		}

		err := retry(ctx, operation, eb)

		assert.Error(t, err, "err should not be nil")
		assert.Equal(t, uint8(failures), eb.GetCurrentAttempt(), "attempt should be equal to failures")
	})

	t.Run("Failure with cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		operation := func() error {
			return errors.New("won't execute")
		}
		eb := &mockExponentialBackoff{
			MaxAttempt: 3,
		}
		cancel()

		err := retry(ctx, operation, eb)

		assert.Error(t, err, "err should not be nil")
		assert.Equal(t, context.Canceled, err, "err should be equal to context.Canceled")
	})

	t.Run("Failure with deadline exceeded context", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		operation := func() error {
			return errors.New("won't execute")
		}
		eb := &mockExponentialBackoff{
			MaxAttempt: 3,
		}
		defer cancel()

		err := retry(ctx, operation, eb)

		assert.Error(t, err, "err should not be nil")
		assert.Equal(t, context.DeadlineExceeded, err, "err should be equal to context.DeadlineExceeded")
	})

	t.Run("Failure with cancelled context after some retries", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		operation := func() error {
			cancel()
			return errors.New("won't execute")
		}
		eb := &mockExponentialBackoff{
			MaxAttempt: 3,
		}

		err := retry(ctx, operation, eb)

		assert.Error(t, err, "err should not be nil")
		assert.Equal(t, context.Canceled, err, "err should be equal to context.Canceled")
	})
}
