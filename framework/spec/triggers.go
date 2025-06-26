package spec

import "time"

// NewTriggerBatch creates a new empty trigger batch
func NewTriggerBatch() TriggerBatch {
	return &triggerBatch{
		triggers: make([]TriggerEvent, 0),
	}
}

// triggerBatch implements TriggerBatch interface
type triggerBatch struct {
	triggers []TriggerEvent
}

func (tb *triggerBatch) Triggers() []TriggerEvent {
	return tb.triggers
}

func (tb *triggerBatch) Append(trigger TriggerEvent) {
	tb.triggers = append(tb.triggers, trigger)
}

// NewTriggerEvent creates a new trigger event
func NewTriggerEvent(source, reference string, metadata map[string]any) TriggerEvent {
	if metadata == nil {
		metadata = make(map[string]any)
	}
	return &triggerEvent{
		source:    source,
		reference: reference,
		metadata:  metadata,
		timestamp: time.Now().UnixNano(),
	}
}

// triggerEvent implements TriggerEvent interface
type triggerEvent struct {
	source    string
	reference string
	metadata  map[string]any
	timestamp int64
}

func (te *triggerEvent) Source() string {
	return te.source
}

func (te *triggerEvent) Reference() string {
	return te.reference
}

func (te *triggerEvent) Metadata() map[string]any {
	return te.metadata
}

func (te *triggerEvent) Timestamp() int64 {
	return te.timestamp
}

// Trigger Source Constants
const (
	TriggerSourceEventBridge = "eventbridge"
	TriggerSourceS3Polling   = "s3-polling"
	TriggerSourceGenerate    = "generate"
	TriggerSourceSQS         = "sqs"
	TriggerSourceFile        = "file"
)

// Common Metadata Keys
const (
	MetadataBucket    = "bucket"
	MetadataKey       = "key"
	MetadataEventName = "event_name"
	MetadataRegion    = "region"
	MetadataTimestamp = "timestamp"
	MetadataSize      = "size"
	MetadataETag      = "etag"
)