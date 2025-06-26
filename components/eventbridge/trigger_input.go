package eventbridge

import (
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/wombatwisdom/components/framework/spec"
)

// NewTriggerInput creates a new EventBridge trigger input component
func NewTriggerInput(ctx spec.ComponentContext, config TriggerInputConfig) (*TriggerInput, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &TriggerInput{
		config: config,
		ctx:    ctx,
		events: make(chan EventBridgeEvent, config.MaxBatchSize*2), // Buffer for events
	}, nil
}

// TriggerInput implements spec.TriggerInput for EventBridge events
type TriggerInput struct {
	config TriggerInputConfig
	ctx    spec.ComponentContext
	client *eventbridge.Client
	events chan EventBridgeEvent
	done   chan struct{}
	closed bool
}

// EventBridgeEvent represents a processed EventBridge event
type EventBridgeEvent struct {
	Source      string                 `json:"source"`
	DetailType  string                 `json:"detail-type"`
	Detail      map[string]interface{} `json:"detail"`
	Time        time.Time              `json:"time"`
	Region      string                 `json:"region"`
	Account     string                 `json:"account"`
	Resources   []string               `json:"resources"`
}

// Init initializes the EventBridge trigger input
func (t *TriggerInput) Init(ctx spec.ComponentContext) error {
	t.ctx = ctx
	
	// Create EventBridge client
	ebClient := eventbridge.NewFromConfig(t.config.Config, func(o *eventbridge.Options) {
		if t.config.EndpointURL != nil {
			o.BaseEndpoint = t.config.EndpointURL
		}
		if t.config.Region != "" {
			o.Region = t.config.Region
		}
	})
	
	t.client = ebClient
	t.done = make(chan struct{})
	
	ctx.Infof("EventBridge trigger input initialized for bus: %s, rule: %s", 
		t.config.EventBusName, t.config.RuleName)
	
	// Start event polling (in real implementation, this would be SQS or Lambda integration)
	go t.pollEvents()
	
	return nil
}

// Close shuts down the EventBridge trigger input
func (t *TriggerInput) Close(ctx spec.ComponentContext) error {
	if t.closed {
		return nil
	}
	t.closed = true
	
	if t.done != nil {
		close(t.done)
		t.done = nil
	}
	if t.events != nil {
		close(t.events)
		t.events = nil
	}
	
	ctx.Infof("EventBridge trigger input closed")
	return nil
}

// ReadTriggers reads trigger events from EventBridge
func (t *TriggerInput) ReadTriggers(ctx spec.ComponentContext) (spec.TriggerBatch, spec.ProcessedCallback, error) {
	batch := spec.NewTriggerBatch()
	
	// If component is closed, return empty batch
	if t.closed {
		return batch, spec.NoopCallback, nil
	}
	
	// Collect events up to max batch size with timeout
	timeout := time.NewTimer(100 * time.Millisecond)
	defer timeout.Stop()
	
collecting:
	for len(batch.Triggers()) < t.config.MaxBatchSize {
		select {
		case event, ok := <-t.events:
			if !ok {
				// Channel closed
				break collecting
			}
			
			trigger := t.convertEventToTrigger(event)
			batch.Append(trigger)
			
		case <-timeout.C:
			// Timeout reached, return what we have
			break collecting
			
		case <-t.done:
			// Component shutting down
			break collecting
		}
	}
	
	return batch, spec.NoopCallback, nil
}

// convertEventToTrigger converts an EventBridge event to a trigger event
func (t *TriggerInput) convertEventToTrigger(event EventBridgeEvent) spec.TriggerEvent {
	// Extract S3 information if this is an S3 event
	reference := t.extractReference(event)
	metadata := t.extractMetadata(event)
	
	return spec.NewTriggerEvent(
		spec.TriggerSourceEventBridge,
		reference,
		metadata,
	)
}

