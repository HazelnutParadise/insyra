package lpgen

import (
	"strings"
)

// 索引字母 I, J, K, L, M, N

type lingoProcessResult struct {
	Tokens []lingoToken
	Funcs  map[string][]lingoToken
}

func LingoProcessor(extractResult *lingoExtractResult) *lingoProcessResult {
	result := &lingoProcessResult{
		Tokens: extractResult.Tokens,
	}

	lingoProcessFuncs(extractResult)

	return result
}

func lingoProcessFuncs(extractResult *lingoExtractResult) map[string][]lingoToken {
	funcs := make(map[string][]lingoToken)

	for funcName, funcTokens := range extractResult.Funcs {
		if strings.HasPrefix(funcName, "$SIZE") {
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
