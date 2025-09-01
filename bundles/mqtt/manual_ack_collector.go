package mqtt

import (
	"fmt"
	"sync"

	"github.com/wombatwisdom/components/framework/spec"
)

// ManualAckCollector wraps a standard collector and tracks messages for manual acknowledgment
type ManualAckCollector struct {
	wrapped      spec.Collector
	messages     map[string]*TrackedMessage
	messagesLock sync.Mutex
	logger       spec.Logger
	semaphore    chan struct{} // For prefetch control
}

// NewManualAckCollector creates a new manual ACK collector
func NewManualAckCollector(wrapped spec.Collector, logger spec.Logger, prefetchCount int) *ManualAckCollector {
	sem := make(chan struct{}, prefetchCount)
	// Pre-fill semaphore
	for i := 0; i < prefetchCount; i++ {
		sem <- struct{}{}
	}

	return &ManualAckCollector{
		wrapped:   wrapped,
		messages:  make(map[string]*TrackedMessage),
		logger:    logger,
		semaphore: sem,
	}
}

// Write processes a message and tracks it for manual acknowledgment
func (c *ManualAckCollector) Write(msg spec.Message) error {
	// Check if this is a tracked message wrapper
	trackedMsg, ok := msg.(*TrackedMessageWrapper)
	if !ok {
		// Not a tracked message, just pass through
		return c.wrapped.Write(msg)
	}

	// Get message ID for tracking
	msgID := fmt.Sprintf("%d", trackedMsg.Tracked.Message.MessageID())

	// Store tracked message
	c.messagesLock.Lock()
	c.messages[msgID] = trackedMsg.Tracked
	c.messagesLock.Unlock()

	// Set metadata for tracking
	trackedMsg.SetMetadata("mqtt_tracked_id", msgID)

	// Write to wrapped collector (pass the wrapper so the adapter can handle it)
	err := c.wrapped.Write(trackedMsg)

	if err != nil {
		// Failed to write, remove from tracking
		c.messagesLock.Lock()
		delete(c.messages, msgID)
		c.messagesLock.Unlock()

		// Signal failure
		trackedMsg.Tracked.Complete(err)
		return err
	}

	// Message written successfully, it will be ACKed when pipeline completes
	return nil
}

// AckMessage acknowledges a message by ID
func (c *ManualAckCollector) AckMessage(msgID string, err error) {
	c.messagesLock.Lock()
	trackedMsg, exists := c.messages[msgID]
	if exists {
		delete(c.messages, msgID)
	}
	c.messagesLock.Unlock()

	if exists {
		// Return semaphore token
		select {
		case c.semaphore <- struct{}{}:
		default:
			// Semaphore full, shouldn't happen
			c.logger.Warnf("Semaphore full when returning token")
		}

		// Signal completion
		trackedMsg.Complete(err)
	}
}

// TryAcquire attempts to acquire a prefetch slot
func (c *ManualAckCollector) TryAcquire() bool {
	select {
	case <-c.semaphore:
		return true
	default:
		return false
	}
}

// Disconnect cleans up any pending messages
func (c *ManualAckCollector) Disconnect() error {
	c.messagesLock.Lock()
	defer c.messagesLock.Unlock()

	// Signal error to all pending messages
	for _, tracked := range c.messages {
		tracked.Complete(fmt.Errorf("collector disconnected"))
	}

	// Clear messages
	c.messages = make(map[string]*TrackedMessage)

	return c.wrapped.Disconnect()
}

// Collect implements the Collector interface
func (c *ManualAckCollector) Collect(msg spec.Message) error {
	return c.Write(msg)
}

// Flush implements the Collector interface
func (c *ManualAckCollector) Flush() (spec.Batch, error) {
	return c.wrapped.Flush()
}

// NewTrackedMessageWrapper creates a new wrapper
func NewTrackedMessageWrapper(tracked *TrackedMessage) spec.Message {
	return &TrackedMessageWrapper{
		Tracked: tracked,
		Message: NewMqttMessage(tracked.Message),
	}
}
