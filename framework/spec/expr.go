package spec

import "encoding/json"

type ExpressionFactory interface {
	ParseExpression(expr string) (Expression, error)
}

type ExpressionContext map[string]any

type Expression interface {
	EvalString(ctx ExpressionContext) (string, error)
	EvalInt(ctx ExpressionContext) (int, error)
	EvalBool(ctx ExpressionContext) (bool, error)
}

func MessageExpressionContext(msg Message) ExpressionContext {
	ctx := make(ExpressionContext)

	// Add message content
	if raw, err := msg.Raw(); err == nil {
		ctx["content"] = string(raw)
		ctx["json"] = parseJSON(raw)
	}

	// Add metadata
	metadata := make(map[string]any)
	for k, v := range msg.Metadata() {
		metadata[k] = v
	}
	ctx["metadata"] = metadata

	return ctx
}

func parseJSON(data []byte) map[string]any {
	var result map[string]any
	// Ignore parse errors, return empty map
	_ = json.Unmarshal(data, &result)
	return result
}
