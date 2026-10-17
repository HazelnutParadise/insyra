package ccl

import (
	"fmt"
)

// StreamRefusal returns why ResolveWholeTable would refuse n, the same check it
// runs before reading anything, or nil when it would not.
func StreamRefusal(n CCLNode) error {
	return refuseUnsupportedParts(n, GetExpressionNode(n))
}

// ReferencedColumns returns the positions of the columns n reads, in increasing
// order and without repeats, for a table of numCols columns. n must already be
// bound with Bind, so every column it names has a position. '@' reads every
// column, and a column range every column between its two ends.
//
// A column at a position outside [0, numCols) is left out: a caller refuses
// such a column with FirstColPastEnd before it asks what is read. The target of
// an assignment is written, not read, so only its right-hand side is walked.
func ReferencedColumns(n CCLNode, numCols int) ([]int, error) {
	if numCols < 0 {
		numCols = 0
	}
	read := make([]bool, numCols)
	// A range is resolved against the table's shape alone; the names are never
	// asked for, because a bound column is found by its position.
	shape := &tableInfoContext{colNames: make([]string, numCols)}

	var walk func(node cclNode) error
	walk = func(node cclNode) error {
		switch t := node.(type) {
		case nil:
			return nil
		case *cclIdentifierNode, *cclColIndexNode, *cclColNameNode:
			return fmt.Errorf("ReferencedColumns needs a bound expression; %s is not bound", describeNode(node))
		case *cclResolvedColNode:
			if t.index >= 0 && t.index < numCols {
				read[t.index] = true
			}
			return nil
		case *cclAtNode:
			for i := range read {
				read[i] = true
			}
			return nil
		}

		for _, child := range childNodes(node) {
			if err := walk(child); err != nil {
				return err
			}
		}

		// The ends of a range are read, and so is every column between them. A
		// range that cannot be resolved (it runs past the table, or backwards)
		// reads only what its ends read, which the walk above has taken: the
		// evaluator reports the range itself where the expression is evaluated.
		if rng, ok := node.(*cclBinaryOpNode); ok && isColumnRangeNode(rng) {
			if v, err := evaluateRange(rng.left, rng.right, shape, 0, 0); err == nil {
				if r, ok := v.(ColumnRange); ok {
					for c := max(r.Start, 0); c <= r.End && c < numCols; c++ {
						read[c] = true
					}
				}
			}
		}
		return nil
	}
	if err := walk(n); err != nil {
		return nil, err
	}

	columns := make([]int, 0, numCols)
	for c, isRead := range read {
		if isRead {
			columns = append(columns, c)
		}
	}
	return columns, nil
}
