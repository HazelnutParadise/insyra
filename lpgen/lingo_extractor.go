package lpgen

import (
	"strings"
	"unicode"

	"github.com/HazelnutParadise/Go-Utils/conv"
)

// 索引字母 I, J, K, L, M, N

type ExtractResult struct {
	tokens    []lingoToken
	Obj       map[string]string   // 用來儲存目標函數
	Variables map[string]string   // 用來儲存變數及其對應的數值
	Data      map[string][]string // 用來儲存數據
	Sets      map[string]Set      // 用來儲存集合及其對應的數值
}

type Set struct {
	Index  []string
	Values []string
}

func LingoExtractor(tokens []lingoToken) *ExtractResult {
	result := &ExtractResult{
		tokens:    tokens,
		Obj:       make(map[string]string),
		Variables: make(map[string]string),
		Data:      make(map[string][]string),
		Sets:      make(map[string]Set),
	}
	result = lingoExtractData(result)
	result = lingoExtractVariablesPureNumbers(result)
	result = lingoExtractSetsNoFuncOrIndex(result)
	result = lingoExtractObj(result)

	return result
}

func lingoExtractObj(result *ExtractResult) *ExtractResult {
	extractObj := false
	objType := ""
	for i, token := range result.tokens {
		upperTokenValue := strings.ToUpper(token.Value)
		if token.Type == "KEYWORD" && upperTokenValue == "MODEL" {
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
		upperTokenValue := strings.ToUpper(token.Value)
		// 如果遇到DATA，則開始提取數據
		if token.Type == "KEYWORD" && upperTokenValue == "DATA" {
			extractData = true
		} else if token.Type == "KEYWORD" && upperTokenValue == "ENDDATA" {
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

func lingoExtractVariablesPureNumbers(result *ExtractResult) *ExtractResult {
	extractVariables := true
	extractingVariableName := ""
	for _, token := range result.tokens {
		upperTokenValue := strings.ToUpper(token.Value)
		if token.Type == "KEYWORD" && (upperTokenValue == "SETS" || upperTokenValue == "DATA") {
			extractVariables = false
		} else if token.Type == "KEYWORD" && (upperTokenValue == "ENDSETS" || upperTokenValue == "ENDDATA") {
			extractVariables = true
		}

		if extractVariables {
			if token.Type == "VARIABLE" {
				extractingVariableName = token.Value
			} else if token.Type == "NUMBER" || token.Type == "OPERATOR" {
				if extractingVariableName != "" && upperTokenValue != "=" {
					result.Variables[extractingVariableName] += token.Value
				}
			}
			if token.Type == "SEPARATOR" {
				extractingVariableName = ""
			}
		}
	}

	// 清理變數值尾端的任何非數字字元
	for variable, value := range result.Variables {
		result.Variables[variable] = strings.TrimFunc(value, func(r rune) bool {
			return !unicode.IsDigit(r)
		})
	}
	return result
}

// TODO
func lingoExtractSetsNoFuncOrIndex(result *ExtractResult) *ExtractResult {
	extractSets := false
	extractingSetName := ""
	extractingSetInsideVariables := false
	for i, token := range result.tokens {
		if i+1 >= len(result.tokens) {
			break
		}
		nextToken := result.tokens[i+1]
		var prevToken lingoToken
		if i-1 < 0 {
			prevToken = token
		} else {
			prevToken = result.tokens[i-1]
		}
		if token.Type == "KEYWORD" && token.Value == "SETS" {
			extractSets = true
		} else if token.Type == "KEYWORD" && token.Value == "ENDSETS" {
			extractSets = false
		}

		if extractSets {
			if token.Type == "VARIABLE" && (nextToken.Type == "OPERATOR" && nextToken.Value == "/") {
				extractingSetName = token.Value
			} else if token.Type == "OPERATOR" && token.Value == "/" {
				if nextToken.Type == "NUMBER" {
					// 取得集合內屬性數量起點
					extractingSetStart := conv.ParseInt(nextToken.Value)
					// 取得集合內屬性數量終點
					extractingSetEnd := conv.ParseInt(result.tokens[i+2].Value)
					// 設定集合內屬性數量
					for j := extractingSetStart; j <= extractingSetEnd; j++ {
						set := Set{
							Index: append(result.Sets[extractingSetName].Index, conv.ToString(j)),
						}
						result.Sets[extractingSetName] = set
					}
				}
			} else if token.Type == "VARIABLE" {
				// 取得集合內屬性
				if prevToken.Type == "OPERATOR" && prevToken.Value == "/" {
					// 開始取得集合內屬性
					extractingSetInsideVariables = true
				}
				if extractingSetInsideVariables {
					set := result.Sets[extractingSetName]
					set.Values = append(set.Values, token.Value)
					result.Sets[extractingSetName] = set
				}
			} else if token.Type == "SEPARATOR" {
				// 結束取得集合內屬性
				extractingSetInsideVariables = false
			}

		}
	}
	return result
}
