//go:build mqclient

package ibm_mq

import (
	"github.com/wombatwisdom/components/framework/spec"
)

// OutputConfig defines configuration for IBM MQ output
type OutputConfig struct {
	CommonMQConfig

	// The IBM MQ queue name to write messages to
	QueueName string `json:"queue_name" yaml:"queue_name"`

	// An expression to dynamically determine the queue name based on message content
	// If set, this overrides the static queue_name for each message
	QueueExpr spec.Expression `json:"queue_expr,omitempty" yaml:"queue_expr,omitempty"`

	// Number of parallel queue connections to use (default: 1)
	NumThreads int `json:"num_threads" yaml:"num_threads"`

	// Metadata configuration for filtering message headers
	Metadata *MetadataConfig `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// MetadataConfig defines metadata filtering options
type MetadataConfig struct {
	// Patterns to match metadata fields
	Patterns []string `json:"patterns" yaml:"patterns"`

	// If true, exclude matching patterns; if false, include only matching patterns
	Invert bool `json:"invert" yaml:"invert"`
}