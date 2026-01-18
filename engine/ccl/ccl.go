package ccl

import internalccl "github.com/HazelnutParadise/insyra/internal/ccl"

// CCL types
type Context = internalccl.Context
type CCLNode = internalccl.CCLNode
type EvaluationResult = internalccl.EvaluationResult
type MapContext = internalccl.MapContext
type Func = internalccl.Func
type AggFunc = internalccl.AggFunc

// NewMapContext creates a map-based CCL context.
func NewMapContext(data map[string][]any) (*MapContext, error) {
	return internalccl.NewMapContext(data)
}

// CompileExpression compiles a CCL expression into an AST.
func CompileExpression(expression string) (CCLNode, error) {
	return internalccl.CompileExpression(expression)
}

// CompileMultiline compiles a multi-line CCL script into AST nodes.
func CompileMultiline(script string) ([]CCLNode, error) {
	return internalccl.CompileMultiline(script)
}

// Bind resolves column references to indices.
func Bind(n CCLNode, colNameMap map[string]int) (CCLNode, error) {
	return internalccl.Bind(n, colNameMap)
}

// Evaluate evaluates a CCL node with the given context.
func Evaluate(n CCLNode, ctx Context) (any, error) {
	return internalccl.Evaluate(n, ctx)
}

// EvaluateStatement evaluates a CCL statement and returns detailed result.
func EvaluateStatement(n CCLNode, ctx Context) (*EvaluationResult, error) {
	return internalccl.EvaluateStatement(n, ctx)
}

// GetAssignmentTarget returns the assignment target column name/index.
func GetAssignmentTarget(n CCLNode) (string, bool) {
	return internalccl.GetAssignmentTarget(n)
}

// GetNewColInfo returns the new column info if the node creates one.
func GetNewColInfo(n CCLNode) (string, CCLNode, bool) {
	return internalccl.GetNewColInfo(n)
}

// GetExpressionNode returns the expression node for a statement.
func GetExpressionNode(n CCLNode) CCLNode {
	return internalccl.GetExpressionNode(n)
}

// IsAssignmentNode reports whether the node is an assignment.
func IsAssignmentNode(n CCLNode) bool {
	return internalccl.IsAssignmentNode(n)
}

// IsNewColNode reports whether the node creates a new column.
func IsNewColNode(n CCLNode) bool {
	return internalccl.IsNewColNode(n)
}

// IsRowDependent reports whether the node depends on row context.
func IsRowDependent(n CCLNode) bool {
	return internalccl.IsRowDependent(n)
}

// RegisterStandardFunctions registers the built-in CCL functions.
func RegisterStandardFunctions() {
	internalccl.RegisterStandardFunctions()
}

// RegisterFunction registers a custom scalar function.
func RegisterFunction(name string, fn Func) {
	internalccl.RegisterFunction(name, fn)
}

// RegisterAggregateFunction registers a custom aggregate function.
func RegisterAggregateFunction(name string, fn AggFunc) {
	internalccl.RegisterAggregateFunction(name, fn)
}

// ResetEvalDepth resets the evaluator recursion depth.
func ResetEvalDepth() {
	internalccl.ResetEvalDepth()
}

// ResetFuncCallDepth resets the function call depth counter.
func ResetFuncCallDepth() {
	internalccl.ResetFuncCallDepth()
}
