package ccl

import (
	"fmt"
	"strings"
	"unicode"
)

func tokenize(input string) ([]cclToken, error) {
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

			// 檢查是否為布林關鍵字或 nil
			if word == "true" || word == "false" {
				tokens = append(tokens, cclToken{typ: tBOOLEAN, value: word})
			} else if word == "nil" || word == "null" {
				tokens = append(tokens, cclToken{typ: tNIL, value: word})
			} else {
				tokens = append(tokens, cclToken{typ: tIDENT, value: word})
			}
		case ch == '@':
			tokens = append(tokens, cclToken{typ: tAT, value: "@"})
			i++
		case ch == '#':
			tokens = append(tokens, cclToken{typ: tROW_INDEX, value: "#"})
			i++
		case isDigit(ch):
			start := i
			for i < len(input) && isDigit(input[i]) {
				i++
			}
			if i < len(input) && input[i] == '.' {
				// 檢查是否為小數點或 . 運算符
				// 如果後面跟著數字，且前一個 token 不是標識符等，則視為小數點
				isDecimal := false
				if i+1 < len(input) && isDigit(input[i+1]) {
					if len(tokens) > 0 {
						lastTok := tokens[len(tokens)-1]
						if lastTok.typ == tIDENT || lastTok.typ == tCOL_INDEX || lastTok.typ == tCOL_NAME || lastTok.typ == tAT || lastTok.typ == tRPAREN {
							isDecimal = false
						} else {
							isDecimal = true
						}
					} else {
						isDecimal = true
					}
				}

				if isDecimal {
					i++
					for i < len(input) && isDigit(input[i]) {
						i++
					}
					tokens = append(tokens, cclToken{typ: tNUMBER, value: input[start:i]})
				} else {
					tokens = append(tokens, cclToken{typ: tNUMBER, value: input[start:i]})
					// 不前進 i，讓下一個 case 處理 '.'
				}
			} else {
				tokens = append(tokens, cclToken{typ: tNUMBER, value: input[start:i]})
			}
		case ch == '.':
			// 檢查是否為數字開頭的小數（如 .5）
			if i+1 < len(input) && isDigit(input[i+1]) {
				isDecimal := true
				if len(tokens) > 0 {
					lastTok := tokens[len(tokens)-1]
					if lastTok.typ == tIDENT || lastTok.typ == tCOL_INDEX || lastTok.typ == tCOL_NAME || lastTok.typ == tAT || lastTok.typ == tRPAREN {
						isDecimal = false
					}
				}
				if isDecimal {
					start := i
					i++
					for i < len(input) && isDigit(input[i]) {
						i++
					}
					tokens = append(tokens, cclToken{typ: tNUMBER, value: input[start:i]})
				} else {
					tokens = append(tokens, cclToken{typ: tDOT, value: "."})
					i++
				}
			} else {
				tokens = append(tokens, cclToken{typ: tDOT, value: "."})
				i++
			}
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
		case ch == '=':
			// Check if it's == (comparison) or = (assignment)
			if i+1 < len(input) && input[i+1] == '=' {
				tokens = append(tokens, cclToken{typ: tOPERATOR, value: "=="})
				i += 2
			} else {
				tokens = append(tokens, cclToken{typ: tASSIGN, value: "="})
				i++
			}
		case ch == '<', ch == '>', ch == '!':
			// 處理比較運算符，包括 <=, >=, !=
			start := i
			i++
			// 檢查是否有後續的 =
			if i < len(input) && input[i] == '=' {
				i++
			}
			tokens = append(tokens, cclToken{typ: tOPERATOR, value: input[start:i]})
		case ch == '&':
			// 處理 & (字串連接) 和 && (邏輯與)
			if i+1 < len(input) && input[i+1] == '&' {
				tokens = append(tokens, cclToken{typ: tOPERATOR, value: "&&"})
				i += 2
			} else {
				tokens = append(tokens, cclToken{typ: tOPERATOR, value: "&"})
				i++
			}
		case ch == '|':
			// 處理 || (邏輯或)
			if i+1 < len(input) && input[i+1] == '|' {
				tokens = append(tokens, cclToken{typ: tOPERATOR, value: "||"})
				i += 2
			} else {
				return nil, fmt.Errorf("invalid operator: single '|' is not supported, use '||' for logical OR")
			}
		case ch == '[':
			// 處理 [colIndex] 或 ['colName'] 語法
			i++ // 跳過 '['
			if i >= len(input) {
				return nil, fmt.Errorf("unclosed bracket at end of input")
			}

			// 檢查是否為 ['colName'] 形式（帶引號的欄位名稱）
			if input[i] == '\'' || input[i] == '"' {
				quoteChar := input[i]
				i++ // 跳過引號
				start := i
				for i < len(input) && input[i] != quoteChar {
					i++
				}
				if i >= len(input) {
					return nil, fmt.Errorf("unclosed string in bracket column reference")
				}
				colName := input[start:i]
				i++ // 跳過結束引號

				// 期望 ']'
				if i >= len(input) || input[i] != ']' {
					return nil, fmt.Errorf("expected ']' after column name reference")
				}
				i++ // 跳過 ']'
				tokens = append(tokens, cclToken{typ: tCOL_NAME, value: colName})
			} else {
				// [colIndex] 形式（不帶引號的欄位索引）
				start := i
				for i < len(input) && input[i] != ']' && !unicode.IsSpace(rune(input[i])) {
					i++
				}
				if i >= len(input) || input[i] != ']' {
					return nil, fmt.Errorf("expected ']' after column index reference")
				}
				colIndex := input[start:i]
				i++ // 跳過 ']'
				tokens = append(tokens, cclToken{typ: tCOL_INDEX, value: colIndex})
			}
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

func isLetter(ch byte) bool { return unicode.IsLetter(rune(ch)) || ch == '_' }
func isDigit(ch byte) bool  { return unicode.IsDigit(rune(ch)) }
func isOperatorChar(ch byte) bool {
	return strings.ContainsRune("+-*/%^", rune(ch))
}
