package remilia

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockClock struct {
	mock.Mock
}

func (m *mockClock) Now() time.Time {
	args := m.Called()
	return args.Get(0).(time.Time)
}

func (m *mockClock) Sleep(d time.Duration) {
	m.Called(d)
}

func TestRatelimit(t *testing.T) {
	t.Run("Successfully build with default value", func(t *testing.T) {
		mockClock := new(mockClock)
		mockClock.On("Now").Return(time.Unix(0, 0))

		bucket := NewBucket(mockClock, 10, 1, 1, 10)
		assert.NotNil(t, bucket, "NewBucket() should return a non-nil bucket")
	})

	t.Run("Successfully Take", func(t *testing.T) {
		mockClock := new(mockClock)
		mockClock.On("Now").Return(time.Unix(0, 0))

		bucket := NewBucket(mockClock, 10, 1, 1, 10)
		duration := bucket.Take(1)
		assert.Equal(t, 0*time.Nanosecond, duration, "Take() should return 0 nanosecond")
	})

	t.Run("Take should return sleep time if the token is not enough", func(t *testing.T) {
		mockClock := new(mockClock)
		// Used for initialization of bucket
		mockClock.On("Now").Return(time.Unix(0, 0)).Once()
		mockClock.On("Now").Return(time.Unix(0, 0)).Once()
		mockClock.On("Now").Return(time.Unix(0, 1)).Once()

		bucket := NewBucket(mockClock, 10, 1*time.Nanosecond, 1, 10)
		bucket.Take(10)
		duration := bucket.Take(2)
		// After 1ns, there is one token in the bucket, so we need to wait for 1ns
		assert.Equal(t, 1*time.Nanosecond, duration, "Take() should return 1 nanosecond")
	})

	t.Run("Bucket should be filled after fillInterval", func(t *testing.T) {
		mockClock := new(mockClock)
		// Used for initialization of bucket
		mockClock.On("Now").Return(time.Unix(0, 0)).Once()
		mockClock.On("Now").Return(time.Unix(0, 0)).Once()
		mockClock.On("Now").Return(time.Unix(0, 1)).Once()
		mockClock.On("Now").Return(time.Unix(0, 2)).Once()

		bucket := NewBucket(mockClock, 10, 1*time.Nanosecond, 1, 10)
		bucket.Take(10)
		bucket.Take(1)
		duration := bucket.Take(1)
		// After 2ns, there are 3 tokens in the bucket, so we need to wait for 0ns
		assert.Equal(t, 0*time.Nanosecond, duration, "Take() should return 0 nanosecond")
	})
}
