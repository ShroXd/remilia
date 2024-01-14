package remilia

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
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
