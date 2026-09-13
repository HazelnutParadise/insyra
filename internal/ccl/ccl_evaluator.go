package ccl

import (
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/HazelnutParadise/insyra/internal/utils"
)

// maxEvalDepth guards against pathological stack growth. It must be well above
// the AST depth of any legitimate expression: a long flat expression such as
// A+B+C+... nests one binary node per term, so a low cap (previously 100) would
// wrongly reject valid formulas. A finite compiled AST cannot recurse infinitely
// (runaway recursion via registered functions is bounded separately by
// funcCallDepth), so a high cap is safe.
var maxEvalDepth = 10000

// EvaluationResult represents the result of evaluating a CCL statement
type EvaluationResult struct {
	Value        any    // The computed value (for expressions)
	IsAssignment bool   // Whether this was an assignment
	Target       string // Assignment target column (if IsAssignment)
	IsNewCol     bool   // Whether this creates a new column
	NewColName   string // New column name (if IsNewCol)
}

// Evaluate evaluates a CCL node with the given context.
func Evaluate(n cclNode, ctx Context) (any, error) {
	return evaluateWithContext(n, ctx, 0)
}

// EvaluateStatement evaluates a CCL statement and returns detailed result
func EvaluateStatement(n cclNode, ctx Context) (*EvaluationResult, error) {
	switch t := n.(type) {
	case *cclAssignmentNode:
		// Evaluate the expression
		val, err := evaluateWithContext(t.expr, ctx, 0)
		if err != nil {
			return nil, err
		}
		return &EvaluationResult{
			Value:        val,
			IsAssignment: true,
			Target:       t.target,
		}, nil
	case *cclNewColNode:
		// Evaluate the expression for new column
		val, err := evaluateWithContext(t.expr, ctx, 0)
		if err != nil {
			return nil, err
		}
		return &EvaluationResult{
			Value:      val,
			IsNewCol:   true,
			NewColName: t.colName,
		}, nil
	default:
		// Regular expression
		val, err := evaluateWithContext(n, ctx, 0)
		if err != nil {
			return nil, err
		}
		return &EvaluationResult{Value: val}, nil
	}
}

// GetAssignmentTarget returns the target column name/index if the node is an assignment
func GetAssignmentTarget(n cclNode) (string, bool) {
	if assign, ok := n.(*cclAssignmentNode); ok {
		return assign.target, true
	}
	return "", false
}

// GetNewColInfo returns the new column info if the node is a NEW function
func GetNewColInfo(n cclNode) (string, cclNode, bool) {
	if newCol, ok := n.(*cclNewColNode); ok {
		return newCol.colName, newCol.expr, true
	}
	return "", nil, false
}

// GetExpressionNode returns the underlying expression node
func GetExpressionNode(n cclNode) cclNode {
	switch t := n.(type) {
	case *cclAssignmentNode:
		return t.expr
	case *cclNewColNode:
		return t.expr
	default:
		return n
	}
}

// IsAssignmentNode checks if the node is an assignment
func IsAssignmentNode(n cclNode) bool {
	_, ok := n.(*cclAssignmentNode)
	return ok
}

// IsNewColNode checks if the node is a NEW column creation
func IsNewColNode(n cclNode) bool {
	_, ok := n.(*cclNewColNode)
	return ok
}

// IsRowDependent checks if the expression depends on the current row.
func IsRowDependent(n cclNode) bool {
	switch t := n.(type) {
	case *cclNumberNode, *cclStringNode, *cclBooleanNode, *cclNilNode, *cclFoldedValueNode:
		return false
	case *cclIdentifierNode, *cclColIndexNode, *cclColNameNode, *cclResolvedColNode, *cclAtNode, *cclRowIndexNode:
		return true
	case *cclBinaryOpNode:
		if t.op == "." {
			return IsRowDependent(t.right)
		}
		if t.op == ":" {
			// 範圍運算符特殊處理：如果兩邊都是靜態欄位引用，則視為行無關
			if isStaticColumnNode(t.left) && isStaticColumnNode(t.right) {
				return false
			}
		}
		return IsRowDependent(t.left) || IsRowDependent(t.right)
	case *cclFoldChainNode:
		// 摺疊鏈不含 '.' 與 ':'（parser 保證），無需比照二元節點做特殊處理。
		if IsRowDependent(t.init) {
			return true
		}
		for _, operand := range t.operands {
			if IsRowDependent(operand) {
				return true
			}
		}
		return false
	case *cclChainedComparisonNode:
		for _, v := range t.values {
			if IsRowDependent(v) {
				return true
			}
		}
		return false
	case *funcCallNode:
		upper := strings.ToUpper(t.name)
		// Sequence functions (LAG, CUMSUM, ROLLING_MEAN, ...) consume whole
		// columns and produce same-length output; they are evaluated once
		// per expression, not per row.
		if IsSequenceFunction(upper) {
			return false
		}
		// Aggregate functions are row-independent unless # appears in args.
		if _, isAgg := lookupAggregateFunction(upper); isAgg {
			return containsRowIndex(t)
		}
		for _, arg := range t.args {
			if IsRowDependent(arg) {
				return true
			}
		}
		return false
	case *cclAssignmentNode:
		return IsRowDependent(t.expr)
	case *cclNewColNode:
		return IsRowDependent(t.expr)
	default:
		return true // Default to dependent for safety
	}
}

