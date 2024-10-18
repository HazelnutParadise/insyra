package lpgen

import (
	"strconv"
	"strings"
)

// 索引字母 I, J, K, L, M, N
var indexLetters = []string{"I", "J", "K", "L", "M", "N"}

type lingoProcessResult struct {
	Tokens []lingoToken
	Funcs  map[string][]lingoToken
}

func LingoProcessor(extractResult *lingoExtractResult) *lingoProcessResult {
	result := &lingoProcessResult{
		Tokens: extractResult.Tokens,
	}

	result.Funcs = lingoProcessFuncs(extractResult)

	return result
}

func lingoProcessFuncs(extractResult *lingoExtractResult) map[string][]lingoToken {
	funcs := make(map[string][]lingoToken)
	// TODO: 重複直到token.Type沒有FUNC

	for funcName, funcTokens := range extractResult.Funcs {
		if strings.HasPrefix(funcName, "$SIZE") {
			funcs[funcName] = lingoProcessFunc_SIZE(funcTokens, extractResult)
			// TODO: 處理 SIZE 函數
		} else if strings.HasPrefix(funcName, "$SUM") {
			// TODO: 處理 SUM 函數
		} else if strings.HasPrefix(funcName, "$FOR") {
			// TODO: 處理 FOR 函數
		} else if strings.HasPrefix(funcName, "$POW") {
			// TODO: 處理 POW 函數
		} else if strings.HasPrefix(funcName, "$BIN") {
			// TODO: 處理 BIN 函數
		}
		// TODO: 其他函數
	}

	return funcs
}

func lingoProcessFunc_SIZE(funcTokens []lingoToken, extractResult *lingoExtractResult) []lingoToken {
	var variableName string
	for _, token := range funcTokens {
		if token.Type == "VARIABLE" {
			variableName = token.Value
		}
	}
	// 去Sets找
	set := extractResult.Sets[variableName]

	return []lingoToken{
		{
			Type:  "NUMBER",
			Value: strconv.Itoa(len(set.Index) * len(set.Values)),
		},
	}
}

func lingoProcessFunc_SUM(funcTokens []lingoToken, extractResult *lingoExtractResult) []lingoToken {
	// TODO
	return nil
}

func lingoProcessFunc_FOR(funcTokens []lingoToken, extractResult *lingoExtractResult) []lingoToken {
	// TODO
	return nil
}

func lingoProcessFunc_POW(funcTokens []lingoToken, extractResult *lingoExtractResult) []lingoToken {
	// TODO
	return nil
}

func lingoProcessFunc_BIN(funcTokens []lingoToken, extractResult *lingoExtractResult) []lingoToken {
	// TODO
	return nil
}
