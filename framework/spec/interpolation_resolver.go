package spec

// interpolationResolver is an interface for resolving a segment of an interpolated string
type interpolationResolver interface {
	// Resolve evaluates this segment and returns a string
	Resolve(ctx ExpressionContext, parser ExpressionFactory) (string, error)
	// IsStatic returns true if this resolver always returns the same value
	IsStatic() bool
}

// staticResolver is a resolver that returns a static string
type staticResolver string

// Resolve returns the static string
func (s staticResolver) Resolve(ctx ExpressionContext, parser ExpressionFactory) (string, error) {
	return string(s), nil
}

// IsStatic always returns true for static resolvers
func (s staticResolver) IsStatic() bool {
	return true
}

// dynamicResolver evaluates an expression within ${!...}
type dynamicResolver struct {
	expression string
}

// Resolve evaluates the expression using the provided parser
func (d *dynamicResolver) Resolve(ctx ExpressionContext, parser ExpressionFactory) (string, error) {
	expr, err := parser.ParseExpression(d.expression)
	if err != nil {
		return "", err
	}
	return expr.EvalString(ctx)
}

// IsStatic always returns false for dynamic resolvers
func (d *dynamicResolver) IsStatic() bool {
	return false
}

// Parsing functions for interpolation

var interpStart = parseTerm("${!")
var interpEscapedStart = parseTerm("${{!")

// parseInterpolation parses a ${!...} interpolation
func parseInterpolation() parseFunc[interpolationResolver] {
	return func(input []rune) parseResult[interpolationResolver] {
		// Try to match ${!
		startRes := interpStart(input)
		if startRes.Err != nil {
			return parseFail[interpolationResolver](startRes.Err, input)
		}

		// Find the closing }
		remaining := startRes.Remaining
		depth := 1
		exprEnd := -1

		for i := 0; i < len(remaining); i++ {
			if remaining[i] == '{' {
				depth++
			} else if remaining[i] == '}' {
				depth--
				if depth == 0 {
					exprEnd = i
					break
				}
			}
		}

		if exprEnd == -1 {
			// We matched ${! but no closing }, this is an error
			return parseFail[interpolationResolver](&parseError{input, "unclosed interpolation"}, input)
		}

		expr := string(remaining[:exprEnd])
		return parseSuccess[interpolationResolver](
			&dynamicResolver{expression: expr},
			remaining[exprEnd+1:],
		)
	}
}

// parseEscapedInterpolation parses a ${{!...}} escaped interpolation
func parseEscapedInterpolation() parseFunc[interpolationResolver] {
	return func(input []rune) parseResult[interpolationResolver] {
		// Try to match ${{!
		startRes := interpEscapedStart(input)
		if startRes.Err != nil {
			return parseFail[interpolationResolver](startRes.Err, input)
		}

		// Find the closing }}
		remaining := startRes.Remaining
		endPos := -1
		for i := 0; i < len(remaining)-1; i++ {
			if remaining[i] == '}' && remaining[i+1] == '}' {
				endPos = i
				break
			}
		}

		if endPos == -1 {
			return parseFail[interpolationResolver](&parseError{input, "unclosed escaped interpolation"}, input)
		}

		content := string(remaining[:endPos])
		escapedContent := "${!" + content + "}"

		return parseSuccess[interpolationResolver](
			staticResolver(escapedContent),
			remaining[endPos+2:],
		)
	}
}

// parseInterpolatedString parses a string with interpolations into resolvers
func parseInterpolatedString(expr string) ([]interpolationResolver, error) {
	var resolvers []interpolationResolver

	remaining := []rune(expr)

	for len(remaining) > 0 {
		// Check for interpolation patterns first
		if len(remaining) >= 4 && string(remaining[0:4]) == "${{!" {
			// Escaped interpolation
			res := parseEscapedInterpolation()(remaining)
			if res.Err != nil {
				return nil, res.Err
			}
			remaining = res.Remaining
			resolvers = append(resolvers, res.Payload)
		} else if len(remaining) >= 3 && string(remaining[0:3]) == "${!" {
			// Regular interpolation - must parse successfully
			res := parseInterpolation()(remaining)
			if res.Err != nil {
				return nil, res.Err
			}
			remaining = res.Remaining
			resolvers = append(resolvers, res.Payload)
		} else {
			// Accumulate static characters until we hit an interpolation or end
			endIdx := len(remaining)

			// Find the next interpolation start
			for i := 0; i < len(remaining); i++ {
				if i+3 <= len(remaining) && string(remaining[i:i+3]) == "${!" {
					endIdx = i
					break
				}
				if i+4 <= len(remaining) && string(remaining[i:i+4]) == "${{!" {
					endIdx = i
					break
				}
			}

			// Add all static characters up to the interpolation (or end)
			if endIdx > 0 {
				resolvers = append(resolvers, staticResolver(string(remaining[0:endIdx])))
				remaining = remaining[endIdx:]
			}
		}
	}

	return resolvers, nil
}