func isStaticColumnNode(n cclNode) bool {
	switch n.(type) {
	case *cclIdentifierNode, *cclColIndexNode, *cclColNameNode, *cclResolvedColNode:
		return true
	default:
		return false
	}
}

// containsRowIndex 檢查表達式是否包含 # 運算符
func containsRowIndex(n cclNode) bool {
	switch t := n.(type) {
	case *cclRowIndexNode:
		return true
	case *cclBinaryOpNode:
		return containsRowIndex(t.left) || containsRowIndex(t.right)
	case *cclFoldChainNode:
		if containsRowIndex(t.init) {
			return true
		}
		for _, operand := range t.operands {
			if containsRowIndex(operand) {
				return true
			}
		}
		return false
	case *cclChainedComparisonNode:
		for _, v := range t.values {
			if containsRowIndex(v) {
				return true
			}
		}
		return false
	case *funcCallNode:
		for _, arg := range t.args {
			if containsRowIndex(arg) {
				return true
			}
		}
		return false
	case *cclAssignmentNode:
		return containsRowIndex(t.expr)
	case *cclNewColNode:
		return containsRowIndex(t.expr)
	default:
		return false
	}
}

func evaluateWithContext(n cclNode, ctx Context, depth int) (any, error) {
	return evaluateWithCallDepth(n, ctx, depth, 0)
}

