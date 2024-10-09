package lpgen

import (
	"fmt"
	"strings"
)

// 定義轉換器
type Converter struct {
	tokens  []Token
	current int
}

// 創建新的轉換器
func NewConverter(tokens []Token) *Converter {
	return &Converter{
		tokens:  tokens,
		current: 0,
	}
}

// 獲取當前 token
func (c *Converter) currentToken() Token {
	if c.current < len(c.tokens) {
		return c.tokens[c.current]
	}
	return Token{} // 空 token 表示結束
}

// 前進到下一個 token
func (c *Converter) nextToken() Token {
	if c.current < len(c.tokens)-1 {
		c.current++
	}
	return c.currentToken()
}

// 匹配指定的 token 類型和值
func (c *Converter) match(tokenType string, value string) bool {
	current := c.currentToken()
	if current.Type == tokenType && (value == "" || current.Value == value) {
		c.nextToken()
		return true
	}
	return false
}

// 處理 @SUM 關鍵字
func (c *Converter) lingoSum() string {
	result := "@SUM("
	c.nextToken() // 跳過 @SUM

	// 處理內部表達式
	result += c.parseExpression()

	if c.match("SEPARATOR", ")") {
		result += ")"
	}
	return result
}

// 處理 @FOR 關鍵字
func (c *Converter) lingoFor() string {
	result := "@FOR("
	c.nextToken() // 跳過 @FOR

	// 處理內部表達式
	result += c.parseExpression()

	if c.match("SEPARATOR", ")") {
		result += ")"
	}
	return result
}

// 處理 @BIN 關鍵字
func (c *Converter) lingoBin() string {
	result := "@BIN("
	c.nextToken() // 跳過 @BIN

	// 處理內部表達式
	result += c.parseExpression()

	if c.match("SEPARATOR", ")") {
		result += ")"
	}
	return result
}

// 處理 @POW 關鍵字
func (c *Converter) lingoPow() string {
	result := "@POW("
	c.nextToken() // 跳過 @POW

	// 處理內部表達式
	result += c.parseExpression()

	if c.match("SEPARATOR", ")") {
		result += ")"
	}
	return result
}

// 解析表達式
func (c *Converter) parseExpression() string {
	var expr strings.Builder

	for c.current < len(c.tokens) {
		token := c.currentToken()
		switch token.Type {
		case "VARIABLE", "NUMBER", "OPERATOR":
			// 變數、數字和操作符直接加入表達式
			expr.WriteString(token.Value)
		case "SEPARATOR":
			if token.Value == "(" || token.Value == ")" {
				// 處理括號
				expr.WriteString(token.Value)
			} else if token.Value == ";" {
				// 到達分號時結束
				return expr.String()
			}
		case "KEYWORD":
			// 根據不同的關鍵字進行對應的處理
			switch token.Value {
			case "@SUM":
				expr.WriteString(c.lingoSum())
			case "@FOR":
				expr.WriteString(c.lingoFor())
			case "@BIN":
				expr.WriteString(c.lingoBin())
			case "@POW":
				expr.WriteString(c.lingoPow())
			}
		default:
			fmt.Printf("Skipping token: Type=%s, Value=%s\n", token.Type, token.Value)
		}
		c.nextToken()
	}

	return expr.String()
}

// 轉換整個 token 列表為 LP 語法
func (c *Converter) Convert() string {
	var result strings.Builder

	for c.current < len(c.tokens) {
		token := c.currentToken()

		switch token.Type {
		case "KEYWORD":
			// 處理關鍵字，匹配並展開相應語法
			switch token.Value {
			case "@SUM":
				result.WriteString(c.lingoSum())
			case "@FOR":
				result.WriteString(c.lingoFor())
			case "@BIN":
				result.WriteString(c.lingoBin())
			case "@POW":
				result.WriteString(c.lingoPow())
			default:
				result.WriteString(token.Value + " ")
			}
		case "VARIABLE", "NUMBER", "OPERATOR":
			// 直接加入變數、數字和操作符
			result.WriteString(token.Value + " ")
		case "SEPARATOR":
			// 加入分號或括號
			result.WriteString(token.Value + " ")
		default:
			fmt.Printf("Skipping token: Type=%s, Value=%s\n", token.Type, token.Value)
		}

		c.nextToken()
	}

	return result.String()
}
