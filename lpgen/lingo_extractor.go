package lpgen

import (
	"strings"
	"unicode"

	"github.com/HazelnutParadise/Go-Utils/conv"
	"github.com/HazelnutParadise/Go-Utils/sliceutil"
)

// 索引字母 I, J, K, L, M, N

type ExtractResult struct {
	Tokens    []lingoToken
	Obj       map[string]string   // 用來儲存目標函數
	Variables map[string]string   // 用來儲存變數及其對應的數值
	Data      map[string][]string // 用來儲存數據
	Sets      map[string]lingoSet // 用來儲存集合及其對應的數值
	Funcs     map[string][]string // 用來儲存函數代號及其對應的式
	funcCount int
}

type lingoSet struct {
	Index  []string
	Values []string
}

var lingoFuncCode = map[string]string{
	"@FOR":  "$FOR",
	"@SUM":  "$SUM",
	"@POW":  "$POW",
	"@BIN":  "$BIN",
	"@LOG":  "$LOG",
	"@ABS":  "$ABS",
	"@SIN":  "$SIN",
	"@COS":  "$COS",
	"@EXP":  "$EXP",
	"@SIZE": "$SIZE",
}

func LingoExtractor(Tokens []lingoToken) *ExtractResult {
	result := &ExtractResult{
		Tokens:    Tokens,
		Obj:       make(map[string]string),
		Variables: make(map[string]string),
		Data:      make(map[string][]string),
		Sets:      make(map[string]lingoSet),
		Funcs:     make(map[string][]string),
		funcCount: 0,
	}
	result = lingoExtractData(result)
	result = lingoExtractVariablesPureNumbers(result)
	result = lingoExtractSetsOneDimension(result)
	result = lingoExtractObj(result)
	// result = lingoExtractFuncsOutermost(result)
	result = lingoProcessNestedParentheses(result)

	return result
}

