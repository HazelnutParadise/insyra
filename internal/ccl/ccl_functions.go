package ccl

import (
	"fmt"
	"strings"
	"sync"
)

type Func = func(args ...any) (any, error)
type AggFunc = func(args ...[]any) (any, error)

// SeqFunc is the signature for CCL sequence (window / lag / cumulative)
// functions. Unlike scalar functions (one row in, one value out) and
// aggregate functions (whole columns in, single value out broadcast back),
// sequence functions take whole columns and return a same-length column,
// enabling LAG, CUMSUM, ROLLING_MEAN, etc.
type SeqFunc = func(args ...[]any) ([]any, error)

// registryMu guards the three function tables: registration may happen at
// any time (users register custom functions), and evaluation reads them
// concurrently from every AddColUsingCCL.
var registryMu sync.RWMutex
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
	registryMu.Lock()
	defer registryMu.Unlock()
	defaultFunctions[strings.ToUpper(name)] = fn
}

func registerAggregateFunction(name string, fn AggFunc) {
	registryMu.Lock()
	defer registryMu.Unlock()
	aggregateFunctions[strings.ToUpper(name)] = fn
}

func registerSequenceFunction(name string, fn SeqFunc) {
	registryMu.Lock()
	defer registryMu.Unlock()
	sequenceFunctions[strings.ToUpper(name)] = fn
}

func lookupFunction(name string) (Func, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	fn, ok := defaultFunctions[strings.ToUpper(name)]
	return fn, ok
}

func lookupAggregateFunction(name string) (AggFunc, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	fn, ok := aggregateFunctions[strings.ToUpper(name)]
	return fn, ok
}

func lookupSequenceFunction(name string) (SeqFunc, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	fn, ok := sequenceFunctions[strings.ToUpper(name)]
	return fn, ok
}

// IsSequenceFunction reports whether name resolves to a registered sequence
// function. Exposed for the evaluator and IsRowDependent.
func IsSequenceFunction(name string) bool {
	_, ok := lookupSequenceFunction(name)
	return ok
}

func callFunction(name string, args []any, callDepth int) (result any, err error) {
	callDepth++
	if callDepth > maxFuncCallDepth {
		return nil, fmt.Errorf("callFunction: maximum function call depth exceeded, possibly recursive function calls")
	}

	fn, ok := lookupFunction(name)
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

func callAggregateFunction(name string, args [][]any) (result any, err error) {
	fn, ok := lookupAggregateFunction(name)
	if !ok {
		return nil, fmt.Errorf("undefined aggregate function: %s", name)
	}
	defer func() {
		if r := recover(); r != nil {
			result = nil
			err = fmt.Errorf("aggregate function %s panicked: %v", name, r)
		}
	}()
	return fn(args...)
}

func callSequenceFunction(name string, args [][]any) (result []any, err error) {
	fn, ok := lookupSequenceFunction(name)
	if !ok {
		return nil, fmt.Errorf("undefined sequence function: %s", name)
	}
	defer func() {
		if r := recover(); r != nil {
			result = nil
			err = fmt.Errorf("sequence function %s panicked: %v", name, r)
		}
	}()
	return fn(args...)
}
