package eventbridge

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
)

// IntegrationMode defines how EventBridge events are consumed
type IntegrationMode string

const (
	// SQSMode uses EventBridge Rules to route events to SQS, then polls SQS
	SQSMode IntegrationMode = "sqs"
	// PipesMode uses EventBridge Pipes for direct streaming integration
	PipesMode IntegrationMode = "pipes"
	// SimulationMode generates fake events for testing
	SimulationMode IntegrationMode = "simulation"
)

// TriggerInputConfig defines the configuration for EventBridge trigger input
type TriggerInputConfig struct {
	// AWS Configuration
	aws.Config

	// Integration Mode
	Mode IntegrationMode `json:"mode" yaml:"mode"` // sqs, pipes, or simulation

	// EventBridge Configuration
	EventBusName string `json:"event_bus_name" yaml:"event_bus_name"`
	RuleName     string `json:"rule_name" yaml:"rule_name"`

	// Event Filtering
	EventSource  string            `json:"event_source" yaml:"event_source"`   // e.g., "aws.s3"
	DetailType   string            `json:"detail_type" yaml:"detail_type"`     // e.g., "Object Created"
	EventFilters map[string]string `json:"event_filters" yaml:"event_filters"` // Additional event filters

	// Processing Configuration
	MaxBatchSize     int  `json:"max_batch_size" yaml:"max_batch_size"`         // Max triggers per batch
	EnableDeadLetter bool `json:"enable_dead_letter" yaml:"enable_dead_letter"` // DLQ for failed events

	// SQS Mode Configuration
	SQSQueueURL          string `json:"sqs_queue_url" yaml:"sqs_queue_url"`
	SQSMaxMessages       int32  `json:"sqs_max_messages" yaml:"sqs_max_messages"`
	SQSWaitTimeSeconds   int32  `json:"sqs_wait_time_seconds" yaml:"sqs_wait_time_seconds"`
	SQSVisibilityTimeout int32  `json:"sqs_visibility_timeout" yaml:"sqs_visibility_timeout"`

	// Pipes Mode Configuration
	PipeName      string `json:"pipe_name" yaml:"pipe_name"`
	PipeSourceARN string `json:"pipe_source_arn" yaml:"pipe_source_arn"`
	PipeTargetARN string `json:"pipe_target_arn" yaml:"pipe_target_arn"`
	PipeBatchSize int32  `json:"pipe_batch_size" yaml:"pipe_batch_size"`

	// AWS SDK Options
	Region             string  `json:"region" yaml:"region"`
	EndpointURL        *string `json:"endpoint_url" yaml:"endpoint_url"`
	ForcePathStyleURLs bool    `json:"force_path_style_urls" yaml:"force_path_style_urls"`
}

// DefaultTriggerInputConfig returns configuration with sensible defaults
func DefaultTriggerInputConfig() TriggerInputConfig {
	return TriggerInputConfig{
		Mode:               SQSMode,
		EventBusName:       "default",
		MaxBatchSize:       10,
		EnableDeadLetter:   false,
		Region:             "us-east-1",
		ForcePathStyleURLs: false,
		// SQS defaults
		SQSMaxMessages:       10,
		SQSWaitTimeSeconds:   20,
		SQSVisibilityTimeout: 30,
		// Pipes defaults
		PipeBatchSize: 10,
	}
}

// Validate checks the configuration for required fields and valid values
func (c *TriggerInputConfig) Validate() error {
	// Validate integration mode
	if c.Mode == "" {
		c.Mode = SQSMode // Default to SQS mode
	}

	if c.Mode != SQSMode && c.Mode != PipesMode && c.Mode != SimulationMode {
		return fmt.Errorf("invalid integration mode: %s (must be sqs, pipes, or simulation)", c.Mode)
	}

	// Common validation
	if c.EventBusName == "" {
		return ErrMissingEventBusName
	}

	if c.EventSource == "" {
		return ErrMissingEventSource
	}

	if c.MaxBatchSize <= 0 {
		c.MaxBatchSize = 10 // Set default
	}

	// Mode-specific validation
	switch c.Mode {
	case SQSMode:
		return c.validateSQSConfig()
	case PipesMode:
		return c.validatePipesConfig()
	case SimulationMode:
		// No additional validation needed for simulation mode
		return nil
	default:
		return fmt.Errorf("unknown integration mode: %s", c.Mode)
	}
}

// validateSQSConfig validates SQS-specific configuration
func (c *TriggerInputConfig) validateSQSConfig() error {
	if c.SQSQueueURL == "" {
		return fmt.Errorf("sqs_queue_url is required for SQS mode")
	}

	if c.RuleName == "" {
		return ErrMissingRuleName
	}

	if c.SQSMaxMessages <= 0 || c.SQSMaxMessages > 10 {
		c.SQSMaxMessages = 10 // AWS SQS limit
	}

	if c.SQSWaitTimeSeconds < 0 || c.SQSWaitTimeSeconds > 20 {
		c.SQSWaitTimeSeconds = 20 // AWS SQS max long polling
	}

	if c.SQSVisibilityTimeout <= 0 {
		c.SQSVisibilityTimeout = 30 // Default visibility timeout
	}

	return nil
}

// validatePipesConfig validates EventBridge Pipes-specific configuration
func (c *TriggerInputConfig) validatePipesConfig() error {
	if c.PipeName == "" {
		return fmt.Errorf("pipe_name is required for Pipes mode")
	}

	if c.PipeSourceARN == "" {
		return fmt.Errorf("pipe_source_arn is required for Pipes mode")
	}

	if c.PipeTargetARN == "" {
		return fmt.Errorf("pipe_target_arn is required for Pipes mode")
	}

	if c.PipeBatchSize <= 0 || c.PipeBatchSize > 10000 {
		c.PipeBatchSize = 10 // Reasonable default
	}

	return nil
}
