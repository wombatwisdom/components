package nats

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/wombatwisdom/components/framework/spec"
)

const (
	StreamInputComponentName = "nats_stream"
)

// StreamInput reads messages from a NATS JetStream stream using pull consumers.
// It provides reliable message delivery with acknowledgment support and consumer management.
type StreamInput struct {
	sys spec.System
	cfg StreamConfig

	js       jetstream.JetStream
	consumer jetstream.Consumer
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewStreamInputFromConfig creates a new NATS Stream input from configuration
func NewStreamInputFromConfig(sys spec.System, config spec.Config) (*StreamInput, error) {
	var cfg StreamConfig
	if err := config.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("failed to decode stream input config: %w", err)
	}

	return &StreamInput{
		sys: sys,
		cfg: cfg,
	}, nil
}

func (si *StreamInput) Init(ctx spec.ComponentContext) error {
	// Get JetStream context from system
	js, ok := si.sys.Client().(jetstream.JetStream)
	if !ok {
		return fmt.Errorf("system client is not a JetStream instance")
	}
	si.js = js

	// Create context for consumer operations
	si.ctx, si.cancel = context.WithCancel(context.Background())

	return si.createConsumer(ctx)
}

func (si *StreamInput) createConsumer(ctx spec.ComponentContext) error {
	// Evaluate stream name
	streamName, err := si.cfg.Stream.Eval(spec.MessageExpressionContext(ctx.NewMessage()))
	if err != nil {
		return fmt.Errorf("failed to evaluate stream name: %w", err)
	}

	// Get or create the stream (assuming it exists for now)
	stream, err := si.js.Stream(si.ctx, streamName)
	if err != nil {
		return fmt.Errorf("failed to get stream %s: %w", streamName, err)
	}

	// Create consumer configuration
	consumerConfig := jetstream.ConsumerConfig{
		DeliverPolicy: jetstream.DeliverNewPolicy,  // Default
		AckPolicy:     jetstream.AckExplicitPolicy, // Default
	}

	// Configure consumer based on config
	if si.cfg.Consumer != nil {
		// Set delivery policy
		switch si.cfg.Consumer.DeliverPolicy {
		case "all":
			consumerConfig.DeliverPolicy = jetstream.DeliverAllPolicy
		case "last":
			consumerConfig.DeliverPolicy = jetstream.DeliverLastPolicy
		case "new":
			consumerConfig.DeliverPolicy = jetstream.DeliverNewPolicy
		default:
			consumerConfig.DeliverPolicy = jetstream.DeliverNewPolicy
		}

		// Set ack policy
		switch si.cfg.Consumer.AckPolicy {
		case "none":
			consumerConfig.AckPolicy = jetstream.AckNonePolicy
		case "all":
			consumerConfig.AckPolicy = jetstream.AckAllPolicy
		case "explicit":
			consumerConfig.AckPolicy = jetstream.AckExplicitPolicy
		default:
			consumerConfig.AckPolicy = jetstream.AckExplicitPolicy
		}

		// Set max deliver
		if si.cfg.Consumer.MaxDeliver > 0 {
			consumerConfig.MaxDeliver = si.cfg.Consumer.MaxDeliver
		}

		// Set ack wait
		if si.cfg.Consumer.AckWait != "" {
			ackWait, err := time.ParseDuration(si.cfg.Consumer.AckWait)
			if err != nil {
				return fmt.Errorf("failed to parse ack_wait duration: %w", err)
			}
			consumerConfig.AckWait = ackWait
		}

		// Set filter subject if provided
		if si.cfg.Subject != nil {
			filterSubject, err := si.cfg.Subject.Eval(spec.MessageExpressionContext(ctx.NewMessage()))
			if err != nil {
				return fmt.Errorf("failed to evaluate filter subject: %w", err)
			}

			consumerConfig.FilterSubject = filterSubject
		}

		// Set consumer name and durable
		if si.cfg.Consumer != nil && si.cfg.Consumer.Name != nil {
			consumerName, err := si.cfg.Consumer.Name.Eval(spec.MessageExpressionContext(ctx.NewMessage()))
			if err != nil {
				return fmt.Errorf("failed to evaluate consumer name: %w", err)
			}

			if si.cfg.Consumer.Durable {
				consumerConfig.Durable = consumerName
			} else {
				consumerConfig.Name = consumerName
			}
		}
	}

	// Create the consumer
	si.consumer, err = stream.CreateOrUpdateConsumer(si.ctx, consumerConfig)
	if err != nil {
		return fmt.Errorf("failed to create consumer: %w", err)
	}

	return nil
}

func (si *StreamInput) Close(ctx spec.ComponentContext) error {
	if si.cancel != nil {
		si.cancel()
	}
	return nil
}

func (si *StreamInput) Read(ctx spec.ComponentContext) (spec.Batch, spec.ProcessedCallback, error) {
	if si.consumer == nil {
		return nil, nil, fmt.Errorf("consumer not initialized")
	}

	// Fetch messages from the consumer
	batchSize := si.cfg.BatchSize
	if batchSize <= 0 {
		batchSize = 1
	}

	msgs, err := si.consumer.Fetch(batchSize, jetstream.FetchMaxWait(30*time.Second))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch messages: %w", err)
	}

	batch := ctx.NewBatch()
	var jetStreamMsgs []jetstream.Msg

	for msg := range msgs.Messages() {
		if msg == nil {
			continue
		}

		jetStreamMsgs = append(jetStreamMsgs, msg)

		// Create message
		m := ctx.NewMessage()
		m.SetRaw(msg.Data())

		// Add NATS headers
		for key, values := range msg.Headers() {
			if len(values) > 0 {
				// For multiple values, join them with commas
				if len(values) == 1 {
					m.SetMetadata(key, values[0])
				} else {
					var combined string
					for i, v := range values {
						if i > 0 {
							combined += ","
						}
						combined += v
					}
					m.SetMetadata(key, combined)
				}
			}
		}

		// Add JetStream metadata if configured
		metadata, err := msg.Metadata()
		if err == nil && metadata != nil {
			m.SetMetadata("jetstream_stream", metadata.Stream)
			m.SetMetadata("jetstream_consumer", metadata.Consumer)
			m.SetMetadata("jetstream_sequence_stream", strconv.FormatUint(metadata.Sequence.Stream, 10))
			m.SetMetadata("jetstream_sequence_consumer", strconv.FormatUint(metadata.Sequence.Consumer, 10))
			m.SetMetadata("jetstream_pending", strconv.FormatUint(metadata.NumPending, 10))
			m.SetMetadata("jetstream_delivered", strconv.FormatUint(metadata.NumDelivered, 10))
			m.SetMetadata("jetstream_timestamp", metadata.Timestamp.Format(time.RFC3339))
		}

		// Add basic NATS message metadata
		m.SetMetadata("nats_subject", msg.Subject())
		if reply := msg.Reply(); reply != "" {
			m.SetMetadata("nats_reply", reply)
		}

		batch.Append(m)
	}

	// Create acknowledgment callback
	ackCallback := func(ctx context.Context, err error) error {
		for _, msg := range jetStreamMsgs {
			if err != nil {
				// On error, we could implement negative acknowledgment or let it time out
				// For now, we'll just let it timeout and retry
				continue
			}

			// Acknowledge successful processing
			if err := msg.Ack(); err != nil {
				return fmt.Errorf("failed to acknowledge message: %w", err)
			}
		}
		return nil
	}

	return batch, ackCallback, nil
}
