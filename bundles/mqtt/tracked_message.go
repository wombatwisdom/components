package mqtt

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/wombatwisdom/components/framework/spec"
)

// TrackedMessage wraps an MQTT message with a response channel for pipeline acknowledgment
type TrackedMessage struct {
	Message  mqtt.Message
	RespChan chan error
}

// NewTrackedMessage creates a new tracked message
func NewTrackedMessage(msg mqtt.Message) *TrackedMessage {
	return &TrackedMessage{
		Message:  msg,
		RespChan: make(chan error, 1),
	}
}

// ToSpecMessage converts the tracked message to a spec.Message
func (tm *TrackedMessage) ToSpecMessage() spec.Message {
	return NewMqttMessage(tm.Message)
}

// Complete signals completion of message processing
func (tm *TrackedMessage) Complete(err error) {
	select {
	case tm.RespChan <- err:
		// Successfully sent response
	default:
		// Channel already has a response, ignore
	}
}

// TrackedMessageWrapper wraps a TrackedMessage with its underlying spec.Message
type TrackedMessageWrapper struct {
	Tracked *TrackedMessage
	spec.Message
}
