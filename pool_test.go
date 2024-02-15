package remilia

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockFactory[T any] struct {
	mock.Mock
}

func (m *mockFactory[T]) New() T {
	args := m.Called()
	return args.Get(0).(T)
}

func (m *mockFactory[T]) Reset(item T) {
	m.Called(item)
}

func TestNewPool(t *testing.T) {
	var mockFactory mockFactory[int]
	mockFactory.On("New").Return(0)

	pool := newPool[int](&mockFactory)

	assert.NotNil(t, pool, "pool should not be nil")
	assert.IsType(t, &abstractPool[int]{}, pool, "pool should be of type *Pool[int]")
}

func TestPoolOperations(t *testing.T) {
	t.Run("Get", func(t *testing.T) {
		var mockFactory mockFactory[int]
		mockFactory.On("New").Return(0)

		pool := newPool[int](&mockFactory)
		item := pool.get()

		assert.Equal(t, 0, item, "item should be 0")
	})

	t.Run("Put", func(t *testing.T) {
		var mockFactory mockFactory[int]
		mockFactory.On("New").Return(0)
		mockFactory.On("Reset", 0).Return()

		pool := newPool[int](&mockFactory)
		item := pool.get()
		pool.put(item)

		retrievedItem := pool.get()
		assert.Equal(t, 0, retrievedItem, "retrievedItem should be 0")
		mockFactory.AssertExpectations(t)
	})
}
