package nats

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/wombatwisdom/components/framework/spec"
)

// JetStreamSystem represents a NATS JetStream system that provides access to
// JetStream features including streams, key-value stores, and object stores.
type JetStreamSystem struct {
	cfg SystemConfig
	nc  *nats.Conn
	js  jetstream.JetStream
}

// NewJetStreamSystemFromConfig creates a JetStream system from a spec.Config interface
func NewJetStreamSystemFromConfig(config spec.Config) (*JetStreamSystem, error) {
	var cfg SystemConfig
	if err := config.Decode(&cfg); err != nil {
		return nil, err
	}

	return &JetStreamSystem{
		cfg: cfg,
	}, nil
}

func (js *JetStreamSystem) Connect(ctx context.Context) error {
	var err error
	var opts []nats.Option

	opts = append(opts, nats.Name(js.cfg.Name))
	if js.cfg.Auth != nil {
		opts = append(opts, nats.UserJWTAndSeed(js.cfg.Auth.Jwt, js.cfg.Auth.Seed))
	}

	js.nc, err = nats.Connect(js.cfg.Url, opts...)
	if err != nil {
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// Create JetStream context with default options
	js.js, err = jetstream.New(js.nc)
	if err != nil {
		js.nc.Close()
		return fmt.Errorf("failed to create JetStream context: %w", err)
	}

	return nil
}

func (js *JetStreamSystem) Client() any {
	return js.js
}

func (js *JetStreamSystem) Close(ctx context.Context) error {
	if js.nc != nil {
		js.nc.Close()
	}
	return nil
}

// NATSConn returns the underlying NATS connection for advanced use cases
func (js *JetStreamSystem) NATSConn() *nats.Conn {
	return js.nc
}

// JetStream returns the JetStream context
func (js *JetStreamSystem) JetStream() jetstream.JetStream {
	return js.js
}
