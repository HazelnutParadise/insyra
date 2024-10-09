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
	originalText := lingoText               // 保留原始字串
	upperText := strings.ToUpper(lingoText) // 全大寫字串用於匹配
	tokens := []Token{}
	tokenPatterns := map[string]string{
		"KEYWORD":   `(@SUM|@FOR|@POW|@BIN|@LOG|@ABS|@SIN|@COS|@EXP|MODEL|SETS|ENDSETS|DATA|ENDDATA|MIN|MINIMIZE|MAX|MAXIMIZE|RHS|IF|THEN|ELSE|ENDIF)`,
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

	// 正則表達式用於全大寫文本進行匹配
	re := regexp.MustCompile(combinedPatternStr)
	matches := re.FindAllStringIndex(upperText, -1) // 基於全大寫文本找到每個匹配的位置

	// 生成 Token 列表，但保留原始大小寫文本的值
	for _, match := range matches {
		start, end := match[0], match[1]
		// 取得對應於全大寫文本的匹配片段
		matchedText := upperText[start:end]

		// 找出是哪種類型的匹配
		for tokenType, pattern := range tokenPatterns {
			if matched, _ := regexp.MatchString(pattern, matchedText); matched {
				// 使用原始文本對應部分的子串作為值，保留大小寫
				originalValue := originalText[start:end]
				tokens = append(tokens, Token{Type: tokenType, Value: originalValue})
				break
			}
		}
	}

	return tokens
}
