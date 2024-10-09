package lpgen

import (
	"fmt"
)

type Extractor struct {
	tokens    []Token
	current   int
	variables map[string][]string // 用來儲存變數及其對應的數值
}

// 初始化
func NewExtractor(tokens []Token) *Extractor {
	return &Extractor{
		tokens:    tokens,
		current:   0,
		variables: make(map[string][]string), // 初始化變數 map
	}
}

// 獲取當前 token
func (e *Extractor) currentToken() Token {
	if e.current < len(e.tokens) {
		return e.tokens[e.current]
	}
	return Token{Type: "EOF"} // 空 token 表示結束
}

// 前進至下一個 token
func (e *Extractor) nextToken() {
	if e.current < len(e.tokens)-1 {
		e.current++
	}
}

// 提取變數
func (e *Extractor) ExtractVariables() {
	var currentVariable string
	for e.current < len(e.tokens) {
		token := e.currentToken()

		switch token.Type {
		case "VARIABLE":
			currentVariable = token.Value
			e.variables[currentVariable] = []string{} // 初始化切片
			fmt.Printf("Found variable: %s\n", currentVariable)

		case "NUMBER":
			if currentVariable != "" {
				fmt.Printf("Adding number %s to variable %s\n", token.Value, currentVariable)
				e.variables[currentVariable] = append(e.variables[currentVariable], token.Value)
			}

		case "SEPARATOR":
			if token.Value == ";" {
				currentVariable = "" // 碰到分號，變數結束
				fmt.Println("End of statement.")
			}

		case "KEYWORD":
			// 遇到結束的關鍵字，直接跳過剩餘處理，並跳過多餘的 ENDSETS
			if token.Value == "ENDSETS" || token.Value == "ENDDATA" {
				fmt.Printf("Skipping keyword: %s\n", token.Value)
				for e.currentToken().Value == token.Value {
					e.nextToken()
				}

			}

		default:
			// 跳過無法處理的 token
			fmt.Printf("Skipping token: Type=%s, Value=%s\n", token.Type, token.Value)
		}

		e.nextToken()
	}
}
