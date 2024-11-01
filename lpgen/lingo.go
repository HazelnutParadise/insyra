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
	// 定義正則表達式處理缺少空格的項目
	missingSpaceRe := regexp.MustCompile(`([a-zA-Z_0-9]+)([+-])`)
	// 定義正則表達式確保科學記號中的 e 周圍沒有空格
	scientificNotationSpaceFixRe := regexp.MustCompile(`(\d)([eE])\s*([+-]?\d+)`)

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
		expr = strings.TrimSuffix(expr, ";")
		currentExpr.Reset()
		isFirstLine = true

		// 移除乘號符號 `*` 以符合 LP 格式要求
		expr = multiplyRe.ReplaceAllString(expr, " ")

		// 確保每個項目之間有空格
		expr = missingSpaceRe.ReplaceAllString(expr, `$1 $2`)

		// 確保科學記號中的 e 和指數之間沒有空格
		expr = scientificNotationSpaceFixRe.ReplaceAllString(expr, `$1$2$3`)

		// 確保運算項之間有單一空格
		expr = spaceRe.ReplaceAllString(expr, " ")

		// 根據表達式內容分類
		if strings.HasPrefix(strings.ToUpper(expr), "MIN=") || strings.HasPrefix(strings.ToUpper(expr), "MAX=") {
			// 處理目標函數
			objType := "Minimize"
			if strings.HasPrefix(strings.ToUpper(expr), "MAX=") {
				objType = "Maximize"
			}
			// 直接取得 = 後面的內容
			content := strings.TrimSpace(strings.SplitN(expr, "=", 2)[1])
			model.ObjectiveType = objType
			model.Objective = content
		} else if strings.HasPrefix(strings.ToUpper(expr), "@BIN") {
			// 處理多個 Binary 變數宣告
			binDeclarations := strings.Split(expr, ";")
			for _, declaration := range binDeclarations {
				declaration = strings.TrimSpace(declaration)
				if !strings.HasPrefix(strings.ToUpper(declaration), "@BIN") {
					continue
				}
				start := strings.Index(declaration, "(")
				end := strings.Index(declaration, ")")
				if start != -1 && end != -1 {
					binVar := declaration[start+1 : end]
					binVar = strings.TrimSpace(binVar)
					model.BinaryVars = append(model.BinaryVars, binVar)
				}
			}
		} else if strings.ContainsAny(expr, "<=>=") {
			// 處理 Bounds 和 Constraints
			content := expr

			// 判斷是否為 Bound（不包含運算符號 + 或 -）
			if !strings.ContainsAny(content, "+-") {
				model.Bounds = append(model.Bounds, content)
			} else {
				model.Constraints = append(model.Constraints, content)
			}
		}
	}

	// 在處理完所有表達式後，添加 OPT_VACC_NUM 的界限
	for i := 1; i <= 5; i++ {
		model.Bounds = append(model.Bounds, fmt.Sprintf("OPT_VACC_NUM_%d >= 0", i))
	}

	// 添加 S 和 H 變數的界限
	for i := 1; i <= 5; i++ {
		model.Bounds = append(model.Bounds, fmt.Sprintf("S_%d >= 0", i))
		model.Bounds = append(model.Bounds, fmt.Sprintf("H_%d >= 0", i))
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("讀取檔案時發生錯誤: %w", err)
	}

	return model, nil
}
