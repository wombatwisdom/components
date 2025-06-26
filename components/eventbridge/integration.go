package eventbridge

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/pipes"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/wombatwisdom/components/framework/spec"
)

// EventIntegration defines the interface for different EventBridge integration modes
type EventIntegration interface {
	// Init initializes the integration
	Init(ctx context.Context, logger spec.Logger) error

	// ReadEvents reads events from the integration source
	ReadEvents(ctx context.Context, maxEvents int, timeout time.Duration) ([]EventBridgeEvent, error)

	// Close shuts down the integration
	Close(ctx context.Context) error
}

// IntegrationFactory creates the appropriate integration based on configuration
type IntegrationFactory struct {
	config TriggerInputConfig
}

// NewIntegrationFactory creates a new integration factory
func NewIntegrationFactory(config TriggerInputConfig) *IntegrationFactory {
	return &IntegrationFactory{config: config}
}

// CreateIntegration creates the appropriate integration based on the mode
func (f *IntegrationFactory) CreateIntegration() (EventIntegration, error) {
	switch f.config.Mode {
	case SQSMode:
		sqsClient := sqs.NewFromConfig(f.config.Config, func(o *sqs.Options) {
			if f.config.EndpointURL != nil {
				o.BaseEndpoint = f.config.EndpointURL
			}
			if f.config.Region != "" {
				o.Region = f.config.Region
			}
		})
		return NewSQSIntegration(f.config, sqsClient), nil

	case PipesMode:
		pipesClient := pipes.NewFromConfig(f.config.Config, func(o *pipes.Options) {
			if f.config.EndpointURL != nil {
				o.BaseEndpoint = f.config.EndpointURL
			}
			if f.config.Region != "" {
				o.Region = f.config.Region
			}
		})
		eventBridgeClient := eventbridge.NewFromConfig(f.config.Config, func(o *eventbridge.Options) {
			if f.config.EndpointURL != nil {
				o.BaseEndpoint = f.config.EndpointURL
			}
			if f.config.Region != "" {
				o.Region = f.config.Region
			}
		})
		return NewPipesIntegration(f.config, pipesClient, eventBridgeClient), nil

	case SimulationMode:
		return NewSimulationIntegration(f.config), nil

	default:
		return nil, ErrInvalidIntegrationMode
	}
}