func evaluateWithCallDepth(n cclNode, ctx Context, depth, callDepth int) (any, error) {
	if depth > maxEvalDepth {
		return nil, fmt.Errorf("evaluate: maximum recursion depth exceeded (%d), possibly infinite recursion", maxEvalDepth)
	}

	switch t := n.(type) {
	case *cclNumberNode:
		return t.value, nil
	case *cclStringNode:
		return t.value, nil
	case *cclBooleanNode:
		return t.value, nil
	case *cclNilNode:
		return nil, nil
	case *cclFoldedValueNode:
		return t.value, nil
	case *cclAtNode:
		return ctx.GetCurrentRow(), nil
	case *cclRowIndexNode:
		return float64(ctx.GetRowIndex()), nil
	case *cclIdentifierNode:
		idx, ok := utils.ParseColIndex(t.name)
		if !ok {
			return nil, fmt.Errorf("invalid column index: %s", t.name)
		}
		return ctx.GetCol(idx), nil
	case *cclColIndexNode:
		// [A] 形式的欄位索引引用
		idx, ok := utils.ParseColIndex(t.index)
		if !ok {
			return nil, fmt.Errorf("invalid column index: %s", t.index)
		}
		return ctx.GetCol(idx), nil
	case *cclColNameNode:
		// ['colName'] 形式的欄位名稱引用
		return ctx.GetColByName(t.name)
	case *cclResolvedColNode:
		return ctx.GetCol(t.index), nil
	case *funcCallNode:
		// Short-circuit special-casing for logical/conditional functions to avoid evaluating
		// arguments that could cause out-of-range access (e.g., IF(#>0, A.(#-1), NULL)).
		upper := strings.ToUpper(t.name)
		if upper == "IF" {
			if len(t.args) != 3 {
				return nil, fmt.Errorf("IF requires 3 arguments")
			}
			functionDepth := callDepth + 1
			condVal, err := evaluateWithCallDepth(t.args[0], ctx, depth+1, functionDepth)
			if err != nil {
				return nil, err
			}
			cond, ok := toBool(condVal)
			if !ok {
				return nil, fmt.Errorf("first argument to IF cannot be converted to boolean: %T", condVal)
			}
			if cond {
				return evaluateWithCallDepth(t.args[1], ctx, depth+1, functionDepth)
			}
			return evaluateWithCallDepth(t.args[2], ctx, depth+1, functionDepth)
		}

		if upper == "AND" {
			functionDepth := callDepth + 1
			for _, arg := range t.args {
				val, err := evaluateWithCallDepth(arg, ctx, depth+1, functionDepth)
				if err != nil {
					return nil, err
				}
				if b, ok := toBool(val); !ok || !b {
					return false, nil
				}
			}
			return true, nil
		}

		if upper == "OR" {
			functionDepth := callDepth + 1
			for _, arg := range t.args {
				val, err := evaluateWithCallDepth(arg, ctx, depth+1, functionDepth)
				if err != nil {
					return nil, err
				}
				if b, ok := toBool(val); ok && b {
					return true, nil
				}
			}
			return false, nil
		}

		// Sequence functions: whole-column input, same-length-column output.
		// Args are evaluated to columns (mirroring aggregate path); the
		// returned []any is consumed directly by the assigner / cell broadcaster
		// at the top level. Nested usage inside arithmetic is not supported
		// in v1 and may produce undefined results.
		if IsSequenceFunction(upper) {
			functionDepth := callDepth + 1
			seqArgs := make([][]any, len(t.args))
			for i, arg := range t.args {
				colData, err := evaluateToColumn(arg, ctx, depth+1, functionDepth)
				if err != nil {
					return nil, fmt.Errorf("sequence function %s: %v", t.name, err)
				}
				seqArgs[i] = colData
			}
			return callSequenceFunction(t.name, seqArgs)
		}

		// 檢查是否為聚合函數
		if _, isAgg := lookupAggregateFunction(upper); isAgg {
			functionDepth := callDepth + 1
			// 如果是行相關的（包含 #），則將其視為普通函數評估（逐行聚合）
			if containsRowIndex(t) {
				args := []any{}
				for _, arg := range t.args {
					val, err := evaluateWithCallDepth(arg, ctx, depth+1, functionDepth)
					if err != nil {
						return nil, err
					}
					args = append(args, val)
				}
				// 呼叫聚合函數，但傳遞的是單行資料（包裝成 []any）
				aggArgs := make([][]any, len(args))
				for i, arg := range args {
					if slice, ok := arg.([]any); ok {
						aggArgs[i] = slice
					} else {
						aggArgs[i] = []any{arg}
					}
				}
				return callAggregateFunction(t.name, aggArgs)
			}

			aggArgs := make([][]any, len(t.args))
			for i, arg := range t.args {
				// 聚合函數的參數必須是欄位引用或能產生整欄資料的表達式
				colData, err := evaluateToColumn(arg, ctx, depth+1, functionDepth)
				if err != nil {
					return nil, fmt.Errorf("aggregate function %s: %v", t.name, err)
				}
				aggArgs[i] = colData
			}
			return callAggregateFunction(t.name, aggArgs)
		}

		// Default: evaluate all args then call function
		functionDepth := callDepth + 1
		args := []any{}
		for _, arg := range t.args {
			val, err := evaluateWithCallDepth(arg, ctx, depth+1, functionDepth)
			if err != nil {
				return nil, err
			}
			args = append(args, val)
		}
		return callFunction(t.name, args, callDepth)
	case *cclBinaryOpNode:
		if t.op == "." {
			return evaluateRowAccess(t.left, t.right, ctx, depth, callDepth)
		}
		// 特殊處理 : 運算符，因為它可能涉及欄位索引的解析，而不是值的評估
		if t.op == ":" {
			return evaluateRange(t.left, t.right, ctx, depth, callDepth)
		}
		left, err := evaluateWithCallDepth(t.left, ctx, depth+1, callDepth)
		if err != nil {
			return nil, err
		}
		right, err := evaluateWithCallDepth(t.right, ctx, depth+1, callDepth)
		if err != nil {
			return nil, err
		}
		return applyOperator(t.op, left, right)
	case *cclFoldChainNode:
		// 左摺疊，運算序列與巢狀二元節點完全一致：逐一「求值運算元、套用
		// 運算子」由左至右，錯誤同樣先到先回。不引入短路（二元的 && / ||
		// 本來就不短路，行為必須一致）。
		acc, err := evaluateWithCallDepth(t.init, ctx, depth+1, callDepth)
		if err != nil {
			return nil, err
		}
		for i, operand := range t.operands {
			rv, err := evaluateWithCallDepth(operand, ctx, depth+1, callDepth)
			if err != nil {
				return nil, err
			}
			acc, err = applyOperator(t.ops[i], acc, rv)
			if err != nil {
				return nil, err
			}
		}
		return acc, nil
	case *cclChainedComparisonNode:
		// 處理連續比較運算
		if len(t.values) != len(t.ops)+1 {
			return nil, fmt.Errorf("invalid chained comparison: number of values (%d) should be one more than number of operators (%d)", len(t.values), len(t.ops))
		}

		// 評估所有值
		values := make([]any, len(t.values))
		for i, valNode := range t.values {
			val, err := evaluateWithCallDepth(valNode, ctx, depth+1, callDepth)
			if err != nil {
				return nil, err
			}
			values[i] = val
		}

		// 逐對比較所有值
		for i := 0; i < len(t.ops); i++ {
			result, err := applyOperator(t.ops[i], values[i], values[i+1])
			if err != nil {
				return nil, err
			}

			// 如果任何一個比較結果為假，整個連續比較的結果就為假
			boolResult, ok := result.(bool)
			if !ok {
				return nil, fmt.Errorf("comparison did not result in a boolean: %v", result)
			}
			if !boolResult {
				return false, nil
			}
		}

		// 所有比較都為真，整個結果為真
		return true, nil
	}

	return nil, fmt.Errorf("invalid node")
}

// FoldRowInvariantAggregates replaces every aggregate call whose answer cannot
// change from row to row with that answer, so the per-row loop does not
// recompute it. On 20,000 rows this is the difference between `A / SUM(A)`
// taking two seconds and taking a millisecond: SUM(A) was summing the whole
// column once per row.
//
// An aggregate that mentions `#` reads the current row and is left alone. So is
// one whose evaluation fails: returning the tree unchanged lets the normal
// per-row path raise the error where it always did, with the row it belongs to,
// rather than moving the message somewhere the caller does not expect.
func FoldRowInvariantAggregates(n cclNode, ctx Context) cclNode {
	folded, _ := foldAggregates(n, ctx)
	return folded
}

