package nats

import (
	"github.com/wombatwisdom/components/framework/spec"
)

type StreamConfig struct {
	// Number of messages to fetch in a single batch. Only applies to inputs. Higher
	// values can improve throughput but increase memory usage.
	//
	BatchSize int

	// Consumer configuration for input components. Only applies to inputs.
	//
	Consumer *StreamConfigConsumer

	// Metadata handling configuration.
	//
	MetadataFilter spec.MetadataFilter

	// The name of the JetStream stream to consume from or publish to. This can be an
	// expression that is evaluated for each message.
	//
	Stream spec.Expression

	// The subject pattern for the stream. For inputs, this is used to filter messages
	// from the stream. For outputs, this is the subject to publish to. This can be an
	// expression that is evaluated for each message.
	//
	Subject spec.Expression
}

// Consumer configuration for input components. Only applies to inputs.
type StreamConfigConsumer struct {
	// The acknowledgment policy for the consumer. - none: No acknowledgment required
	// - all: Acknowledge all messages in order - explicit: Acknowledge each message
	// individually
	//
	AckPolicy StreamConfigConsumerAckPolicy

	// Time to wait for acknowledgment before redelivering a message. Use Go duration
	// format (e.g., "30s", "5m", "1h").
	//
	AckWait string

	// The delivery policy for the consumer. - all: Deliver all messages in the stream
	// - last: Deliver only the last message per subject - new: Deliver only new
	// messages (from now)
	//
	DeliverPolicy StreamConfigConsumerDeliverPolicy

	// Whether to create a durable consumer. If true, the consumer will persist across
	// restarts and continue from where it left off.
	//
	Durable bool

	// Additional subject filter for the consumer. If provided, the consumer will only
	// receive messages matching this subject pattern.
	//
	FilterSubject spec.Expression

	// Maximum number of delivery attempts for a message. After this many attempts,
	// the message will be considered failed.
	//
	MaxDeliver int

	// The name of the consumer. If not provided, an ephemeral consumer will be
	// created.
	//
	Name spec.Expression
}

type StreamConfigConsumerAckPolicy string

const StreamConfigConsumerAckPolicyAll StreamConfigConsumerAckPolicy = "all"
const StreamConfigConsumerAckPolicyExplicit StreamConfigConsumerAckPolicy = "explicit"
const StreamConfigConsumerAckPolicyNone StreamConfigConsumerAckPolicy = "none"

type StreamConfigConsumerDeliverPolicy string

const StreamConfigConsumerDeliverPolicyAll StreamConfigConsumerDeliverPolicy = "all"
const StreamConfigConsumerDeliverPolicyLast StreamConfigConsumerDeliverPolicy = "last"
const StreamConfigConsumerDeliverPolicyNew StreamConfigConsumerDeliverPolicy = "new"
