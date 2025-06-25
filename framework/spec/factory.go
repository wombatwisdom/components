package spec

// ComponentFactory provides methods to create component instances.
// This interface enables benthos-compatible component construction patterns
// while maintaining the System-first architecture.
type ComponentFactory interface {
	// NewInput creates a new Input component using the provided system and configuration
	NewInput(sys System, cfg Config) (Input, error)

	// NewOutput creates a new Output component using the provided system and configuration
	NewOutput(sys System, cfg Config) (Output, error)
}

// SystemFactory creates System instances from configuration.
type SystemFactory interface {
	// NewSystem creates a new System instance from raw configuration
	NewSystem(cfg Config) (System, error)
}

// ComponentConstructor is a function type for creating components.
// This mirrors benthos patterns while maintaining system dependency injection.
type ComponentConstructor[T Component] func(sys System, cfg Config) (T, error)

// SystemConstructor is a function type for creating systems.
type SystemConstructor func(cfg Config) (System, error)
