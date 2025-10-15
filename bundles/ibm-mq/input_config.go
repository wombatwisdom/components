//go:build mqclient

package ibm_mq

// InputConfig defines configuration for IBM MQ input
type InputConfig struct {
	CommonMQConfig

	// The IBM MQ queue name to read messages from
	QueueName string `json:"queue_name" yaml:"queue_name"`

	// Number of parallel workers for processing messages (default: 1)
	NumWorkers int `json:"num_workers" yaml:"num_workers"`

	// Maximum number of messages to batch together (default: 1)
	BatchSize int `json:"batch_size" yaml:"batch_size"`

	// Poll interval when queue is empty
	PollInterval string `json:"poll_interval" yaml:"poll_interval"`

	NumThreads int `json:"num_threads" yaml:"num_threads"`

	WaitTime string `json:"wait_time" yaml:"wait_time"`

	BatchCount int `json:"batch_count" yaml:"batch_count"`
}
