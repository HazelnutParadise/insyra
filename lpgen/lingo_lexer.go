package main

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
		"KEYWORD":   `(@SUM|@FOR|@BIN|@POW|SETS|DATA|ENDSETS|ENDDATA|MIN|MAX|RHS)`, // 支持@開頭的語法
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

func main() {
	lingoText := `
	SETS:
	group /1..5/: group_size, vaccine_coverage, next_generation, eignvector_subgroup;
	ENDSETS

	DATA: 
	group_size = 77 241 375 204 103;
	vaccine_coverage = 0.1 0.2 0.3 0.4 0.5;
	next_generation = 1.1 1.2 2.1 2.2 2.3;
	ENDDATA

	@SUM(group(I): group_size(I) * vaccine_coverage(I));
	@FOR(group(I): @SUM(group(J): next_generation(I,J) * eignvector_subgroup(J)) <= 1);
	@BIN(x(I,K));
	@POW(2,(-1)*K);
	`

	// 使用 Lexer 解析 Lingo 代碼
	tokens := Lexer(lingoText)
	for _, token := range tokens {
		fmt.Printf("Token Type: %s, Value: %s\n", token.Type, token.Value)
	}
}
