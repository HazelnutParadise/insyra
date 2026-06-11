package ccl

import (
	"fmt"
	"strings"
)

type Func = func(args ...any) (any, error)
type AggFunc = func(args ...[]any) (any, error)

// SeqFunc is the signature for CCL sequence (window / lag / cumulative)
// functions. Unlike scalar functions (one row in, one value out) and
// aggregate functions (whole columns in, single value out broadcast back),
// sequence functions take whole columns and return a same-length column,
// enabling LAG, CUMSUM, ROLLING_MEAN, etc.
type SeqFunc = func(args ...[]any) ([]any, error)

var defaultFunctions = map[string]Func{}
var aggregateFunctions = map[string]AggFunc{}
var sequenceFunctions = map[string]SeqFunc{}
var funcCallDepth int = 0
var maxFuncCallDepth int = 20 // 合理的函數調用深度上限

func ResetFuncCallDepth() {
	funcCallDepth = 0
}

// RegisterFunction registers a custom scalar function for CCL evaluation.
func RegisterFunction(name string, fn Func) {
	registerFunction(name, fn)
}

// RegisterAggregateFunction registers a custom aggregate function for CCL evaluation.
func RegisterAggregateFunction(name string, fn AggFunc) {
	registerAggregateFunction(name, fn)
}

// RegisterSequenceFunction registers a custom sequence function (whole-column
// input, same-length-column output) for CCL evaluation.
func RegisterSequenceFunction(name string, fn SeqFunc) {
	registerSequenceFunction(name, fn)
}

func registerFunction(name string, fn Func) {
	defaultFunctions[strings.ToUpper(name)] = fn
}

func registerAggregateFunction(name string, fn AggFunc) {
	aggregateFunctions[strings.ToUpper(name)] = fn
}

func registerSequenceFunction(name string, fn SeqFunc) {
	sequenceFunctions[strings.ToUpper(name)] = fn
}

// IsSequenceFunction reports whether name resolves to a registered sequence
// function. Exposed for the evaluator and IsRowDependent.
func IsSequenceFunction(name string) bool {
	_, ok := sequenceFunctions[strings.ToUpper(name)]
	return ok
}

func callFunction(name string, args []any) (any, error) {
	// 防止過深調用
	funcCallDepth++
	if funcCallDepth > maxFuncCallDepth {
		funcCallDepth = 0
		return nil, fmt.Errorf("callFunction: maximum function call depth exceeded, possibly recursive function calls")
	}

	// 使用 defer 確保退出前減少深度計數
	defer func() {
		funcCallDepth--
	}()

	fn, ok := defaultFunctions[strings.ToUpper(name)]
	if !ok {
		return nil, fmt.Errorf("undefined function: %s", name)
	}

	// 添加 panic 恢復機制
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("函數呼叫錯誤: %v\n", r)
		}
	}()

	return fn(args...)
}

func callAggregateFunction(name string, args [][]any) (any, error) {
	fn, ok := aggregateFunctions[strings.ToUpper(name)]
	if !ok {
		return nil, fmt.Errorf("undefined aggregate function: %s", name)
	}

	return fn(args...)
}

func callSequenceFunction(name string, args [][]any) ([]any, error) {
	fn, ok := sequenceFunctions[strings.ToUpper(name)]
	if !ok {
		return nil, fmt.Errorf("undefined sequence function: %s", name)
	}
	return fn(args...)
}
