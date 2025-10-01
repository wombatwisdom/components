package spec

import "context"

type ComponentContext interface {
	Logger
	ExpressionFactory
	MessageFactory
	MetadataFilterFactory

	Context() context.Context

	ParseInterpolatedExpression(expr string) (InterpolatedExpression, error)
}
