package ccl

import (
	"fmt"
	"strings"

	"github.com/HazelnutParadise/insyra/internal/utils"
)

// CompileExpression compiles a CCL expression string into an AST.
// It checks for forbidden syntax (assignment, NEW) for expression mode.
func CompileExpression(expression string) (CCLNode, error) {
	tokens, err := tokenize(expression)
	if err != nil {
		return nil, err
	}

	if err := checkExpressionMode(tokens); err != nil {
		return nil, err
	}

	node, err := parseExpression(tokens)
	if err != nil {
		return nil, err
	}
	if err := checkASTDepth(node); err != nil {
		return nil, err
	}
	return node, nil
}

// compileStatement compiles a CCL statement (expression or assignment).
func compileStatement(statement string) (CCLNode, error) {
	tokens, err := tokenize(statement)
	if err != nil {
		return nil, err
	}
	node, err := parseStatement(tokens)
	if err != nil {
		return nil, err
	}
	if err := checkASTDepth(node); err != nil {
		return nil, err
	}
	return node, nil
}

// checkASTDepth rejects ASTs deeper than maxEvalDepth. maxParseDepth only
// bounds parse recursion; a left-associative chain like "1+1+1+..." parses
// with O(1) recursion (iterative precedence loop) yet builds an O(n)-deep
// AST that would fatally overflow the stack in Bind/IsRowDependent before
// the evaluator's own depth guard could run. Anything deeper than
// maxEvalDepth could never be evaluated anyway, so nothing usable is lost.
func checkASTDepth(n cclNode) error {
	if exceedsDepth(n, maxEvalDepth) {
		return fmt.Errorf("expression too complex: nesting depth exceeds max %d", maxEvalDepth)
	}
	return nil
}

// exceedsDepth reports whether the AST is deeper than limit. Iterative
// (explicit stack) on purpose: a recursive walker would overflow on exactly
// the inputs this exists to reject. Container node types must all be
// enumerated here — see the note at the node declarations in ccl_types.go.
func exceedsDepth(n cclNode, limit int) bool {
	type entry struct {
		node  cclNode
		depth int
	}
	stack := []entry{{n, 1}}
	for len(stack) > 0 {
		e := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if e.depth > limit {
			return true
		}
		switch t := e.node.(type) {
		case *cclBinaryOpNode:
			stack = append(stack, entry{t.left, e.depth + 1}, entry{t.right, e.depth + 1})
		case *cclFoldChainNode:
			// 攤平鏈的深度是 1 + max(子節點深度)，與鏈長無關。
			stack = append(stack, entry{t.init, e.depth + 1})
			for _, operand := range t.operands {
				stack = append(stack, entry{operand, e.depth + 1})
			}
		case *cclChainedComparisonNode:
			for _, v := range t.values {
				stack = append(stack, entry{v, e.depth + 1})
			}
		case *funcCallNode:
			for _, arg := range t.args {
				stack = append(stack, entry{arg, e.depth + 1})
			}
		case *cclAssignmentNode:
			stack = append(stack, entry{t.expr, e.depth + 1})
		case *cclNewColNode:
			stack = append(stack, entry{t.expr, e.depth + 1})
		}
	}
	return false
}

