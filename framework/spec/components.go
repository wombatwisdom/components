package spec

// Component is the base interface for all pipeline components
type Component interface {
	Init(ctx ComponentContext) error
	Close(ctx ComponentContext) error
}

// Input represents a component that reads data from external sources
type Input interface {
	Component
	Read(ctx ComponentContext) (Batch, ProcessedCallback, error)
}

// Output represents a component that writes data to external destinations
type Output interface {
	Component
	Write(ctx ComponentContext, batch Batch) error
}

// Processor represents a component that transforms data between inputs and outputs
type Processor interface {
	Component
	Process(ctx ComponentContext, batch Batch) (Batch, ProcessedCallback, error)
}

// Trigger-Retrieval Pattern Interfaces
// ===================================

// TriggerInput emits lightweight trigger events that reference data locations
// without fetching the actual data. Designed for event-driven architectures.
type TriggerInput interface {
	Component
	ReadTriggers(ctx ComponentContext) (TriggerBatch, ProcessedCallback, error)
}

// RetrievalProcessor fetches actual data based on trigger events.
// Enables filtering and validation before expensive retrieval operations.
type RetrievalProcessor interface {
	Component
	Retrieve(ctx ComponentContext, triggers TriggerBatch) (Batch, ProcessedCallback, error)
}

// SelfContainedInput represents inputs where the trigger IS the data
// (like NATS, MQTT, Kafka). No separation of trigger/retrieval needed.
type SelfContainedInput interface {
	Input // Inherits standard Input behavior
}

// TriggerBatch contains lightweight trigger events
type TriggerBatch interface {
	Triggers() []TriggerEvent
	Append(trigger TriggerEvent)
}

// TriggerEvent represents a lightweight reference to data that can be retrieved
type TriggerEvent interface {
	// Source identifies the trigger mechanism (e.g., "eventbridge", "s3-polling", "generate")
	Source() string

	// Reference provides the data location/identifier (e.g., S3 key, event ID)
	Reference() string

	// Metadata contains trigger-specific context
	Metadata() map[string]any

	// Timestamp when the trigger was generated
	Timestamp() int64
}
