package main // 記得改掉

import (
	"fmt"
	"regexp"
	"strings"
)

// 變數替換函數：將 Lingo 變數轉換成 LP 格式
func replaceLingoVariables(lingoText string) string {
	// 匹配變數形式，例如 X(I)
	re := regexp.MustCompile(`(\w+)\((\w+)\)`)

	// 替換變數為 LP 標準變數 x1, x2...
	replacedText := re.ReplaceAllStringFunc(lingoText, func(match string) string {
		subMatches := re.FindStringSubmatch(match)
		varName := strings.ToLower(subMatches[1]) // 變數名稱，如 X 或 Cost
		index := subMatches[2]                    // 索引名稱，如 I

		// 假設索引是 I 或 J，動態生成索引對應數字
		// TODO: 動態生成索引對應數字
		indexNumber := map[string]int{
			"I": 1, // 替換 I 為 1
			"J": 2, // 替換 J 為 2
		}

		// 將變數轉換成 LP 格式變數
		if num, exists := indexNumber[index]; exists {
			return fmt.Sprintf("%s%d", varName, num) // 替換成 x1, cost1 等
		}
		return match // 若無對應索引，保持原狀
	})

	return replacedText
}

// 解析並轉換目標函數
func convertObjectiveFunction(lingoText string) string {
	// 匹配 MIN 或 MAX 語法
	re := regexp.MustCompile(`(MIN|MAX) = @SUM\((\w+): (.+)\)`)
	subMatches := re.FindStringSubmatch(lingoText)

	if len(subMatches) == 0 {
		return ""
	}

	objectiveType := strings.ToLower(subMatches[1]) // MIN 或 MAX
	sumExpression := subMatches[3]                  // 目標函數中的運算部分

	// 進行變數替換
	lpExpression := replaceLingoVariables(sumExpression)

	// 生成 LP 語法
	if objectiveType == "min" {
		return fmt.Sprintf("Minimize\n  obj: %s", lpExpression)
	}
	return fmt.Sprintf("Maximize\n  obj: %s", lpExpression)
}

// 解析並轉換約束條件
func convertConstraints(lingoText string) string {
	// 匹配 @FOR 和 @SUM 結構的約束條件
	re := regexp.MustCompile(`@FOR\((\w+): @SUM\((\w+): (.+)\) <= (.+)\)`)
	subMatches := re.FindStringSubmatch(lingoText)

	if len(subMatches) == 0 {
		return ""
	}

	// 解析運算部分並進行變數替換
	sumExpression := subMatches[3]
	rhs := subMatches[4]

	lpExpression := replaceLingoVariables(sumExpression)
	lpRHS := replaceLingoVariables(rhs)

	// 生成 LP 格式的約束條件
	return fmt.Sprintf("Subject To\n  constraint1: %s <= %s", lpExpression, lpRHS)
}

func main() {
	// 假設這是你的 Lingo 模型
	lingoText := `
	MIN = @SUM(I: Cost(I) * X(I));
	@FOR(I: @SUM(J: Coeff(I, J) * X(J)) <= RHS(I));
	`

	// 轉換目標函數
	objectiveFunction := convertObjectiveFunction(lingoText)

	// 轉換約束條件
	constraints := convertConstraints(lingoText)

	// 最終輸出 LP 模型
	fmt.Println(objectiveFunction)
	fmt.Println(constraints)
}
