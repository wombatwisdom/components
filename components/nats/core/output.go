package nats

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

// Output sends messages to a NATS subject.
// The NATS core output allows you to send messages to a subject. It does so by creating a NATS publisher and
// converting each message into a NATS message and send each individually to the NATS server. The subject is an expression
// that is constructed based on the message being processed.
type Output struct {
	sys spec.System
	cfg OutputConfig

	subject        spec.Expression
	metadataFilter spec.MetadataFilter

	nc *nats.Conn
}

func (o *Output) Init(ctx spec.ComponentContext) error {
	var ok bool
	if o.nc, ok = o.sys.Client().(*nats.Conn); !ok {
		return fmt.Errorf("nats client is not of type *nats.Conn")
	}

	// -- create the subject expression
	if o.cfg.Subject == "" {
		return fmt.Errorf("subject is required")
	}

	var err error
	if o.subject, err = ctx.ParseExpression(o.cfg.Subject); err != nil {
		return fmt.Errorf("subject: %w", err)
	}

	if o.cfg.Metadata != nil {
		if o.metadataFilter, err = ctx.BuildMetadataFilter(o.cfg.Metadata.Patterns, o.cfg.Metadata.Invert); err != nil {
			return fmt.Errorf("metadata: %w", err)
		}
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
	subject, err := o.subject.EvalString(spec.MessageExpressionContext(message))
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
		if o.metadataFilter != nil && !o.metadataFilter.Include(key) {
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
