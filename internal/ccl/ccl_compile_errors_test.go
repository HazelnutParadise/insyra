package ccl

import (
	"testing"
	"time"
)

// compileWithTimeout runs CompileExpression and fails the test if it does not
// return within the deadline. Regression guard for issue #184: certain
// malformed inputs used to make the tokenizer loop forever instead of
// returning an error, so "did not return" is itself the failure being tested.
func compileWithTimeout(t *testing.T, expr string) (CCLNode, error) {
	t.Helper()
	type result struct {
		node CCLNode
		err  error
	}
	done := make(chan result, 1)
	go func() {
		node, err := CompileExpression(expr)
		done <- result{node, err}
	}()
	select {
	case r := <-done:
		return r.node, r.err
	case <-time.After(5 * time.Second):
		t.Fatalf("CompileExpression(%q) did not return within 5s — possible tokenizer/parser non-termination (issue #184 regression)", expr)
		return nil, nil
	}
}

// TestCompileExpression_MalformedInputsReturnError pins the fix for issue #184:
// every malformed input must return an error promptly, never hang.
func TestCompileExpression_MalformedInputsReturnError(t *testing.T) {
	cases := []struct {
		name string
		expr string
	}{
		// Unrecognized characters — these hung the tokenizer before the
		// no-progress guard in tokenize's default case.
		{"issue 184 repro", "this is junk @#$"},
		{"lone dollar", "$"},
		{"lone tilde", "~"},
		{"dollar between idents", "A $ B"},
		{"fullwidth punctuation", "中文？"},
		{"backslash", `A \ B`},
		// Trailing tokens after a complete expression — previously silently
		// dropped, now rejected by the EOF check in parseExpression.
		{"trailing numbers", "1 2 3"},
		{"trailing ident", "5 * 3 garbage"},
		// Other prompt-error paths worth pinning.
		{"empty input", ""},
		{"dangling operator", "A +"},
		{"unclosed parens", "((((1"},
		{"unclosed call", "F(1,"},
		{"unclosed string", "'abc"},
		{"single pipe", "A | B"},
		{"unclosed bracket ref", "['name'"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := compileWithTimeout(t, tc.expr); err == nil {
				t.Errorf("CompileExpression(%q) = nil error, want parse/tokenize error", tc.expr)
			}
		})
	}
}

// TestCompileExpression_ValidInputsStillCompile guards against the error
// checks above becoming over-eager.
func TestCompileExpression_ValidInputsStillCompile(t *testing.T) {
	cases := []string{
		"A + B",
		"IF(A > 0, 1, 0)",
		"1 < A + 100 < 5",
		"A > 15 && B > 10",
		"['col name'] * 2",
		"-A + +B",
	}
	for _, expr := range cases {
		if _, err := compileWithTimeout(t, expr); err != nil {
			t.Errorf("CompileExpression(%q) returned unexpected error: %v", expr, err)
		}
	}
}

// TestCompileMultiline_MalformedStatementReturnsError covers the statement
// path used by ExecuteCCL for the same issue-#184 input class.
func TestCompileMultiline_MalformedStatementReturnsError(t *testing.T) {
	cases := []struct {
		name   string
		script string
	}{
		{"unknown char in assignment", "NEW('Z') = A + $"},
		{"trailing tokens in assignment", "A = 1 2"},
		{"unknown char statement", "~"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			done := make(chan error, 1)
			go func() {
				_, err := CompileMultiline(tc.script)
				done <- err
			}()
			select {
			case err := <-done:
				if err == nil {
					t.Errorf("CompileMultiline(%q) = nil error, want parse/tokenize error", tc.script)
				}
			case <-time.After(5 * time.Second):
				t.Fatalf("CompileMultiline(%q) did not return within 5s — possible non-termination (issue #184 regression)", tc.script)
			}
		})
	}
}
