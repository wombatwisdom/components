package spec

import (
	"fmt"
	"strings"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
)

type part interface {
	Eval(ctx ExpressionContext) (any, error)
}

type stringPart struct {
	value string
}

func (s *stringPart) Eval(ctx ExpressionContext) (any, error) {
	return s.value, nil
}

type exprPart struct {
	source string
	ex     *vm.Program
}

func (e *exprPart) Eval(ctx ExpressionContext) (any, error) {
	return vm.Run(e.ex, ctx)
}

func NewExprLangExpression(exprStr string) (Expression, error) {
	splits := strings.Split(exprStr, "${!")
	parts := make([]part, len(splits))
	for idx, s := range splits {
		if strings.Contains(s, "}") {
			expression := strings.TrimSuffix(s, "}")

			// Compile the expression
			program, err := expr.Compile(expression)
			if err != nil {
				return nil, fmt.Errorf("failed to compile expression '%s': %w", exprStr, err)
			}

			parts[idx] = &exprPart{source: expression, ex: program}
		} else {
			parts[idx] = &stringPart{value: s}
		}
	}

	return &exprLangExpression{
		parts:  parts,
		source: exprStr,
	}, nil
}

// exprLangExpression implements Expression using expr-lang
type exprLangExpression struct {
	parts  []part
	source string
}

func (e *exprLangExpression) Eval(ctx ExpressionContext) (string, error) {
	result := strings.Builder{}
	for _, p := range e.parts {
		res, err := p.Eval(ctx)
		if err != nil {
			return "", err
		}

		if res != nil {
			result.WriteString(fmt.Sprintf("%v", res))
		}
	}
	return result.String(), nil
}
