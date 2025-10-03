package spec

// InterpolatedExpression represents an expression that may contain ${!...} interpolations
type InterpolatedExpression interface {
	Expression

	// IsStatic returns true if the expression contains no interpolations
	IsStatic() bool

	// StaticValue returns the static value if IsStatic() is true
	StaticValue() string
}

// InterpolatedExpressionFactory creates interpolated expressions with a specific backend
type InterpolatedExpressionFactory struct {
	exprFactory ExpressionFactory
}

// NewInterpolatedExpressionFactory creates a factory that uses the given expression parser
// for evaluating dynamic segments within ${!...} blocks
func NewInterpolatedExpressionFactory(exprFactory ExpressionFactory) *InterpolatedExpressionFactory {
	return &InterpolatedExpressionFactory{
		exprFactory: exprFactory,
	}
}

// ParseInterpolatedExpression parses a string that may contain ${!...} interpolations
func (f *InterpolatedExpressionFactory) ParseInterpolatedExpression(expr string) (InterpolatedExpression, error) {
	resolvers, err := parseInterpolatedString(expr)
	if err != nil {
		return nil, err
	}

	return newInterpolatedExpression(resolvers, f.exprFactory), nil
}