// foldAggregates returns the rewritten node and whether anything changed.
func foldAggregates(n cclNode, ctx Context) (cclNode, bool) {
	switch t := n.(type) {
	case *funcCallNode:
		upper := strings.ToUpper(t.name)
		if _, isAgg := lookupAggregateFunction(upper); isAgg && !containsRowIndex(t) {
			val, err := evaluateWithCallDepth(t, ctx, 0, 0)
			if err != nil {
				return n, false
			}
			// Only a scalar is safe to inline. A column-shaped result belongs
			// to the sequence path, which spreads it across the rows itself.
			if _, isSlice := val.([]any); isSlice {
				return n, false
			}
			return literalNode(val), true
		}
		changed := false
		args := make([]cclNode, len(t.args))
		for i, arg := range t.args {
			a, c := foldAggregates(arg, ctx)
			args[i] = a
			changed = changed || c
		}
		if !changed {
			return n, false
		}
		return &funcCallNode{name: t.name, args: args}, true
	case *cclBinaryOpNode:
		// ':' resolves column references structurally; folding either side
		// would turn `A:B` into numbers and change what it means.
		if t.op == ":" {
			return n, false
		}
		left, lc := foldAggregates(t.left, ctx)
		right, rc := foldAggregates(t.right, ctx)
		if !lc && !rc {
			return n, false
		}
		return &cclBinaryOpNode{op: t.op, left: left, right: right}, true
	case *cclFoldChainNode:
		init, changed := foldAggregates(t.init, ctx)
		operands := make([]cclNode, len(t.operands))
		for i, operand := range t.operands {
			o, c := foldAggregates(operand, ctx)
			operands[i] = o
			changed = changed || c
		}
		if !changed {
			return n, false
		}
		return &cclFoldChainNode{init: init, ops: t.ops, operands: operands}, true
	case *cclChainedComparisonNode:
		changed := false
		values := make([]cclNode, len(t.values))
		for i, v := range t.values {
			nv, c := foldAggregates(v, ctx)
			values[i] = nv
			changed = changed || c
		}
		if !changed {
			return n, false
		}
		return &cclChainedComparisonNode{ops: t.ops, values: values}, true
	case *cclAssignmentNode:
		expr, changed := foldAggregates(t.expr, ctx)
		if !changed {
			return n, false
		}
		return &cclAssignmentNode{target: t.target, expr: expr}, true
	case *cclNewColNode:
		expr, changed := foldAggregates(t.expr, ctx)
		if !changed {
			return n, false
		}
		return &cclNewColNode{colName: t.colName, expr: expr}, true
	}
	return n, false
}

// literalNode wraps an already-computed value so the evaluator can read it back
// without recomputing anything.
func literalNode(v any) cclNode {
	switch x := v.(type) {
	case float64:
		return &cclNumberNode{value: x}
	case string:
		return &cclStringNode{value: x}
	case bool:
		return &cclBooleanNode{value: x}
	case nil:
		return &cclNilNode{}
	}
	return &cclFoldedValueNode{value: v}
}

// clampedInt turns a character count, character position or digit count
// into an int, clamping it to the int32 range first. A bare int(f) past the
// int range differs by platform: amd64 gives the most negative int and arm64
// the most positive, so MID('abc', 2, 10^300) was "" on amd64 and "bc" on
// arm64. Clamped, a count past the end of a string means "to the end"
// everywhere. NaN is no count at all.
func clampedInt(f float64, what string) (int, error) {
	if math.IsNaN(f) {
		return 0, fmt.Errorf("%s must be a number, got NaN", what)
	}
	return int(max(min(f, math.MaxInt32), math.MinInt32)), nil
}

// durationOf converts f units into a Duration. time.Duration(f) is undefined
// past ±2^63 nanoseconds (about 292 years) and for NaN, and the platforms
// disagree there, so those report false instead.
func durationOf(f float64, unit time.Duration) (time.Duration, bool) {
	d := f * float64(unit)
	if math.IsNaN(d) || d >= 1<<63 || d < -(1<<63) {
		return 0, false
	}
	return time.Duration(d), true
}

// dayShift converts a number of days added to or subtracted from a date into
// the Duration the date moves by. It keeps the arithmetic date ± number has
// always used, time.Duration(days*24) * time.Hour, so a fraction of an hour is
// dropped, and refuses only what that arithmetic cannot hold: NaN, the
// infinities, and a shift past about 292 years, where the conversion is
// undefined or the product wraps around.
func dayShift(days float64) (time.Duration, error) {
	hours := days * 24.0
	if math.IsNaN(hours) || hours >= 1<<63 || hours < -(1<<63) {
		return 0, fmt.Errorf("a shift of %v days is out of range", days)
	}
	h := time.Duration(hours)
	if h > math.MaxInt64/time.Hour || h < math.MinInt64/time.Hour {
		return 0, fmt.Errorf("a shift of %v days is out of range", days)
	}
	return h * time.Hour, nil
}

// rangeBound turns a row range bound into an int the way the range operator
// always has, dropping a fraction. NaN, the infinities and values past the
// int32 range are refused: int(f) is undefined past the int range, and amd64
// and arm64 disagree there.
func rangeBound(f float64, what string) (int, error) {
	if math.IsNaN(f) || f > math.MaxInt32 || f < math.MinInt32 {
		return 0, fmt.Errorf("row range %s %v is out of range", what, f)
	}
	return int(f), nil
}

