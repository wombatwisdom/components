package spec

import (
	"fmt"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
)

// ExprLangExpressionFactory implements ExpressionFactory using expr-lang
type ExprLangExpressionFactory struct{}

// NewExprLangExpressionFactory creates a new expression factory using expr-lang
func NewExprLangExpressionFactory() ExpressionFactory {
	return &ExprLangExpressionFactory{}
}

func (e *ExprLangExpressionFactory) ParseExpression(exprStr string) (Expression, error) {
	// Compile the expression
	program, err := expr.Compile(exprStr)
	if err != nil {
		return nil, fmt.Errorf("failed to compile expression '%s': %w", exprStr, err)
	}

	return &exprLangExpression{
		program: program,
		source:  exprStr,
	}, nil
}

// exprLangExpression implements Expression using expr-lang
type exprLangExpression struct {
	program *vm.Program
	source  string
}

func (e *exprLangExpression) EvalString(ctx ExpressionContext) (string, error) {
	result, err := vm.Run(e.program, ctx)
	if err != nil {
		return "", fmt.Errorf("failed to evaluate expression '%s': %w", e.source, err)
	}

	if result == nil {
		return "", nil
	}

	if str, ok := result.(string); ok {
		return str, nil
	}

	return fmt.Sprintf("%v", result), nil
}

func (e *exprLangExpression) EvalInt(ctx ExpressionContext) (int, error) {
	result, err := vm.Run(e.program, ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to evaluate expression '%s': %w", e.source, err)
	}

	if result == nil {
		return 0, nil
	}

	switch v := result.(type) {
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case float64:
		return int(v), nil
	default:
		return 0, fmt.Errorf("expression '%s' evaluated to %T, expected numeric type", e.source, result)
	}
}

func (e *exprLangExpression) EvalBool(ctx ExpressionContext) (bool, error) {
	result, err := vm.Run(e.program, ctx)
	if err != nil {
		return false, fmt.Errorf("failed to evaluate expression '%s': %w", e.source, err)
	}

	if result == nil {
		return false, nil
	}

	if b, ok := result.(bool); ok {
		return b, nil
	}

	return false, fmt.Errorf("expression '%s' evaluated to %T, expected bool", e.source, result)
}