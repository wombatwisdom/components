package core

type ExpressionFactory interface {
	ParseExpression(expr string) (Expression, error)
}

type ExpressionContext map[string]any

type Expression interface {
	EvalString(ctx ExpressionContext) (string, error)
	EvalInt(ctx ExpressionContext) (int, error)
	EvalBool(ctx ExpressionContext) (bool, error)
}

// MessageExpressionContext creates an expression context from a message.
// It exposes the message's raw content and metadata for use in expressions.
func MessageExpressionContext(msg Message) ExpressionContext {
	ctx := make(ExpressionContext)

	// Add raw content if available
	if raw, err := msg.Raw(); err == nil {
		ctx["content"] = string(raw)
		ctx["raw"] = raw
	}

	// Add all metadata
	metadata := make(map[string]any)
	for key, value := range msg.Metadata() {
		metadata[key] = value
		// Also expose metadata at top level with "meta_" prefix
		ctx["meta_"+key] = value
	}
	ctx["metadata"] = metadata

	return ctx
}
