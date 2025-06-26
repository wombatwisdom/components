package eventbridge

import (
	"fmt"
	"strings"
	"time"

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
	}, nil
}

// TriggerInput implements spec.TriggerInput for EventBridge events
type TriggerInput struct {
	config      TriggerInputConfig
	ctx         spec.ComponentContext
	integration EventIntegration
	closed      bool
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
	
	// Create the appropriate integration
	factory := NewIntegrationFactory(t.config)
	integration, err := factory.CreateIntegration()
	if err != nil {
		return fmt.Errorf("failed to create integration: %w", err)
	}
	
	t.integration = integration
	
	// Initialize the integration
	err = t.integration.Init(ctx.Context(), ctx)
	if err != nil {
		return fmt.Errorf("failed to initialize integration: %w", err)
	}
	
	ctx.Infof("EventBridge trigger input initialized with %s mode for bus: %s", 
		t.config.Mode, t.config.EventBusName)
	
	return nil
}

// Close shuts down the EventBridge trigger input
func (t *TriggerInput) Close(ctx spec.ComponentContext) error {
	if t.closed {
		return nil
	}
	t.closed = true
	
	if t.integration != nil {
		err := t.integration.Close(ctx.Context())
		if err != nil {
			ctx.Warnf("Error closing integration: %v", err)
		}
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
	
	// Read events from the integration
	timeout := 100 * time.Millisecond
	events, err := t.integration.ReadEvents(ctx.Context(), t.config.MaxBatchSize, timeout)
	if err != nil {
		return batch, spec.NoopCallback, fmt.Errorf("failed to read events: %w", err)
	}
	
	// Convert events to triggers
	for _, event := range events {
		trigger := t.convertEventToTrigger(event)
		batch.Append(trigger)
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

