package eventbridge

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/pipes"
	"github.com/wombatwisdom/components/framework/spec"
)

// PipesIntegration implements EventIntegration for EventBridge Pipes
type PipesIntegration struct {
	config            TriggerInputConfig
	pipesClient       *pipes.Client
	eventBridgeClient *eventbridge.Client
	logger            spec.Logger
	eventChan         chan EventBridgeEvent
	stopChan          chan struct{}
}

// NewPipesIntegration creates a new EventBridge Pipes integration
func NewPipesIntegration(config TriggerInputConfig, pipesClient *pipes.Client, eventBridgeClient *eventbridge.Client) *PipesIntegration {
	return &PipesIntegration{
		config:            config,
		pipesClient:       pipesClient,
		eventBridgeClient: eventBridgeClient,
		eventChan:         make(chan EventBridgeEvent, config.MaxBatchSize*2),
		stopChan:          make(chan struct{}),
	}
}

// Init initializes the EventBridge Pipes integration
func (p *PipesIntegration) Init(ctx context.Context, logger spec.Logger) error {
	p.logger = logger
	
	// Verify pipe exists and get its configuration
	describePipeInput := &pipes.DescribePipeInput{
		Name: aws.String(p.config.PipeName),
	}
	
	pipeResp, err := p.pipesClient.DescribePipe(ctx, describePipeInput)
	if err != nil {
		return fmt.Errorf("failed to describe EventBridge pipe %s: %w", p.config.PipeName, err)
	}
	
	// Verify pipe is in running state
	if pipeResp.CurrentState != "RUNNING" {
		return fmt.Errorf("EventBridge pipe %s is not in RUNNING state: %s", p.config.PipeName, string(pipeResp.CurrentState))
	}
	
	p.logger.Infof("EventBridge Pipes integration initialized for pipe: %s", p.config.PipeName)
	
	// Note: In a real implementation, EventBridge Pipes would push events to a target
	// (like SQS, Kinesis, or Lambda). For this implementation, we'll simulate
	// receiving events through the pipe configuration.
	// In production, you'd typically configure the pipe to push to an SQS queue
	// and then poll that queue, or use a Lambda function as the target.
	
	return nil
}

// ReadEvents reads events from EventBridge Pipes
// Note: This is a simplified implementation. In reality, EventBridge Pipes
// pushes events to targets (SQS, Kinesis, Lambda, etc.) rather than being polled.
func (p *PipesIntegration) ReadEvents(ctx context.Context, maxEvents int, timeout time.Duration) ([]EventBridgeEvent, error) {
	events := make([]EventBridgeEvent, 0, maxEvents)
	
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	
	// In a real implementation, this would:
	// 1. Poll the target configured in the pipe (e.g., SQS queue)
	// 2. Or receive events via webhook/API if using HTTP targets
	// 3. Or be called by Lambda if using Lambda targets
	
	// For now, we'll return an empty batch as Pipes is a push-based system
	// In production, you'd configure the pipe target appropriately
	
	select {
	case <-timeoutCtx.Done():
		// Timeout reached, return what we have
		p.logger.Debugf("EventBridge Pipes timeout reached, returning %d events", len(events))
		return events, nil
	case <-p.stopChan:
		// Component shutting down
		return events, nil
	}
}

// Close shuts down the EventBridge Pipes integration
func (p *PipesIntegration) Close(ctx context.Context) error {
	close(p.stopChan)
	close(p.eventChan)
	p.logger.Infof("EventBridge Pipes integration closed")
	return nil
}

// GetPipeConfiguration returns the current pipe configuration
func (p *PipesIntegration) GetPipeConfiguration(ctx context.Context) (*pipes.DescribePipeOutput, error) {
	input := &pipes.DescribePipeInput{
		Name: aws.String(p.config.PipeName),
	}
	
	return p.pipesClient.DescribePipe(ctx, input)
}

// StartPipe starts the EventBridge pipe if it's stopped
func (p *PipesIntegration) StartPipe(ctx context.Context) error {
	input := &pipes.StartPipeInput{
		Name: aws.String(p.config.PipeName),
	}
	
	_, err := p.pipesClient.StartPipe(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to start EventBridge pipe %s: %w", p.config.PipeName, err)
	}
	
	p.logger.Infof("Started EventBridge pipe: %s", p.config.PipeName)
	return nil
}

// StopPipe stops the EventBridge pipe
func (p *PipesIntegration) StopPipe(ctx context.Context) error {
	input := &pipes.StopPipeInput{
		Name: aws.String(p.config.PipeName),
	}
	
	_, err := p.pipesClient.StopPipe(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to stop EventBridge pipe %s: %w", p.config.PipeName, err)
	}
	
	p.logger.Infof("Stopped EventBridge pipe: %s", p.config.PipeName)
	return nil
}