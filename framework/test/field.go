package test

import (
	"fmt"
	"slices"
	"strings"

	"github.com/wombatwisdom/components/framework/spec"
)

type dynamicFieldFactory struct {
	exprFactory spec.ExpressionFactory
}

func (d *dynamicFieldFactory) NewDynamicField(expr string) spec.DynamicField {
	if expr == "" {
		return &constantField{}
	}

	if !strings.HasPrefix(expr, "${!") || !strings.HasSuffix(expr, "}") {
		return &constantField{value: expr}
	}

	// Remove the ${! and } wrapper to get the actual expression
	actualExpr := strings.TrimSuffix(strings.TrimPrefix(expr, "${!"), "}")
	
	parsedExpr, err := d.exprFactory.ParseExpression(actualExpr)
	if err != nil {
		// Return constant field on error for simplicity
		return &constantField{value: expr}
	}

	return &dynamicField{expr: parsedExpr}
}

type constantField struct {
	value string
}

func (c *constantField) String() string {
	return c.value
}

func (c *constantField) Int() int {
	return 0 // Simplified implementation
}

func (c *constantField) Bool() bool {
	return slices.Contains([]string{"true", "yes", "on", "1"}, strings.ToLower(c.value))
}

func (c *constantField) AsString(_ spec.Message) (string, error) {
	return c.value, nil
}

func (c *constantField) AsBool(_ spec.Message) (bool, error) {
	return slices.Contains([]string{"true", "yes", "on", "1"}, strings.ToLower(c.value)), nil
}

type dynamicField struct {
	expr spec.Expression
}

func (d *dynamicField) String() string {
	return "" // Simplified implementation
}

func (d *dynamicField) Int() int {
	return 0 // Simplified implementation
}

func (d *dynamicField) Bool() bool {
	return false // Simplified implementation
}

func (d *dynamicField) AsString(msg spec.Message) (string, error) {
	res, err := d.AsAny(msg)
	if err != nil {
		return "", err
	}

	if res == nil {
		return "", nil
	}

	if str, ok := res.(string); ok {
		return str, nil
	}

	return "", fmt.Errorf("expected string, got %T", res)
}

func (d *dynamicField) AsBool(msg spec.Message) (bool, error) {
	res, err := d.AsAny(msg)
	if err != nil {
		return false, err
	}

	if res == nil {
		return false, nil
	}

	switch k := res.(type) {
	case bool:
		return k, nil
	case string:
		return slices.Contains([]string{"true", "yes", "on", "1"}, strings.ToLower(k)), nil
	}

	return false, fmt.Errorf("expected bool, got %T", res)
}

func (d *dynamicField) AsAny(msg spec.Message) (any, error) {
	if d.expr == nil {
		return nil, nil
	}

	// Create expression context from message
	ctx := spec.MessageExpressionContext(msg)
	ctx["this"] = msg

	// Try to evaluate as string first, then fallback to other types
	result, err := d.expr.EvalString(ctx)
	if err == nil {
		return result, nil
	}

	// Try as bool
	boolResult, boolErr := d.expr.EvalBool(ctx)
	if boolErr == nil {
		return boolResult, nil
	}

	// Try as int
	intResult, intErr := d.expr.EvalInt(ctx)
	if intErr == nil {
		return intResult, nil
	}

	return nil, fmt.Errorf("evaluation error: %s", err)
}
