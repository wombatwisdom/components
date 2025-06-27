package core

import (
	"iter"

	"github.com/wombatwisdom/components/framework/spec"
)

// exprEvaluatorFactory implements spec.ExpressionFactory using our FieldEvaluator
type exprEvaluatorFactory struct{}

// NewExpressionFactory creates a new expression factory using expr-lang
func NewExpressionFactory() spec.ExpressionFactory {
	return &exprEvaluatorFactory{}
}

// ParseExpression creates a new Expression from the given expression string
func (f *exprEvaluatorFactory) ParseExpression(expr string) (spec.Expression, error) {
	evaluator, err := NewFieldEvaluator(expr)
	if err != nil {
		return nil, err
	}

	return &exprEvaluatorExpression{
		evaluator: evaluator,
	}, nil
}

// exprEvaluatorExpression implements spec.Expression using FieldEvaluator
type exprEvaluatorExpression struct {
	evaluator *FieldEvaluator
}

// convertContextToBatch converts an ExpressionContext to our spec interfaces for evaluation
func (e *exprEvaluatorExpression) convertContextToBatch(ctx spec.ExpressionContext) (spec.Batch, error) {
	// Create a mock message from the context
	msg := spec.NewBytesMessage([]byte(""))

	// Set content if available
	if content, ok := ctx["content"].(string); ok {
		msg.SetRaw([]byte(content))
	}

	// Set metadata if available
	if metadata, ok := ctx["metadata"].(map[string]any); ok {
		for k, v := range metadata {
			msg.SetMetadata(k, v)
		}
	}

	// Create a batch with this single message
	batch := &mockBatch{messages: []spec.Message{msg}}
	return batch, nil
}

// EvalString evaluates the expression and returns the result as a string
func (e *exprEvaluatorExpression) EvalString(ctx spec.ExpressionContext) (string, error) {
	batch, err := e.convertContextToBatch(ctx)
	if err != nil {
		return "", err
	}

	return e.evaluator.EvalString(batch, 0)
}

// EvalInt evaluates the expression and returns the result as an integer
func (e *exprEvaluatorExpression) EvalInt(ctx spec.ExpressionContext) (int, error) {
	batch, err := e.convertContextToBatch(ctx)
	if err != nil {
		return 0, err
	}

	return e.evaluator.EvalInt(batch, 0)
}

// EvalBool evaluates the expression and returns the result as a boolean
func (e *exprEvaluatorExpression) EvalBool(ctx spec.ExpressionContext) (bool, error) {
	batch, err := e.convertContextToBatch(ctx)
	if err != nil {
		return false, err
	}

	return e.evaluator.EvalBool(batch, 0)
}

// mockBatch implements spec.Batch for internal use in the expression bridge
type mockBatch struct {
	messages []spec.Message
}

func (mb *mockBatch) Messages() iter.Seq2[int, spec.Message] {
	return func(yield func(int, spec.Message) bool) {
		for i, msg := range mb.messages {
			if !yield(i, msg) {
				return
			}
		}
	}
}

func (mb *mockBatch) Append(msg spec.Message) {
	mb.messages = append(mb.messages, msg)
}