// extractReference creates a reference string from the event
func (t *TriggerInput) extractReference(event EventBridgeEvent) string {
	// For S3 events, create s3:// URI
	if event.Source == "aws.s3" {
		if bucket, ok := event.Detail["bucket"].(map[string]interface{}); ok {
			if bucketName, ok := bucket["name"].(string); ok {
				if object, ok := event.Detail["object"].(map[string]interface{}); ok {
					if key, ok := object["key"].(string); ok {
						return fmt.Sprintf("s3://%s/%s", bucketName, key)
					}
				}
			}
		}
	}
	
	// For other events, use a generic reference
	if len(event.Resources) > 0 {
		return event.Resources[0]
	}
	
	return fmt.Sprintf("%s-%d", event.Source, time.Now().UnixNano())
}

// extractMetadata extracts relevant metadata from the event
func (t *TriggerInput) extractMetadata(event EventBridgeEvent) map[string]any {
	metadata := map[string]any{
		"event_source":    event.Source,
		"detail_type":     event.DetailType,
		"event_time":      event.Time.Unix(),
		"event_region":    event.Region,
		"event_account":   event.Account,
		"event_resources": event.Resources,
	}
	
	// Extract S3-specific metadata
	if event.Source == "aws.s3" {
		t.extractS3Metadata(event, metadata)
	}
	
	// Add custom detail fields
	for key, value := range event.Detail {
		metadata[fmt.Sprintf("detail_%s", key)] = value
	}
	
	return metadata
}

// extractS3Metadata extracts S3-specific metadata from the event
func (t *TriggerInput) extractS3Metadata(event EventBridgeEvent, metadata map[string]any) {
	if bucket, ok := event.Detail["bucket"].(map[string]interface{}); ok {
		if bucketName, ok := bucket["name"].(string); ok {
			metadata[spec.MetadataBucket] = bucketName
		}
	}
	
	if object, ok := event.Detail["object"].(map[string]interface{}); ok {
		if key, ok := object["key"].(string); ok {
			metadata[spec.MetadataKey] = key
		}
		if size, ok := object["size"].(float64); ok {
			metadata[spec.MetadataSize] = int64(size)
		}
		if etag, ok := object["etag"].(string); ok {
			metadata[spec.MetadataETag] = strings.Trim(etag, "\"")
		}
	}
	
	if eventName, ok := event.Detail["eventName"].(string); ok {
		metadata[spec.MetadataEventName] = eventName
	}
}

// pollEvents simulates polling for EventBridge events
// In a real implementation, this would integrate with SQS, Lambda, or EventBridge Pipes
func (t *TriggerInput) pollEvents() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			// Simulate receiving an S3 event (in real implementation, this would come from SQS/Lambda)
			if t.shouldSimulateEvent() {
				event := t.createSimulatedS3Event()
				select {
				case t.events <- event:
					t.ctx.Debugf("Received EventBridge event: %s", event.DetailType)
				default:
					// Channel full, drop event
					t.ctx.Warnf("Event channel full, dropping event")
				}
			}
			
		case <-t.done:
			return
		}
	}
}

// shouldSimulateEvent determines if we should create a simulated event (for testing)
func (t *TriggerInput) shouldSimulateEvent() bool {
	// Disable event simulation in test environment
	// In real implementation, this would be controlled by configuration
	return false
}

// createSimulatedS3Event creates a sample S3 event for testing
func (t *TriggerInput) createSimulatedS3Event() EventBridgeEvent {
	return EventBridgeEvent{
		Source:     "aws.s3",
		DetailType: "Object Created",
		Detail: map[string]interface{}{
			"eventName": "ObjectCreated:Put",
			"bucket": map[string]interface{}{
				"name": "test-bucket",
			},
			"object": map[string]interface{}{
				"key":  fmt.Sprintf("data/file-%d.json", time.Now().Unix()),
				"size": float64(1024),
				"etag": "\"d41d8cd98f00b204e9800998ecf8427e\"",
			},
		},
		Time:      time.Now(),
		Region:    t.config.Region,
		Account:   "123456789012",
		Resources: []string{"arn:aws:s3:::test-bucket"},
	}
}