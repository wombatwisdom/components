package spec

import "context"

type ComponentContext interface {
	Logger
	ExpressionFactory
	MessageFactory
	MetadataFilterFactory

	Context() context.Context

	ParseInterpolatedExpression(expr string) (InterpolatedExpression, error)
	
	// CreateExpressionContext creates an ExpressionContext from a Message
	// This allows runtimes to provide their own context creation logic
	CreateExpressionContext(msg Message) ExpressionContext
}
