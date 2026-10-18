package lpgen

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// parseLingo reads a LINGO model from r and turns it into an LPModel. A
// statement it does not recognise is an error naming the line, because
// dropping it would hand back a model the text does not describe.
func parseLingo(r io.Reader) (*LPModel, error) {
	model := &LPModel{
		Constraints: make([]string, 0),
		Bounds:      make([]string, 0),
		BinaryVars:  make([]string, 0),
		IntegerVars: make([]string, 0),
	}

	// 用於累積多行表達式
	var currentExpr strings.Builder
	// 敘述開始的行號，供報錯使用；以 scanner 讀到的行數計。
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

		parts := strings.Split(line, ";")
		for i, part := range parts {
			part = strings.TrimSpace(part)
			if currentExpr.Len() == 0 && part != "" {
				stmtLine = lineNo
			}
			if strings.HasPrefix(part, "!") {
				currentExpr.Reset()
				continue
			}
			if part != "" {
				if currentExpr.Len() > 0 && !strings.HasPrefix(part, "+") && !strings.HasPrefix(part, "-") {
					currentExpr.WriteByte(' ')
				}
				currentExpr.WriteString(part)
			}
			if i < len(parts)-1 {
				expr := cleanLingoExpr(currentExpr.String())
				currentExpr.Reset()
				if expr != "" {
					if err := lingoStatement(model, expr, stmtLine); err != nil {
						return nil, err
					}
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if rest := strings.TrimSpace(currentExpr.String()); rest != "" {
		return nil, fmt.Errorf("line %d: LINGO statement has no closing ';': %s", stmtLine, rest)
	}

	return model, nil
}

func cleanLingoExpr(expr string) string {
	expr = lingoSciNotationRe.ReplaceAllString(expr, "$1$2$3")
	expr = lingoLinearMultiplyRe.ReplaceAllString(expr, "$1 $2")
	expr = lingoMissingSpaceRe.ReplaceAllString(expr, "$1 $2")
	return strings.TrimSpace(lingoSpaceRe.ReplaceAllString(expr, " "))
}

// lingoStatement reads one statement of a model. It
// knows the objective, the relations, the @BIN, @INT and @GIN declarations,
// @FREE(x) and @BND(l, x, u), and it skips a comment. Anything else is an error
// naming the line the statement starts on: dropping it would hand back a model
// the text does not describe.
func lingoStatement(model *LPModel, expr string, line int) error {
	declErr := func() error {
		return fmt.Errorf("line %d: LINGO declaration not read: %s", line, expr)
	}

	switch {
	case strings.HasPrefix(expr, "!"):
		// LINGO 註解，不是敘述。
		return nil
	case strings.HasPrefix(strings.ToUpper(expr), "MIN="), strings.HasPrefix(strings.ToUpper(expr), "MAX="):
		if !lingoIsLinear(strings.TrimSpace(strings.SplitN(expr, "=", 2)[1])) {
			return fmt.Errorf("line %d: invalid linear objective: %s", line, expr)
		}
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
		parts := lingoRelSplitRe.Split(expr, -1)
		if len(parts) < 2 {
			return fmt.Errorf("line %d: invalid relation: %s", line, expr)
		}
		for _, part := range parts {
			if !lingoIsLinear(strings.TrimSpace(part)) {
				return fmt.Errorf("line %d: invalid linear expression: %s", line, expr)
			}
		}
		lingoAddRelation(model, expr)
	default:
		return fmt.Errorf("line %d: LINGO statement not recognised: %s", line, expr)
	}
	return nil
}

func lingoIsLinear(expr string) bool {
	tokens := strings.Fields(expr)
	if len(tokens) == 0 {
		return false
	}
	expectTerm, previousNumber := true, false
	for _, token := range tokens {
		if token == "+" || token == "-" {
			if expectTerm {
				continue
			}
			expectTerm, previousNumber = true, false
			continue
		}
		if !expectTerm && (!previousNumber || !lingoBareVarRe.MatchString(token)) {
			return false
		}
		if lingoIsNumber(token) {
			if !expectTerm {
				return false
			}
			previousNumber, expectTerm = true, false
			continue
		}
		if lingoBareVarRe.MatchString(token) {
			if !expectTerm && !previousNumber {
				return false
			}
			previousNumber, expectTerm = false, false
			continue
		}
		return false
	}
	return !expectTerm
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
	lp, err := parseLingo(strings.NewReader(model))
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

	lp, err := parseLingo(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read LINGO model %s: %w", path, err)
	}
	return lp, nil
}

// Compiled once: the parser rebuilt these on every call.
var (
	lingoRowLabelRe       = regexp.MustCompile(`^\[\_\d+\]\s*`)
	lingoSpaceRe          = regexp.MustCompile(`\s+`)
	lingoMissingSpaceRe   = regexp.MustCompile(`([a-zA-Z_0-9]+)([+-])`)
	lingoSciNotationRe    = regexp.MustCompile(`(\d)([eE])\s*([+-]?\d+)`)
	lingoLinearMultiplyRe = regexp.MustCompile(`([+-]?(?:\d+\.?\d*|\.\d+)(?:[eE][+-]?\d+)?)\s*\*\s*([A-Za-z_][A-Za-z0-9_]*)`)
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
