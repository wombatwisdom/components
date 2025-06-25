package test

import (
	"fmt"
	"slices"
	"strings"

	"github.com/google/cel-go/cel"
	"github.com/wombatwisdom/components/framework/spec"
)

type dynamicFieldFactory struct {
	env *cel.Env
}

func (d *dynamicFieldFactory) NewDynamicField(expr string) spec.DynamicField {
	if expr == "" {
		return &constantField{}
	}

	if !strings.HasPrefix(expr, "${!") || !strings.HasSuffix(expr, "}") {
		return &constantField{value: expr}
	}

	ast, issues := d.env.Compile(expr)
	if issues != nil && issues.Err() != nil {
		// Return constant field on error for simplicity
		return &constantField{value: expr}
	}

	prg, err := d.env.Program(ast)
	if err != nil {
		// Return constant field on error for simplicity
		return &constantField{value: expr}
	}

	return &dynamicField{expr: prg}
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
	expr cel.Program
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

	res, _, err := d.expr.Eval(map[string]interface{}{
		"this": msg,
	})

	return res, fmt.Errorf("evaluation error: %s", err)
}
