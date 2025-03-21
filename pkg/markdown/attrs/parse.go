package attrs

import (
	"errors"
	"fmt"
	"strings"
)

// Values are the extended attributes of a Markdown node.
//
//	```go {name="foo.go" description="bar"}
//	func foo() {}
//	```
type Values struct {
	Name        string
	Description string
}

// ParseValues parses the extended attribute string and returns a Values struct.
func ParseValues(expr string) (Values, error) {
	var err error
	s := strings.TrimSpace(expr)
	offs := 0

	if len(s) == 0 {
		return Values{}, errors.New("empty extended attributes")
	}
	if s[0] != '{' {
		return Values{}, fmt.Errorf("expected '{' for extended attributes in code block:\n%s", s)
	}
	offs++

	vals := Values{}
	for offs < len(s) {
		offs = skipWhitespace(s, offs)
		if offs >= len(s) {
			return Values{}, fmt.Errorf("extend attribute not closed")
		}

		if s[offs] == '}' {
			break
		}

		// Parse the field name.
		var fieldName string
		fieldName, offs, err = parseFieldName(s, offs)
		if err != nil {
			return Values{}, fmt.Errorf("parse field name: %w", err)
		}

		offs = skipWhitespace(s, offs)

		// Parse the equal sign.
		if offs >= len(s) || s[offs] != '=' {
			return Values{}, fmt.Errorf("expected '=' after field name %q", fieldName)
		}
		offs++

		// Skip whitespace.
		offs = skipWhitespace(s, offs)

		// Parse the quoted value.
		var val string
		val, offs, err = parseQuotedValue(s, offs)
		if err != nil {
			return Values{}, fmt.Errorf("parse field %q value: %w", fieldName, err)
		}

		if offs >= len(s) || (!isWhitespace(s[offs]) && s[offs] != '}') {
			return Values{}, fmt.Errorf("field %q not separated with whitespace", fieldName)
		}

		// Assign the value.
		switch fieldName {
		case "name":
			vals.Name = val
		case "description":
			vals.Description = val
		default:
			return Values{}, fmt.Errorf("unsupported field name %q", fieldName)
		}
	}

	offs = skipWhitespace(s, offs)
	if offs >= len(s) {
		return Values{}, fmt.Errorf("missing closing delimiter '}' in: %s", s)
	}
	if s[offs] != '}' {
		return Values{}, fmt.Errorf("expected '}' for extended attributes in: %s", s)
	}
	offs++

	if offs != len(s) {
		return Values{}, fmt.Errorf("expr not empty after parsing closing brace in: %s", s)
	}
	return vals, nil
}

// skipWhitespace advances the offset past any whitespace.
func skipWhitespace(s string, offs int) int {
	for offs < len(s) && isWhitespace(s[offs]) {
		offs++
	}
	return offs
}

// parseFieldName parses the field name from the expr string.
func parseFieldName(s string, offs int) (string, int, error) {
	start := offs
	for offs < len(s) && isFieldNameChar(s[offs]) {
		offs++
	}
	if start == offs {
		return "", offs, fmt.Errorf("expected field name after %s; got %q", s[:start], s[offs])
	}
	name := s[start:offs]
	return name, offs, nil
}

// isWhitespace checks if a character is a whitespace character.
func isWhitespace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

// parseQuotedValue parses a quoted string value from the expr string.
func parseQuotedValue(s string, offs int) (string, int, error) {
	if offs >= len(s) || (s[offs] != '"' && s[offs] != '\'') {
		return "", offs, errors.New("missing start quote")
	}

	startQuote := s[offs]
	offs++ // skip the opening quote

	start := offs
	for offs < len(s) && s[offs] != startQuote {
		offs++
	}

	if offs >= len(s) {
		return "", offs, errors.New("missing closing quote")
	}

	value := s[start:offs]
	offs++ // skip the closing quote

	return value, offs, nil
}

func isFieldNameChar(c byte) bool {
	isUpperAlpha := 'A' <= c && c <= 'Z'
	isLowerAlpha := 'a' <= c && c <= 'z'
	isDigit := '0' <= c && c <= '9'
	isValidPunctuation := c == '-' || c == '_'
	return isUpperAlpha || isLowerAlpha || isDigit || isValidPunctuation
}
