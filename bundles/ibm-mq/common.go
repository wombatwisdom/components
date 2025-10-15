//go:build mqclient

package ibm_mq

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
}