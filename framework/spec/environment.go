package spec

// Environment provides environment variable and dynamic field access.
// This interface maintains compatibility with existing components.
type Environment interface {
	Logger
	GetString(key string) string
	GetInt(key string) int
	GetBool(key string) bool
}

// Collector
// deprecated: Not to be used in new components.
type Collector interface {
	Collect(msg Message) error
	Flush() (Batch, error)
	// Legacy methods for backward compatibility
	Write(msg Message) error
	Disconnect() error
}
