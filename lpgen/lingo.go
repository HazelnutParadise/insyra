package lpgen

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/HazelnutParadise/insyra"
)

// parseLingo reads a LINGO model from r and turns it into an LPModel. With
// strict off it skips whatever it does not recognise, so a model that is read
// but not understood comes back without an error; only text r cannot yield is
// one. With strict on, the same skip is an error naming the line, because the
// caller asked for a model and got a different one.
func parseLingo(r io.Reader, strict bool) (*LPModel, error) {
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
	// 敘述開始的行號，供嚴格模式報錯使用；以 scanner 讀到的行數計。
	var stmtLine int
	lineNo := 0

	// 逐行讀取文件
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())

		// 跳過不必要的行
		if line == "MODEL:" || len(line) == 0 || line == "END" {
			continue
		}

		// 移除方括號和內部數字
		line = lingoRowLabelRe.ReplaceAllString(line, "")
		line = strings.TrimSpace(line)

		// 累積當前行到表達式
		if line != "" {
			if !isFirstLine {
				if !strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "+") {
					currentExpr.WriteString(" ")
				}
			} else {
				// 這一行是這句敘述的開頭。
				stmtLine = lineNo
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

		if strict {
			if err := lingoStrictStatement(model, expr, stmtLine); err != nil {
				return nil, err
			}
			continue
		}

		// 判斷和處理目標函數
		if strings.HasPrefix(strings.ToUpper(expr), "MIN=") || strings.HasPrefix(strings.ToUpper(expr), "MAX=") {
			lingoSetObjective(model, expr)
		} else if strings.HasPrefix(strings.ToUpper(expr), "@BIN") {
			// 處理 Binary 變數宣告
			lingo_handleVariableDeclarations(expr, "@BIN", &model.BinaryVars)
		} else if strings.HasPrefix(strings.ToUpper(expr), "@INT") {
			// 處理 Integer 變數宣告
			lingo_handleVariableDeclarations(expr, "@INT", &model.IntegerVars)
		} else if strings.ContainsAny(expr, "<=>=") {
			lingoAddRelation(model, expr)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if strict {
		if rest := strings.TrimSpace(currentExpr.String()); rest != "" {
			return nil, fmt.Errorf("line %d: LINGO statement has no closing ';': %s", stmtLine, rest)
		}
	}

	return model, nil
}

// lingoStrictStatement reads one statement of a model read with strict on. It
// knows the objective, the relations, the @BIN, @INT and @GIN declarations,
// @FREE(x) and @BND(l, x, u), and it skips a comment. Anything else is an error
// naming the line the statement starts on: dropping it would hand back a model
// the text does not describe.
func lingoStrictStatement(model *LPModel, expr string, line int) error {
	declErr := func() error {
		return fmt.Errorf("line %d: LINGO declaration not read: %s", line, expr)
	}

	switch {
	case strings.HasPrefix(expr, "!"):
		// LINGO 註解，不是敘述。
		return nil
	case strings.HasPrefix(strings.ToUpper(expr), "MIN="), strings.HasPrefix(strings.ToUpper(expr), "MAX="):
		lingoSetObjective(model, expr)
	case strings.HasPrefix(strings.ToUpper(expr), "@BIN"), strings.HasPrefix(strings.ToUpper(expr), "@INT"),
		strings.HasPrefix(strings.ToUpper(expr), "@GIN"):
		// @GIN is LINGO's general integer, which belongs with @INT.
		prefix, target := "@BIN", &model.BinaryVars
		if !strings.HasPrefix(strings.ToUpper(expr), "@BIN") {
			prefix, target = "@INT", &model.IntegerVars
		}
		if strings.HasPrefix(strings.ToUpper(expr), "@GIN") {
			prefix = "@GIN"
		}
		name, ok := lingoDeclaredName(expr, prefix)
		if !ok {
			return declErr()
		}
		*target = append(*target, name)
	case strings.HasPrefix(strings.ToUpper(expr), "@FREE"):
		name, ok := lingoDeclaredName(expr, "@FREE")
		if !ok {
			return declErr()
		}
		model.Bounds = append(model.Bounds, name+" free")
	case strings.HasPrefix(strings.ToUpper(expr), "@BND"):
		bound, ok := lingoBnd(expr)
		if !ok {
			return declErr()
		}
		model.Bounds = append(model.Bounds, bound)
	case strings.ContainsAny(expr, "<=>"):
		lingoAddRelation(model, expr)
	default:
		return fmt.Errorf("line %d: LINGO statement not recognised: %s", line, expr)
	}
	return nil
}

// lingoDeclaredName returns the variable a declaration such as "@BIN(X)"
// declares. prefix is stripped without regard to case and what is left must be
// a parenthesised bare variable name, so a declaration the reader cannot read is
// reported rather than turned into an empty name.
func lingoDeclaredName(expr, prefix string) (string, bool) {
	inner, ok := lingoDeclaredArgs(expr, prefix)
	if !ok || !lingoBareVarRe.MatchString(inner) {
		return "", false
	}
	return inner, true
}

// lingoDeclaredArgs returns what a declaration puts between its parentheses,
// with prefix stripped without regard to case and the outside trimmed. What is
// left is not checked further: @BIN names one variable, @BND three values.
func lingoDeclaredArgs(expr, prefix string) (string, bool) {
	if len(expr) < len(prefix) || !strings.EqualFold(expr[:len(prefix)], prefix) {
		return "", false
	}
	rest := strings.TrimSpace(expr[len(prefix):])
	if len(rest) < 2 || !strings.HasPrefix(rest, "(") || !strings.HasSuffix(rest, ")") {
		return "", false
	}
	return strings.TrimSpace(rest[1 : len(rest)-1]), true
}

// lingoBnd turns a @BND(l, x, u) declaration into the bound "l <= x <= u". The
// two limits keep the text they were written as, since a value the reader
// understands exactly need not be reformatted.
func lingoBnd(expr string) (string, bool) {
	inner, ok := lingoDeclaredArgs(expr, "@BND")
	if !ok {
		return "", false
	}
	parts := strings.Split(inner, ",")
	if len(parts) != 3 {
		return "", false
	}
	for i, s := range parts {
		parts[i] = strings.TrimSpace(s)
	}
	if !lingoIsNumber(parts[0]) || !lingoBareVarRe.MatchString(parts[1]) || !lingoIsNumber(parts[2]) {
		return "", false
	}
	return fmt.Sprintf("%s <= %s <= %s", parts[0], parts[1], parts[2]), true
}

// lingoIsNumber reports whether s is a number the reader takes: the shape the
// rest of the parser uses, or the scientific notation it writes back without
// spaces (1e-5), which lingoNumRe does not match.
func lingoIsNumber(s string) bool {
	if lingoNumRe.MatchString(s) {
		return true
	}
	if _, err := strconv.ParseFloat(s, 64); err == nil {
		return true
	}
	return false
}

// lingoSetObjective records a MIN= or MAX= statement's objective.
func lingoSetObjective(model *LPModel, expr string) {
	objType := "Minimize"
	if strings.HasPrefix(strings.ToUpper(expr), "MAX=") {
		objType = "Maximize"
	}
	content := strings.TrimSpace(strings.SplitN(expr, "=", 2)[1])
	model.ObjectiveType = objType
	model.Objective = content
}

// lingoAddRelation records a relation, in Bounds when it is a single variable
// against constants and in Constraints otherwise.
func lingoAddRelation(model *LPModel, expr string) {
	if lingoIsBound(expr) {
		model.Bounds = append(model.Bounds, expr)
		return
	}
	model.Constraints = append(model.Constraints, expr)
}

// ParseLingo reads a LINGO model from text and turns it into an LPModel. The
// text is what LINGO shows under LINGO > Generate > Display Model. Text that
// cannot be read, such as a line of 64 KiB or more, is an error.
//
// A statement it does not recognise, a declaration whose variable it cannot
// read, or a last statement without its closing ';' is an error giving the
// line the statement starts on. Besides the objective, the constraints and
// the bounds, it reads @BIN, @INT and @GIN declarations, @FREE(x) as the bound
// "x free" and @BND(l, x, u) as "l <= x <= u", and it skips a comment, a
// statement starting with '!'.
func ParseLingo(model string) (*LPModel, error) {
	lp, err := parseLingo(strings.NewReader(model), true)
	if err != nil {
		return nil, fmt.Errorf("failed to read LINGO model: %w", err)
	}
	return lp, nil
}

// ParseLingoFile reads a LINGO model from the file at path, the way ParseLingo
// reads it from text. A file that cannot be opened or read is an error; a
// missing file matches fs.ErrNotExist.
//
// It reads a file as ParseLingo reads text: a statement it does not recognise,
// a declaration whose variable it cannot read, or a last statement without its
// closing ';' is an error giving the line the statement starts on. Besides the
// objective, the constraints and the bounds, it reads @BIN, @INT and @GIN
// declarations, @FREE(x) as the bound "x free" and @BND(l, x, u) as
// "l <= x <= u", and it skips a comment, a statement starting with '!'.
func ParseLingoFile(path string) (*LPModel, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open LINGO model: %w", err)
	}
	defer func() { _ = file.Close() }()

	lp, err := parseLingo(file, true)
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
	model, err := parseLingo(strings.NewReader(modelStr), false)
	if err != nil {
		// strict off: the message and the nil are the ones this name has always
		// given on text it cannot read at all.
		insyra.LogWarning("lpgen", "ParseLingoModel_str", "%s", fmt.Errorf("failed to read LINGO model: %w", err).Error())
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
	file, err := os.Open(filePath)
	if err != nil {
		insyra.LogWarning("lpgen", "ParseLingoModel_txt", "%s", fmt.Errorf("failed to open LINGO model: %w", err).Error())
		return nil
	}
	defer func() { _ = file.Close() }()

	model, err := parseLingo(file, false)
	if err != nil {
		insyra.LogWarning("lpgen", "ParseLingoModel_txt", "%s", fmt.Errorf("failed to read LINGO model %s: %w", filePath, err).Error())
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
