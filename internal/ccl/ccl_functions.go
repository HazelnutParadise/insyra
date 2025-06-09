package ccl

import (
	"fmt"
	"strings"
)

type Func = func(args ...any) (any, error)

var defaultFunctions = map[string]Func{}
var funcCallDepth int = 0
var maxFuncCallDepth int = 20 // 合理的函數調用深度上限

func ResetFuncCallDepth() {
	funcCallDepth = 0
}

func RegisterFunction(name string, fn Func) {
	defaultFunctions[strings.ToUpper(name)] = fn
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
