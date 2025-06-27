package core

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	"github.com/wombatwisdom/components/framework/spec"
)

// ExpressionSegment represents a part of the field - either literal text or a compiled expression
type ExpressionSegment struct {
	IsExpression bool
	Text         string      // For literal text
	Program      *vm.Program // For compiled expressions
	RawExpr      string      // Original expression text (for error reporting)
}

// FieldEvaluator holds precompiled expressions and can evaluate them against multiple data sets
type FieldEvaluator struct {
	segments []ExpressionSegment
	original string
}

// NewFieldEvaluator creates a new FieldEvaluator by parsing and precompiling expressions in the field
func NewFieldEvaluator(field string) (*FieldEvaluator, error) {
	evaluator := &FieldEvaluator{
		original: field,
		segments: make([]ExpressionSegment, 0),
	}

	if !strings.Contains(field, "${!") {
		// No expressions found, treat as single literal segment
		evaluator.segments = append(evaluator.segments, ExpressionSegment{
			IsExpression: false,
			Text:         field,
		})
		return evaluator, nil
	}

	remaining := field

	for {
		// Find the start of an expression
		startIdx := strings.Index(remaining, "${!")
		if startIdx == -1 {
			// No more expressions, add the rest as literal text if not empty
			if len(remaining) > 0 {
				evaluator.segments = append(evaluator.segments, ExpressionSegment{
					IsExpression: false,
					Text:         remaining,
				})
			}
			break
		}

		// Add literal text before the expression if any
		if startIdx > 0 {
			evaluator.segments = append(evaluator.segments, ExpressionSegment{
				IsExpression: false,
				Text:         remaining[:startIdx],
			})
		}

		// Find the end of the expression
		endIdx := strings.Index(remaining[startIdx:], "}")
		if endIdx == -1 {
			return nil, fmt.Errorf("unclosed expression starting at position %d", len(field)-len(remaining)+startIdx)
		}

		// Adjust endIdx to be relative to the remaining string
		endIdx += startIdx

		// Extract the expression (without the ${! and } delimiters)
		expression := remaining[startIdx+3 : endIdx]

		// Compile the expression (we'll use a nil environment for now, actual data comes at eval time)
		program, err := expr.Compile(expression)
		if err != nil {
			return nil, fmt.Errorf("failed to compile expression '%s': %w", expression, err)
		}

		// Add the compiled expression segment
		evaluator.segments = append(evaluator.segments, ExpressionSegment{
			IsExpression: true,
			Program:      program,
			RawExpr:      expression,
		})

		// Move past this expression
		remaining = remaining[endIdx+1:]
	}

	return evaluator, nil
}

// Eval evaluates the precompiled field against the provided batch and index, returns the result
func (fe *FieldEvaluator) Eval(batch spec.Batch, index int) (interface{}, error) {
	// Convert batch to slice for index validation
	messages := make([]spec.Message, 0)
	for _, msg := range batch.Messages() {
		messages = append(messages, msg)
	}

	// Validate batch and index bounds
	if len(messages) == 0 {
		return nil, fmt.Errorf("cannot evaluate expression on empty batch")
	}
	if index < 0 {
		return nil, fmt.Errorf("negative index %d is not allowed", index)
	}
	if index >= len(messages) {
		return nil, fmt.Errorf("index %d is out of bounds for batch of size %d", index, len(messages))
	}

	message := messages[index]
	rawData, err := message.Raw()
	if err != nil {
		return nil, fmt.Errorf("failed to get message raw data: %w", err)
	}

	// Helper functions for metadata access
	metaGet := func(key string) any {
		for k, v := range message.Metadata() {
			if k == key {
				return v
			}
		}
		return "" // Return empty string instead of nil for missing keys
	}

	metaList := func() map[string]any {
		result := make(map[string]any)
		for k, v := range message.Metadata() {
			result[k] = v
		}
		return result
	}

	// Create a helper structure for batch messages that provides convenient access
	batchData := make([]map[string]any, len(messages))
	for i, msg := range messages {
		msgRaw, _ := msg.Raw()
		msgMeta := make(map[string]any)
		for k, v := range msg.Metadata() {
			msgMeta[k] = v
		}

		// Create a closure that captures the metadata for this specific message
		metaCapture := msgMeta // Capture the current msgMeta
		batchData[i] = map[string]any{
			"payload":  string(msgRaw),
			"raw":      msgRaw,
			"metadata": msgMeta,
			"GetHeader": func(key string) any {
				if val, exists := metaCapture[key]; exists {
					return val
				}
				return "" // Return empty string instead of nil for missing keys
			},
		}
	}

	data := map[string]any{
		"batch":        batchData, // Use structured data with helper methods
		"index":        index,
		"payload":      string(rawData),
		"payloadBytes": rawData,
		"meta":         metaGet,
		"metas":        metaList(), // Call the function to get the map
	}

	// If there's only one segment and it's an expression, return the raw value
	if len(fe.segments) == 1 && fe.segments[0].IsExpression {
		output, err := expr.Run(fe.segments[0].Program, data)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate expression '%s': %w", fe.segments[0].RawExpr, err)
		}
		return output, nil
	}

	// Multiple segments or mixed content - concatenate to string
	var result strings.Builder
	for _, segment := range fe.segments {
		if segment.IsExpression {
			// Execute the compiled expression
			output, err := expr.Run(segment.Program, data)
			if err != nil {
				return nil, fmt.Errorf("failed to evaluate expression '%s': %w", segment.RawExpr, err)
			}
			// Convert output to string and append to result
			result.WriteString(fmt.Sprintf("%v", output))
		} else {
			// Append literal text
			result.WriteString(segment.Text)
		}
	}

	return result.String(), nil
}

