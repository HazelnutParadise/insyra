package ccl

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Batches reads a table one batch at a time, in row order: each call is one
// full pass, giving yield one GlobalRowContext per batch, whose rows continue
// where the previous batch's ended. It returns yield's error, or its own.
type Batches func(yield func(batch GlobalRowContext) error) error

// cclFailedPartNode stands for a whole-table part that could not be computed.
// Evaluating it returns that error, so only a row that reaches the part fails,
// the way the part fails on the whole table only where it is evaluated.
type cclFailedPartNode struct{ err error }

// failedPart is how a part that could not be computed is carried in the values
// replaceWholeTableParts substitutes.
type failedPart struct{ err error }

// ResolveWholeTable returns n with every part whose value depends on rows
// other than the current one replaced by that value over the whole table, so
// the result can be evaluated batch by batch and give what n gives on the
// whole table. totalRows is the table's row count and colNames its columns in
// order. It reads the table through batches only when n has such a part. A
// part it cannot compute this way is named in an error returned before
// anything is read — an empty table included, since a table with no rows is no
// reason to accept an expression batched reading cannot answer.
//
// A part is whole-table when its value cannot change from row to row: an
// aggregate call whose arguments do not mention `#`, and a row access whose
// row operand is not row-dependent (`A.0`, `@.3`, `A:C.5`, `A.3:20`). `X.#`
// and an aggregate reading `#` are row-local, because the evaluator already
// computes them per row. Parts are resolved innermost first: a part is ready
// when no unresolved part is left inside it, every ready part is resolved in
// one pass over the table, and the walk repeats.
//
// A part that is resolved but cannot be computed — a STDEV over one value, a
// row past the end of the table — becomes a node that raises that error where
// it is evaluated, so only the rows that reach it fail, the way the whole
// table answers them.
func ResolveWholeTable(n CCLNode, totalRows int, colNames []string, batches Batches) (CCLNode, error) {
	if err := refuseUnsupportedParts(n); err != nil {
		return nil, err
	}
	if totalRows == 0 {
		return n, nil
	}

	current := n
	for {
		parts := collectWholeTableParts(current)
		if len(parts) == 0 {
			return current, nil
		}
		readyFixed, readyAggregates := readyParts(parts)
		if len(readyFixed) == 0 && len(readyAggregates) == 0 {
			return current, nil
		}

		values := make(map[cclNode]any, len(readyFixed)+len(readyAggregates))
		if len(readyFixed) > 0 {
			fixedValues, err := resolveFixedRows(readyFixed, totalRows, colNames, batches)
			if err != nil {
				return nil, err
			}
			for node, v := range fixedValues {
				values[node] = v
			}
		}
		if len(readyAggregates) > 0 {
			aggregateValues, err := resolveAggregates(readyAggregates, totalRows, colNames, batches)
			if err != nil {
				return nil, err
			}
			for node, v := range aggregateValues {
				values[node] = v
			}
		}
		current = replaceWholeTableParts(current, values)
		for node := range values {
			// A part still standing here is handed back by the next round's walk
			// and read from the table again, for as long as it stays. Say so
			// rather than keep passing over the table.
			if containsNode(current, node) {
				return nil, fmt.Errorf("internal error: %s was computed but could not be replaced", describeNode(node))
			}
		}
	}
}

// refuseUnsupportedParts names the parts ResolveWholeTable cannot compute yet,
// before anything is read.
func refuseUnsupportedParts(n cclNode) error {
	switch t := n.(type) {
	case *funcCallNode:
		upper := strings.ToUpper(t.name)
		if IsSequenceFunction(upper) {
			return fmt.Errorf("sequence function %s is not supported when the table is read in batches", upper)
		}
		if _, isAgg := lookupAggregateFunction(upper); isAgg && len(t.args) > 0 && !containsRowIndex(t) {
			if _, streams := NewStreamingAggregate(upper); !streams {
				return fmt.Errorf("aggregate function %s is not supported when the table is read in batches", upper)
			}
		}
		if isWholeTablePart(t) {
			for _, arg := range t.args {
				if buried, ok := rangeInsideAggregateArgument(arg); ok {
					return fmt.Errorf("a column range inside an aggregate's argument (%s) is not supported when the table is read in batches", describeNode(buried))
				}
			}
		}
	case *cclBinaryOpNode:
		if t.op == "." && IsRowDependent(t.right) {
			if _, isRowIndex := t.right.(*cclRowIndexNode); !isRowIndex {
				return fmt.Errorf("a row reference computed from the current row (%s) is not supported when the table is read in batches", describeNode(n))
			}
		}
	}
	for _, child := range childNodes(n) {
		if err := refuseUnsupportedParts(child); err != nil {
			return err
		}
	}
	return nil
}