func applyOperator(op string, left, right any) (any, error) {
	// Try to interpret date-like operands first (time.Time or parseable date strings)
	parseTimeLike := func(v any) (time.Time, bool) {
		switch x := v.(type) {
		case time.Time:
			return x, true
		case string:
			// Every layout below starts with a four-digit year, so a string
			// that does not start with a digit cannot match any of them.
			// Skipping the four time.Parse calls takes a 100k-row text column
			// from 45ms to roughly what a numeric one costs.
			if len(x) == 0 || x[0] < '0' || x[0] > '9' {
				return time.Time{}, false
			}
			formats := []string{time.RFC3339, time.RFC3339Nano, "2006-01-02", "2006-01-02T15:04:05Z07:00"}
			for _, f := range formats {
				if t, err := time.Parse(f, x); err == nil {
					return t, true
				}
			}
		}
		return time.Time{}, false
	}

	// Special-case: date/time arithmetic and comparisons
	if lt, lok := parseTimeLike(left); lok {
		if rt, rok := parseTimeLike(right); rok {
			// both are times
			switch op {
			case "-":
				// return difference as time.Duration (left - right)
				return lt.Sub(rt), nil
			case ">":
				return lt.After(rt), nil
			case "<":
				return lt.Before(rt), nil
			case ">=":
				return !lt.Before(rt), nil
			case "<=":
				return !lt.After(rt), nil
			case "==":
				return lt.Equal(rt), nil
			case "!=":
				return !lt.Equal(rt), nil
			case "+":
				return nil, fmt.Errorf("operator + not supported between two dates")
			case "*", "/", "^":
				return nil, fmt.Errorf("operator %s not supported between dates", op)
			}
		}
		// right is not time; if it's numeric, allow time +/- number (days)
		if rf, ok := toFloat64(right); ok {
			switch op {
			case "+":
				d, err := dayShift(rf)
				if err != nil {
					return nil, err
				}
				return lt.Add(d), nil
			case "-":
				d, err := dayShift(rf)
				if err != nil {
					return nil, err
				}
				return lt.Add(-d), nil
			}
		}
	}

	// symmetric: left is not time but right might be (e.g., number + date)
	if rt, rok := parseTimeLike(right); rok {
		if lf, ok := toFloat64(left); ok {
			switch op {
			case "+":
				d, err := dayShift(lf)
				if err != nil {
					return nil, err
				}
				return rt.Add(d), nil
			case "-":
				// number - date doesn't make sense
				return nil, fmt.Errorf("invalid operands for -: %v, %v", left, right)
			}
		}
	}

	// 特殊情況處理：其中一方為nil
	if left == nil || right == nil {
		switch op {
		case "==":
			// 只有當兩者都是nil時才相等
			return left == nil && right == nil, nil
		case "!=":
			// 只要有一方不是nil就不相等
			return left != nil || right != nil, nil
		case ">", "<", ">=", "<=":
			// nil與任何值進行大小比較都返回false
			return false, nil
		case "+", "-", "*", "/": // Added "^" to this line in the original thought process, but it's handled by toFloat64
			// 將 nil 視為 0 進行算術運算
			lf, lok := toFloat64(left)
			rf, rok := toFloat64(right)
			if !lok || !rok { // Should not happen if toFloat64 handles nil
				return nil, fmt.Errorf("internal error: failed to convert nil to float64 for operator %s", op)
			}
			switch op {
			case "+":
				return lf + rf, nil
			case "-":
				return lf - rf, nil
			case "*":
				return lf * rf, nil
			case "/":
				if rf == 0 {
					return nil, fmt.Errorf("division by zero")
				}
				return lf / rf, nil
			}
		}
	}

	// 處理 & 運算符（字串連接，等同於 CONCAT）
	if op == "&" {
		leftStr := fmt.Sprintf("%v", left)
		rightStr := fmt.Sprintf("%v", right)
		return leftStr + rightStr, nil
	}
	// 處理 && 運算符（邏輯與，等同於 AND）
	if op == "&&" {
		lb, lok := toBool(left)
		rb, rok := toBool(right)
		if !lok || !rok {
			return nil, fmt.Errorf("invalid operands for &&: %v, %v (both must be boolean)", left, right)
		}
		return lb && rb, nil
	}

	// 處理 || 運算符（邏輯或，等同於 OR）
	if op == "||" {
		lb, lok := toBool(left)
		rb, rok := toBool(right)
		if !lok || !rok {
			return nil, fmt.Errorf("invalid operands for ||: %v, %v (both must be boolean)", left, right)
		}
		return lb || rb, nil
	}

	// 對於比較運算，嘗試將值轉換為數字進行比較
	lf, lok := toFloat64(left)
	rf, rok := toFloat64(right)
	if lok && rok {
		// 兩者都能轉換為數字，使用數字比較
		switch op {
		case "+":
			return lf + rf, nil
		case "-":
			return lf - rf, nil
		case "*":
			return lf * rf, nil
		case "/":
			// Consistent with the nil-operand path and the '%' operator: division
			// by zero is a reported error, not a silent +Inf.
			if rf == 0 {
				return nil, fmt.Errorf("division by zero")
			}
			return lf / rf, nil
		case "%":
			// '%' is tokenized and given precedence; implement it here (mirrors
			// the MOD function) instead of erroring on every use.
			if rf == 0 {
				return nil, fmt.Errorf("modulo by zero")
			}
			return math.Mod(lf, rf), nil
		case "^":
			return math.Pow(lf, rf), nil
		case ">":
			return lf > rf, nil
		case "<":
			return lf < rf, nil
		case ">=":
			return lf >= rf, nil
		case "<=":
			return lf <= rf, nil
		case "==":
			return lf == rf, nil
		case "!=":
			return lf != rf, nil
		default:
			return nil, fmt.Errorf("unsupported operator: %s", op)
		}
	}

	// 如果不能都轉換為數字，對於==和!=，檢查是否都是字串
	if op == "==" || op == "!=" {
		ls, lokStr := left.(string)
		rs, rokStr := right.(string)
		if lokStr && rokStr {
			if op == "==" {
				return ls == rs, nil
			} else {
				return ls != rs, nil
			}
		}

		// 如果是布林值比較
		lb, lokBool := left.(bool)
		rb, rokBool := right.(bool)
		if lokBool && rokBool {
			if op == "==" {
				return lb == rb, nil
			} else {
				return lb != rb, nil
			}
		}

		// 不同數據類型間的比較，直接返回false或true
		if op == "==" {
			return false, nil
		} else { // op == "!="
			return true, nil
		}
	}

	// 對於大小比較，如果不能轉換為數字，返回false
	if op == ">" || op == "<" || op == ">=" || op == "<=" {
		return false, nil
	}

	// 處理範圍運算符 :
	if op == ":" {
		// 1. 數字範圍 (Row Range)
		lf, lok := toFloat64(left)
		rf, rok := toFloat64(right)
		if lok && rok {
			start, err := rangeBound(lf, "start")
			if err != nil {
				return nil, err
			}
			end, err := rangeBound(rf, "end")
			if err != nil {
				return nil, err
			}
			// 支援負數索引
			// 但這裡我們不知道總行數，所以無法在這裡處理負數索引轉換
			// 負數索引轉換應該在 evaluateRowAccess 中處理
			// 這裡只返回原始範圍
			return []int{start, end}, nil
		}

		// 2. 欄位範圍 (Column Range)
		// 這裡我們假設 left 和 right 已經被解析為欄位索引或名稱
		// 但 evaluateWithContext 對於 Identifier 會返回欄位值
		// 如果我們想要欄位範圍，我們需要在 evaluateToColumn 或 evaluateRowAccess 中特殊處理
		// 或者在這裡返回一個特殊的 ColumnRange 對象
		// 但 left 和 right 已經是值了
		// 如果是 A:C，A 會被評估為 A 欄的值（如果行相關）或 A 欄第一列的值（如果行無關）
		// 這不是我們想要的。
		// 我們想要的是 A 和 C 的索引。
		// 這意味著 : 運算符不能像普通運算符那樣先評估左右運算元。
		// 它需要在 evaluateWithContext 中特殊處理 BinaryOpNode{op: ":"}
	}

	return nil, fmt.Errorf("invalid operands for %s: %v, %v", op, left, right)
}

