package main

import (
	"fmt"
	"regexp"
	"strings"
)

// 迴圈處理函數，根據索引執行指定的運算邏輯
func lingoFor(index int, action func(i int) string) string {
	result := ""
	for i := 1; i <= index; i++ {
		result += action(i) + "\n"
	}
	return result
}

// 變數替換函數：將 Lingo 變數轉換成 LP 格式，但不替換 RHS
func replaceLingoVariables(lingoText string, varIndexMap map[string]int) string {
	// 匹配變數形式，例如 X(I)、Cost(I)，但跳過 RHS(I)
	re := regexp.MustCompile(`(\w+)\(([^)]+)\)`)

	replacedText := re.ReplaceAllStringFunc(lingoText, func(match string) string {
		subMatches := re.FindStringSubmatch(match)
		varName := strings.ToLower(subMatches[1]) // 變數名稱，如 X 或 Cost
		indexes := subMatches[2]                  // 索引名稱，如 I 或 I, J

		// 跳過 RHS，保持不變
		if strings.ToLower(varName) == "rhs" {
			return match
		}

		// 將索引部分拆解並轉換成 _ 分隔的格式，如 I, J -> 1_2
		indexList := strings.Split(indexes, ",")
		var indexStringBuilder strings.Builder

		for i, index := range indexList {
			// 去掉空格
			index = strings.TrimSpace(index)

			// 確定索引是否已經分配
			if _, exists := varIndexMap[index]; !exists {
				varIndexMap[index] = 1 // 從 1 開始
			}

			// 如果不是第一個索引，前面加上 _
			if i > 0 {
				indexStringBuilder.WriteString("_")
			}
			indexStringBuilder.WriteString(fmt.Sprintf("%d", varIndexMap[index]))

			// 每次索引遞增
			varIndexMap[index]++
		}

		// 將變數轉換成 LP 格式變數，並用 _ 分隔索引
		return fmt.Sprintf("%s%s", varName, indexStringBuilder.String())
	})

	return replacedText
}

// 處理目標函數邏輯 (如 MIN, MAX)
func lingoObjectiveFunction(lingoText string, varIndexMap map[string]int) string {
	// 匹配 MIN 或 MAX 語法
	re := regexp.MustCompile(`(MIN|MAX) = @SUM\((\w+): (.+)\)`)
	subMatches := re.FindStringSubmatch(lingoText)

	if len(subMatches) == 0 {
		return ""
	}

	objectiveType := strings.ToLower(subMatches[1]) // MIN 或 MAX
	sumExpression := subMatches[3]                  // 目標函數中的運算部分

	// 進行變數替換，考慮索引
	lpExpression := replaceLingoVariables(sumExpression, varIndexMap)

	// 生成 LP 語法
	if objectiveType == "min" {
		return fmt.Sprintf("Minimize\n  obj: %s", lpExpression)
	}
	return fmt.Sprintf("Maximize\n  obj: %s", lpExpression)
}

// 處理 @SUM 函數邏輯
func lingoSum(index int, sumExpression string, varIndexMap map[string]int) string {
	return lingoFor(index, func(i int) string {
		// 根據索引 i 替換 sumExpression 中的變數
		lpExpression := replaceLingoVariables(sumExpression, varIndexMap)
		return fmt.Sprintf("sum_term%d: %s", i, lpExpression)
	})
}

// 處理 @FOR 函數邏輯
func lingoForConstraints(index int, body string, varIndexMap map[string]int) string {
	// 使用 lingoFor 函數展開 @FOR 迴圈
	return lingoFor(index, func(i int) string {
		lpExpression := replaceLingoVariables(body, varIndexMap)
		return fmt.Sprintf("for_constraint%d: %s", i, lpExpression)
	})
}

// 處理所有 Lingo 函數的統一入口
func convertLingoFunction(lingoText string, varIndexMap map[string]int) string {
	// 根據函數類型匹配並處理
	if strings.Contains(lingoText, "@SUM") {
		// 使用正則表達式提取出 SUM 的內容
		re := regexp.MustCompile(`@SUM\((\w+): (.+)\)`)
		subMatches := re.FindStringSubmatch(lingoText)
		if len(subMatches) > 0 {
			sumExpression := subMatches[2]
			return lingoSum(3, sumExpression, varIndexMap) // 假設 I 的範圍是 3
		}
	} else if strings.Contains(lingoText, "@FOR") {
		// 使用正則表達式提取出 FOR 的內容
		re := regexp.MustCompile(`@FOR\((\w+): (.+)\)`)
		subMatches := re.FindStringSubmatch(lingoText)
		if len(subMatches) > 0 {
			forBody := subMatches[2]
			return lingoForConstraints(3, forBody, varIndexMap) // 假設 I 的範圍是 3
		}
	}
	return ""
}

// 整體轉換邏輯
func LingoToLP(lingoText string) string {
	result := ""
	varIndexMap := make(map[string]int) // 追蹤索引狀況

	// 轉換目標函數
	result += lingoObjectiveFunction(lingoText, varIndexMap) + "\n"

	// 假設 @FOR 的索引範圍
	result += convertLingoFunction(lingoText, varIndexMap) + "\n"

	// 結束
	result += "End"

	return result
}

func main() {
	// 假設這是你的 Lingo 模型
	lingoText := `
MIN = @SUM(I: Cost(I) * X(I));
@FOR(I: @SUM(J: Coeff(I, J) * X(J)) <= RHS(I));
`

	// 最終輸出 LP 模型
	fmt.Println(LingoToLP(lingoText))
}
