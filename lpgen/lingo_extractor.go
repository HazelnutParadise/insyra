package lpgen

import (
	"strings"
	"unicode"

	"github.com/HazelnutParadise/Go-Utils/conv"
	"github.com/HazelnutParadise/Go-Utils/sliceutil"
)

// 索引字母 I, J, K, L, M, N

type ExtractResult struct {
	Tokens      []lingoToken
	Obj         map[string]string       // 用來儲存目標函數
	Variables   map[string]string       // 用來儲存變數及其對應的數值
	Data        map[string][]string     // 用來儲存數據
	Sets        map[string]lingoSet     // 用來儲存集合及其對應的數值
	Funcs       map[string][]lingoToken // 用來儲存函數代號及其對應的式
	nextFuncNum int
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

func LingoExtractor(Tokens *[]lingoToken) *ExtractResult {
	result := &ExtractResult{
		Tokens:      *Tokens,
		Obj:         make(map[string]string),
		Variables:   make(map[string]string),
		Data:        make(map[string][]string),
		Sets:        make(map[string]lingoSet),
		Funcs:       make(map[string][]lingoToken),
		nextFuncNum: 0,
	}
	result = lingoExtractData(result)
	result = lingoExtractVariablesPureNumbers(result)
	result = lingoExtractSetsOneDimension(result)
	result = lingoExtractObj(result)
	result = lingoProcessNestedParentheses(result)
	result = lingoProcessParenthesesInFuncs(result)

	// TODO: 處理多維度Sets
	// TODO: 處理有索引的變數

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

// 擷取tokens內的括號
func lingoProcessNestedParentheses(result *ExtractResult) *ExtractResult {
	var stopExtract bool = false
	for !stopExtract {
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

					// 檢查是否為函數調用
					if 左括號索引 > 0 {
						// 由於lexer有小bug，有些@不會被當成KEYWORD，所以這邊不加入KEYWORD的判斷
						if _, exists := lingoFuncCode[strings.ToUpper(result.Tokens[左括號索引-1].Value)]; exists {
							// 處理函數代號及其括號內的內容
							result, 函數代號tokens := lingoExtractFuncs(result, 左括號索引-1, i)

							// 更新 token 並將結果寫回
							if 函數代號tokens != nil {
								result.Tokens, _ = sliceutil.ReplaceWithSlice(result.Tokens, 左括號索引-1, i, 函數代號tokens)

								// 調整索引，以應對已經替換的 token
								i = 左括號索引 - 1
							}
						}
					}
				}
			}
		}

		var hasFunc bool = false
		for _, token := range result.Tokens {
			// 由於lexer有小bug，有些@不會被當成KEYWORD，所以這邊不加入KEYWORD的判斷
			if strings.Contains(strings.ToUpper(token.Value), "@") {
				hasFunc = true
				break
			}
		}
		if !hasFunc {
			stopExtract = true
		}
	}
	return result
}

func lingoExtractFuncs(result *ExtractResult, funcStartIndex int, funEndIndex int) (*ExtractResult, []lingoToken) {
	函數代號 := ""
	if code, exists := lingoFuncCode[strings.ToUpper(result.Tokens[funcStartIndex].Value)]; exists {
		函數代號 = code + conv.ToString(result.nextFuncNum)
		result.nextFuncNum++
	} else {
		return nil, nil
	}

	// 將括號內的所有內容放入對應的函數
	for i := funcStartIndex + 1; i <= funEndIndex; i++ {
		result.Funcs[函數代號] = append(result.Funcs[函數代號], result.Tokens[i])
	}

	// 返回新構造的函數 token
	return result, []lingoToken{{Type: "FUNC", Value: 函數代號}}
}

// 須在Funcs內再遞迴處理一次括號
func lingoProcessParenthesesInFuncs(result *ExtractResult) *ExtractResult {
	var stopExtract = false
	for !stopExtract {
		leftParenthesesStack := []int{}
		leftParenthesesIndex := -1
		rightParenthesesIndex := -1
		for key, funcTokens := range result.Funcs {
			for i, funcToken := range funcTokens {
				if funcToken.Type == "SEPARATOR" && funcToken.Value == "(" {
					leftParenthesesStack = append(leftParenthesesStack, i)
				} else if funcToken.Type == "SEPARATOR" && funcToken.Value == ")" && len(leftParenthesesStack) > 0 {
					leftParenthesesStack = leftParenthesesStack[:len(leftParenthesesStack)-1]
					rightParenthesesIndex = i
				}
				if leftParenthesesIndex > 0 {
					// 由於lexer有小bug，有些@不會被當成KEYWORD，所以這邊不加入KEYWORD的判斷
					if code, exists := lingoFuncCode[strings.ToUpper(funcTokens[leftParenthesesIndex-1].Value)]; exists {
						// 處理函數代號及其括號內的內容
						funcCode := code + conv.ToString(result.nextFuncNum)
						result.nextFuncNum++
						// 將括號內的所有內容放入對應的函數
						for i := leftParenthesesIndex + 1; i <= rightParenthesesIndex; i++ {
							result.Funcs[funcCode] = append(result.Funcs[funcCode], funcTokens[i])
						}
						// 將原本的函數代號及其括號內的內容刪除
						result.Funcs[key], _ = sliceutil.ReplaceWithSlice(funcTokens, leftParenthesesIndex, rightParenthesesIndex, []lingoToken{{Type: "FUNC", Value: funcCode}})
						// 調整索引，以應對已經替換的 token

					}
				}
			}
		}
		var hasFunc bool = false
		for _, funcTokens := range result.Funcs {
			for _, funcToken := range funcTokens {
				// 由於lexer有小bug，有些@不會被當成KEYWORD，所以這邊不加入KEYWORD的判斷
				if strings.Contains(strings.ToUpper(funcToken.Value), "@") {
					hasFunc = true
					break
				}
			}
		}
		if !hasFunc {
			stopExtract = true
		}
	}
	return result
}