func evaluateToColumn(n cclNode, ctx Context, depth, callDepth int) ([]any, error) {
	// 1. 針對直接欄位引用的優化
	switch t := n.(type) {
	case *cclAtNode:
		// Special case for @ in aggregate functions: return all data in the context
		return ctx.GetAllData()
	case *cclIdentifierNode:
		idx, ok := utils.ParseColIndex(t.name)
		if !ok {
			return nil, fmt.Errorf("invalid column index: %s", t.name)
		}
		// Try index
		col, err := ctx.GetColData(idx)
		if err == nil {
			return col, nil
		}
		// Try name
		return ctx.GetColDataByName(t.name)
	case *cclColIndexNode:
		idx, ok := utils.ParseColIndex(t.index)
		if !ok {
			return nil, fmt.Errorf("invalid column index: %s", t.index)
		}
		return ctx.GetColData(idx)
	case *cclColNameNode:
		return ctx.GetColDataByName(t.name)
	case *cclResolvedColNode:
		return ctx.GetColData(t.index)
	}

	// 2. 檢查是否為行無關表達式（如 A.0, @.0, 1+2）
	if !IsRowDependent(n) {
		// Evaluate once
		val, err := evaluateWithCallDepth(n, ctx, depth, callDepth)
		if err != nil {
			return nil, err
		}

		// Expand ranges
		if cr, ok := val.(ColumnRange); ok {
			var allData []any
			for i := cr.Start; i <= cr.End; i++ {
				col, err := ctx.GetColData(i)
				if err != nil {
					return nil, err
				}
				allData = append(allData, col...)
			}
			return allData, nil
		}

		if _, ok := val.(RowRange); ok {
			return nil, fmt.Errorf("raw row range cannot be used as a data source; use @.start:end instead")
		}

		// A sequence function (LAG, CUMSUM, ...) already produced a whole
		// column; hand it through so nesting keeps every row.
		if col, ok := val.([]any); ok && len(col) == ctx.GetRowCount() {
			return col, nil
		}
		// Otherwise a scalar: aggregates see it as a one-element column.
		return []any{val}, nil
	}

	// 3. 行相關表達式（如 A+B, IF(A>0, B, C)）：對每一行進行評估
	// We need to iterate all rows.
	rowCount := ctx.GetRowCount()
	results := make([]any, rowCount)

	// Save current row index to restore later
	originalRowIdx := ctx.GetRowIndex()
	defer func() {
		if err := ctx.SetRowIndex(originalRowIdx); err != nil {
			log.Printf("ccl: failed to restore row index: %v", err)
		}
	}()

	for i := 0; i < rowCount; i++ {
		if err := ctx.SetRowIndex(i); err != nil {
			return nil, err
		}
		val, err := evaluateWithCallDepth(n, ctx, depth, callDepth)
		if err != nil {
			return nil, err
		}

		// Expand ranges if returned
		if cr, ok := val.(ColumnRange); ok {
			var allData []any
			for j := cr.Start; j <= cr.End; j++ {
				col, err := ctx.GetColData(j)
				if err != nil {
					return nil, err
				}
				allData = append(allData, col...)
			}
			results[i] = allData
		} else if _, ok := val.(RowRange); ok {
			return nil, fmt.Errorf("raw row range cannot be used as a data source; use @.start:end instead")
		} else {
			results[i] = val
		}
	}
	return results, nil
}

