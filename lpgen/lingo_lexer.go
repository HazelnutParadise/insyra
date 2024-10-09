package lpgen

import (
	"fmt"
	"regexp"
	"strings"
)

// Token 定義
type Token struct {
	Type  string
	Value string
}

// 定義 Lexer 函數
func Lexer(lingoText string) []Token {
	tokens := []Token{}
	tokenPatterns := map[string]string{
		"KEYWORD":   `(@SUM|@FOR|@POW|@BIN|@LOG|@ABS|@SIN|@COS|@EXP|SETS|ENDSETS|DATA|ENDDATA|MIN|MAX|RHS|IF|THEN|ELSE|ENDIF)`,
		"VARIABLE":  `\b[a-zA-Z_]\w*\b`,
		"NUMBER":    `\b\d+(\.\d+)?\b`,
		"OPERATOR":  `[+\-*/=<>]`,
		"SEPARATOR": `[();]`,
	}

	// 合併所有正則表達式
	var combinedPattern strings.Builder
	for key, pattern := range tokenPatterns {
		combinedPattern.WriteString(fmt.Sprintf("(?P<%s>%s)|", key, pattern))
	}
	combinedPatternStr := strings.TrimSuffix(combinedPattern.String(), "|")

	// 匹配 Token
	re := regexp.MustCompile(combinedPatternStr)
	matches := re.FindAllStringSubmatch(lingoText, -1)

	// 生成 Token 列表
	for _, match := range matches {
		for i, group := range match {
			if group != "" {
				for name, _ := range tokenPatterns {
					if re.SubexpNames()[i] == name {
						tokens = append(tokens, Token{Type: name, Value: group})
					}
				}
			}
		}
	}

	return tokens
}
