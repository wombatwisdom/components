package spec

import (
	"bytes"
	"fmt"
)

// interpolatedExpression represents an interpolated string expression
type interpolatedExpression struct {
	resolvers   []interpolationResolver
	staticValue string
	isStatic    bool
	exprFactory ExpressionFactory
}

// newInterpolatedExpression creates a new interpolated expression from resolvers
func newInterpolatedExpression(resolvers []interpolationResolver, exprFactory ExpressionFactory) *interpolatedExpression {
	e := &interpolatedExpression{
		resolvers:   resolvers,
		exprFactory: exprFactory,
		isStatic:    true,
	}

	// Check if all resolvers are static
	var staticBuf bytes.Buffer
	for _, r := range resolvers {
		if !r.IsStatic() {
			e.isStatic = false
			return e
		}
		// Since it's static, we can resolve it once with nil context
		if val, err := r.Resolve(nil, nil); err == nil {
			staticBuf.WriteString(val)
		}
	}

	// If all static, cache the value
	if e.isStatic {
		e.staticValue = staticBuf.String()
	}

	return e
}

// EvalString evaluates the expression and returns a string
func (e *interpolatedExpression) EvalString(ctx ExpressionContext) (string, error) {
	// Fast path for static strings
	if e.isStatic {
		return e.staticValue, nil
	}

	var result bytes.Buffer
	for _, resolver := range e.resolvers {
		val, err := resolver.Resolve(ctx, e.exprFactory)
		if err != nil {
			return "", err
		}
		result.WriteString(val)
	}

	return result.String(), nil
}

// EvalInt evaluates the expression and returns an int
func (e *interpolatedExpression) EvalInt(ctx ExpressionContext) (int, error) {
	str, err := e.EvalString(ctx)
	if err != nil {
		return 0, err
	}

	// Try to parse the entire string as an expression if it contains interpolations
	if !e.isStatic && e.exprFactory != nil {
		expr, err := e.exprFactory.ParseExpression(str)
		if err != nil {
			return 0, fmt.Errorf("failed to parse result as expression: %w", err)
		}
		return expr.EvalInt(ctx)
	}

	return 0, fmt.Errorf("interpolated expressions do not support direct int evaluation")
}

// EvalBool evaluates the expression and returns a bool
func (e *interpolatedExpression) EvalBool(ctx ExpressionContext) (bool, error) {
	str, err := e.EvalString(ctx)
	if err != nil {
		return false, err
	}

	// Try to parse the entire string as an expression if it contains interpolations
	if !e.isStatic && e.exprFactory != nil {
		expr, err := e.exprFactory.ParseExpression(str)
		if err != nil {
			return false, fmt.Errorf("failed to parse result as expression: %w", err)
		}
		return expr.EvalBool(ctx)
	}

	return false, fmt.Errorf("interpolated expressions do not support direct bool evaluation")
}

// IsStatic returns true if the expression contains no interpolations
func (e *interpolatedExpression) IsStatic() bool {
	return e.isStatic
}

// StaticValue returns the static value if IsStatic() is true
func (e *interpolatedExpression) StaticValue() string {
	if !e.isStatic {
		return ""
	}
	return e.staticValue
}