// EvalString evaluates the field and returns the result as a string.
// All types are converted to their string representation using fmt.Sprintf.
func (fe *FieldEvaluator) EvalString(batch spec.Batch, index int) (string, error) {
	result, err := fe.Eval(batch, index)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%v", result), nil
}

// EvalInt evaluates the field and returns the result as an integer.
// Supports conversion from numeric types, strings (if parseable), and booleans.
// Returns error if the result cannot be converted to int.
func (fe *FieldEvaluator) EvalInt(batch spec.Batch, index int) (int, error) {
	result, err := fe.Eval(batch, index)
	if err != nil {
		return 0, err
	}

	switch v := result.(type) {
	case int:
		return v, nil
	case int8:
		return int(v), nil
	case int16:
		return int(v), nil
	case int32:
		return int(v), nil
	case int64:
		return int(v), nil
	case uint:
		return int(v), nil
	case uint8:
		return int(v), nil
	case uint16:
		return int(v), nil
	case uint32:
		return int(v), nil
	case uint64:
		return int(v), nil
	case float32:
		return int(v), nil
	case float64:
		return int(v), nil
	case string:
		parsed, parseErr := strconv.Atoi(v)
		if parseErr != nil {
			return 0, fmt.Errorf("cannot convert string '%s' to int: %w", v, parseErr)
		}
		return parsed, nil
	case bool:
		if v {
			return 1, nil
		}
		return 0, nil
	default:
		return 0, fmt.Errorf("cannot convert %T to int", result)
	}
}

// EvalBool evaluates the field and returns the result as a boolean.
// Supports conversion from booleans, numeric types (0 = false, non-zero = true),
// and strings ("true"/"1"/"yes"/"on" = true, "false"/"0"/"no"/"off"/"" = false).
// Returns error if the result cannot be converted to bool.
func (fe *FieldEvaluator) EvalBool(batch spec.Batch, index int) (bool, error) {
	result, err := fe.Eval(batch, index)
	if err != nil {
		return false, err
	}

	switch v := result.(type) {
	case bool:
		return v, nil
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%v", v) != "0", nil
	case float32, float64:
		return fmt.Sprintf("%v", v) != "0", nil
	case string:
		switch strings.ToLower(v) {
		case "true", "1", "yes", "on":
			return true, nil
		case "false", "0", "no", "off", "":
			return false, nil
		default:
			return false, fmt.Errorf("cannot convert string '%s' to bool", v)
		}
	default:
		return false, fmt.Errorf("cannot convert %T to bool", result)
	}
}

// EvalFloat64 evaluates the field and returns the result as a float64.
// Supports conversion from numeric types, strings (if parseable), and booleans.
// Returns error if the result cannot be converted to float64.
func (fe *FieldEvaluator) EvalFloat64(batch spec.Batch, index int) (float64, error) {
	result, err := fe.Eval(batch, index)
	if err != nil {
		return 0, err
	}

	switch v := result.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int8:
		return float64(v), nil
	case int16:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case uint:
		return float64(v), nil
	case uint8:
		return float64(v), nil
	case uint16:
		return float64(v), nil
	case uint32:
		return float64(v), nil
	case uint64:
		return float64(v), nil
	case string:
		parsed, parseErr := strconv.ParseFloat(v, 64)
		if parseErr != nil {
			return 0, fmt.Errorf("cannot convert string '%s' to float64: %w", v, parseErr)
		}
		return parsed, nil
	case bool:
		if v {
			return 1.0, nil
		}
		return 0.0, nil
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", result)
	}
}

// GetOriginal returns the original field string
func (fe *FieldEvaluator) GetOriginal() string {
	return fe.original
}

// HasExpressions returns true if the field contains any expressions
func (fe *FieldEvaluator) HasExpressions() bool {
	for _, segment := range fe.segments {
		if segment.IsExpression {
			return true
		}
	}
	return false
}

// GetExpressionCount returns the number of expressions in the field
func (fe *FieldEvaluator) GetExpressionCount() int {
	count := 0
	for _, segment := range fe.segments {
		if segment.IsExpression {
			count++
		}
	}
	return count
}
