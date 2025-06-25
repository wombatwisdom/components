package test

import (
	"fmt"
	"github.com/google/cel-go/cel"
	"github.com/wombatwisdom/components/spec"
	"slices"
	"strings"
)

type dynamicFieldFactory struct {
	env *cel.Env
}

func (d *dynamicFieldFactory) NewDynamicField(expr string) (spec.DynamicField, error) {
	if expr == "" {
		return &constantField{}, nil
	}

	if !strings.HasPrefix(expr, "${!") || !strings.HasSuffix(expr, "}") {
		return &constantField{value: expr}, nil
	}

	ast, issues := d.env.Compile(expr)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("failed to compile expression: %s", issues.Err())
	}

	prg, err := d.env.Program(ast)
	if err != nil {
		return nil, fmt.Errorf("program construction error: %s", err)
	}

	return &dynamicField{expr: prg}, nil
}

type constantField struct {
	value string
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
