package spec

import "context"

type ComponentContext interface {
	Logger
	MessageFactory
	MetadataFilterFactory

	Context() context.Context
}
