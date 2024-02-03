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

func TestNewPipelineSuccess(t *testing.T) {
	_, err := newPipeline[any](MockProcessorDef, MockProcessorDef, MockProcessorDef)
	assert.NoError(t, err, "newPipeline should not return error")
}

func TestNewPipelineProducerError(t *testing.T) {
	errorProducerDef := func() (*processor[any], error) {
		return nil, errors.New("producer error")
	}
	_, err := newPipeline[any](errorProducerDef, MockProcessorDef)
	assert.Error(t, err, "newPipeline should have failed with producer error")
}

func TestNewPipelineStageError(t *testing.T) {
	errorStageDef := func() (*processor[any], error) {
		return nil, errors.New("stage error")
	}
	normalStageDef := func() (*processor[any], error) {
		return &processor[any]{}, nil
	}

	_, err := newPipeline[any](MockProcessorDef, errorStageDef, normalStageDef)
	assert.Error(t, err, "newPipeline should have failed with stage error")
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

func TestExecutor(t *testing.T) {
	t.Run("Successful execute", func(t *testing.T) {
		mockExec := new(mockExecutor)
		eg := new(errgroup.Group)

		mockExec.On("execute").Return(nil)
		mockExec.On("outputChannelCloser").Return(func() {})
		mockExec.On("exhaustInputChannel").Return()

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

		execute(eg, mockExec)
		err := eg.Wait()

		assert.Error(t, err, "execute should return error")
		assert.Equal(t, "test execute error", err.Error(), "execute should return correct error")
		mockExec.AssertExpectations(t)
	})
}
