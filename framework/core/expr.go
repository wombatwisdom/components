package core

import (
	"github.com/wombatwisdom/components/framework/spec"
)

// Legacy interfaces - deprecated in favor of spec package equivalents
type ExpressionFactory = spec.ExpressionFactory
type ExpressionContext = spec.ExpressionContext
type Expression = spec.Expression

// Default creates a default expression factory using expr-lang
func Default() spec.ExpressionFactory {
	return NewExpressionFactory()
}

// MessageExpressionContext creates an expression context from a message.
// It exposes the message's raw content and metadata for use in expressions.
// Deprecated: Use spec.MessageExpressionContext instead
func MessageExpressionContext(msg spec.Message) ExpressionContext {
	return spec.MessageExpressionContext(msg)
}
