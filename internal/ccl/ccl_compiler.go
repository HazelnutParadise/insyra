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

	return parseExpression(tokens)
}

// compileStatement compiles a CCL statement (expression or assignment).
func compileStatement(statement string) (CCLNode, error) {
	tokens, err := tokenize(statement)
	if err != nil {
		return nil, err
	}
	return parseStatement(tokens)
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
		idx := utils.ParseColIndex(t.name)
		if idx != -1 {
			return &cclResolvedColNode{index: idx, name: t.name}, nil
		}
		if idx, ok := colNameMap[t.name]; ok {
			return &cclResolvedColNode{index: idx, name: t.name}, nil
		}
		return t, nil
	case *cclColIndexNode:
		idx := utils.ParseColIndex(t.index)
		if idx != -1 {
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
