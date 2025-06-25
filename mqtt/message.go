package mqtt

import (
	"iter"
	"slices"

	mqtt2 "github.com/eclipse/paho.mqtt.golang"
	"github.com/wombatwisdom/components/spec"
)

const (
	MetaDuplicate = "mqtt_duplicate"
	MetaQos       = "mqtt_qos"
	MetaRetained  = "mqtt_retained"
	MetaTopic     = "mqtt_topic"
	MetaMessageId = "mqtt_message_id"
)

func NewMqttMessage(m mqtt2.Message) spec.Message {
	return &message{
		m:        m,
		metadata: make(map[string]any),
	}
}

type message struct {
	m        mqtt2.Message
	metadata map[string]any
}

func (m *message) SetMetadata(key string, value any) {
	m.metadata[key] = value
}

func (m *message) SetRaw(b []byte) {
	// MQTT messages are immutable, so this is a no-op
	// In a real implementation, you might store this separately
}

func (m *message) Raw() ([]byte, error) {
	return m.m.Payload(), nil
}

func (m *message) Metadata() iter.Seq2[string, any] {
	return func(yield func(string, any) bool) {
		// Add MQTT-specific metadata
		metadata := map[string]any{
			MetaDuplicate: m.m.Duplicate(),
			MetaQos:       int(m.m.Qos()),
			MetaRetained:  m.m.Retained(),
			MetaTopic:     m.m.Topic(),
			MetaMessageId: int(m.m.MessageID()),
		}

		// Add any custom metadata
		for k, v := range m.metadata {
			metadata[k] = v
		}

		// Yield all metadata
		for k, v := range metadata {
			if !yield(k, v) {
				return
			}
		}
	}
}

// Legacy methods for backward compatibility
func (m *message) Keys() iter.Seq[string] {
	return slices.Values([]string{MetaDuplicate, MetaQos, MetaRetained, MetaTopic, MetaMessageId})
}

func (m *message) Get(key string) any {
	switch key {
	case MetaDuplicate:
		return m.m.Duplicate()
	case MetaQos:
		return int(m.m.Qos())
	case MetaRetained:
		return m.m.Retained()
	case MetaTopic:
		return m.m.Topic()
	case MetaMessageId:
		return int(m.m.MessageID())
	}

	return m.metadata[key]
}

func (m *message) Meta() spec.Metadata {
	return m
}

func (m *message) Ack() error {
	m.m.Ack()
	return nil
}

func (m *message) Nack() error { return nil }
