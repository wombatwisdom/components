package spec

import "context"

type ComponentContext interface {
	Logger
	ExpressionFactory
	MessageFactory
	MetadataFilterFactory

	Context() context.Context

	// Resource management
	Resources() ResourceManager

	// Component access (for cross-component communication)
	Input(name string) (Input, error)
	Output(name string) (Output, error)

	// System access (for shared client usage)
	System(name string) (System, error)
}
