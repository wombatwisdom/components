//go:build mqclient

package ibm_mq

import "github.com/wombatwisdom/components/framework/spec"

// CommonMQConfig contains shared configuration for IBM MQ connections
type CommonMQConfig struct {
	// The IBM MQ Queue Manager name
	QueueManagerName string `json:"queue_manager_name" yaml:"queue_manager_name"`

	// The IBM MQ channel name for client connections
	ChannelName string `json:"channel_name" yaml:"channel_name"`

	// The IBM MQ connection name in the format hostname(port)
	ConnectionName string `json:"connection_name" yaml:"connection_name"`

	// Optional: The IBM MQ user ID for authentication
	UserId string `json:"user_id" yaml:"user_id"`

	// Optional: The IBM MQ user password for authentication
	Password string `json:"password" yaml:"password"`

	// Optional: Application name for MQ connection identification
	ApplicationName string `json:"application_name" yaml:"application_name"`

	// Optional: TLS/SSL configuration for secure connections
	TLS *TLSConfig `json:"tls,omitempty" yaml:"tls,omitempty"`
}

// TLSConfig contains TLS/SSL configuration for secure IBM MQ connections
type TLSConfig struct {
	// Enable TLS encryption for the connection
	Enabled bool `json:"enabled" yaml:"enabled"`

	// The cipher specification to use for TLS
	// Example: "TLS_RSA_WITH_AES_128_CBC_SHA256", "TLS_RSA_WITH_AES_256_CBC_SHA256", "ANY_TLS12_OR_HIGHER"
	CipherSpec string `json:"cipher_spec,omitempty" yaml:"cipher_spec,omitempty"`

	// Path to the key repository containing certificates
	// For example: "/opt/mqm/ssl/key" (without file extension)
	// The actual files would be key.kdb, key.sth, etc.
	KeyRepository string `json:"key_repository,omitempty" yaml:"key_repository,omitempty"`

	// Password for the key repository
	KeyRepositoryPassword string `json:"key_repository_password,omitempty" yaml:"key_repository_password,omitempty"`

	// Certificate label to use from the key repository
	// If empty, the default certificate will be used
	CertificateLabel string `json:"certificate_label,omitempty" yaml:"certificate_label,omitempty"`

	// Optional: Peer name for SSL/TLS validation
	// Used to verify the DN of the certificate from the peer queue manager or client
	SSLPeerName string `json:"ssl_peer_name,omitempty" yaml:"ssl_peer_name,omitempty"`

	// Require FIPS 140-2 compliant algorithms
	FipsRequired bool `json:"fips_required,omitempty" yaml:"fips_required,omitempty"`
}

// InputConfig defines configuration for IBM MQ input
type InputConfig struct {
	CommonMQConfig

	// The IBM MQ queue name to read messages from
	QueueName string `json:"queue_name" yaml:"queue_name"`

	// The number of messages to fetch in a single batch
	// Default: 1
	BatchSize int `json:"batch_size" yaml:"batch_size"`

	// Maximum time to wait for a complete batch before returning partial batch
	// Format: duration string (e.g., "100ms", "1s", "500ms")
	// Default: "100ms"
	BatchWaitTime string `json:"batch_wait_time" yaml:"batch_wait_time"`
}

// OutputConfig defines configuration for IBM MQ output
type OutputConfig struct {
	CommonMQConfig

	QueueExpr spec.Expression `json:"queue_expr,omitempty" yaml:"queue_expr,omitempty"`

	// Metadata configuration for filtering message headers
	Metadata *MetadataConfig `json:"metadata,omitempty" yaml:"metadata,omitempty"`

	// The format of the message data (e.g., "MQSTR" for string, "MQHRF2" for RFH2 headers)
	// Default: "MQSTR"
	Format string `json:"format,omitempty" yaml:"format,omitempty"`

	// The Coded Character Set Identifier for the message
	// Common values: "1208" (UTF-8), "819" (ISO-8859-1)
	// Default: "1208"
	Ccsid string `json:"ccsid,omitempty" yaml:"ccsid,omitempty"`

	// The encoding of numeric data in the message
	// Common values: "546" (Linux/Windows little-endian), "273" (big-endian)
	// Default: "546"
	Encoding string `json:"encoding,omitempty" yaml:"encoding,omitempty"`
}

// MetadataConfig defines metadata filtering options
type MetadataConfig struct {
	// Patterns to match metadata fields
	Patterns []string `json:"patterns" yaml:"patterns"`

	// If true, exclude matching patterns; if false, include only matching patterns
	Invert bool `json:"invert" yaml:"invert"`
}
