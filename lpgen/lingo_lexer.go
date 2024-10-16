package lpgen

import (
	"fmt"
	"regexp"
	"strings"
)

// Token 定義
type lingoToken struct {
	Type  string
	Value string
}

// 定義 Lexer 函數
func LingoLexer(lingoText string) []lingoToken {
	// 移除 Lingo 註釋
	lingoText = removeLingoComments(lingoText)

	originalText := lingoText               // 保留原始字串
	upperText := strings.ToUpper(lingoText) // 全大寫字串用於匹配
	tokens := []lingoToken{}
	tokenPatterns := map[string]string{
		"KEYWORD":   `@SIZE|@SUM|@FOR|@POW|@BIN|@LOG|@ABS|@SIN|@COS|@EXP|MODEL|SETS|ENDSETS|DATA|ENDDATA|MIN|MINIMIZE|MAX|MAXIMIZE|RHS|IF|THEN|ELSE|ENDIF`,
		"VARIABLE":  `[a-zA-Z_]\w*`,
		"NUMBER":    `\b\d+(\.\d+)?\b`,
		"OPERATOR":  `[+\-*/=<>]`,
		"SEPARATOR": `[();:,]`,
	}

	// 按順序合併所有正則表達式
	var combinedPattern strings.Builder
	keys := []string{"KEYWORD", "VARIABLE", "NUMBER", "OPERATOR", "SEPARATOR"}
	for _, key := range keys {
		pattern := tokenPatterns[key]
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

		// 先檢查是否為關鍵字
		if tokenType, exists := checkKeyword(matchedText); exists {
			originalValue := originalText[start:end]
			tokens = append(tokens, lingoToken{Type: tokenType, Value: originalValue})
			continue
		}

		// 找出是哪種類型的匹配
		for tokenType, pattern := range tokenPatterns {
			if matched, _ := regexp.MatchString(pattern, matchedText); matched {
				// 使用原始文本對應部分的子串作為值，保留大小寫
				originalValue := originalText[start:end]
				tokens = append(tokens, lingoToken{Type: tokenType, Value: originalValue})
				break
			}
		}
	}

	return tokens
}

// 檢查是否為關鍵字
func checkKeyword(text string) (string, bool) {
	keywords := []string{"@SIZE", "@SUM", "@FOR", "@POW", "@BIN", "@LOG", "@ABS", "@SIN", "@COS", "@EXP", "MODEL", "SETS", "ENDSETS", "DATA", "ENDDATA", "MIN", "MINIMIZE", "MAX", "MAXIMIZE", "RHS", "IF", "THEN", "ELSE", "ENDIF"}
	for _, keyword := range keywords {
		if text == keyword {
			return "KEYWORD", true
		}
	}
	return "", false
}

func removeLingoComments(text string) string {
	// 移除單行註釋
	singleLineCommentRegex := regexp.MustCompile(`!.*`)
	text = singleLineCommentRegex.ReplaceAllString(text, "")

	// 移除多行註釋
	multiLineCommentRegex := regexp.MustCompile(`(?s)/\*.*?\*/`)
	text = multiLineCommentRegex.ReplaceAllString(text, "")

	return strings.TrimSpace(text)
}