func lingoExtractObj(result *ExtractResult) *ExtractResult {
	extractObj := false
	objType := ""
	for i, token := range result.Tokens {
		upperTokenValue := strings.ToUpper(token.Value)
		if token.Type == "KEYWORD" && upperTokenValue == "MODEL" {
			// 如果遇到MODEL，則開始提取目標函數
			extractObj = true
			// 前往下一個token
			continue
		} else if token.Type == "KEYWORD" && i != 2 {
			// 如果遇到其他關鍵字，則停止提取目標函數
			extractObj = false
		} else if token.Type == "OPERATOR" && i == 3 {
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
	for _, token := range result.Tokens {
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
	for i, token := range result.Tokens {
		upperTokenValue := strings.ToUpper(token.Value)
		if i+2 >= len(result.Tokens) {

			break
		}
		nextToken := result.Tokens[i+1]
		if token.Type == "KEYWORD" && (upperTokenValue == "SETS" || upperTokenValue == "DATA") {
			extractVariables = false
		} else if token.Type == "KEYWORD" && (upperTokenValue == "ENDSETS" || upperTokenValue == "ENDDATA") {
			extractVariables = true
		}

		if extractVariables {
			if token.Type == "VARIABLE" && (nextToken.Type == "OPERATOR" && nextToken.Value == "=") && (result.Tokens[i+2].Type == "NUMBER") {
				extractingVariableName = token.Value
			} else if token.Type == "NUMBER" || (nextToken.Type == "SEPARATOR" && nextToken.Value == ";") {
				if extractingVariableName != "" && upperTokenValue != "=" {
					result.Variables[extractingVariableName] += token.Value
				}
			}
			if token.Type == "SEPARATOR" && token.Value == ";" {
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

func lingoExtractSetsOneDimension(result *ExtractResult) *ExtractResult {
	extractSets := false
	extractingSetName := ""
	extractingSetInsideVariables := false
	for i, token := range result.Tokens {
		if i+1 >= len(result.Tokens) {
			break
		}
		nextToken := result.Tokens[i+1]
		var prevToken lingoToken
		if i-1 < 0 {
			prevToken = token
		} else {
			prevToken = result.Tokens[i-1]
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
					extractingSetEnd := conv.ParseInt(result.Tokens[i+2].Value)
					// 設定集合內屬性數量
					for j := extractingSetStart; j <= extractingSetEnd; j++ {
						set := lingoSet{
							Index: append(result.Sets[extractingSetName].Index, conv.ToString(j)),
						}
						result.Sets[extractingSetName] = set
					}
				}
			} else if token.Type == "VARIABLE" {
				// 取得集合內屬性
				if prevToken.Type == "SEPARATOR" && prevToken.Value == ":" {
					// 開始取得集合內屬性
					extractingSetInsideVariables = true
				}
				if extractingSetInsideVariables && extractingSetName != "" {
					set := result.Sets[extractingSetName]
					set.Values = append(set.Values, token.Value)
					result.Sets[extractingSetName] = set
				}
			}
			if token.Type == "SEPARATOR" && token.Value == ";" {
				// 結束取得集合內屬性
				extractingSetInsideVariables = false
				extractingSetName = ""
			}

		}
	}
	return result
}

// WARNING: 無法處理某些函數
func lingoExtractFuncsOutermost(result *ExtractResult) *ExtractResult {
	extractingFuncName := ""

	for i, token := range result.Tokens {
		if token.Type == "KEYWORD" {
			if code, exists := lingoFuncCode[strings.ToUpper(token.Value)]; exists {
				codeWithNumber := code + conv.ToString(result.funcCount)
				extractingFuncName = codeWithNumber

				// 將 token 的值轉換為 codeWithNumber
				token.Value = codeWithNumber
				result.Tokens[i] = token

				// 增加函數編號
				result.funcCount++
			}
		}
		if extractingFuncName != "" {
			if token.Type == "SEPARATOR" && token.Value == ";" {
				extractingFuncName = ""
				continue
			}
			if token.Type != "KEYWORD" {
				result.Funcs[extractingFuncName] = append(result.Funcs[extractingFuncName], token.Value)
			}
		}
	}
	return result
}

// TODO
func lingoProcessNestedParentheses(result *ExtractResult) *ExtractResult {
	左括號堆疊 := []int{} // 堆疊來儲存左括號的索引
	for i := 0; i < len(result.Tokens); i++ {
		if result.Tokens[i].Type == "SEPARATOR" {
			if result.Tokens[i].Value == "(" {
				// 將左括號的位置推入堆疊
				左括號堆疊 = append(左括號堆疊, i)
			} else if result.Tokens[i].Value == ")" && len(左括號堆疊) > 0 {
				// 取出最近的一個左括號索引
				左括號索引 := 左括號堆疊[len(左括號堆疊)-1]
				// 從堆疊中移除
				左括號堆疊 = 左括號堆疊[:len(左括號堆疊)-1]

				// 處理括號內的所有內容，包括函數
				updatedTokens, startIndex, endIndex := processParenthesesContents(result, 左括號索引, i)

				// 如果成功提取到內容，進行替換
				if updatedTokens != nil {
					// 替換原來的括號內容
					result.Tokens, _ = sliceutil.ReplaceWithSlice(result.Tokens, startIndex, endIndex, updatedTokens)
					// 調整索引，因為我們已經替換了部分 tokens
					i = startIndex + len(updatedTokens) - 1
				}
			}
		}
	}
	return result
}

func processParenthesesContents(result *ExtractResult, 左括號索引 int, 右括號索引 int) ([]lingoToken, int, int) {
	內容tokens := []lingoToken{}
	startIndex := 左括號索引
	endIndex := 右括號索引

	// 遍歷括號內的所有 token，識別函數並處理
	for i := 左括號索引 + 1; i < 右括號索引; i++ {
		token := result.Tokens[i]
		if token.Type == "KEYWORD" {
			// 如果是函數，呼叫 lingoExtractFuncs 來提取函數
			if _, exists := lingoFuncCode[strings.ToUpper(token.Value)]; exists {
				extractedResult, newTokens := lingoExtractFuncs(result, i, 右括號索引)
				result = extractedResult

				// 更新函數代號的 tokens
				內容tokens = append(內容tokens, newTokens...)
			} else {
				// 非函數關鍵字，保持原樣
				內容tokens = append(內容tokens, token)
			}
		} else {
			// 其他非函數的 token 保持不變
			內容tokens = append(內容tokens, token)
		}
	}

	return 內容tokens, startIndex, endIndex
}

// func processNonFuncContent(result *ExtractResult, 左括號索引 int, 右括號索引 int) *ExtractResult {
// 	// 遍歷括號內的 token，保持原內容不變
// 	nonFuncContent := []lingoToken{}
// 	for i := 左括號索引 + 1; i < 右括號索引; i++ {
// 		nonFuncContent = append(nonFuncContent, result.Tokens[i])
// 	}
// 	// 如果需要，這裡可以加入對於非函數表達式的額外處理邏輯
// 	// 例如數學運算符號、變數操作等
// 	return result
// }

func lingoExtractFuncs(result *ExtractResult, funcStartIndex int, funEndIndex int) (*ExtractResult, []lingoToken) {
	函數代號 := ""
	if code, exists := lingoFuncCode[strings.ToUpper(result.Tokens[funcStartIndex].Value)]; exists {
		函數代號 = code + conv.ToString(result.funcCount)
		result.funcCount++
	} else {
		return nil, nil
	}

	// 將括號內的所有內容放入對應的函數
	for i := funcStartIndex + 1; i <= funEndIndex; i++ {
		result.Funcs[函數代號] = append(result.Funcs[函數代號], result.Tokens[i].Value)
	}

	// 返回新構造的函數 token
	return result, []lingoToken{{Type: "FUNC", Value: 函數代號}}
}
