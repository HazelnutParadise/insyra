package lpgen

import (
	"bufio"
	"os"
	"regexp"
	"strings"

	"github.com/HazelnutParadise/insyra"
)

// ParseLingoModel_txt parse lingo model from txt file.
// It turns LINGO model to standard lp model.
// Go to `LINGO > Generate > Display Model` in LINGO to get the model.
func ParseLingoModel_txt(filePath string) *LPModel {
	// 讀取文件
	file, err := os.Open(filePath)
	if err != nil {
		insyra.LogWarning("lpgen", "ParseLingoModel_txt", "%s", err.Error())
		return nil
	}
	defer file.Close()

	// 初始化 LPModel
	model := &LPModel{
		Constraints: make([]string, 0),
		Bounds:      make([]string, 0),
		BinaryVars:  make([]string, 0),
		IntegerVars: make([]string, 0),
	}

	// 正則表達式
	re := regexp.MustCompile(`^\[\_\d+\]\s*`)
	multiplyRe := regexp.MustCompile(`\s*\*\s*`)
	spaceRe := regexp.MustCompile(`\s+`)
	missingSpaceRe := regexp.MustCompile(`([a-zA-Z_0-9]+)([+-])`)
	scientificNotationSpaceFixRe := regexp.MustCompile(`(\d)([eE])\s*([+-]?\d+)`)

	// 用於累積多行表達式
	var currentExpr strings.Builder
	var isFirstLine bool = true

	// 逐行讀取文件
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 跳過不必要的行
		if line == "MODEL:" || len(line) == 0 || line == "END" {
			continue
		}

		// 移除方括號和內部數字
		line = re.ReplaceAllString(line, "")
		line = strings.TrimSpace(line)

		// 累積當前行到表達式
		if !isFirstLine && line != "" {
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

		// 清理表達式格式
		expr = multiplyRe.ReplaceAllString(expr, " ")
		expr = missingSpaceRe.ReplaceAllString(expr, `$1 $2`)
		expr = scientificNotationSpaceFixRe.ReplaceAllString(expr, `$1$2$3`)
		expr = spaceRe.ReplaceAllString(expr, " ")

		// 判斷和處理目標函數
		if strings.HasPrefix(strings.ToUpper(expr), "MIN=") || strings.HasPrefix(strings.ToUpper(expr), "MAX=") {
			objType := "Minimize"
			if strings.HasPrefix(strings.ToUpper(expr), "MAX=") {
				objType = "Maximize"
			}
			content := strings.TrimSpace(strings.SplitN(expr, "=", 2)[1])
			model.ObjectiveType = objType
			model.Objective = content
		} else if strings.HasPrefix(strings.ToUpper(expr), "@BIN") {
			// 處理 Binary 變數宣告
			lingo_handleVariableDeclarations(expr, "@BIN", &model.BinaryVars)
		} else if strings.HasPrefix(strings.ToUpper(expr), "@INT") {
			// 處理 Integer 變數宣告
			lingo_handleVariableDeclarations(expr, "@INT", &model.IntegerVars)
		} else if strings.ContainsAny(expr, "<=>=") {
			// 處理 Bounds 和 Constraints
			content := expr
			if !strings.ContainsAny(content, "+-") {
				model.Bounds = append(model.Bounds, content)
			} else {
				model.Constraints = append(model.Constraints, content)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		insyra.LogWarning("lpgen", "ParseLingoModel_txt", "%s", err.Error())
		return nil
	}

	return model
}

// ParseLingoModel_str parse lingo model from string.
// It turns LINGO model to standard lp model.
// Go to `LINGO > Generate > Display Model` in LINGO to get the model.
func ParseLingoModel_str(modelStr string) *LPModel {
	// 初始化 LPModel
	model := &LPModel{
		Constraints: make([]string, 0),
		Bounds:      make([]string, 0),
		BinaryVars:  make([]string, 0),
		IntegerVars: make([]string, 0),
	}

	// 正則表達式
	re := regexp.MustCompile(`^\[\_\d+\]\s*`)
	multiplyRe := regexp.MustCompile(`\s*\*\s*`)
	spaceRe := regexp.MustCompile(`\s+`)
	missingSpaceRe := regexp.MustCompile(`([a-zA-Z_0-9]+)([+-])`)
	scientificNotationSpaceFixRe := regexp.MustCompile(`(\d)([eE])\s*([+-]?\d+)`)

	// 用於累積多行表達式
	var currentExpr strings.Builder
	var isFirstLine bool = true

	// 將字串分割成行
	scanner := bufio.NewScanner(strings.NewReader(modelStr))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 跳過不必要的行
		if line == "MODEL:" || len(line) == 0 || line == "END" {
			continue
		}

		// 移除方括號和內部數字
		line = re.ReplaceAllString(line, "")
		line = strings.TrimSpace(line)

		// 累積當前行到表達式
		if !isFirstLine && line != "" {
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

		// 清理表達式格式
		expr = multiplyRe.ReplaceAllString(expr, " ")
		expr = missingSpaceRe.ReplaceAllString(expr, `$1 $2`)
		expr = scientificNotationSpaceFixRe.ReplaceAllString(expr, `$1$2$3`)
		expr = spaceRe.ReplaceAllString(expr, " ")

		// 判斷和處理目標函數
		if strings.HasPrefix(strings.ToUpper(expr), "MIN=") || strings.HasPrefix(strings.ToUpper(expr), "MAX=") {
			objType := "Minimize"
			if strings.HasPrefix(strings.ToUpper(expr), "MAX=") {
				objType = "Maximize"
			}
			content := strings.TrimSpace(strings.SplitN(expr, "=", 2)[1])
			model.ObjectiveType = objType
			model.Objective = content
		} else if strings.HasPrefix(strings.ToUpper(expr), "@BIN") {
			// 處理 Binary 變數宣告
			lingo_handleVariableDeclarations(expr, "@BIN", &model.BinaryVars)
		} else if strings.HasPrefix(strings.ToUpper(expr), "@INT") {
			// 處理 Integer 變數宣告
			lingo_handleVariableDeclarations(expr, "@INT", &model.IntegerVars)
		} else if strings.ContainsAny(expr, "<=>=") {
			// 處理 Bounds 和 Constraints
			content := expr
			if !strings.ContainsAny(content, "+-") {
				model.Bounds = append(model.Bounds, content)
			} else {
				model.Constraints = append(model.Constraints, content)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		insyra.LogWarning("lpgen", "ParseLingoModel_str", "%s", err.Error())
		return nil
	}

	return model
}

// handleVariableDeclarations 處理變數宣告並將變數名稱添加到相應的列表中
func lingo_handleVariableDeclarations(expr string, declarationType string, targetList *[]string) {
	// 切分宣告語句並處理每個宣告
	declarations := strings.Split(expr, ";")
	for _, declaration := range declarations {
		declaration = strings.TrimSpace(declaration)
		if !strings.HasPrefix(strings.ToUpper(declaration), declarationType) {
			continue
		}
		// 提取括號內的變數名稱
		start := strings.Index(declaration, "(")
		end := strings.Index(declaration, ")")
		if start != -1 && end != -1 {
			varName := declaration[start+1 : end]
			*targetList = append(*targetList, strings.TrimSpace(varName))
		}
	}
}
