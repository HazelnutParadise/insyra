package lpgen

type ExtractResult struct {
	tokens    []Token
	Variables map[string][]string // 用來儲存變數及其對應的數值
	Data      map[string][]string // 用來儲存數據
}

func LingoExtractor(tokens []Token) *ExtractResult {
	result := &ExtractResult{
		tokens:    tokens,
		Variables: make(map[string][]string),
		Data:      make(map[string][]string),
	}
	result = lingoExtractData(result)
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
