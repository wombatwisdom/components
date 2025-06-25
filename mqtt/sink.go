package mqtt

import (
	"context"
	"errors"
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/wombatwisdom/components/spec"
	"sync"
	"time"
)

type SinkConfig struct {
	CommonMQTTConfig

	TopicExpr    string        `json:"topic_expr" yaml:"topic_expr"`
	WriteTimeout time.Duration `json:"write_timeout" yaml:"write_timeout"`
	RetainedExpr string        `json:"retained_expr" yaml:"retained_expr"`
	QOS          byte          `json:"qos" yaml:"qos"`
}

func NewSink(env spec.Environment, config SinkConfig) (*Sink, error) {
	topic := env.NewDynamicField(config.TopicExpr)

	var retained spec.DynamicField
	if config.RetainedExpr != "" {
		retained = env.NewDynamicField(config.RetainedExpr)
	}

	return &Sink{
		config: config,
		log:    env,

		topic:    topic,
		retained: retained,
	}, nil
}

type Sink struct {
	config SinkConfig

	log spec.Logger

	topic    spec.DynamicField
	retained spec.DynamicField

	client  mqtt.Client
	connMut sync.RWMutex
}

func (m *Sink) Connect(ctx context.Context) error {
	m.connMut.Lock()
	defer m.connMut.Unlock()

	if m.client != nil {
		return nil
	}

	opts := NewClientOptions(m.config.CommonMQTTConfig).
		SetConnectionLostHandler(func(client mqtt.Client, reason error) {
			client.Disconnect(0)
			m.log.Errorf("Connection lost due to: %v", reason)
		}).
		SetWriteTimeout(m.config.WriteTimeout)

	client := mqtt.NewClient(opts)

	tok := client.Connect()
	tok.Wait()
	if err := tok.Error(); err != nil {
		return err
	}

	m.client = client
	return nil
}

func (m *Sink) Write(ctx context.Context, message spec.Message) error {
	m.connMut.RLock()
	client := m.client
	m.connMut.RUnlock()

	if client == nil {
		return spec.ErrNotConnected
	}

	var err error
	retained := false
	if m.retained != nil {
		retained, err = m.retained.AsBool(message)
		if err != nil {
			m.log.Errorf("Retained interpolation error: %v", err)
		}
	}

	topicStr, err := m.topic.AsString(message)
	if err != nil {
		return fmt.Errorf("topic interpolation error: %w", err)
	}

	mb, err := message.Raw()
	if err != nil {
		return fmt.Errorf("failed to access message data: %w", err)
	}

	mtok := client.Publish(topicStr, m.config.QOS, retained, mb)
	mtok.Wait()
	sendErr := mtok.Error()
	if errors.Is(sendErr, mqtt.ErrNotConnected) {
		m.connMut.RLock()
		m.client = nil
		m.connMut.RUnlock()
		sendErr = spec.ErrNotConnected
	}

	if sendErr == nil {
		m.log.Infof("Message sent to topic %s", topicStr)
	} else {
		m.log.Errorf("Failed to send message to topic %s: %v", topicStr, sendErr)
	}

	return sendErr
}

func (m *Sink) Disconnect(context.Context) error {
	m.connMut.Lock()
	defer m.connMut.Unlock()

	if m.client != nil {
		m.client.Disconnect(0)
		m.client = nil
	}
	return nil
}
