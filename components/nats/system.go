package nats

import (
	"context"
	_ "embed"

	"github.com/nats-io/nats.go"
	"github.com/wombatwisdom/components/framework/spec"
)

func NewSystem(rawConfig string) (*System, error) {
	var cfg SystemConfig
	if err := cfg.UnmarshalJSON([]byte(rawConfig)); err != nil {
		return nil, err
	}

	return &System{
		cfg: cfg,
	}, nil
}

// NewSystemFromConfig creates a system from a spec.Config interface
func NewSystemFromConfig(config spec.Config) (*System, error) {
	var cfg SystemConfig
	if err := config.Decode(&cfg); err != nil {
		return nil, err
	}

	return &System{
		cfg: cfg,
	}, nil
}

// System represents a NATS system.
//
// In case your NATS server requires authentication, you can provide the necessary credentials through the auth section
// of the configuration. This section gives you the ability to provide the NKey seed and JWT token for the user. Since
// both of these are sensitive information, it is recommended to expose them through environment variables and not make
// them explicit in the configuration.
type System struct {
	cfg SystemConfig
	nc  *nats.Conn
}

func (c *System) Connect(ctx context.Context) error {
	var err error
	var opts []nats.Option

	opts = append(opts, nats.Name(c.cfg.Name))
	if c.cfg.Auth != nil {
		opts = append(opts, nats.UserJWTAndSeed(c.cfg.Auth.Jwt, c.cfg.Auth.Seed))
	}

	c.nc, err = nats.Connect(c.cfg.Url, opts...)
	return err
}

func (c *System) Client() any {
	return c.nc
}

func (c *System) Close(ctx context.Context) error {
	if c.nc != nil {
		c.nc.Close()
	}

	return nil
}
