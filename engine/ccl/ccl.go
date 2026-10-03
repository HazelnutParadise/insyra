// Package ccl compiles and evaluates CCL expressions outside a DataTable, and
// registers the functions CCL calls.
//
// # The function registry
//
// CCL has one registry of functions for the whole process. Every DataTable, the
// parquet CCL functions, the CLI and this package read the same one, so a
// function registered here is available to all of them. Registration and
// evaluation are safe from every goroutine: each registration takes effect as a
// whole, and evaluating an expression never sees a half-made entry. An
// expression evaluated while a registration is under way may still call the old
// function for some rows and the new one for later rows, so register before you
// evaluate when that matters. Names are matched in any letter case. There is no
// way to remove a function; registering a name again replaces it.
package ccl

import internalccl "github.com/HazelnutParadise/insyra/internal/ccl"

// CCL types
type Context = internalccl.Context
type CCLNode = internalccl.CCLNode
type EvaluationResult = internalccl.EvaluationResult
type MapContext = internalccl.MapContext
type Func = internalccl.Func
type AggFunc = internalccl.AggFunc

// SeqFunc is the signature of a sequence function: it takes whole columns and
// returns a column of the same length, the way LAG, CUMSUM and ROLLING_MEAN do.
type SeqFunc = internalccl.SeqFunc

// CompileError reports an expression that could not be compiled, carrying the
// byte offset into the expression and the text at it. Match it with errors.As.
type CompileError = internalccl.CompileError

// EvalError reports a failure while evaluating a compiled expression, carrying
// the row it happened on (-1 when the expression does not depend on the row)
// and wrapping the cause. Match it with errors.As on an error that came out of
// a DataTable — AddColUsingCCL, ExecuteCCL and their siblings attach the row
// and the expression. Evaluate below returns the underlying error as it is,
// because it evaluates one node against one context and has no statement or
// row loop to name.
type EvalError = internalccl.EvalError

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

// CompiledStatement pairs a compiled statement with the source line it came
// from, so a failure part-way through a script can say which line failed.
type CompiledStatement = internalccl.CompiledStatement

// CompileMultilineStatements compiles a script and keeps each statement's
// source text alongside its AST.
func CompileMultilineStatements(script string) ([]CompiledStatement, error) {
	return internalccl.CompileMultilineStatements(script)
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

// RegisterStandardFunctions registers the built-in CCL functions. Importing the
// insyra package already does this. Calling it again puts every built-in back,
// and it replaces a function a caller registered under a built-in's name.
func RegisterStandardFunctions() {
	internalccl.RegisterStandardFunctions()
}

// RegisterFunction registers a custom scalar function: one value per argument
// in, one value out, called once for each row. The registry is shared by every
// goroutine and every DataTable in the process; the name is matched in any letter
// case, and a later registration under the same name replaces this one,
// including a built-in's. A panic inside fn is returned as an error from the
// evaluation that called it.
func RegisterFunction(name string, fn Func) {
	internalccl.RegisterFunction(name, fn)
}

// RegisterAggregateFunction registers a custom aggregate function: whole
// columns in, one value out, which every row receives. The registry is shared by
// every goroutine and every DataTable in the process; the name is matched in any
// letter case, and a later registration under the same name replaces this one,
// including a built-in's. A function registered under a built-in's name, such as
// SUM, is the caller's from then on: nothing in insyra computes that name with
// the built-in's arithmetic any more, so parquet's FilterWithCCL and ApplyCCL
// compute it over the whole column instead of batch by batch.
func RegisterAggregateFunction(name string, fn AggFunc) {
	internalccl.RegisterAggregateFunction(name, fn)
}

// RegisterSequenceFunction registers a custom sequence function: whole columns
// in, a column of the same length out, of which each row receives its own cell.
// The registry is shared by every goroutine and every DataTable in the process;
// the name is matched in any letter case, and a later registration under the
// same name replaces this one, including a built-in's. A function registered
// under a built-in's name, such as CUMSUM, is the caller's from then on: nothing
// in insyra computes that name with the built-in's arithmetic any more.
func RegisterSequenceFunction(name string, fn SeqFunc) {
	internalccl.RegisterSequenceFunction(name, fn)
}

// ResetEvalDepth does nothing. Recursion depth has been threaded on the call
// stack since thread-ccl-eval-depth (issue #191), so there is no global state
// left to reset.
//
// Deprecated: it does nothing; remove the call. Removed in the release after the
// one that deprecated it.
func ResetEvalDepth() {
}

// ResetFuncCallDepth does nothing, for the same reason as ResetEvalDepth.
//
// Deprecated: it does nothing; remove the call. Removed in the release after the
// one that deprecated it.
func ResetFuncCallDepth() {
}