func evaluateRowAccess(left, right cclNode, ctx Context, depth, callDepth int) (any, error) {
	// 1. Determine row index/indices from right
	rowVal, err := evaluateWithCallDepth(right, ctx, depth+1, callDepth)
	if err != nil {
		return nil, err
	}

	var rowIndices []int
	var isRowRange bool

	switch v := rowVal.(type) {
	case RowRange:
		isRowRange = true
		start, end := v.Start, v.End
		// Generate indices
		if start <= end {
			for i := start; i <= end; i++ {
				rowIndices = append(rowIndices, i)
			}
		}
	case float64:
		rowIndices = []int{int(v)}
	case int:
		rowIndices = []int{v}
	case string:
		idx, err := ctx.GetRowIndexByName(v)
		if err != nil {
			return nil, err
		}
		rowIndices = []int{idx}
	default:
		return nil, fmt.Errorf("invalid row index type: %T", rowVal)
	}

	// 2. Prepare to fetch data from left
	var colRange *ColumnRange
	var leftSimple = false

	switch left.(type) {
	case *cclAtNode, *cclResolvedColNode, *cclIdentifierNode, *cclColIndexNode, *cclColNameNode:
		leftSimple = true
	default:
		// Evaluate left once to see if it's a ColumnRange or other value
		lVal, err := evaluateWithCallDepth(left, ctx, depth+1, callDepth)
		if err != nil {
			return nil, err
		}
		if cr, ok := lVal.(ColumnRange); ok {
			colRange = &cr
		} else {
			// If it's not a ColumnRange, it might be an error or unsupported type for row access
			// But wait, if left is (1+2), we can't do (1+2).1
			// Row access is typically on columns.
			return nil, fmt.Errorf("invalid left operand for row access: %T", lVal)
		}
	}

	results := make([]any, 0, len(rowIndices))

	for _, rIdx := range rowIndices {
		var val any
		var err error

		if colRange != nil {
			// Return list of values for this row
			rowRes := make([]any, 0, colRange.End-colRange.Start+1)
			for c := colRange.Start; c <= colRange.End; c++ {
				v, err := ctx.GetCell(c, rIdx)
				if err != nil {
					return nil, err
				}
				rowRes = append(rowRes, v)
			}
			val = rowRes
		} else if leftSimple {
			switch l := left.(type) {
			case *cclAtNode:
				val, err = ctx.GetRowAt(rIdx)
			case *cclResolvedColNode:
				if l.index != -1 {
					val, err = ctx.GetCell(l.index, rIdx)
				} else {
					val, err = ctx.GetCellByName(l.name, rIdx)
				}
			case *cclIdentifierNode:
				idx, ok := utils.ParseColIndex(l.name)
				// Try as index first
				if ok {
					val, err = ctx.GetCell(idx, rIdx)
				} else {
					err = fmt.Errorf("invalid column index")
				}

				if err != nil {
					// Fallback to name
					val, err = ctx.GetCellByName(l.name, rIdx)
				}
			case *cclColIndexNode:
				idx, ok := utils.ParseColIndex(l.index)
				if ok {
					val, err = ctx.GetCell(idx, rIdx)
				} else {
					err = fmt.Errorf("invalid column index")
				}

				if err != nil {
					val, err = ctx.GetCellByName(l.index, rIdx)
				}
			case *cclColNameNode:
				val, err = ctx.GetCellByName(l.name, rIdx)
			}
		}

		if err != nil {
			return nil, err
		}
		results = append(results, val)
	}

	// If it was a single row access (not a range), return the single value
	if !isRowRange && len(results) == 1 {
		return results[0], nil
	}

	return results, nil
}

// ColumnRange represents a range of column indices [Start, End]
type ColumnRange struct {
	Start int
	End   int
}

// RowRange represents a range of row indices [Start, End]
type RowRange struct {
	Start int
	End   int
}

// describeNode names an AST node the way the caller wrote it, for an error
// message. Printing the node itself dumps a Go struct, pointers included.
func describeNode(n cclNode) string {
	switch t := n.(type) {
	case *cclIdentifierNode:
		return t.name
	case *cclColIndexNode:
		return "[" + t.index + "]"
	case *cclColNameNode:
		return "['" + t.name + "']"
	case *cclResolvedColNode:
		if t.name != "" {
			return t.name
		}
		return fmt.Sprintf("column %d", t.index)
	case *cclNumberNode:
		return strconv.FormatFloat(t.value, 'g', -1, 64)
	case *cclStringNode:
		return "'" + t.value + "'"
	case *cclBooleanNode:
		return strconv.FormatBool(t.value)
	case *cclNilNode:
		return "nil"
	case *cclFoldedValueNode:
		return fmt.Sprint(t.value)
	case *cclAtNode:
		return "@"
	case *cclRowIndexNode:
		return "#"
	case *funcCallNode:
		return t.name + "(...)"
	case *cclBinaryOpNode:
		return describeNode(t.left) + " " + t.op + " " + describeNode(t.right)
	default:
		return "expression"
	}
}

