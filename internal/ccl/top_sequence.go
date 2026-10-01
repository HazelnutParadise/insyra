package ccl

import (
	"fmt"
	"strings"
)

// TopSequence is a built-in sequence function that is a whole expression,
// computed batch by batch: its column argument is read from each batch the way
// the evaluator reads it, and its outputs come out in row order, possibly some
// rows behind its input when it looks ahead.
type TopSequence struct {
	name   string // as the caller wrote it, the way the evaluator's errors read
	arg    cclNode
	stream SequenceStream
}

// NewTopSequence returns n as a TopSequence when n is a call to a sequence
// function. ok is false when n is not a sequence call. The error says why a
// sequence call cannot be computed this way: it has no streaming form, its
// column argument holds another sequence function or does not change from row
// to row, or another argument is not a constant. totalRows and colNames are the
// table's shape, which a constant argument may read.
func NewTopSequence(n CCLNode, totalRows int, colNames []string) (seq *TopSequence, ok bool, err error) {
	call, isCall := n.(*funcCallNode)
	if !isCall {
		return nil, false, nil
	}
	upper := strings.ToUpper(call.name)
	if !IsSequenceFunction(upper) {
		return nil, false, nil
	}
	if len(call.args) == 0 {
		return nil, true, fmt.Errorf("sequence function %s needs a column argument", upper)
	}

	// The column argument is read per batch, so it has to be a column of this
	// table. A nested sequence function would need one stream feeding another,
	// and an argument that does not change from row to row is a constant the
	// stream would read again per batch for nothing.
	column := call.args[0]
	if nested, buried := sequenceCallIn(column); buried {
		return nil, true, fmt.Errorf("sequence function %s inside another sequence function (%s) is not supported when the table is read in batches", nested, describeNode(n))
	}
	if !isSequenceRowShaped(column) && !IsRowDependent(column) {
		return nil, true, fmt.Errorf("sequence function %s reads a column that does not change from row to row, which is not supported when the table is read in batches", upper)
	}

	// The rest are the periods and windows: one value each, read against the
	// table's shape rather than against its rows.
	info := &tableInfoContext{totalRows: totalRows, colNames: colNames}
	params := make([][]any, 0, len(call.args)-1)
	for i, arg := range call.args[1:] {
		_, buried := sequenceCallIn(arg)
		if buried || IsRowDependent(arg) {
			return nil, true, fmt.Errorf("sequence function %s: argument %d must be a constant", upper, i+2)
		}
		param, err := evaluateToColumn(arg, info, 0, 0)
		if err != nil {
			return nil, true, err
		}
		params = append(params, param)
	}

	stream, streams, err := NewSequenceStream(call.name, params)
	if err != nil {
		return nil, true, err
	}
	if !streams {
		return nil, true, fmt.Errorf("sequence function %s is not supported when the table is read in batches", upper)
	}
	return &TopSequence{name: call.name, arg: column, stream: stream}, true, nil
}

// Push reads the column argument from every row of batch and returns the
// outputs for the rows answered so far.
func (s *TopSequence) Push(batch Context) ([]any, error) {
	values, err := sequenceColumn(s.name, s.arg, batch, 0, 0)
	if err != nil {
		return nil, err
	}
	return s.stream.Push(values)
}

// Flush returns the outputs for the rows still held, once the table has ended.
func (s *TopSequence) Flush() ([]any, error) {
	return s.stream.Flush()
}

// isSequenceRowShaped reports whether a sequence function's argument stands for
// a whole row rather than a column of values: '@', or a range of whole columns.
// Such an argument is read one row per element even though it does not change
// from row to row.
func isSequenceRowShaped(arg cclNode) bool {
	if _, ok := arg.(*cclAtNode); ok {
		return true
	}
	return isColumnRangeNode(arg)
}

// sequenceCallIn returns the name of the first sequence function call anywhere
// inside n, n itself included, so that a column argument holding one is named
// by the call the caller wrote.
func sequenceCallIn(n cclNode) (string, bool) {
	if call, ok := n.(*funcCallNode); ok {
		if upper := strings.ToUpper(call.name); IsSequenceFunction(upper) {
			return upper, true
		}
	}
	for _, child := range childNodes(n) {
		if name, ok := sequenceCallIn(child); ok {
			return name, true
		}
	}
	return "", false
}
