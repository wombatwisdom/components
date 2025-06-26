package eventbridge

import (
	"context"
	"fmt"
	"time"

	"github.com/wombatwisdom/components/framework/spec"
)

// SimulationIntegration implements EventIntegration for testing/development
type SimulationIntegration struct {
	config TriggerInputConfig
	logger spec.Logger
}

// NewSimulationIntegration creates a new simulation integration
func NewSimulationIntegration(config TriggerInputConfig) *SimulationIntegration {
	return &SimulationIntegration{
		config: config,
	}
}

// Init initializes the simulation integration
func (s *SimulationIntegration) Init(ctx context.Context, logger spec.Logger) error {
	s.logger = logger
	s.logger.Infof("Simulation integration initialized for event source: %s", s.config.EventSource)
	return nil
}

// ReadEvents generates simulated events for testing
func (s *SimulationIntegration) ReadEvents(ctx context.Context, maxEvents int, timeout time.Duration) ([]EventBridgeEvent, error) {
	events := make([]EventBridgeEvent, 0, maxEvents)
	
	// Only generate events occasionally to avoid flooding tests
	if s.shouldGenerateEvent() {
		event := s.createSimulatedEvent()
		events = append(events, event)
		s.logger.Debugf("Generated simulated event: %s", event.DetailType)
	}
	
	// Wait for the timeout to simulate real polling behavior
	select {
	case <-time.After(timeout):
		// Timeout reached, return what we have
		return events, nil
	case <-ctx.Done():
		// Context cancelled
		return events, ctx.Err()
	}
}

// shouldGenerateEvent determines if we should create a simulated event
func (s *SimulationIntegration) shouldGenerateEvent() bool {
	// Don't generate events during tests to avoid interference
	// In a real simulation scenario, you might want controllable event generation
	return false
}

// createSimulatedEvent creates a sample event based on the configured event source
func (s *SimulationIntegration) createSimulatedEvent() EventBridgeEvent {
	now := time.Now()
	
	switch s.config.EventSource {
	case "aws.s3":
		return s.createS3Event(now)
	case "aws.ec2":
		return s.createEC2Event(now)
	case "aws.rds":
		return s.createRDSEvent(now)
	default:
		return s.createGenericEvent(now)
	}
}

// createS3Event creates a simulated S3 event
func (s *SimulationIntegration) createS3Event(eventTime time.Time) EventBridgeEvent {
	return EventBridgeEvent{
		Source:     "aws.s3",
		DetailType: "Object Created",
		Detail: map[string]interface{}{
			"eventName": "ObjectCreated:Put",
			"bucket": map[string]interface{}{
				"name": "example-bucket",
			},
			"object": map[string]interface{}{
				"key":  fmt.Sprintf("data/simulated-file-%d.json", eventTime.Unix()),
				"size": float64(2048),
				"etag": "\"d41d8cd98f00b204e9800998ecf8427e\"",
			},
		},
		Time:      eventTime,
		Region:    s.config.Region,
		Account:   "123456789012",
		Resources: []string{"arn:aws:s3:::example-bucket"},
	}
}

// createEC2Event creates a simulated EC2 event
func (s *SimulationIntegration) createEC2Event(eventTime time.Time) EventBridgeEvent {
	return EventBridgeEvent{
		Source:     "aws.ec2",
		DetailType: "EC2 Instance State-change Notification",
		Detail: map[string]interface{}{
			"instance-id": "i-1234567890abcdef0",
			"state":       "running",
		},
		Time:      eventTime,
		Region:    s.config.Region,
		Account:   "123456789012",
		Resources: []string{"arn:aws:ec2:us-east-1:123456789012:instance/i-1234567890abcdef0"},
	}
}

// createRDSEvent creates a simulated RDS event
func (s *SimulationIntegration) createRDSEvent(eventTime time.Time) EventBridgeEvent {
	return EventBridgeEvent{
		Source:     "aws.rds",
		DetailType: "RDS DB Instance Event",
		Detail: map[string]interface{}{
			"EventID":         12345,
			"SourceID":        "mydb-instance",
			"EventCategories": []string{"backup"},
			"Message":         "Automated backup started",
		},
		Time:      eventTime,
		Region:    s.config.Region,
		Account:   "123456789012",
		Resources: []string{"arn:aws:rds:us-east-1:123456789012:db:mydb-instance"},
	}
}

// createGenericEvent creates a generic simulated event
func (s *SimulationIntegration) createGenericEvent(eventTime time.Time) EventBridgeEvent {
	return EventBridgeEvent{
		Source:     s.config.EventSource,
		DetailType: "Generic Event",
		Detail: map[string]interface{}{
			"message":   "This is a simulated event",
			"timestamp": eventTime.Unix(),
		},
		Time:      eventTime,
		Region:    s.config.Region,
		Account:   "123456789012",
		Resources: []string{fmt.Sprintf("arn:aws:%s:%s:123456789012:resource/simulated", s.config.EventSource, s.config.Region)},
	}
}

// Close shuts down the simulation integration
func (s *SimulationIntegration) Close(ctx context.Context) error {
	s.logger.Infof("Simulation integration closed")
	return nil
}