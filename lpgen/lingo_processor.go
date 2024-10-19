package lpgen

import (
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
			// TODO: 處理 SUM 函數
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
	// TODO
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

	// 有問題
	for toMerge != nil {
		for _, token := range funcTokens {
			if token.Type == "TO_MERGE" {
				// 去掉 #
				nowMerge := conv.ParseInt(token.Value[1:])
				expandedTokens = append(expandedTokens, []lingoToken{})
				// 邏輯有問題
				slice := toMerge[nowMerge]
				poped, err := sliceutil.Drt_PopFrom(&slice)
				if err != nil {
					return nil, err
				}
				toMerge[nowMerge] = slice
				expandedTokens[nowMerge] = append(expandedTokens[nowMerge], lingoToken{
					Type:  "VARIABLE",
					Value: poped,
				})

				if len(toMerge[nowMerge]) == 0 {
					delete(toMerge, nowMerge)
					break
				}

			}
		}
	}
	return expandedTokens, nil
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

func lingoProcessGetSetsInFunc(funcTokens []lingoToken) map[string]string {
	sets := make(map[string]string)
	nowSetName := ""

	for i, token := range funcTokens {
		if token.Value == ":" {
			break
		} else if token.Value == "," {
			nowSetName = ""
			continue
		}

		if token.Type == "VARIABLE" && i+2 < len(funcTokens) {
			nowSetName = token.Value
			indexToken := funcTokens[i+2]
			for _, indexLetter := range indexLetters {
				if indexLetter == indexToken.Value {
					sets[indexLetter] = nowSetName
				}
			}
		}
	}
	return sets
}