func evaluateRange(left, right cclNode, ctx Context, depth, callDepth int) (any, error) {
	// 1. 嘗試解析為欄位範圍 (Column Range)
	// 檢查左右是否為欄位引用
	lIdx, lIsCol := resolveColumnIndex(left, ctx)
	rIdx, rIsCol := resolveColumnIndex(right, ctx)

	if lIsCol && rIsCol {
		// Check for invalid indices (-1) which resolveColumnIndex might return if not found but "looks like" a column
		if lIdx == -1 {
			return nil, fmt.Errorf("column not found: %s", describeNode(left))
		}
		if rIdx == -1 {
			return nil, fmt.Errorf("column not found: %s", describeNode(right))
		}

		// Check bounds
		colCount := ctx.GetColCount()
		if lIdx < 0 || lIdx >= colCount {
			return nil, fmt.Errorf("column index %d out of range (total columns: %d)", lIdx, colCount)
		}
		if rIdx < 0 || rIdx >= colCount {
			return nil, fmt.Errorf("column index %d out of range (total columns: %d)", rIdx, colCount)
		}

		if lIdx > rIdx {
			return nil, fmt.Errorf("invalid column range: start index %d > end index %d", lIdx, rIdx)
		}
		return ColumnRange{Start: lIdx, End: rIdx}, nil
	}

	// 2. 嘗試解析為數字範圍 (Row Range)
	// 這裡我們需要評估表達式，因為可能是 1+1 : 5
	lVal, err := evaluateWithCallDepth(left, ctx, depth+1, callDepth)
	if err != nil {
		return nil, err
	}
	rVal, err := evaluateWithCallDepth(right, ctx, depth+1, callDepth)
	if err != nil {
		return nil, err
	}

	lf, lok := toFloat64(lVal)
	rf, rok := toFloat64(rVal)

	if lok && rok {
		lRowIdx, err := rangeBound(lf, "start")
		if err != nil {
			return nil, err
		}
		rRowIdx, err := rangeBound(rf, "end")
		if err != nil {
			return nil, err
		}
		rowCount := ctx.GetRowCount()
		if lRowIdx < 0 || lRowIdx >= rowCount {
			return nil, fmt.Errorf("row index %d out of range (total rows: %d)", lRowIdx, rowCount)
		}
		if rRowIdx < 0 || rRowIdx >= rowCount {
			return nil, fmt.Errorf("row index %d out of range (total rows: %d)", rRowIdx, rowCount)
		}
		return RowRange{Start: lRowIdx, End: rRowIdx}, nil
	}

	// 3. 嘗試解析為行名稱或混合範圍 (Row Name/Index Range)
	// Helper to resolve row index from value (int/float or string)
	resolveRowIdx := func(val any) (int, error) {
		if f, ok := toFloat64(val); ok {
			return rangeBound(f, "bound")
		}
		if s, ok := val.(string); ok {
			return ctx.GetRowIndexByName(s)
		}
		return 0, fmt.Errorf("invalid row reference: %v", val)
	}

	lRowIdx, lErr := resolveRowIdx(lVal)
	rRowIdx, rErr := resolveRowIdx(rVal)

	if lErr == nil && rErr == nil {
		rowCount := ctx.GetRowCount()
		if lRowIdx < 0 || lRowIdx >= rowCount {
			return nil, fmt.Errorf("row index %d out of range (total rows: %d)", lRowIdx, rowCount)
		}
		if rRowIdx < 0 || rRowIdx >= rowCount {
			return nil, fmt.Errorf("row index %d out of range (total rows: %d)", rRowIdx, rowCount)
		}
		return RowRange{Start: lRowIdx, End: rRowIdx}, nil
	}

	// %v on the nodes printed the AST structs, pointers and all.
	return nil, fmt.Errorf("invalid range operands: %s : %s", describeNode(left), describeNode(right))
}

func resolveColumnIndex(n cclNode, ctx Context) (int, bool) {
	switch t := n.(type) {
	case *cclIdentifierNode:
		idx, ok := utils.ParseColIndex(t.name)
		if ok {
			return idx, true
		}
		// Try to resolve by name
		if idx, err := ctx.GetColIndexByName(t.name); err == nil {
			return idx, true
		}
		// If not found, return -1 but indicate it WAS an identifier/name attempt
		// This allows the caller to decide if -1 is an error or just "not a column"
		// But wait, if we return true, the caller thinks it IS a column index.
		// If we return -1, the caller might use it as an index.
		// We should return false if it's not a valid column.
		return -1, false
	case *cclColIndexNode:
		idx, ok := utils.ParseColIndex(t.index)
		if !ok {
			return -1, false
		}
		return idx, true
	case *cclResolvedColNode:
		return t.index, true
	case *cclColNameNode:
		if idx, err := ctx.GetColIndexByName(t.name); err == nil {
			return idx, true
		}
		return -1, false
	}
	return -1, false
}
