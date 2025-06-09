package ccl

import (
	"fmt"
	"strings"
	"unicode"
)

func Tokenize(input string) ([]cclToken, error) {
	tokens := []cclToken{}
	i := 0
	for i < len(input) {
		ch := input[i]
		switch {
		case unicode.IsSpace(rune(ch)):
			i++
		case isLetter(ch):
			start := i
			for i < len(input) && (isLetter(input[i]) || isDigit(input[i])) {
				i++
			}
			word := input[start:i]

			// 檢查是否為布林關鍵字
			if word == "true" || word == "false" {
				tokens = append(tokens, cclToken{typ: tBOOLEAN, value: word})
			} else {
				tokens = append(tokens, cclToken{typ: tIDENT, value: word})
			}
		case isDigit(ch) || ch == '.':
			start := i
			for i < len(input) && (isDigit(input[i]) || input[i] == '.') {
				i++
			}
			tokens = append(tokens, cclToken{typ: tNUMBER, value: input[start:i]})
		case ch == '"' || ch == '\'':
			quoteChar := ch // 保存當前引號字符
			i++
			start := i
			for i < len(input) && input[i] != quoteChar {
				i++
			}
			if i >= len(input) {
				return nil, fmt.Errorf("unclosed string starting with %c", quoteChar)
			}
			tokens = append(tokens, cclToken{typ: tSTRING, value: input[start:i]})
			i++
		case ch == '(':
			tokens = append(tokens, cclToken{typ: tLPAREN, value: "("})
			i++
		case ch == ')':
			tokens = append(tokens, cclToken{typ: tRPAREN, value: ")"})
			i++
		case ch == ',':
			tokens = append(tokens, cclToken{typ: tCOMMA, value: ","})
			i++
		default:
			start := i
			for i < len(input) && isOperatorChar(input[i]) {
				i++
			}
			tokens = append(tokens, cclToken{typ: tOPERATOR, value: input[start:i]})
		}
	}
	tokens = append(tokens, cclToken{typ: tEOF})
	return tokens, nil
}

func isLetter(ch byte) bool { return unicode.IsLetter(rune(ch)) }
func isDigit(ch byte) bool  { return unicode.IsDigit(rune(ch)) }
func isOperatorChar(ch byte) bool {
	return strings.ContainsRune("+-*/%^=<>!", rune(ch))
}
