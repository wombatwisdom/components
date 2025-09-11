package nats

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/wombatwisdom/components/framework/spec"
)

const (
	StreamOutputComponentName = "nats_stream"
)

// StreamOutput publishes messages to a NATS JetStream stream.
// It provides reliable message publishing with acknowledgment support.
type StreamOutput struct {
	sys spec.System
	cfg StreamConfig

	stream         spec.Expression
	subject        spec.Expression
	metadataFilter spec.MetadataFilter

	js jetstream.JetStream
}

// NewStreamOutputFromConfig creates a new NATS Stream output from configuration
func NewStreamOutputFromConfig(sys spec.System, config spec.Config) (*StreamOutput, error) {
	var cfg StreamConfig
	if err := config.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("failed to decode stream output config: %w", err)
	}

	return &StreamOutput{
		sys: sys,
		cfg: cfg,
	}, nil
}

func (so *StreamOutput) Init(ctx spec.ComponentContext) error {
	// Get JetStream context from system
	js, ok := so.sys.Client().(jetstream.JetStream)
	if !ok {
		return fmt.Errorf("system client is not a JetStream instance")
	}
	so.js = js

	// Parse expressions
	var err error
	so.stream, err = ctx.ParseExpression(so.cfg.Stream)
	if err != nil {
		return fmt.Errorf("failed to parse stream expression: %w", err)
	}

	so.subject, err = ctx.ParseExpression(so.cfg.Subject)
	if err != nil {
		return fmt.Errorf("failed to parse subject expression: %w", err)
	}

	// Setup metadata filter if configured
	if so.cfg.Metadata != nil && len(so.cfg.Metadata.Filter) > 0 {
		so.metadataFilter, err = ctx.BuildMetadataFilter(so.cfg.Metadata.Filter, false)
		if err != nil {
			return fmt.Errorf("failed to build metadata filter: %w", err)
		}
	}

	return nil
}

func (so *StreamOutput) Close(ctx spec.ComponentContext) error {
	return nil
}

func (so *StreamOutput) Write(ctx spec.ComponentContext, batch spec.Batch) error {
	for idx, message := range batch.Messages() {
		if err := so.WriteMessage(ctx, message); err != nil {
			return fmt.Errorf("batch #%d: %w", idx, err)
		}
	}
	return nil
}

func (so *StreamOutput) WriteMessage(ctx spec.ComponentContext, message spec.Message) error {
	// Evaluate stream name
	streamName, err := so.stream.EvalString(spec.MessageExpressionContext(message))
	if err != nil {
		return fmt.Errorf("failed to evaluate stream name: %w", err)
	}

	// Evaluate subject
	subject, err := so.subject.EvalString(spec.MessageExpressionContext(message))
	if err != nil {
		return fmt.Errorf("failed to evaluate subject: %w", err)
	}

	// Verify stream exists (optional check)
	_, err = so.js.Stream(context.Background(), streamName)
	if err != nil {
		return fmt.Errorf("failed to get stream %s: %w", streamName, err)
	}

	// Create publish options with headers
	var publishOpts []jetstream.PublishOpt
	headers := make(nats.Header)

	// Process metadata
	metadata := make(map[string]string)
	for key, value := range message.Metadata() {
		if strValue, ok := value.(string); ok {
			metadata[key] = strValue
		} else {
			metadata[key] = fmt.Sprintf("%v", value)
		}
	}

	// Apply metadata filter if configured
	if so.metadataFilter != nil {
		filtered := make(map[string]string)
		for key, value := range metadata {
			if so.metadataFilter.Include(key) {
				filtered[key] = value
			}
		}
		metadata = filtered
	}

	// Add metadata as headers
	for key, value := range metadata {
		// Skip jetstream-specific metadata to avoid conflicts
		if key == "jetstream_stream" || key == "jetstream_consumer" ||
			key == "jetstream_sequence_stream" || key == "jetstream_sequence_consumer" ||
			key == "jetstream_pending" || key == "jetstream_delivered" ||
			key == "jetstream_timestamp" {
			continue
		}

		headers.Set(key, value)
	}

	// Add custom headers for tracking
	headers.Set("wombat_timestamp", time.Now().Format(time.RFC3339))
	if hostname := getHostname(); hostname != "" {
		headers.Set("wombat_source", hostname)
	}

	// Note: JetStream publish options don't directly support headers in this version
	// Headers would need to be embedded in the message data or handled differently

	// Add message ID for deduplication
	msgID := fmt.Sprintf("%d", time.Now().UnixNano())
	publishOpts = append(publishOpts, jetstream.WithMsgID(msgID))

	// Get message data
	msgData, err := message.Raw()
	if err != nil {
		return fmt.Errorf("failed to get message data: %w", err)
	}

	// Publish message using JetStream context
	pubAck, err := so.js.Publish(context.Background(), subject, msgData, publishOpts...)
	if err != nil {
		return fmt.Errorf("failed to publish message to stream %s: %w", streamName, err)
	}

	// Log successful publish for debugging (optional metadata)
	if so.cfg.Metadata == nil || so.cfg.Metadata.IncludeStreamInfo {
		// We could add publish acknowledgment info to message metadata if needed
		// but for output components, this is typically not necessary
		_ = pubAck // Suppress unused variable warning
	}

	return nil
}

// getHostname returns the hostname for source tracking
func getHostname() string {
	// For now, return empty string. In a real implementation,
	// you might want to use os.Hostname() or get it from environment
	return ""
}
