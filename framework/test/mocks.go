package test

import (
	"github.com/wombatwisdom/components/framework/spec"
)

// NewMockTriggerBatch creates a new mock trigger batch for testing
func NewMockTriggerBatch() spec.TriggerBatch {
	return spec.NewTriggerBatch()
}

// NewMockBatch creates a new mock batch for testing
func NewMockBatch() spec.Batch {
	return &mockBatch{
		messages: make([]spec.Message, 0),
	}
}

// NewMockMessage creates a new mock message for testing
func NewMockMessage(data []byte) spec.Message {
	return spec.NewBytesMessage(data)
}
