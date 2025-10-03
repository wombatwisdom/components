package core

import (
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/wombatwisdom/components/framework/spec"
)

const (
	OutputComponentName = "nats_core"
)

func NewOutput(sys spec.System, cfg OutputConfig) *Output {
	return &Output{
		sys: sys,
		cfg: cfg,
	}
}

// NewOutputFromConfig creates an output from a spec.Config interface
func NewOutputFromConfig(sys spec.System, config spec.Config) (*Output, error) {
	var cfg OutputConfig
	if err := config.Decode(&cfg); err != nil {
		return nil, err
	}
	return NewOutput(sys, cfg), nil
}

// Output sends messages to a NATS subject.
// The NATS core output allows you to send messages to a subject. It does so by creating a NATS publisher and
// converting each message into a NATS message and send each individually to the NATS server. The subject is an expression
// that is constructed based on the message being processed.
type Output struct {
	sys spec.System
	cfg OutputConfig

	nc *nats.Conn
}

func (o *Output) Init(ctx spec.ComponentContext) error {
	var ok bool
	if o.nc, ok = o.sys.Client().(*nats.Conn); !ok {
		return fmt.Errorf("nats client is not of type *nats.Conn")
	}

	if o.cfg.Subject == nil {
		return fmt.Errorf("subject must be specified")
	}

	return nil
}

func (o *Output) Close(ctx spec.ComponentContext) error {
	return nil
}

func (o *Output) Write(ctx spec.ComponentContext, batch spec.Batch) error {
	for idx, message := range batch.Messages() {
		if err := o.WriteMessage(ctx, message); err != nil {
			return fmt.Errorf("batch #%d: %w", idx, err)
		}
	}

	return nil
}

func (o *Output) WriteMessage(ctx spec.ComponentContext, message spec.Message) error {
	subject, err := o.cfg.Subject.Eval(spec.MessageExpressionContext(message))
	if err != nil {
		return fmt.Errorf("subject: %w", err)
	}

	msg := nats.NewMsg(subject)

	msg.Data, err = message.Raw()
	if err != nil {
		return fmt.Errorf("payload: %w", err)
	}

	msg.Header = make(map[string][]string)
	for key, value := range message.Metadata() {
		// -- skip the metadata if the filter is set and the key is not included
		if o.cfg.MetadataFilter != nil && !o.cfg.MetadataFilter.Include(key) {
			return nil
		}

		msg.Header[key] = append(msg.Header[key], fmt.Sprintf("%v", value))
		return nil
	}

	if err := o.nc.PublishMsg(msg); err != nil {
		return fmt.Errorf("publish: %w", err)
	}

	return nil
}
