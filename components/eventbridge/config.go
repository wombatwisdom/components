package eventbridge

import (
	"github.com/aws/aws-sdk-go-v2/aws"
)

// TriggerInputConfig defines the configuration for EventBridge trigger input
type TriggerInputConfig struct {
	// AWS Configuration
	aws.Config

	// EventBridge Configuration
	EventBusName string `json:"event_bus_name" yaml:"event_bus_name"`
	RuleName     string `json:"rule_name" yaml:"rule_name"`

	// Event Filtering
	EventSource     string            `json:"event_source" yaml:"event_source"`           // e.g., "aws.s3"
	DetailType      string            `json:"detail_type" yaml:"detail_type"`             // e.g., "Object Created"
	EventFilters    map[string]string `json:"event_filters" yaml:"event_filters"`         // Additional event filters
	
	// Processing Configuration  
	MaxBatchSize    int  `json:"max_batch_size" yaml:"max_batch_size"`       // Max triggers per batch
	EnableDeadLetter bool `json:"enable_dead_letter" yaml:"enable_dead_letter"` // DLQ for failed events

	// AWS SDK Options
	Region              string  `json:"region" yaml:"region"`
	EndpointURL         *string `json:"endpoint_url" yaml:"endpoint_url"`
	ForcePathStyleURLs  bool    `json:"force_path_style_urls" yaml:"force_path_style_urls"`
}

// DefaultTriggerInputConfig returns configuration with sensible defaults
func DefaultTriggerInputConfig() TriggerInputConfig {
	return TriggerInputConfig{
		EventBusName:       "default",
		MaxBatchSize:       10,
		EnableDeadLetter:   false,
		Region:             "us-east-1",
		ForcePathStyleURLs: false,
	}
}

// Validate checks the configuration for required fields and valid values
func (c *TriggerInputConfig) Validate() error {
	if c.EventBusName == "" {
		return ErrMissingEventBusName
	}
	
	if c.RuleName == "" {
		return ErrMissingRuleName
	}
	
	if c.EventSource == "" {
		return ErrMissingEventSource
	}
	
	if c.MaxBatchSize <= 0 {
		c.MaxBatchSize = 10 // Set default
	}
	
	return nil
}