// CompileMultiline compiles a multi-line CCL script into a list of AST nodes.
// It splits the script by ';' or newline and compiles each statement individually.
func CompileMultiline(script string) ([]CCLNode, error) {
	// Split by ; or newline
	// We need a more robust splitter that respects strings, but for now simple split is used
	// consistent with previous implementation.
	// TODO: Implement a proper lexer-based splitter if needed.
	var lines []string
	var currentLine strings.Builder
	inString := false
	var stringChar rune

	for _, r := range script {
		if inString {
			currentLine.WriteRune(r)
			if r == stringChar {
				inString = false
			}
		} else {
			switch r {
			case '\'', '"':
				inString = true
				stringChar = r
				currentLine.WriteRune(r)
			case ';', '\n':
				line := strings.TrimSpace(currentLine.String())
				if line != "" {
					lines = append(lines, line)
				}
				currentLine.Reset()
			default:
				currentLine.WriteRune(r)
			}
		}
	}
	// Add last line
	line := strings.TrimSpace(currentLine.String())
	if line != "" {
		lines = append(lines, line)
	}

	nodes := make([]CCLNode, 0, len(lines))
	for _, line := range lines {
		node, err := compileStatement(line)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}

// Bind traverses the AST and resolves column references to indices.
// It returns a new AST with resolved nodes.
func Bind(n cclNode, colNameMap map[string]int) (cclNode, error) {
	switch t := n.(type) {
	case *cclIdentifierNode:
		idx, ok := utils.ParseColIndex(t.name)
		if ok {
			return &cclResolvedColNode{index: idx, name: t.name}, nil
		}
		if idx, ok := colNameMap[t.name]; ok {
			return &cclResolvedColNode{index: idx, name: t.name}, nil
		}
		return t, nil
	case *cclColIndexNode:
		idx, ok := utils.ParseColIndex(t.index)
		if ok {
			return &cclResolvedColNode{index: idx, name: t.index}, nil
		}
		return t, nil
	case *cclColNameNode:
		if idx, ok := colNameMap[t.name]; ok {
			return &cclResolvedColNode{index: idx, name: t.name}, nil
		}
		return nil, fmt.Errorf("column name '%s' not found", t.name)
	case *cclBinaryOpNode:
		l, err := Bind(t.left, colNameMap)
		if err != nil {
			return nil, err
		}
		r, err := Bind(t.right, colNameMap)
		if err != nil {
			return nil, err
		}
		return &cclBinaryOpNode{op: t.op, left: l, right: r}, nil
	case *cclFoldChainNode:
		init, err := Bind(t.init, colNameMap)
		if err != nil {
			return nil, err
		}
		newOperands := make([]cclNode, len(t.operands))
		for i, operand := range t.operands {
			bound, err := Bind(operand, colNameMap)
			if err != nil {
				return nil, err
			}
			newOperands[i] = bound
		}
		return &cclFoldChainNode{init: init, ops: t.ops, operands: newOperands}, nil
	case *cclChainedComparisonNode:
		newValues := make([]cclNode, len(t.values))
		for i, v := range t.values {
			nv, err := Bind(v, colNameMap)
			if err != nil {
				return nil, err
			}
			newValues[i] = nv
		}
		return &cclChainedComparisonNode{ops: t.ops, values: newValues}, nil
	case *funcCallNode:
		newArgs := make([]cclNode, len(t.args))
		for i, arg := range t.args {
			na, err := Bind(arg, colNameMap)
			if err != nil {
				return nil, err
			}
			newArgs[i] = na
		}
		return &funcCallNode{name: t.name, args: newArgs}, nil
	case *cclAssignmentNode:
		expr, err := Bind(t.expr, colNameMap)
		if err != nil {
			return nil, err
		}
		return &cclAssignmentNode{target: t.target, expr: expr}, nil
	case *cclNewColNode:
		expr, err := Bind(t.expr, colNameMap)
		if err != nil {
			return nil, err
		}
		return &cclNewColNode{colName: t.colName, expr: expr}, nil
	default:
		return n, nil
	}
}

// MaxResolvedColIndex walks a bound AST and returns the largest column index
// referenced by an Excel-style or resolved column node, or -1 when none.
// Callers use it to reject A..Z references past the last column before
// evaluation, instead of silently reading nil.
func MaxResolvedColIndex(n cclNode) int {
	maxIdx := -1
	var walk func(n cclNode)
	walk = func(n cclNode) {
		switch t := n.(type) {
		case nil:
			return
		case *cclResolvedColNode:
			if t.index > maxIdx {
				maxIdx = t.index
			}
		case *cclColIndexNode:
			if idx, ok := utils.ParseColIndex(t.index); ok && idx > maxIdx {
				maxIdx = idx
			}
		case *cclBinaryOpNode:
			walk(t.left)
			walk(t.right)
		case *cclFoldChainNode:
			walk(t.init)
			for _, o := range t.operands {
				walk(o)
			}
		case *cclChainedComparisonNode:
			for _, v := range t.values {
				walk(v)
			}
		case *funcCallNode:
			for _, a := range t.args {
				walk(a)
			}
		case *cclAssignmentNode:
			walk(t.expr)
		case *cclNewColNode:
			walk(t.expr)
		}
	}
	walk(n)
	return maxIdx
}
