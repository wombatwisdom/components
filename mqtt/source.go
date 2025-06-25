package mqtt

import (
	"context"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/wombatwisdom/components/spec"
	"time"
)

type SourceConfig struct {
	CommonMQTTConfig

	// Filters is a map of topics and QoS levels to subscribe to
	Filters map[string]byte

	// CleanSession
	CleanSession bool

	// ClientId is an optional unique identifier for the client
	ClientId string
}

func NewSource(env spec.Environment, config SourceConfig) (*Source, error) {
	return &Source{
		SourceConfig: config,
		log:          env,

		done: make(chan struct{}),
	}, nil
}

type Source struct {
	SourceConfig

	client mqtt.Client
	done   chan struct{}

	log spec.Logger
}

func (m *Source) Connect(ctx context.Context, collector spec.Collector) error {
	if m.client != nil {
		return spec.ErrAlreadyConnected
	}

	opts := NewClientOptions(m.SourceConfig.CommonMQTTConfig).
		SetCleanSession(m.CleanSession).
		SetConnectionLostHandler(func(client mqtt.Client, reason error) {
			client.Disconnect(0)
			_ = collector.Disconnect(ctx)
			m.log.Errorf("Connection lost due to: %v\n", reason)
		}).
		SetOnConnectHandler(func(c mqtt.Client) {
			tok := c.SubscribeMultiple(m.Filters, func(c mqtt.Client, msg mqtt.Message) {
				message := NewMqttMessage(msg)

				// not being able to write a message will never call the ack function. This means
				// that the message will be redelivered by the mqtt broker.
				if err := collector.Write(ctx, message); err != nil {
					m.log.Warnf("Failed to write message: %v", err)
				}
			})
			tok.Wait()
			if err := tok.Error(); err != nil {
				m.log.Errorf("Failed to subscribe using filters '%v': %v", m.Filters, err)
				m.log.Errorf("Shutting connection down.")
				_ = collector.Disconnect(ctx)
			}
		})

	client := mqtt.NewClient(opts)

	tok := client.Connect()
	tok.Wait()
	if err := tok.Error(); err != nil {
		return err
	}

	go func() {
		for {
			select {
			case <-time.After(time.Second):
				if !client.IsConnected() {
					_ = collector.Disconnect(ctx)
					m.log.Errorf("Connection lost for unknown reasons.")
					return
				}
			case <-m.done:
				return
			}
		}
	}()

	m.client = client
	return nil
}

func (m *Source) Disconnect(ctx context.Context) (err error) {
	if m.client != nil {
		m.client.Disconnect(0)
		m.client = nil
		close(m.done)
	}

	return
}
