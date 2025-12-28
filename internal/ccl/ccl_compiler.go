package ccl

import (
	"fmt"

	"github.com/HazelnutParadise/insyra/internal/utils"
)

// CompileExpression compiles a CCL expression string into an AST.
// It checks for forbidden syntax (assignment, NEW) for expression mode.
func CompileExpression(expression string) (CCLNode, error) {
	tokens, err := Tokenize(expression)
	if err != nil {
		return nil, err
	}

	if err := CheckExpressionMode(tokens); err != nil {
		return nil, err
	}

	return ParseExpression(tokens)
}

// CompileStatement compiles a CCL statement (expression or assignment).
func CompileStatement(statement string) (CCLNode, error) {
	tokens, err := Tokenize(statement)
	if err != nil {
		return nil, err
	}
	return ParseStatement(tokens)
}

// Bind traverses the AST and resolves column references to indices.
// It returns a new AST with resolved nodes.
func Bind(n cclNode, colNameMap map[string]int) (cclNode, error) {
	switch t := n.(type) {
	case *cclIdentifierNode:
		idx := utils.ParseColIndex(t.name)
		return &cclResolvedColNode{index: idx, name: t.name}, nil
	case *cclColIndexNode:
		idx := utils.ParseColIndex(t.index)
		return &cclResolvedColNode{index: idx, name: t.index}, nil
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
