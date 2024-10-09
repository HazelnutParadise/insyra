package lpgen

import (
	"strings"
)

type ExtractResult struct {
	tokens    []lingoToken
	Obj       map[string]string   // 用來儲存目標函數
	Variables map[string][]string // 用來儲存變數及其對應的數值
	Data      map[string][]string // 用來儲存數據
}

func LingoExtractor(tokens []lingoToken) *ExtractResult {
	result := &ExtractResult{
		tokens:    tokens,
		Obj:       make(map[string]string),
		Variables: make(map[string][]string),
		Data:      make(map[string][]string),
	}
	result = lingoExtractObj(result)
	result = lingoExtractData(result)
	return result
}

func lingoExtractObj(result *ExtractResult) *ExtractResult {
	extractObj := false
	objType := ""
	for i, token := range result.tokens {
		if token.Type == "KEYWORD" && strings.ToUpper(token.Value) == "MODEL" {
			// 如果遇到MODEL，則開始提取目標函數
			extractObj = true
			// 前往下一個token
			continue
		} else if token.Type == "KEYWORD" && i != 1 {
			// 如果遇到其他關鍵字，則停止提取目標函數
			extractObj = false
		} else if token.Type == "OPERATOR" && i == 2 {
			continue
		}

		if extractObj {
			if token.Type == "KEYWORD" {
				objType = token.Value
			} else if token.Type != "SEPARATOR" {
				result.Obj[objType] += token.Value
			}
		}
	}
	return result
}

func lingoExtractData(result *ExtractResult) *ExtractResult {
	extractData := false
	extractingVariableName := "" // 目前正在提取的變數名稱
	for _, token := range result.tokens {
		// 如果遇到DATA，則開始提取數據
		if token.Type == "KEYWORD" && token.Value == "DATA" {
			extractData = true
		} else if token.Type == "KEYWORD" && token.Value == "ENDDATA" {
			// 如果遇到ENDDATA，則停止提取數據
			extractData = false
		}

		if extractData {
			switch token.Type {
			case "VARIABLE":
				extractingVariableName = token.Value
			case "NUMBER":
				result.Data[extractingVariableName] = append(result.Data[extractingVariableName], token.Value)
			}
		}
	}
	return result
}
