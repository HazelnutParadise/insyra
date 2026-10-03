package dsl

import "strings"

// Tokenize splits a command line into its words, the way the REPL, scripts and
// Session.Execute read it: words are separated by spaces or tabs, a single or
// double quote keeps the spaces inside it, and a backslash takes the next
// character as it is.
func Tokenize(input string) []string {
	result := []string{}
	var builder strings.Builder
	quote := rune(0)
	escaped := false

	flush := func() {
		if builder.Len() == 0 {
			return
		}
		result = append(result, builder.String())
		builder.Reset()
	}

	for _, ch := range input {
		if escaped {
			builder.WriteRune(ch)
			escaped = false
			continue
		}
		if ch == '\\' {
			escaped = true
			continue
		}
		if quote != 0 {
			if ch == quote {
				quote = 0
				continue
			}
			builder.WriteRune(ch)
			continue
		}
		if ch == '"' || ch == '\'' {
			quote = ch
			continue
		}
		if ch == ' ' || ch == '\t' {
			flush()
			continue
		}
		builder.WriteRune(ch)
	}
	flush()
	return result
}
