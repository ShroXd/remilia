package remilia

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewBucket(t *testing.T) {
	t.Run("Successfully build with default value", func(t *testing.T) {
		bucket := NewBucket(realClock{}, 10, 1, 1, 10)
		assert.NotNil(t, bucket, "NewBucket() should return a non-nil bucket")
	})
}

func TestRatelimit(t *testing.T) {
	t.Run("Successfully Take", func(t *testing.T) {
		bucket := NewBucket(realClock{}, 10, 1, 1, 10)
		ok, _ := bucket.Take(1)
		assert.True(t, ok, "Take() should return true")
	})

	t.Run("Delay Take", func(t *testing.T) {
		bucket := NewBucket(realClock{}, 10, 1, 1, 10)
		bucket.Take(12)
		ok, duration := bucket.Take(1)
		assert.False(t, ok, "Take() should return false")
		assert.NotEqual(t, 0, duration, "Take() should return a non-zero duration")
	})
}
