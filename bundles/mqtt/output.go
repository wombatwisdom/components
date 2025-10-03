package mqtt

import (
	"errors"
	"fmt"
	"sync"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/wombatwisdom/components/framework/spec"
)

type Config struct {
	Mqtt MqttConfig
}

func NewOutput(env spec.Environment, config Config) (*Output, error) {
	return &Output{
		config: config,
		log:    env,
	}, nil
}

type Output struct {
	config Config

	log spec.Logger

	topic spec.InterpolatedExpression

	client  mqtt.Client
	connMut sync.RWMutex // TODO: replace this with something more idiomatic
}

func (m *Output) Init(ctx spec.ComponentContext) error {
	m.connMut.Lock()
	defer m.connMut.Unlock()

	if m.client != nil {
		return nil
	}

	var err error
	m.topic, err = ctx.ParseInterpolatedExpression(m.config.Mqtt.Topic)
	if err != nil {
		return fmt.Errorf("failed to parse Topic expression: %w", err)
	}

	opts := NewClientOptions(m.config.Mqtt).
		SetConnectionLostHandler(func(client mqtt.Client, reason error) {
			client.Disconnect(0)
			m.log.Errorf("Connection lost due to: %v", reason)
		}).
		SetWriteTimeout(m.config.Mqtt.WriteTimeout)

	client := mqtt.NewClient(opts)

	tok := client.Connect()
	tok.Wait()
	if err := tok.Error(); err != nil {
		return err
	}

	m.client = client
	return nil
}

func (m *Output) Close(ctx spec.ComponentContext) error {
	m.connMut.Lock()
	defer m.connMut.Unlock()

	if m.client != nil {
		m.client.Disconnect(0)
		m.client = nil
	}
	return nil
}

func (m *Output) Write(ctx spec.ComponentContext, batch spec.Batch) error {
	m.connMut.RLock()
	client := m.client
	m.connMut.RUnlock()

	if client == nil {
		return spec.ErrNotConnected
	}

	var errs error
	for _, message := range batch.Messages() {
		blobExprCtx := ctx.CreateExpressionContext(message)

		var err error

		topicStr, err := m.topic.EvalString(blobExprCtx)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("topic interpolation error: %w", err))

			if m.config.Mqtt.FailBatchOnError {
				break
			} else {
				continue
			}
		}

		mb, err := message.Raw()
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to access message data: %w", err))
			if m.config.Mqtt.FailBatchOnError {
				break
			} else {
				continue
			}
		}

		mtok := client.Publish(topicStr, m.config.Mqtt.QOS, m.config.Mqtt.Retained, mb)
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
			errs = errors.Join(errs, sendErr)
			if m.config.Mqtt.FailBatchOnError {
				break
			} else {
				continue
			}
		}
	}

	return errs
}
