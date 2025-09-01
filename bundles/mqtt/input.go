package mqtt

import (
	"context"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/wombatwisdom/components/framework/spec"
)

type InputConfig struct {
	CommonMQTTConfig

	// Filters is a map of topics and QoS levels to subscribe to
	Filters map[string]byte

	// CleanSession
	CleanSession bool

	// ClientId is an optional unique identifier for the client
	ClientId string

	// SetAutoAckDisabled disables automatic acknowledgment for at-least-once delivery (paho SetAutoAckDisabled)
	SetAutoAckDisabled *bool

	// PrefetchCount limits concurrent message processing (only used when SetAutoAckDisabled is true)
	PrefetchCount int
}

func NewInput(env spec.Environment, config InputConfig) (*Input, error) {
	// Set defaults
	if config.PrefetchCount == 0 {
		config.PrefetchCount = 10
	}
	if config.SetAutoAckDisabled == nil {
		defaultValue := true // Default to safe behavior (disable auto ACK)
		config.SetAutoAckDisabled = &defaultValue
	}

	return &Input{
		InputConfig: config,
		log:         env,

		done: make(chan struct{}),
	}, nil
}

type Input struct {
	InputConfig

	client       mqtt.Client
	done         chan struct{}
	manualAckCol *ManualAckCollector

	log spec.Logger
}

func (m *Input) Connect(ctx context.Context, collector spec.Collector) error {
	if m.client != nil {
		return spec.ErrAlreadyConnected
	}

	// Wrap collector for manual ACK if auto ACK is disabled
	if m.SetAutoAckDisabled != nil && *m.SetAutoAckDisabled {
		m.manualAckCol = NewManualAckCollector(collector, m.log, m.PrefetchCount)
		collector = m.manualAckCol
	}

	opts := NewClientOptions(m.InputConfig.CommonMQTTConfig).
		SetCleanSession(m.CleanSession).
		SetConnectionLostHandler(func(client mqtt.Client, reason error) {
			client.Disconnect(0)
			_ = collector.Disconnect()
			m.log.Errorf("Connection lost due to: %v\n", reason)
		})

	// Configure auto ACK based on setting
	if m.SetAutoAckDisabled != nil && *m.SetAutoAckDisabled {
		opts = opts.SetAutoAckDisabled(true)
	}

	opts = opts.SetOnConnectHandler(func(c mqtt.Client) {
		if m.SetAutoAckDisabled != nil && *m.SetAutoAckDisabled {
			// Manual ACK mode - block callback until pipeline completes
			tok := c.SubscribeMultiple(m.Filters, func(c mqtt.Client, msg mqtt.Message) {
				// Try to acquire prefetch slot
				if !m.manualAckCol.TryAcquire() {
					// No slot available, message will be redelivered
					m.log.Warnf("Prefetch limit reached, rejecting message on topic %s", msg.Topic())
					return
				}

				// Create tracked message
				tracked := NewTrackedMessage(msg)
				wrapper := NewTrackedMessageWrapper(tracked)

				// Send to collector (non-blocking)
				if err := collector.Write(wrapper); err != nil {
					// Return prefetch slot
					m.manualAckCol.semaphore <- struct{}{}
					m.log.Warnf("Failed to write message: %v", err)
					return
				}

				// Block waiting for pipeline result
				err := <-tracked.RespChan

				// Only ACK if pipeline succeeded
				if err == nil {
					msg.Ack()
				} else {
					m.log.Warnf("Pipeline failed, not ACKing message: %v", err)
				}
			})
			tok.Wait()
			if err := tok.Error(); err != nil {
				m.log.Errorf("Failed to subscribe using filters '%v': %v", m.Filters, err)
				m.log.Errorf("Shutting connection down.")
				_ = collector.Disconnect()
			}
		} else {
			// Standard mode - auto ACK
			tok := c.SubscribeMultiple(m.Filters, func(c mqtt.Client, msg mqtt.Message) {
				message := NewMqttMessage(msg)

				// not being able to write a message will never call the ack function. This means
				// that the message will be redelivered by the mqtt broker.
				if err := collector.Write(message); err != nil {
					m.log.Warnf("Failed to write message: %v", err)
				}
			})
			tok.Wait()
			if err := tok.Error(); err != nil {
				m.log.Errorf("Failed to subscribe using filters '%v': %v", m.Filters, err)
				m.log.Errorf("Shutting connection down.")
				_ = collector.Disconnect()
			}
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
					_ = collector.Disconnect()
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

func (m *Input) Disconnect(ctx context.Context) (err error) {
	if m.client != nil {
		m.client.Disconnect(0)
		m.client = nil
		close(m.done)
	}

	return
}

// GetManualAckCollector returns the manual ACK collector if enabled
func (m *Input) GetManualAckCollector() *ManualAckCollector {
	return m.manualAckCol
}
