// Package spec defines the core interfaces and abstractions for WombatWisdom components.
//
// WombatWisdom implements a component-based data processing framework with two main patterns:
//
// # Standard Pattern
//
// Traditional data processing components:
//   - Input: Reads data from external sources
//   - Processor: Transforms data between inputs and outputs
//   - Output: Writes data to external destinations
//
// # Trigger-Retrieval Pattern
//
// Event-driven pattern that separates event detection from data retrieval:
//   - TriggerInput: Emits lightweight events referencing data locations
//   - RetrievalProcessor: Fetches actual data based on trigger events
//
// This pattern enables:
//   - Efficient filtering before expensive operations
//   - Event deduplication and ordering
//   - Graceful handling of data availability delays
//   - Better resource utilization
//
// # Self-Contained Pattern
//
// For streaming systems where events contain the actual data:
//   - SelfContainedInput: Events include all necessary data payload
//
// Examples: NATS messages, MQTT payloads, Kafka records.
package spec

// Component is the base interface for all pipeline components.
// All components must be able to initialize and cleanup their resources.
type Component interface {
	// Init initializes the component with the given context.
	// This is called once before the component starts processing.
	Init(ctx ComponentContext) error

	// Close releases any resources held by the component.
	// This is called when the component is being shut down.
	Close(ctx ComponentContext) error
}

// Input represents a component that reads data from external sources.
// Examples include database readers, file parsers, and message queue consumers.
type Input interface {
	Component

	// Read retrieves a batch of data from the external source.
	// Returns the data batch and a callback to acknowledge processing.
	Read(ctx ComponentContext) (Batch, ProcessedCallback, error)
}

// Output represents a component that writes data to external destinations.
// Examples include database writers, file exporters, and message queue producers.
type Output interface {
	Component

	// Write sends a batch of data to the external destination.
	// The batch should be written atomically where possible.
	Write(ctx ComponentContext, batch Batch) error
}

// Processor represents a component that transforms data between inputs and outputs.
// Examples include data filters, enrichers, parsers, and format converters.
type Processor interface {
	Component

	// Process transforms the input batch and returns the processed result.
	// Returns the transformed batch and a callback to acknowledge processing.
	Process(ctx ComponentContext, batch Batch) (Batch, ProcessedCallback, error)
}

// Trigger-Retrieval Pattern Interfaces
// ===================================

// TriggerInput emits lightweight trigger events that reference data locations
// without fetching the actual data. Designed for event-driven architectures.
//
// This pattern separates event detection from data retrieval, enabling:
// - Efficient filtering before expensive operations
// - Event deduplication and ordering
// - Graceful handling of data availability delays
//
// Examples: EventBridge listeners, S3 notifications, webhook receivers.
type TriggerInput interface {
	Component

	// ReadTriggers returns a batch of trigger events referencing available data.
	// These are lightweight references, not the actual data.
	ReadTriggers(ctx ComponentContext) (TriggerBatch, ProcessedCallback, error)
}

// RetrievalProcessor fetches actual data based on trigger events.
// Enables filtering and validation before expensive retrieval operations.
//
// This component works with TriggerInput to implement the trigger-retrieval pattern:
// 1. Receives lightweight trigger events
// 2. Filters and validates triggers before retrieval
// 3. Fetches the actual data only when needed
// 4. Returns the retrieved data as a standard batch
//
// Examples: S3 object retrievers, database record fetchers, API data pullers.
type RetrievalProcessor interface {
	Component

	// Retrieve fetches actual data based on the provided trigger events.
	// Can filter triggers before retrieval to optimize performance.
	Retrieve(ctx ComponentContext, triggers TriggerBatch) (Batch, ProcessedCallback, error)
}

// SelfContainedInput represents inputs where the trigger IS the data.
// Used for streaming systems where events contain the actual data payload.
//
// Unlike the trigger-retrieval pattern, these inputs don't need separate
// data fetching because the event notification contains all necessary data.
//
// Examples: NATS messages, MQTT payloads, Kafka records, Redis streams.
type SelfContainedInput interface {
	Input // Inherits standard Input behavior
}

// TriggerBatch contains a collection of lightweight trigger events.
// Used to group related triggers for efficient processing.
type TriggerBatch interface {
	// Triggers returns all trigger events in this batch.
	Triggers() []TriggerEvent

	// Append adds a new trigger event to this batch.
	Append(trigger TriggerEvent)
}

// TriggerEvent represents a lightweight reference to data that can be retrieved.
// This is the core abstraction of the trigger-retrieval pattern, providing
// just enough information to identify and locate data without fetching it.
type TriggerEvent interface {
	// Source identifies the trigger mechanism that generated this event.
	// Examples: "eventbridge", "s3-polling", "webhook", "generate"
	Source() string

	// Reference provides the data location or identifier for retrieval.
	// Examples: S3 object key, database record ID, API endpoint URL
	Reference() string

	// Metadata contains trigger-specific context and additional parameters.
	// This can include filtering criteria, authentication tokens, or processing hints.
	Metadata() map[string]any

	// Timestamp returns when the trigger was generated (Unix timestamp).
	// Used for ordering, deduplication, and expiration logic.
	Timestamp() int64
}
