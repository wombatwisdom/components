package spec_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wombatwisdom/components/framework/spec"
)

// mockExpression implements spec.Expression for testing
type mockExpression struct {
	value string
}

func (m *mockExpression) EvalString(ctx spec.ExpressionContext) (string, error) {
	// Mock expr-lang style access to json data
	if m.value == "json.topic" {
		if jsonData, ok := ctx["json"].(map[string]any); ok {
			if val, ok := jsonData["topic"].(string); ok {
				return val, nil
			}
		}
	}
	// Default: return the expression itself
	return m.value, nil
}

func (m *mockExpression) EvalInt(ctx spec.ExpressionContext) (int, error) {
	return 0, nil
}

func (m *mockExpression) EvalBool(ctx spec.ExpressionContext) (bool, error) {
	return false, nil
}

// mockExpressionFactory creates mock expressions
type mockExpressionFactory struct{}

func (m *mockExpressionFactory) ParseExpression(expr string) (spec.Expression, error) {
	return &mockExpression{value: expr}, nil
}

func TestInterpolatedExpression(t *testing.T) {
	factory := spec.NewInterpolatedExpressionFactory(&mockExpressionFactory{})

	tests := []struct {
		name         string
		input        string
		context      spec.ExpressionContext
		expectResult string
		expectStatic bool
	}{
		{
			name:         "static string",
			input:        "hello world",
			expectResult: "hello world",
			expectStatic: true,
		},
		{
			name:  "simple interpolation",
			input: "Hello ${!name}!",
			context: spec.ExpressionContext{
				"json": map[string]any{"name": "World"},
			},
			expectResult: "Hello name!",
			expectStatic: false,
		},
		{
			name:  "topic interpolation",
			input: "test/output/${!json.topic}",
			context: spec.ExpressionContext{
				"json": map[string]any{"topic": "sensor-data"},
			},
			expectResult: "test/output/sensor-data",
			expectStatic: false,
		},
		{
			name:         "escaped interpolation",
			input:        "Template: ${{!example}}",
			expectResult: "Template: ${!example}",
			expectStatic: true,
		},
		{
			name:         "dollar sign handling",
			input:        "Price: $100",
			expectResult: "Price: $100",
			expectStatic: true,
		},
		{
			name:  "multiple interpolations",
			input: "${!first}/${!second}",
			context: spec.ExpressionContext{
				"json": map[string]any{"first": "a", "second": "b"},
			},
			expectResult: "first/second",
			expectStatic: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := factory.ParseInterpolatedExpression(tt.input)
			require.NoError(t, err)

			assert.Equal(t, tt.expectStatic, expr.IsStatic())

			if tt.expectStatic {
				assert.Equal(t, tt.expectResult, expr.StaticValue())
			}

			result, err := expr.EvalString(tt.context)
			require.NoError(t, err)
			assert.Equal(t, tt.expectResult, result)
		})
	}
}

func TestInterpolatedExpressionOptimized(t *testing.T) {
	factory := spec.NewInterpolatedExpressionFactory(&mockExpressionFactory{})

	tests := []struct {
		name         string
		input        string
		expectResult string
	}{
		{
			name:         "single interpolation with static prefix",
			input:        "test/output/${!json.topic}",
			expectResult: "test/output/json.topic",
		},
		{
			name:         "multiple interpolations",
			input:        "${!first}/middle/${!second}/end",
			expectResult: "first/middle/second/end",
		},
		{
			name:         "static string only",
			input:        "just/a/static/string",
			expectResult: "just/a/static/string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := factory.ParseInterpolatedExpression(tt.input)
			require.NoError(t, err)

			// Verify the result is correct
			result, err := expr.EvalString(nil)
			require.NoError(t, err)
			assert.Equal(t, tt.expectResult, result)

			// The key test is that this should be efficient
			// We can't easily test the internal structure without reflection
			// but we can verify the behavior is correct
		})
	}
}

func TestInterpolatedExpressionEdgeCases(t *testing.T) {
	factory := spec.NewInterpolatedExpressionFactory(&mockExpressionFactory{})

	tests := []struct {
		name        string
		input       string
		expectError bool
	}{
		{
			name:        "unclosed interpolation",
			input:       "${!unclosed", // Simpler test case
			expectError: true,
		},
		{
			name:        "nested braces",
			input:       "test ${!json{nested}}",
			expectError: false, // Should handle nested braces
		},
		{
			name:        "empty interpolation",
			input:       "test ${!}",
			expectError: false, // Empty expression is valid
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := factory.ParseInterpolatedExpression(tt.input)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
