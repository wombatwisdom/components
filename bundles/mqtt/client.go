package mqtt

import (
	"crypto/tls"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type MqttConfig struct {
	ClientId string   `json:"client_id" yaml:"client_id"`
	Urls     []string `json:"urls" yaml:"urls"`
	Topic    string   `json:"Topic" yaml:"Topic"`

	ConnectTimeout *time.Duration `json:"connect_timeout" yaml:"connect_timeout"`
	KeepAlive      *time.Duration `json:"keepalive" yaml:"keepalive"`

	Username         string        `json:"username" yaml:"username"`
	Password         string        `json:"password" yaml:"password"`
	WriteTimeout     time.Duration `json:"write_timeout" yaml:"write_timeout"`
	Retained         bool          `json:"retained" yaml:"retained"`
	QOS              byte          `json:"qos" yaml:"qos"`
	FailBatchOnError bool          `json:"fail_batch_on_error" yaml:"fail_batch_on_error"`
	TLS              *tls.Config   `json:"tls" yaml:"tls"`

	Will *WillConfig `json:"will" yaml:"will"`
}

func (c *MqttConfig) apply(opts *mqtt.ClientOptions) *mqtt.ClientOptions {
	opts = opts.SetAutoReconnect(false).
		SetClientID(c.ClientId).SetWriteTimeout(c.WriteTimeout)

	if c.ConnectTimeout != nil {
		opts = opts.SetConnectTimeout(*c.ConnectTimeout)
	}

	if c.KeepAlive != nil {
		opts = opts.SetKeepAlive(*c.KeepAlive)
	}

	if c.Will != nil {
		opts = c.Will.apply(opts)
	}

	if c.TLS != nil {
		opts = opts.SetTLSConfig(c.TLS)
	}

	if c.Username != "" {
		opts = opts.SetUsername(c.Username)
	}

	if c.Password != "" {
		opts = opts.SetPassword(c.Password)
	}

	for _, u := range c.Urls {
		opts = opts.AddBroker(u)
	}

	return opts
}

func NewClientOptions(config MqttConfig) *mqtt.ClientOptions {
	return config.apply(mqtt.NewClientOptions())
}

type WillConfig struct {
	QoS      uint8
	Retained bool
	Topic    string
	Payload  string
}

func (w *WillConfig) apply(opts *mqtt.ClientOptions) *mqtt.ClientOptions {
	if w == nil {
		return opts
	}

	opts = opts.SetWill(w.Topic, w.Payload, w.QoS, w.Retained)
	return opts
}
