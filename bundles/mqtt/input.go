package mqtt

import (
	"context"
	"errors"
	"maps"
	"sync"
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

	// EnableAutoAck enables automatic acknowledgment for at-least-once delivery (paho SetAutoAckDisabled)
	EnableAutoAck bool
}

func NewInput(env spec.Environment, config InputConfig) (*Input, error) {
	return &Input{
		InputConfig: config,
		log:         env,
	}, nil
}

type Input struct {
	InputConfig

	client mqtt.Client

	msgChan     chan mqtt.Message
	msgChanLock sync.Mutex

	log spec.Logger
}

func (m *Input) closeMsgChan() bool {
	m.msgChanLock.Lock()
	defer m.msgChanLock.Unlock()

	chanOpen := m.msgChan != nil
	if chanOpen {
		close(m.msgChan)
		m.msgChan = nil
	}
	return chanOpen
}

func (m *Input) Init(ctx spec.ComponentContext) error {
	if m.client != nil {
		return spec.ErrAlreadyConnected
	}

	var msgMut sync.Mutex
	msgChan := make(chan mqtt.Message)

	opts := NewClientOptions(m.InputConfig.CommonMQTTConfig).
		SetCleanSession(m.CleanSession).
		SetConnectionLostHandler(func(client mqtt.Client, reason error) {
			client.Disconnect(0)
			m.closeMsgChan()
			m.log.Errorf("Connection lost due to: %v\n", reason)
		}).
		SetOnConnectHandler(func(client mqtt.Client) {
			tok := client.SubscribeMultiple(m.Filters, func(_ mqtt.Client, msg mqtt.Message) {
				msgMut.Lock()
				defer msgMut.Unlock()

				if msgChan != nil {
					select {
					case msgChan <- msg:
					case <-ctx.Context().Done():
					}
				}
			})
			tok.Wait()
			if err := tok.Error(); err != nil {
				m.log.Errorf("Failed to subscribe to topics '%v': %v", maps.Keys(m.InputConfig.Filters), err)
				m.log.Errorf("Shutting connection down.")
				m.closeMsgChan()
			}
		}).
		SetAutoAckDisabled(!m.EnableAutoAck)

	client := mqtt.NewClient(opts)

	tok := client.Connect()
	tok.Wait()
	if err := tok.Error(); err != nil {
		return err
	}

	m.log.Infof("Connected to MQTT broker")

	go func() {
		for {
			select {
			case <-time.After(time.Second):
				if !client.IsConnected() {
					if m.closeMsgChan() {
						m.log.Errorf("Connection lost for unknown reasons.")
					}
					return
				}
			case <-ctx.Context().Done():
				return
			}
		}
	}()

	m.client = client
	m.msgChan = msgChan
	return nil
}

func (m *Input) Close(ctx spec.ComponentContext) error {
	m.msgChanLock.Lock()
	defer m.msgChanLock.Unlock()

	if m.client != nil {
		m.client.Disconnect(0)
		m.client = nil
	}

	return nil
}

func (m *Input) Read(ctx spec.ComponentContext) (spec.Batch, spec.ProcessedCallback, error) {
	m.msgChanLock.Lock()
	msgChan := m.msgChan
	m.msgChanLock.Unlock()

	if msgChan == nil {
		return nil, nil, spec.ErrNotConnected
	}

	select {
	case msg, open := <-msgChan:
		if !open {
			m.closeMsgChan()
			return nil, nil, spec.ErrNotConnected
		}

		specMsg := ctx.NewMessage()
		specMsg.SetRaw(msg.Payload())

		specMsg.SetMetadata("mqtt_duplicate", msg.Duplicate())
		specMsg.SetMetadata("mqtt_qos", int(msg.Qos()))
		specMsg.SetMetadata("mqtt_retained", msg.Retained())
		specMsg.SetMetadata("mqtt_topic", msg.Topic())
		specMsg.SetMetadata("mqtt_message_id", int(msg.MessageID()))

		return ctx.NewBatch(specMsg), func(ackCtx context.Context, res error) error {
			// check for any errors in the component context
			if err := ackCtx.Err(); err != nil {
				if !m.EnableAutoAck {
					var reason string
					switch {
					case errors.Is(err, context.Canceled):
						reason = "context cancellation"
					case errors.Is(err, context.DeadlineExceeded):
						reason = "deadline exceeded"
					default:
						reason = "context error: " + err.Error()
					}
					m.log.Infof("Skipping ACK for message (topic: %s, id: %d) due to %s - message will be redelivered",
						msg.Topic(), msg.MessageID(), reason)
				}
				return nil
			}

			if res == nil {
				if !m.EnableAutoAck {
					// Check if client is still connected before ACKing
					if m.client != nil && m.client.IsConnected() {
						msg.Ack()
					} else {
						m.log.Infof("Skipping ACK for message (topic: %s, id: %d) - client disconnected, message will be redelivered",
							msg.Topic(), msg.MessageID())
					}
				}
			}
			return nil
		}, nil
	case <-ctx.Context().Done():
		return nil, nil, ctx.Context().Err()
	}
}
