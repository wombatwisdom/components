package mqtt

import (
    "bytes"
    mqtt2 "github.com/eclipse/paho.mqtt.golang"
    "github.com/wombatwisdom/components/spec"
    "io"
    "iter"
    "slices"
)

const (
    MetaDuplicate = "mqtt_duplicate"
    MetaQos       = "mqtt_qos"
    MetaRetained  = "mqtt_retained"
    MetaTopic     = "mqtt_topic"
    MetaMessageId = "mqtt_message_id"
)

func NewMqttMessage(m mqtt2.Message) spec.Message {
    return &message{m: m}
}

type message struct {
    m mqtt2.Message
}

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

    return nil
}

func (m *message) Meta() spec.Metadata {
    return m
}

func (m *message) Data() (io.Reader, error) {
    return bytes.NewBuffer(m.m.Payload()), nil
}

func (m *message) Ack() error {
    m.m.Ack()
    return nil
}

func (m *message) Nack() error { return nil }
