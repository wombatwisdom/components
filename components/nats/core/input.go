package core

import (
	"fmt"
	"strings"

	"github.com/nats-io/nats.go"
	"github.com/wombatwisdom/components/framework/spec"
)

const (
	InputComponentName = "nats_core"
)

func NewInput(sys spec.System, rawConfig spec.Config) (*Input, error) {
	var cfg InputConfig
	if err := rawConfig.Decode(&cfg); err != nil {
		return nil, err
	}

	return &Input{
		sys: sys,
		cfg: cfg,
	}, nil
}

// NewInputFromConfig creates an input from a spec.Config interface
func NewInputFromConfig(sys spec.System, config spec.Config) (*Input, error) {
	return NewInput(sys, config)
}

// Input receives messages from a NATS subject.
//
// The NATS core input allows you to read messages from a subject. It does so by creating a NATS subscriber and fetch
// batches of messages at a time. By default, the batch count is set to 1, which is quite conservative since it means
// a new message is only fetched once the current one has been processed.
//
// ## Queue Groups
// Each input with the same queue name will be load balancing messages across all members of the group. This is useful
// when you want to scale the processing of messages across multiple instances.
type Input struct {
	sys spec.System
	cfg InputConfig

	sub *nats.Subscription
}

func (i *Input) Init(ctx spec.ComponentContext) error {
	client, ok := i.sys.Client().(*nats.Conn)
	if !ok {
		return fmt.Errorf("nats client is not of type *nats.Conn")
	}

	// create the subscription
	var err error
	if i.cfg.Queue == nil {
		i.sub, err = client.SubscribeSync(i.cfg.Subject)
	} else {
		i.sub, err = client.QueueSubscribeSync(i.cfg.Subject, *i.cfg.Queue)
	}
	return err
}

func (i *Input) Close(ctx spec.ComponentContext) error {
	if i.sub != nil {
		if err := i.sub.Unsubscribe(); err != nil {
			return err
		}
	}

	return nil
}

func (i *Input) Read(ctx spec.ComponentContext) (spec.Batch, spec.ProcessedCallback, error) {
	msgs, err := i.sub.Fetch(i.cfg.BatchCount)
	if err != nil {
		return nil, nil, err
	}

	batch := ctx.NewBatch()
	for _, msg := range msgs {
		m := ctx.NewMessage()
		m.SetRaw(msg.Data)

		for k, v := range msg.Header {
			if len(v) == 1 {
				m.SetMetadata(k, v[0])
			} else {
				m.SetMetadata(k, strings.Join(v, ","))
			}
		}

		// add message metadata as headers
		m.SetMetadata("nats_subject", msg.Subject)
		m.SetMetadata("nats_reply", msg.Reply)

		batch.Append(m)
	}

	return batch, spec.NoopCallback, nil
}
