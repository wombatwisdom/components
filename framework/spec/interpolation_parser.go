package spec

// parseResult represents the result of a parser function
type parseResult[T any] struct {
	Payload   T
	Err       *parseError
	Remaining []rune
}

// parseFunc is a parser function that attempts to parse a specific pattern
type parseFunc[T any] func([]rune) parseResult[T]

// parseError represents a parsing error
type parseError struct {
	Input   []rune
	Message string
}

func (e *parseError) Error() string {
	preview := string(e.Input)
	if len(preview) > 20 {
		preview = preview[:20] + "..."
	}
	return "parse error: " + e.Message + " (near: " + preview + ")"
}

// Parser helper functions

func parseSuccess[T any](payload T, remaining []rune) parseResult[T] {
	return parseResult[T]{
		Payload:   payload,
		Remaining: remaining,
	}
}

func parseFail[T any](err *parseError, input []rune) parseResult[T] {
	return parseResult[T]{
		Err:       err,
		Remaining: input,
	}
}

func parseTerm(str string) parseFunc[string] {
	runes := []rune(str)
	return func(input []rune) parseResult[string] {
		if len(input) < len(runes) {
			return parseFail[string](&parseError{input, "expected: " + str}, input)
		}
		for i, r := range runes {
			if input[i] != r {
				return parseFail[string](&parseError{input, "expected: " + str}, input)
			}
		}
		return parseSuccess(str, input[len(runes):])
	}
}
