package ccl

import (
	"strings"
	"testing"
	"time"
)

// Regression tests for the compile-time recursion bounds (change
// bound-ccl-compile-recursion): over-deep input must return an error
// promptly — before these guards it fatally overflowed the stack, which
// recover() cannot catch and which kills the whole process.

// TestCompileExpression_OverDeepInputsReturnError covers both crash axes:
// deep parse recursion (nesting) and deep AST from iteratively-built
// left-associative chains.
func TestCompileExpression_OverDeepInputsReturnError(t *testing.T) {
	cases := []struct {
		name string
		expr string
	}{
		{"nested parens", strings.Repeat("(", maxParseDepth+1) + "1" + strings.Repeat(")", maxParseDepth+1)},
		{"unclosed nested parens", strings.Repeat("(", maxParseDepth+1) + "1"},
		{"nested calls", strings.Repeat("F(", maxParseDepth+1) + "1" + strings.Repeat(")", maxParseDepth+1)},
		{"unary chain", strings.Repeat("- ", maxParseDepth+1) + "5"},
		{"left-associative chain", "1" + strings.Repeat("+1", maxEvalDepth+1)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := compileWithTimeout(t, tc.expr); err == nil {
				t.Errorf("CompileExpression on over-deep input (%s, len %d) = nil error, want depth-limit error", tc.name, len(tc.expr))
			}
		})
	}
}

// TestCompileMultiline_OverDeepStatementReturnsError covers the statement
// path used by ExecuteCCL.
func TestCompileMultiline_OverDeepStatementReturnsError(t *testing.T) {
	script := "NEW('X') = 1" + strings.Repeat("+1", maxEvalDepth+1)
	done := make(chan error, 1)
	go func() {
		_, err := CompileMultiline(script)
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Error("CompileMultiline on over-deep statement = nil error, want depth-limit error")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("CompileMultiline on over-deep statement did not return within 5s")
	}
}

// TestCompileExpression_DeepButLegalInputsStillCompile guards against the
// depth limits becoming over-eager: realistic depth (hundreds of levels)
// and width (thousands of flat arguments/terms) must keep compiling.
func TestCompileExpression_DeepButLegalInputsStillCompile(t *testing.T) {
	cases := []struct {
		name string
		expr string
	}{
		{"500 nested parens", strings.Repeat("(", 500) + "1" + strings.Repeat(")", 500)},
		{"500-term chain", "1" + strings.Repeat("+1", 500)},
		{"wide flat call", "F(" + strings.Repeat("1,", 4999) + "1)"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := compileWithTimeout(t, tc.expr); err != nil {
				t.Errorf("CompileExpression on legal input (%s) returned unexpected error: %v", tc.name, err)
			}
		})
	}
}
