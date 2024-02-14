package remilia

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/sync/errgroup"
)

func MockProcessorDef() (*processor[any], error) {
	return &processor[any]{}, nil
}

func MockStageDef() (*stage[any], error) {
	return &stage[any]{}, nil
}

func TestNewPipeline(t *testing.T) {
	t.Run("NewPipeline with no stages", func(t *testing.T) {
		_, err := newPipeline[any](MockProcessorDef, MockStageDef, MockStageDef)
		assert.NoError(t, err, "newPipeline should not return error")
	})

	t.Run("NewPipeline with processor which returns error", func(t *testing.T) {
		errorProducerDef := func() (*processor[any], error) {
			return nil, errors.New("producer error")
		}
		_, err := newPipeline[any](errorProducerDef, MockStageDef)
		assert.Error(t, err, "newPipeline should have failed with producer error")
	})

	t.Run("NewPipeline with stage which returns error", func(t *testing.T) {
		errorStageDef := func() (*stage[any], error) {
			return nil, errors.New("stage error")
		}
		normalStageDef := func() (*stage[any], error) {
			return &stage[any]{}, nil
		}

		_, err := newPipeline[any](MockProcessorDef, errorStageDef, normalStageDef)
		assert.Error(t, err, "newPipeline should have failed with stage error")
	})
}

func TestPipelineExecute(t *testing.T) {
	t.Run("Successful execute", func(t *testing.T) {
		generator := NewProcessor[int](func(get Get[int], put, chew Put[int]) error {
			put(1)
			return nil
		})

		processor := NewStage[int](func(get Get[int], put, chew Put[int]) error {
			arr, _ := get()
			put(arr * 2)
			// TODO: chew has bug with closed channel
			// chew(item * 3)
			return nil
		})

		pipeline, _ := newPipeline[int](generator, processor)
		err := pipeline.execute()

		assert.NoError(t, err, "execute should not return error")
	})

	t.Run("Failed execute when any stage returns error", func(t *testing.T) {
		generator := NewProcessor[int](func(get Get[int], put, chew Put[int]) error {
			put(1)
			return nil
		})

		errProcessor := NewStage[int](func(get Get[int], put, chew Put[int]) error {
			return errors.New("test error")
		})

		pipeline, _ := newPipeline[int](generator, errProcessor)
		err := pipeline.execute()

		assert.Error(t, err, "execute should return error")
		assert.Equal(t, "test error", err.Error(), "execute should return correct error")
	})
}

type mockExecutor struct {
	executeErr error
	mock.Mock
}

func (m *mockExecutor) execute() error {
	args := m.Called()
	return args.Error(0)
}

func (m *mockExecutor) outputChannelCloser() func() {
	args := m.Called()
	return args.Get(0).(func())
}

func (m *mockExecutor) exhaustInputChannel() {
	m.Called()
}

func (m *mockExecutor) concurrency() uint {
	args := m.Called()
	return args.Get(0).(uint)
}

func TestExecutor(t *testing.T) {
	t.Run("Successful execute", func(t *testing.T) {
		mockExec := new(mockExecutor)
		eg := new(errgroup.Group)

		mockExec.On("execute").Return(nil)
		mockExec.On("outputChannelCloser").Return(func() {})
		mockExec.On("exhaustInputChannel").Return()
		mockExec.On("concurrency").Return(uint(1))

		execute(eg, mockExec)
		err := eg.Wait()

		assert.NoError(t, err, "execute should not return error")
		mockExec.AssertExpectations(t)
	})

	t.Run("Failed execute", func(t *testing.T) {
		mockExec := new(mockExecutor)
		eg := new(errgroup.Group)

		mockExec.On("execute").Return(errors.New("test execute error"))
		mockExec.On("outputChannelCloser").Return(func() {})
		mockExec.On("exhaustInputChannel").Return()
		mockExec.On("concurrency").Return(uint(1))

		execute(eg, mockExec)
		err := eg.Wait()

		assert.Error(t, err, "execute should return error")
		assert.Equal(t, "test execute error", err.Error(), "execute should return correct error")
		mockExec.AssertExpectations(t)
	})
}