// isColumnRangeNode reports whether n is a range of whole columns, A:B.
func isColumnRangeNode(n cclNode) bool {
	t, ok := n.(*cclBinaryOpNode)
	return ok && t.op == ":" && isStaticColumnNode(t.left) && isStaticColumnNode(t.right)
}

// rangeInsideAggregateArgument returns the first column range buried inside one
// argument of a whole-table aggregate. evaluateToColumn expands a range into
// whatever the context holds, which over the whole table is the whole column
// and in a batch only that batch, so a range an aggregate's argument merely
// mentions cannot be given the same answer both ways.
//
// A range handed to an aggregate as one of its arguments is expanded by
// resolveAggregates, column by column, and a range naming the columns a row
// access reads (`A:B.3`) is read row by row, so neither is buried. Both hold at
// every depth: in `AVG(A - SUM(A:B))` the range is the inner SUM's own
// argument, and in `SUM(A:B.3 * 2)` it is the left of a row access.
func rangeInsideAggregateArgument(arg cclNode) (cclNode, bool) {
	if isColumnRangeNode(arg) {
		return nil, false
	}

	var found cclNode
	var walk func(cclNode)
	walk = func(node cclNode) {
		if found != nil {
			return
		}
		switch t := node.(type) {
		case *funcCallNode:
			if _, isAgg := lookupAggregateFunction(t.name); isAgg {
				for _, inner := range t.args {
					if !isColumnRangeNode(inner) {
						walk(inner)
					}
				}
				return
			}
		case *cclBinaryOpNode:
			if t.op == "." && isColumnRangeNode(t.left) {
				walk(t.right)
				return
			}
		}
		if isColumnRangeNode(node) {
			found = node
			return
		}
		for _, child := range childNodes(node) {
			walk(child)
		}
	}
	walk(arg)
	return found, found != nil
}

// collectWholeTableParts returns every whole-table part of n, in the order the
// walk finds them.
func collectWholeTableParts(n cclNode) []cclNode {
	var parts []cclNode
	var walk func(cclNode)
	walk = func(node cclNode) {
		if isWholeTablePart(node) {
			parts = append(parts, node)
		}
		for _, child := range childNodes(node) {
			walk(child)
		}
	}
	walk(n)
	return parts
}

// isWholeTablePart reports whether the node's value cannot change from row to
// row.
func isWholeTablePart(n cclNode) bool {
	switch t := n.(type) {
	case *funcCallNode:
		upper := strings.ToUpper(t.name)
		if _, isAgg := lookupAggregateFunction(upper); isAgg {
			return len(t.args) > 0 && !containsRowIndex(t)
		}
	case *cclBinaryOpNode:
		return t.op == "." && !IsRowDependent(t.right)
	}
	return false
}

// readyParts splits parts into those with no other part left inside them and
// those still waiting on one.
func readyParts(parts []cclNode) (fixed []*cclBinaryOpNode, aggregates []*funcCallNode) {
	for _, part := range parts {
		if holdsAnotherPart(part, parts) {
			continue
		}
		switch t := part.(type) {
		case *cclBinaryOpNode:
			fixed = append(fixed, t)
		case *funcCallNode:
			aggregates = append(aggregates, t)
		}
	}
	return fixed, aggregates
}

// holdsAnotherPart reports whether any part other than itself sits inside node.
func holdsAnotherPart(node cclNode, parts []cclNode) bool {
	for _, other := range parts {
		if other == node {
			continue
		}
		if containsNode(node, other) {
			return true
		}
	}
	return false
}

// containsNode reports whether target sits anywhere inside node, node itself
// included.
func containsNode(node, target cclNode) bool {
	if node == target {
		return true
	}
	for _, child := range childNodes(node) {
		if containsNode(child, target) {
			return true
		}
	}
	return false
}

