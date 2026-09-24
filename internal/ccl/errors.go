package ccl

import "fmt"

// CompileError reports an expression that could not be turned into an AST, or
// that could not be bound to a table. It carries where in the source text the
// problem is, so the caller can point at it.
//
// Offset is a byte offset into Expr, and is -1 when the failure is not about a
// particular place in the text (a name that does not resolve to a column, for
// example). Near is the source text at Offset.
type CompileError struct {
	Expr   string
	Offset int
	Near   string
	Msg    string
}

func (e *CompileError) Error() string {
	if e.Offset < 0 {
		return fmt.Sprintf("cannot compile %q: %s", e.Expr, e.Msg)
	}
	if e.Near == "" {
		return fmt.Sprintf("cannot compile %q at offset %d: %s", e.Expr, e.Offset, e.Msg)
	}
	return fmt.Sprintf("cannot compile %q at offset %d (near %q): %s", e.Expr, e.Offset, e.Near, e.Msg)
}

// EvalError reports a failure while evaluating a compiled expression. Row is
// the row it happened on, or -1 when the expression does not depend on the row
// and is therefore evaluated once.
type EvalError struct {
	Expr string
	Row  int
	Err  error
}

func (e *EvalError) Error() string {
	if e.Row < 0 {
		return fmt.Sprintf("cannot evaluate %q: %v", e.Expr, e.Err)
	}
	return fmt.Sprintf("cannot evaluate %q at row %d: %v", e.Expr, e.Row, e.Err)
}

func (e *EvalError) Unwrap() error { return e.Err }

// compileErrorAt builds a CompileError pointing at a token. A token that
// carries no offset (the synthetic EOF) yields an offset of -1 rather than a
// position the reader cannot find.
func compileErrorAt(expr string, tok cclToken, format string, args ...any) *CompileError {
	e := &CompileError{Expr: expr, Offset: -1, Msg: fmt.Sprintf(format, args...)}
	if tok.pos >= 0 && tok.pos <= len(expr) {
		e.Offset = tok.pos
		e.Near = tok.value
	}
	return e
}

// asCompileError wraps err as a CompileError unless it already is one. Used at
// the compiler's edges so every failure that reaches a caller carries the
// expression, even the ones raised deep inside without it.
func asCompileError(expr string, err error) error {
	if err == nil {
		return nil
	}
	if ce, ok := err.(*CompileError); ok {
		if ce.Expr == "" {
			ce.Expr = expr
		}
		return ce
	}
	return &CompileError{Expr: expr, Offset: -1, Msg: err.Error()}
}

// AttachExpr fills in the expression on an error raised somewhere that did not
// know the source text — a statement inside a multi-statement script, for
// instance, where only the caller knows which line is running. An error that is
// neither of the two CCL types is wrapped as an evaluation failure, because
// that is the only place this is reached from.
func AttachExpr(expr string, err error) error {
	if err == nil {
		return nil
	}
	switch e := err.(type) {
	case *EvalError:
		if e.Expr == "" {
			e.Expr = expr
		}
		return e
	case *CompileError:
		if e.Expr == "" {
			e.Expr = expr
		}
		return e
	}
	return &EvalError{Expr: expr, Row: -1, Err: err}
}
