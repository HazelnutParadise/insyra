package lpgen

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/HazelnutParadise/insyra"
)

// parseLingo reads a LINGO model from r and turns it into an LPModel. It skips
// whatever it does not recognise, so a model that is read but not understood
// comes back without an error; only text r cannot yield is one.
func parseLingo(r io.Reader) (*LPModel, error) {
	// 初始化 LPModel
	model := &LPModel{
		Constraints: make([]string, 0),
		Bounds:      make([]string, 0),
		BinaryVars:  make([]string, 0),
		IntegerVars: make([]string, 0),
	}

	// 用於累積多行表達式
	var currentExpr strings.Builder
	var isFirstLine = true

	// 逐行讀取文件
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 跳過不必要的行
		if line == "MODEL:" || len(line) == 0 || line == "END" {
			continue
		}

		// 移除方括號和內部數字
		line = lingoRowLabelRe.ReplaceAllString(line, "")
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
		expr = lingoMultiplyRe.ReplaceAllString(expr, " ")
		expr = lingoMissingSpaceRe.ReplaceAllString(expr, `$1 $2`)
		expr = lingoSciNotationRe.ReplaceAllString(expr, `$1$2$3`)
		expr = lingoSpaceRe.ReplaceAllString(expr, " ")

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
			// 處理 Bounds 和 Constraints：以「變數項數量」判斷，而非 +/- 字元。
			if lingoIsBound(expr) {
				model.Bounds = append(model.Bounds, expr)
			} else {
				model.Constraints = append(model.Constraints, expr)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return model, nil
}

// ParseLingo reads a LINGO model from text and turns it into an LPModel. The
// text is what LINGO shows under LINGO > Generate > Display Model. Text that
// cannot be read, such as a line of 64 KiB or more, is an error.
func ParseLingo(model string) (*LPModel, error) {
	lp, err := parseLingo(strings.NewReader(model))
	if err != nil {
		return nil, fmt.Errorf("failed to read LINGO model: %w", err)
	}
	return lp, nil
}

// ParseLingoFile reads a LINGO model from the file at path, the way ParseLingo
// reads it from text. A file that cannot be opened or read is an error; a
// missing file matches fs.ErrNotExist.
func ParseLingoFile(path string) (*LPModel, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open LINGO model: %w", err)
	}
	defer func() { _ = file.Close() }()

	lp, err := parseLingo(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read LINGO model %s: %w", path, err)
	}
	return lp, nil
}

// ParseLingoModel_str reads a LINGO model from text, or returns nil and logs a
// warning when the text cannot be read.
//
// Deprecated: use ParseLingo, which returns the failure as an error. Removed
// in the release after the one that deprecated it.
func ParseLingoModel_str(modelStr string) *LPModel {
	model, err := ParseLingo(modelStr)
	if err != nil {
		insyra.LogWarning("lpgen", "ParseLingoModel_str", "%s", err.Error())
		return nil
	}
	return model
}

// ParseLingoModel_txt reads a LINGO model from a file, or returns nil and logs
// a warning when the file cannot be opened or read.
//
// Deprecated: use ParseLingoFile, which returns the failure as an error.
// Removed in the release after the one that deprecated it.
func ParseLingoModel_txt(filePath string) *LPModel {
	model, err := ParseLingoFile(filePath)
	if err != nil {
		insyra.LogWarning("lpgen", "ParseLingoModel_txt", "%s", err.Error())
		return nil
	}
	return model
}

// Compiled once: the parser rebuilt these five on every call.
var (
	lingoRowLabelRe     = regexp.MustCompile(`^\[\_\d+\]\s*`)
	lingoMultiplyRe     = regexp.MustCompile(`\s*\*\s*`)
	lingoSpaceRe        = regexp.MustCompile(`\s+`)
	lingoMissingSpaceRe = regexp.MustCompile(`([a-zA-Z_0-9]+)([+-])`)
	lingoSciNotationRe  = regexp.MustCompile(`(\d)([eE])\s*([+-]?\d+)`)
)

// handleVariableDeclarations 處理變數宣告並將變數名稱添加到相應的列表中
var lingoBareVarRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
var lingoNumRe = regexp.MustCompile(`^[+-]?(\d+\.?\d*|\.\d+)$`)
var lingoRelSplitRe = regexp.MustCompile(`<=|>=|==|<|>|=`)

// lingoIsBound reports whether a LINGO relation is a simple variable bound
// (a single bare variable compared to constants, e.g. "X1 >= -5" or
// "0 <= X1 <= 10") rather than a general constraint (multiple terms or a
// non-unit coefficient, e.g. "3 X1 <= 10" or "X1 + X2 <= 10"). The previous
// heuristic keyed only off the presence of '+'/'-', which misclassified both.
func lingoIsBound(expr string) bool {
	segs := lingoRelSplitRe.Split(expr, -1)
	varCount := 0
	for _, s := range segs {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		switch {
		case lingoNumRe.MatchString(s):
			// constant operand
		case lingoBareVarRe.MatchString(s):
			varCount++
		default:
			return false
		}
	}
	return varCount == 1
}

func lingo_handleVariableDeclarations(expr string, declarationType string, targetList *[]string) {
	// 切分宣告語句並處理每個宣告
	declarations := strings.Split(expr, ";")
	for _, declaration := range declarations {
		declaration = strings.TrimSpace(declaration)
		if !strings.HasPrefix(strings.ToUpper(declaration), declarationType) {
			continue
		}
		// 提取括號內的變數名稱。start 與 end 都是「第一個」出現的位置，
		// 順序顛倒時（例如 `@BIN)X(;`）切片邊界會反過來而 panic，
		// 因此要求 end 在 start 之後；讀不懂的宣告與其他讀不懂的行一樣略過。
		start := strings.Index(declaration, "(")
		end := strings.Index(declaration, ")")
		if start != -1 && end > start {
			varName := declaration[start+1 : end]
			*targetList = append(*targetList, strings.TrimSpace(varName))
		}
	}
}