// childNodes returns the nodes directly under n.
func childNodes(n cclNode) []cclNode {
	switch t := n.(type) {
	case *funcCallNode:
		return t.args
	case *cclBinaryOpNode:
		return []cclNode{t.left, t.right}
	case *cclFoldChainNode:
		children := make([]cclNode, 0, len(t.operands)+1)
		children = append(children, t.init)
		children = append(children, t.operands...)
		return children
	case *cclChainedComparisonNode:
		return t.values
	case *cclAssignmentNode:
		return []cclNode{t.expr}
	case *cclNewColNode:
		return []cclNode{t.expr}
	}
	return nil
}

// resolveFixedRows computes every ready row access over the whole table and
// returns each one's value, keyed by its node. The rows they name are captured
// in one pass over the table.
//
// A node whose row operand, or whose captured rows, cannot be read is recorded
// as a failedPart of its own, so it fails only where it is evaluated and the
// nodes beside it keep their values. An error from reading the table itself has
// no such reading — the capture is what every node needs — so it is returned.
func resolveFixedRows(nodes []*cclBinaryOpNode, totalRows int, colNames []string, batches Batches) (map[cclNode]any, error) {
	info := &tableInfoContext{totalRows: totalRows, colNames: colNames}
	values := make(map[cclNode]any, len(nodes))

	wanted := make(map[int]struct{})
	for _, node := range nodes {
		rows, err := fixedRowIndices(node.right, info)
		if err != nil {
			values[node] = failedPart{err: err}
			continue
		}
		for _, r := range rows {
			wanted[r] = struct{}{}
		}
	}
	indices := make([]int, 0, len(wanted))
	for r := range wanted {
		indices = append(indices, r)
	}
	sort.Ints(indices)

	captured := make(map[int][]any, len(indices))
	// Nothing to read when no node named a row: a reversed range names none,
	// and reading the table would not give the rows those nodes ask for either.
	if len(indices) > 0 {
		if err := batches(func(batch GlobalRowContext) error {
			if batch == nil {
				return nil
			}
			rows := batch.GetRowCount()
			if rows <= 0 {
				return nil
			}
			if err := batch.SetRowIndex(0); err != nil {
				return err
			}
			offset := batch.GlobalRowIndex()
			for _, r := range indices {
				if r < offset || r >= offset+rows {
					continue
				}
				row := make([]any, len(colNames))
				for c := range colNames {
					cell, err := batch.GetCell(c, r-offset)
					if err != nil {
						return err
					}
					row[c] = cell
				}
				captured[r] = row
			}
			return nil
		}); err != nil {
			return nil, err
		}
	}

	served := &capturedRowsContext{tableInfoContext: info, rows: captured}
	for _, node := range nodes {
		if _, recorded := values[node]; recorded {
			continue
		}
		v, err := evaluateWithCallDepth(node, served, 0, 0)
		if err != nil {
			values[node] = failedPart{err: err}
			continue
		}
		values[node] = v
	}
	return values, nil
}

// fixedRowIndices evaluates a row operand against the table's shape and returns
// the row positions it names, a range standing for every row it spans. A range
// that ends before it starts names no row, which is how evaluateRowAccess reads
// it on the whole table.
func fixedRowIndices(row cclNode, info *tableInfoContext) ([]int, error) {
	v, err := evaluateWithCallDepth(row, info, 0, 0)
	if err != nil {
		return nil, err
	}
	switch value := v.(type) {
	case RowRange:
		if value.End < value.Start {
			return nil, nil
		}
		indices := make([]int, 0, value.End-value.Start+1)
		for i := value.Start; i <= value.End; i++ {
			indices = append(indices, i)
		}
		return indices, nil
	case float64:
		idx, err := wholeIndex(value, "row index")
		if err != nil {
			return nil, err
		}
		return []int{idx}, nil
	case int:
		return []int{value}, nil
	case string:
		return nil, fmt.Errorf("row names are not available when the table is read in batches")
	default:
		return nil, fmt.Errorf("invalid row index type: %T", v)
	}
}

