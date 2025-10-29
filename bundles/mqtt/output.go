package mqtt

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/url"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/wombatwisdom/components/framework/spec"
)

type OutputConfig struct {
	CommonMQTTConfig

	TopicExpr        spec.Expression `json:"topic_expr" yaml:"topic_expr"`
	WriteTimeout     time.Duration   `json:"write_timeout" yaml:"write_timeout"`
	Retained         bool            `json:"retained" yaml:"retained"`
	QOS              byte            `json:"qos" yaml:"qos"`
	FailBatchOnError bool            `json:"fail_batch_on_error" yaml:"fail_batch_on_error"`
	CleanSession     bool            `json:"clean_session" yaml:"clean_session"`
}

func NewOutput(env spec.Environment, config OutputConfig) (*Output, error) {
	return &Output{
		config: config,
		log:    env,
	}, nil
}

type Output struct {
	config OutputConfig

	log spec.Logger

	client  mqtt.Client
	connMut sync.RWMutex
}

func (m *Output) Init(ctx spec.ComponentContext) error {
	m.connMut.Lock()
	defer m.connMut.Unlock()

	if m.client != nil {
		return nil
	}

	opts := NewClientOptions(m.config.CommonMQTTConfig).
		SetConnectionLostHandler(func(client mqtt.Client, reason error) {
			m.log.Errorf("Connection lost due to: %v", reason)
		}).
		SetOnConnectHandler(func(client mqtt.Client) {
			m.log.Infof("Connected to MQTT broker")
		}).
		SetReconnectingHandler(func(_ mqtt.Client, _ *mqtt.ClientOptions) {
			m.log.Infof("Reconnecting to MQTT broker...")
		}).
		SetConnectionAttemptHandler(func(broker *url.URL, tlsCfg *tls.Config) *tls.Config {
			m.log.Infof("Attempting to reconnect to MQTT broker at %s", broker)
			return tlsCfg
		}).
		SetWriteTimeout(m.config.WriteTimeout).
		SetCleanSession(m.config.CleanSession)

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
		exprCtx := spec.MessageExpressionContext(message)

		var err error

		topicStr, err := m.config.TopicExpr.Eval(exprCtx)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("topic interpolation error: %w", err))

			if m.config.FailBatchOnError {
				break
			} else {
				continue
			}
		}

		mb, err := message.Raw()
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to access message data: %w", err))
			if m.config.FailBatchOnError {
				break
			} else {
				continue
			}
		}

		mtok := client.Publish(topicStr, m.config.QOS, m.config.Retained, mb)
		mtok.Wait()

		sendErr := mtok.Error()
		if sendErr == nil {
			m.log.Infof("Message sent to topic %s", topicStr)
		} else {
			m.log.Errorf("Failed to send message to topic %s: %v", topicStr, sendErr)

			if errors.Is(sendErr, mqtt.ErrNotConnected) {
				errs = errors.Join(errs, spec.ErrNotConnected)
			} else {
				errs = errors.Join(errs, sendErr)
			}

			if m.config.FailBatchOnError {
				break
			} else {
				continue
			}
		}
	}

	return errs
}
