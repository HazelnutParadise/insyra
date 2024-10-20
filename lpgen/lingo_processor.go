package lpgen

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/HazelnutParadise/Go-Utils/conv"
	"github.com/HazelnutParadise/Go-Utils/sliceutil"
)

// 索引字母 I, J, K, L, M, N
var indexLetters = []string{"I", "J", "K", "L", "M", "N"}

type lingoProcessResult struct {
	Tokens []lingoToken
	Funcs  map[string][][]lingoToken
}

func LingoProcessor(extractResult *lingoExtractResult) *lingoProcessResult {
	result := &lingoProcessResult{
		Tokens: extractResult.Tokens,
		Funcs:  make(map[string][][]lingoToken),
	}
	var err error
	result.Funcs, err = lingoProcessFuncs(extractResult)
	if err != nil {
		panic(err)
	}

	return result
}

func lingoProcessFuncs(extractResult *lingoExtractResult) (map[string][][]lingoToken, error) {
	funcs := make(map[string][][]lingoToken)
	// TODO: 重複直到token.Type沒有FUNC

processFunc:
	for funcName, funcTokens := range extractResult.Funcs {
		for _, token := range funcTokens {
			if token.Type == "FUNC" {
				// 先解決裡面沒有其他函數的
				// 遇到裡面有其他函數的，先跳過
				continue processFunc
			}
		}
		setsMap := lingoProcessGetSetsInFunc(funcTokens)
		toCutIndex := 0
		for i, token := range funcTokens {
			if token.Value == ":" {
				toCutIndex = i
				break
			}
		}
		funcTokens, err := sliceutil.ReplaceWithSlice(funcTokens, 0, toCutIndex, []lingoToken{})
		if err != nil {
			return nil, err
		}
		if strings.HasPrefix(funcName, "$SIZE") {
			funcs[funcName] = append(funcs[funcName], lingoProcessFunc_SIZE(funcTokens, extractResult))
		} else if strings.HasPrefix(funcName, "$SUM") {
			sumTokens, err := lingoProcessFunc_SUM(funcTokens, extractResult, setsMap)
			if err != nil {
				return nil, err
			}
			funcs[funcName] = append(funcs[funcName], sumTokens...)
		} else if strings.HasPrefix(funcName, "$FOR") {
			// TODO: 處理 FOR 函數
		} else if strings.HasPrefix(funcName, "$POW") {
			// TODO: 處理 POW 函數
		} else if strings.HasPrefix(funcName, "$BIN") {
			// TODO: 處理 BIN 函數
		}
		// TODO: 其他函數
	}

	return funcs, nil
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

func lingoProcessFunc_SUM(funcTokens []lingoToken, extractResult *lingoExtractResult, setsMap map[string]string) ([][]lingoToken, error) {
	expandedTokens := make([][]lingoToken, 0)
	toMerge := make(map[int][]string)
	variableToMergeCount := 0

	for i, token := range funcTokens {
		if i+1 >= len(funcTokens) {
			break
		}
		nextToken := funcTokens[i+1]
		if nextToken.Value == "(" {
			thisVariable := token.Value
			// 取得該遍歷的Set
			set := setsMap[funcTokens[i+2].Value]

			// 取得該遍歷的Set的子元素
			setIndex := extractResult.Sets[set].Index

			// 開始展開邏輯
			for _, index := range setIndex {
				subVariable := thisVariable + "_" + index
				toMerge[variableToMergeCount] = append(toMerge[variableToMergeCount], subVariable)
			}

			// 原本含索引的變數替換成符號
			var err error
			funcTokens, err = sliceutil.ReplaceWithSlice(funcTokens, i, i+3, []lingoToken{
				{
					Type:  "TO_MERGE",
					Value: "#" + strconv.Itoa(variableToMergeCount),
				},
			})
			if err != nil {
				return nil, err
			}

			variableToMergeCount++
		}
	}

	maxMergeLength := 0
	for _, mergeList := range toMerge {
		if len(mergeList) > maxMergeLength {
			maxMergeLength = len(mergeList)
		}
	}

	for mergeIndex := 0; mergeIndex < maxMergeLength; mergeIndex++ {
		currentExpression := make([]lingoToken, 0)
		for _, token := range funcTokens {
			if token.Type == "TO_MERGE" {
				nowMerge := conv.ParseInt(token.Value[1:])
				if mergeIndex < len(toMerge[nowMerge]) {
					currentExpression = append(currentExpression, lingoToken{
						Type:  "VARIABLE",
						Value: toMerge[nowMerge][mergeIndex],
					})
				}
			} else {
				currentExpression = append(currentExpression, token)
			}

		}

		// 移除最後一個加號（如果存在）
		if len(currentExpression) > 0 && currentExpression[len(currentExpression)-1].Value == "+" {
			currentExpression = currentExpression[:len(currentExpression)-1]
		}

		// 移除最後一個右括號
		if len(currentExpression) > 0 && currentExpression[len(currentExpression)-1].Value == ")" {
			currentExpression = currentExpression[:len(currentExpression)-1]
		}

		if len(currentExpression) > 0 {
			expandedTokens = append(expandedTokens, currentExpression)
		}
	}

	result := make([]lingoToken, 0)
	for i, token := range expandedTokens {
		result = append(result, token...)
		if i < len(expandedTokens)-1 {
			result = append(result, lingoToken{
				Type:  "OPERATOR",
				Value: "+",
			})
		}
	}

	return [][]lingoToken{result}, nil
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

// 取得函數中的Set
// 已完成，沒問題
func lingoProcessGetSetsInFunc(funcTokens []lingoToken) map[string]string {
	sets := make(map[string]string)
	nowSetName := ""
	indexTokens := make([]lingoToken, 0)

	for i, token := range funcTokens {
		if token.Value == ":" {
			break
		} else if token.Value == "," {
			continue
		}

		if token.Type == "VARIABLE" && i+2 < len(funcTokens) && funcTokens[i+1].Value == "(" {
			nowSetName = token.Value
		} else if token.Type == "VARIABLE" && (funcTokens[i-1].Value == "(" || funcTokens[i-1].Value == ",") {
			indexTokens = append(indexTokens, token)
		}
	}
	for _, indexToken := range indexTokens {
		for _, indexLetter := range indexLetters {
			if indexLetter == indexToken.Value {
				sets[indexLetter] = nowSetName
			}
		}
	}
	fmt.Println(sets)
	return sets
}