// resolveAggregates computes every ready aggregate over the whole table, in as
// few passes as the expressions allow.
//
// Each aggregate is fed its arguments in the order the aggregate function sees
// them: each argument whole before the next, a column range or `@` one column
// at a time, and the whole sequence once more for every extra pass the
// aggregate asks for. An aggregate's feeding is a list of steps, one per
// source; every round the aggregates settle the steps that need no rows of the
// table (a constant argument), and the ones left standing on a step that does
// share one read of the table, so independent aggregates cost as many reads as
// the slowest of them rather than the sum. An aggregate nested inside another
// is not ready until the inner one is resolved, which ResolveWholeTable's loop
// takes care of one level at a time.
//
// An aggregate that cannot be computed is recorded as a failedPart of its own
// and takes no further part, so it fails where it is evaluated and the
// aggregates beside it keep theirs. Only an error from reading the table itself
// is returned: nothing else is needed by every run in the round.
func resolveAggregates(nodes []*funcCallNode, totalRows int, colNames []string, batches Batches) (map[cclNode]any, error) {
	info := &tableInfoContext{totalRows: totalRows, colNames: colNames}
	values := make(map[cclNode]any, len(nodes))

	runs := make([]*aggregateRun, 0, len(nodes))
	for _, node := range nodes {
		name := strings.ToUpper(node.name)
		if len(node.args) == 0 {
			v, err := callAggregateFunction(name, nil)
			if err != nil {
				values[node] = failedPart{err: fmt.Errorf("aggregate function %s: %w", name, err)}
				continue
			}
			values[node] = v
			continue
		}
		stream, ok := NewStreamingAggregate(name)
		if !ok {
			return nil, fmt.Errorf("aggregate function %s is not supported when the table is read in batches", name)
		}
		sources, err := aggregateSources(node, info, len(colNames))
		if err != nil {
			values[node] = failedPart{err: fmt.Errorf("aggregate function %s: %w", name, err)}
			continue
		}
		runs = append(runs, newAggregateRun(node, name, stream, sources))
	}

	for {
		// Settle every step that reads nothing from the table.
		for _, run := range runs {
			for !run.done() && run.current().kind == sourceConstant {
				column, err := evaluateToColumn(run.current().arg, info, 0, 0)
				if err != nil {
					run.fail(fmt.Errorf("aggregate function %s: %w", run.name, err))
					continue
				}
				run.stream.Add(column)
				run.advance()
			}
		}

		// Whoever is left needs the table: one read serves all of them.
		var readers []*aggregateRun
		for _, run := range runs {
			if !run.done() {
				readers = append(readers, run)
			}
		}
		if len(readers) == 0 {
			break
		}
		err := batches(func(batch GlobalRowContext) error {
			if batch == nil || batch.GetRowCount() <= 0 {
				return nil
			}
			for _, run := range readers {
				if run.done() {
					continue
				}
				column, err := run.read(batch)
				if err != nil {
					run.fail(fmt.Errorf("aggregate function %s: %w", run.name, err))
					continue
				}
				run.stream.Add(column)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
		for _, run := range readers {
			run.advance()
		}
	}

	for _, run := range runs {
		if run.err != nil {
			values[run.node] = failedPart{err: run.err}
			continue
		}
		v, err := run.stream.Result()
		if err != nil {
			values[run.node] = failedPart{err: fmt.Errorf("aggregate function %s: %w", run.name, err)}
			continue
		}
		values[run.node] = v
	}
	return values, nil
}

// aggregateSourceKind says where one stretch of an aggregate's values comes
// from.
type aggregateSourceKind int

const (
	// sourceColumn is a column of the table, read as a batch's own column.
	sourceColumn aggregateSourceKind = iota
	// sourceExpression is an argument that varies from row to row, evaluated
	// over each batch.
	sourceExpression
	// sourceConstant is an argument that does not depend on the current row,
	// evaluated once when its turn comes, without reading the table.
	sourceConstant
)

// aggregateSource is one stretch of values an aggregate is fed, whole, before
// the next stretch.
type aggregateSource struct {
	kind aggregateSourceKind
	col  int     // sourceColumn
	arg  cclNode // sourceExpression and sourceConstant
}

// aggregateSources lists the stretches of values node's arguments make, in the
// order the aggregate function sees them.
func aggregateSources(node *funcCallNode, info *tableInfoContext, colCount int) ([]aggregateSource, error) {
	var sources []aggregateSource
	addColumns := func(from, to int) {
		for c := from; c <= to; c++ {
			sources = append(sources, aggregateSource{kind: sourceColumn, col: c})
		}
	}
	for _, arg := range node.args {
		switch a := arg.(type) {
		case *cclAtNode:
			addColumns(0, colCount-1)
			continue
		case *cclBinaryOpNode:
			if isColumnRangeNode(a) {
				v, err := evaluateRange(a.left, a.right, info, 0, 0)
				if err != nil {
					return nil, err
				}
				r, ok := v.(ColumnRange)
				if !ok {
					return nil, fmt.Errorf("raw row range cannot be used as a data source; use @.start:end instead")
				}
				addColumns(r.Start, r.End)
				continue
			}
		}
		if IsRowDependent(arg) {
			sources = append(sources, aggregateSource{kind: sourceExpression, arg: arg})
		} else {
			sources = append(sources, aggregateSource{kind: sourceConstant, arg: arg})
		}
	}
	return sources, nil
}

// aggregateRun is one aggregate being fed: its steps are every source in turn,
// once for each pass the aggregate asks for, and pass and index are where it
// stands among them.
type aggregateRun struct {
	node    *funcCallNode
	name    string // upper case
	stream  StreamingAggregate
	sources []aggregateSource
	passes  int
	pass    int
	index   int
	err     error // set when the run cannot be fed, which settles it
}

func newAggregateRun(node *funcCallNode, name string, stream StreamingAggregate, sources []aggregateSource) *aggregateRun {
	run := &aggregateRun{node: node, name: name, stream: stream, sources: sources, passes: stream.Passes()}
	stream.BeginPass(0)
	if len(sources) == 0 {
		// Nothing to feed: the passes still happen, so the aggregate settles
		// the way it does for a call whose arguments held no values.
		for p := 1; p < run.passes; p++ {
			stream.BeginPass(p)
		}
		run.pass = run.passes
	}
	return run
}

// done reports whether every step has been fed, which a run that failed is as
// surely as one that finished.
func (r *aggregateRun) done() bool { return r.err != nil || r.pass >= r.passes }

// fail records why the run cannot be fed. It keeps the first reason: the one
// that stopped it is the one a row reaching it would see.
func (r *aggregateRun) fail(err error) {
	if r.err == nil {
		r.err = err
	}
}

// current is the source the next step feeds. It is only valid while the run is
// not done.
func (r *aggregateRun) current() aggregateSource { return r.sources[r.index] }

// advance moves past the step just fed, starting the next pass when the
// sources have all been fed.
func (r *aggregateRun) advance() {
	r.index++
	if r.index < len(r.sources) {
		return
	}
	r.index = 0
	r.pass++
	if r.pass < r.passes {
		r.stream.BeginPass(r.pass)
	}
}

// read returns the current source's values in one batch.
func (r *aggregateRun) read(batch GlobalRowContext) ([]any, error) {
	src := r.current()
	if src.kind == sourceColumn {
		return batch.GetColData(src.col)
	}
	return evaluateToColumn(src.arg, batch, 0, 0)
}

// replaceWholeTableParts rebuilds n with each node in values replaced by its
// value, the same substitution foldAggregates makes on the DataTable path.
func replaceWholeTableParts(n cclNode, values map[cclNode]any) cclNode {
	if len(values) == 0 {
		return n
	}
	if v, ok := values[n]; ok {
		if failed, isFailed := v.(failedPart); isFailed {
			return &cclFailedPartNode{err: failed.err}
		}
		return literalNode(v)
	}
	switch t := n.(type) {
	case *funcCallNode:
		changed := false
		args := make([]cclNode, len(t.args))
		for i, arg := range t.args {
			a := replaceWholeTableParts(arg, values)
			args[i] = a
			changed = changed || a != arg
		}
		if !changed {
			return n
		}
		return &funcCallNode{name: t.name, args: args}
	case *cclBinaryOpNode:
		// `:` is walked like any other operator. Only an aggregate call and a
		// `.` node are ever in values, and a column node never is, so `A:B`
		// keeps naming the columns it wrote even though both its sides were
		// offered for substitution.
		left := replaceWholeTableParts(t.left, values)
		right := replaceWholeTableParts(t.right, values)
		if left == t.left && right == t.right {
			return n
		}
		return &cclBinaryOpNode{op: t.op, left: left, right: right}
	case *cclFoldChainNode:
		init := replaceWholeTableParts(t.init, values)
		changed := init != t.init
		operands := make([]cclNode, len(t.operands))
		for i, operand := range t.operands {
			o := replaceWholeTableParts(operand, values)
			operands[i] = o
			changed = changed || o != operand
		}
		if !changed {
			return n
		}
		return &cclFoldChainNode{init: init, ops: t.ops, operands: operands}
	case *cclChainedComparisonNode:
		changed := false
		values2 := make([]cclNode, len(t.values))
		for i, v := range t.values {
			nv := replaceWholeTableParts(v, values)
			values2[i] = nv
			changed = changed || nv != v
		}
		if !changed {
			return n
		}
		return &cclChainedComparisonNode{ops: t.ops, values: values2}
	case *cclAssignmentNode:
		expr := replaceWholeTableParts(t.expr, values)
		if expr == t.expr {
			return n
		}
		return &cclAssignmentNode{target: t.target, expr: expr}
	case *cclNewColNode:
		expr := replaceWholeTableParts(t.expr, values)
		if expr == t.expr {
			return n
		}
		return &cclNewColNode{colName: t.colName, expr: expr}
	}
	return n
}

// tableInfoContext answers what a row operand can be checked against — the
// table's shape — without holding any rows of it.
type tableInfoContext struct {
	totalRows int
	colNames  []string
}

func (c *tableInfoContext) GetCol(index int) any { return nil }

func (c *tableInfoContext) GetColByName(name string) (any, error) {
	return nil, errors.New("no rows are available here")
}

func (c *tableInfoContext) GetRowIndex() int { return 0 }

func (c *tableInfoContext) GetCurrentRow() any { return nil }

func (c *tableInfoContext) GetCell(colIndex, rowIndex int) (any, error) {
	return nil, errors.New("no rows are available here")
}

func (c *tableInfoContext) GetCellByName(colName string, rowIndex int) (any, error) {
	return nil, errors.New("no rows are available here")
}

func (c *tableInfoContext) GetRowAt(rowIndex int) (any, error) {
	return nil, errors.New("no rows are available here")
}

func (c *tableInfoContext) GetRowIndexByName(rowName string) (int, error) {
	return 0, errors.New("row names are not available when the table is read in batches")
}

func (c *tableInfoContext) GetColData(index int) ([]any, error) {
	return nil, errors.New("no rows are available here")
}

func (c *tableInfoContext) GetColDataByName(name string) ([]any, error) {
	return nil, errors.New("no rows are available here")
}

func (c *tableInfoContext) GetColIndexByName(colName string) (int, error) {
	for i, name := range c.colNames {
		if name == colName {
			return i, nil
		}
	}
	return -1, fmt.Errorf("column name '%s' not found", colName)
}

func (c *tableInfoContext) GetColCount() int { return len(c.colNames) }

func (c *tableInfoContext) GetRowCount() int { return c.totalRows }

func (c *tableInfoContext) SetRowIndex(index int) error {
	return errors.New("no rows are available here")
}

func (c *tableInfoContext) GetAllData() ([]any, error) {
	return nil, errors.New("no rows are available here")
}

// capturedRowsContext serves the whole rows resolveFixedRows captured, by
// their position in the whole table. It is deliberately not a
// GlobalRowContext: a row written in the expression is already the position
// this context reads by.
type capturedRowsContext struct {
	*tableInfoContext
	rows map[int][]any
}

func (c *capturedRowsContext) GetCell(colIndex, rowIndex int) (any, error) {
	if colIndex < 0 || colIndex >= len(c.colNames) {
		return nil, fmt.Errorf("column index %d out of range", colIndex)
	}
	row, ok := c.rows[rowIndex]
	if !ok {
		return nil, c.missingRow(rowIndex)
	}
	return row[colIndex], nil
}

func (c *capturedRowsContext) GetCellByName(colName string, rowIndex int) (any, error) {
	colIndex, err := c.GetColIndexByName(colName)
	if err != nil {
		return nil, err
	}
	return c.GetCell(colIndex, rowIndex)
}

func (c *capturedRowsContext) GetRowAt(rowIndex int) (any, error) {
	row, ok := c.rows[rowIndex]
	if !ok {
		return nil, c.missingRow(rowIndex)
	}
	return row, nil
}

func (c *capturedRowsContext) missingRow(rowIndex int) error {
	return fmt.Errorf("row index %d out of range (total rows: %d)", rowIndex, c.totalRows)
}
