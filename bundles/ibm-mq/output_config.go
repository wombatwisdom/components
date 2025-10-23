//go:build mqclient

package ibm_mq

import (
	"github.com/wombatwisdom/components/framework/spec"
)

// OutputConfig defines configuration for IBM MQ output
type OutputConfig struct {
	CommonMQConfig

	// An expression to dynamically determine the queue name based on message content
	// If set, this overrides the static queue_name for each message
	QueueExpr spec.Expression `json:"queue_expr,omitempty" yaml:"queue_expr,omitempty"`

	// Number of parallel queue connections to use (default: 1)
	NumThreads int `json:"num_threads" yaml:"num_threads"`

	// Metadata configuration for filtering message headers
	Metadata *MetadataConfig `json:"metadata,omitempty" yaml:"metadata,omitempty"`

	// The format of the message data (e.g., "MQSTR" for string, "MQHRF2" for RFH2 headers)
	// Default: "MQSTR"
	Format string `json:"format,omitempty" yaml:"format,omitempty"`

	// The Coded Character Set Identifier for the message
	// Common values: "1208" (UTF-8), "819" (ISO-8859-1)
	// Default: "1208"
	Ccsid string `json:"ccsid,omitempty" yaml:"ccsid,omitempty"`

	// The encoding of numeric data in the message
	// Common values: "546" (Linux/Windows little-endian), "273" (big-endian)
	// Default: "546"
	Encoding string `json:"encoding,omitempty" yaml:"encoding,omitempty"`
}

// MetadataConfig defines metadata filtering options
type MetadataConfig struct {
	// Patterns to match metadata fields
	Patterns []string `json:"patterns" yaml:"patterns"`

	// If true, exclude matching patterns; if false, include only matching patterns
	Invert bool `json:"invert" yaml:"invert"`
}
