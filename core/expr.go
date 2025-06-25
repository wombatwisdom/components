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

func MessageExpressionContext(msg Message) ExpressionContext {
    panic("not implemented")
}
