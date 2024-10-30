package lpgen

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// ParseLingoFile 解析 Lingo 檔案並返回 LPModel
func ParseLingoFile(filePath string) (*LPModel, error) {
	// 讀取文件
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("無法開啟檔案: %w", err)
	}
	defer file.Close()

	// 初始化 LPModel
	model := &LPModel{
		Objective:     "",
		ObjectiveType: "",
		Constraints:   make([]string, 0),
		Bounds:        make([]string, 0),
		BinaryVars:    make([]string, 0),
		IntegerVars:   make([]string, 0),
	}

	// 定義正則表達式去除方括號和內部數字
	re := regexp.MustCompile(`^\[\_\d+\]\s*`)
	// 定義正則表達式處理乘號
	multiplyRe := regexp.MustCompile(`\s*\*\s*`)
	// 定義正則表達式處理多餘空格
	spaceRe := regexp.MustCompile(`\s+`)

	// 用於累積多行表達式
	var currentExpr strings.Builder
	var isFirstLine bool = true

	// 逐行讀取文件
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 跳過 "MODEL:" 行和空行或 "END"
		if line == "MODEL:" || len(line) == 0 || line == "END" {
			continue
		}

		// 移除方括號和內部數字
		line = re.ReplaceAllString(line, "")
		line = strings.TrimSpace(line)

		// 累積當前行到表達式
		if !isFirstLine && line != "" {
			// 如果當前行不以運算符開頭，添加空格分隔
			if !strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "+") {
				currentExpr.WriteString(" ")
			}
		}
		currentExpr.WriteString(line)
		isFirstLine = false

		// 如果行尾沒有分號，繼續累積下一行
		if !strings.HasSuffix(line, ";") {
			continue
		}

		// 獲取完整表達式並清理
		expr := currentExpr.String()
		expr = strings.TrimSpace(expr)
		if strings.HasSuffix(expr, ";") {
			expr = expr[:len(expr)-1]
		}
		currentExpr.Reset()
		isFirstLine = true

		// 根據表達式內容分類
		if strings.HasPrefix(strings.ToUpper(expr), "MIN=") || strings.HasPrefix(strings.ToUpper(expr), "MAX=") {
			// 處理目標函數
			objType := "MIN"
			if strings.HasPrefix(strings.ToUpper(expr), "MAX=") {
				objType = "MAX"
			}
			content := strings.TrimSpace(strings.TrimPrefix(strings.ToUpper(expr), objType+"="))
			model.ObjectiveType = objType

			// 格式化目標函數
			content = formatExpression(content, multiplyRe, spaceRe)
			model.Objective = content

		} else if strings.HasPrefix(strings.ToUpper(expr), "@BIN") {
			// 處理 Binary
			start := strings.Index(expr, "(")
			end := strings.Index(expr, ")")
			if start != -1 && end != -1 {
				vars := expr[start+1 : end]
				// 分割多個二進制變數
				binVars := strings.Split(vars, ",")
				for _, binVar := range binVars {
					binVar = strings.TrimSpace(binVar)
					model.BinaryVars = append(model.BinaryVars, binVar)
				}
			}
		} else if strings.ContainsAny(expr, "<=>=") {
			// 處理 Bounds 和 Constraints
			content := formatExpression(expr, multiplyRe, spaceRe)

			// 判斷是否為 Bound（不包含運算符號 + 或 -）
			if !strings.ContainsAny(content, "+-") {
				model.Bounds = append(model.Bounds, content)
			} else {
				model.Constraints = append(model.Constraints, content)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("讀取檔案時發生錯誤: %w", err)
	}

	return model, nil
}

// formatExpression 格式化表達式，確保運算符前後有單一空格，並移除乘號
func formatExpression(expr string, multiplyRe, spaceRe *regexp.Regexp) string {
	// 移除乘號並保持係數和變數之間的空格
	expr = multiplyRe.ReplaceAllString(expr, " ")
	// 替換多餘空格為單一空格
	expr = spaceRe.ReplaceAllString(expr, " ")
	// 確保運算符前後有單一空格
	expr = regexp.MustCompile(`([+-])`).ReplaceAllString(expr, " $1 ")
	// 替換多個空格為單一空格
	expr = spaceRe.ReplaceAllString(expr, " ")
	// 去除表達式前後的空格
	expr = strings.TrimSpace(expr)
	return expr
}

// PrintModel 輸出模型內容
func (m *LPModel) PrintModel() {
	fmt.Println("目標函數（Objectives）:")
	fmt.Printf(" - %s: %s\n", m.ObjectiveType, m.Objective)

	fmt.Println("\nBinary 限制（Binary Constraints）:")
	for _, bin := range m.BinaryVars {
		fmt.Println(" -", bin)
	}

	fmt.Println("\n限制式（Constraints）:")
	for _, constraint := range m.Constraints {
		fmt.Println(" -", constraint)
	}

	fmt.Println("\n界限（Bounds）:")
	for _, bound := range m.Bounds {
		fmt.Println(" -", bound)
	}
}
