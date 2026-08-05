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
var maxFuncCallDepth = 20 // 合理的函數調用深度上限

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

func callFunction(name string, args []any, callDepth int) (result any, err error) {
	callDepth++
	if callDepth > maxFuncCallDepth {
		return nil, fmt.Errorf("callFunction: maximum function call depth exceeded, possibly recursive function calls")
	}

	fn, ok := defaultFunctions[strings.ToUpper(name)]
	if !ok {
		return nil, fmt.Errorf("undefined function: %s", name)
	}

	// panic 恢復：以具名返回值將 panic 轉為錯誤回報給呼叫端，
	// 而非靜默吞掉（原本 fmt.Printf 後回傳 nil,nil）。
	defer func() {
		if r := recover(); r != nil {
			result = nil
			err = fmt.Errorf("function %s panicked: %v", name, r)
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
