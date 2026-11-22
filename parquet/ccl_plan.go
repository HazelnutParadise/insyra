package parquet

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/HazelnutParadise/insyra/internal/ccl"
	"github.com/apache/arrow-go/v18/parquet/file"
)

// CCLPlan says how FilterWithCCL, or one statement of ApplyCCL, would run on a
// file. ExplainFilterWithCCL and ExplainApplyCCL read it from the file's
// metadata and the expression alone, without evaluating a row.
//
// MinDataPasses and HeldCells are exact for a run that succeeds, unless
// MayReadMore says the data can make it read more. A run that fails on the
// data can stop sooner. EstimatedHeldBytes is an estimate.
type CCLPlan struct {
	// Mode is how the rows are evaluated.
	Mode CCLMode
	// Reasons says why Mode is not CCLStreamed, why the run may change once
	// the data is read, and what Explain noticed that the call reports only
	// when it evaluates a row, such as a column name the file does not have.
	Reasons []string

	// NeededColumns are the columns the expression reads, in file order, as
	// the loaded table binds it. Every pass reads every column of each batch.
	NeededColumns []string
	// HeldColumns are the columns loaded whole into memory. It is empty unless
	// Mode is CCLWholeColumns.
	HeldColumns []string
	// NumRows is the file's row count, from its metadata.
	NumRows int64

	// MinDataPasses is how many times the run reads the file's data. For
	// FilterWithCCL it includes the read that keeps the rows. For a statement
	// of ApplyCCL it counts the reads that statement adds before the file is
	// written; ApplyCCL then reads the file once more to write it.
	MinDataPasses int
	// HeldCells is how many values the run holds in memory besides the rows it
	// keeps: for CCLWholeColumns the held columns and one value per row of the
	// answer; 0 otherwise.
	HeldCells int64
	// HoldsResult is true for FilterWithCCL, which holds every row it keeps
	// until it returns; ApplyCCL writes its rows to the file instead.
	HoldsResult bool

	// MayReadMore is true when only the data decides whether the run reads
	// more than MinDataPasses or loads whole columns; Reasons says why.
	MayReadMore bool

	// EstimatedHeldBytes is an estimate: the uncompressed size HeldColumns
	// have in the file, from its metadata. Their size in memory is usually
	// larger. A column an earlier ApplyCCL statement creates has no size in
	// the file and is not counted.
	EstimatedHeldBytes int64
}

// CCLMode is how FilterWithCCL or ApplyCCL evaluates the rows.
type CCLMode int

const (
	// CCLStreamed evaluates the rows batch by batch.
	CCLStreamed CCLMode = iota
	// CCLStreamedSequence evaluates a sequence function that is the whole
	// expression, such as LAG, CUMSUM or ROLLING_MEAN, batch by batch, holding
	// the rows a later batch is needed for.
	CCLStreamedSequence
	// CCLWholeColumns loads the columns the expression reads whole and
	// evaluates them the way the DataTable CCL methods do.
	CCLWholeColumns
)

// String returns the mode's name, or CCLMode(n) for a value that is not one.
func (m CCLMode) String() string {
	switch m {
	case CCLStreamed:
		return "CCLStreamed"
	case CCLStreamedSequence:
		return "CCLStreamedSequence"
	case CCLWholeColumns:
		return "CCLWholeColumns"
	}
	return fmt.Sprintf("CCLMode(%d)", int(m))
}

// String returns the plan as a few lines of text.
func (p CCLPlan) String() string {
	var b strings.Builder
	switch p.Mode {
	case CCLStreamed:
		b.WriteString("Plan: batch by batch")
	case CCLStreamedSequence:
		b.WriteString("Plan: batch by batch, as one sequence")
	case CCLWholeColumns:
		b.WriteString("Plan: whole columns")
	default:
		b.WriteString("Plan: " + p.Mode.String())
	}
	if len(p.Reasons) > 0 {
		b.WriteString(" (" + strings.Join(p.Reasons, "; ") + ")")
	}
	b.WriteString("\nReads:   ")
	if len(p.NeededColumns) == 0 {
		b.WriteString("no column")
	} else {
		b.WriteString(strings.Join(p.NeededColumns, ", "))
	}
	if len(p.HeldColumns) > 0 {
		fmt.Fprintf(&b, "\nHolds:   %s, %s rows", strings.Join(p.HeldColumns, ", "), groupDigits(p.NumRows))
		if p.EstimatedHeldBytes > 0 {
			fmt.Fprintf(&b, " (about %s on disk, uncompressed)", humanBytes(p.EstimatedHeldBytes))
		}
	}
	fmt.Fprintf(&b, "\nPasses:  at least %d", p.MinDataPasses)
	if p.MayReadMore {
		b.WriteString(", possibly more")
	}
	if p.HoldsResult {
		b.WriteString("\nResult:  every kept row is held in memory")
	} else {
		b.WriteString("\nResult:  written to the file")
	}
	return b.String()
}

// Show prints String to standard output.
func (p CCLPlan) Show() { fmt.Println(p.String()) }

func groupDigits(n int64) string {
	s := fmt.Sprintf("%d", n)
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	if neg {
		s = "-" + s
	}
	return s
}

func humanBytes(n int64) string {
	const unit = 1000
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.0f %cB", float64(n)/float64(div), "kMGTPE"[exp])
}

// FilterWithCCLOptions limits what FilterWithCCL may hold in memory. The zero
// value sets no limit.
type FilterWithCCLOptions struct {
	// MaxRows is the most rows the filter may keep; 0 means no limit. The call
	// stops reading at the first kept row past it and returns nil and an error
	// matching ErrCCLRowLimit. It limits the result only, not the columns
	// CCLWholeColumns holds. A negative value is an error.
	MaxRows int64
	// RefuseWholeColumns makes an expression that would load whole columns an
	// error matching ErrCCLWholeColumnsRefused, before any data is read when
	// the plan already says so, and as soon as the data decides it otherwise.
	RefuseWholeColumns bool
}

var (
	// ErrCCLRowLimit is the error FilterWithCCL returns when it would keep
	// more rows than FilterWithCCLOptions.MaxRows.
	ErrCCLRowLimit = errors.New("parquet: the filter kept more rows than MaxRows")
	// ErrCCLWholeColumnsRefused is the error FilterWithCCL returns when the
	// expression needs whole columns and FilterWithCCLOptions.RefuseWholeColumns
	// forbids it.
	ErrCCLWholeColumnsRefused = errors.New("parquet: the expression needs whole columns, which RefuseWholeColumns forbids")
)

// filterOptions returns the one FilterWithCCLOptions a call may pass.
func filterOptions(opts []FilterWithCCLOptions) (FilterWithCCLOptions, error) {
	if len(opts) > 1 {
		return FilterWithCCLOptions{}, fmt.Errorf("FilterWithCCL takes at most one FilterWithCCLOptions, got %d", len(opts))
	}
	if len(opts) == 0 {
		return FilterWithCCLOptions{}, nil
	}
	if opts[0].MaxRows < 0 {
		return FilterWithCCLOptions{}, fmt.Errorf("FilterWithCCLOptions.MaxRows is %d; it must be 0 for no limit or a positive number", opts[0].MaxRows)
	}
	return opts[0], nil
}

// cclRoute is how one expression or statement is evaluated over a file.
type cclRoute struct {
	// resolved is the node with its whole-file parts computed; on the
	// whole-column route it is the node itself, since nothing was resolved.
	resolved     ccl.CCLNode
	seq          *ccl.TopSequence
	isSequence   bool
	wholeColumns bool
	reason       string // why wholeColumns
}

// routeCCL decides how node is evaluated over a file of totalRows rows whose
// columns are names, reading the whole-file parts from batches. FilterWithCCL,
// ApplyCCL and the plans all decide here, so a plan and its call agree. An
// expression the batches cannot answer, or a top-level sequence function the
// stream cannot take, is computed over whole columns.
func routeCCL(node ccl.CCLNode, totalRows int, names []string, batches ccl.Batches) (cclRoute, error) {
	// A refusal is not a failure: it chooses the whole-column route, and its
	// message is the reason the plan gives.
	if refusal := ccl.StreamRefusal(node); refusal != nil {
		return wholeColumnRoute(node, refusal), nil
	}
	resolved, err := ccl.ResolveWholeTable(node, totalRows, names, batches)
	if err != nil {
		return cclRoute{}, err
	}
	seq, isSequence, refusal := ccl.NewTopSequence(ccl.GetExpressionNode(resolved), totalRows, names)
	if refusal != nil {
		return wholeColumnRoute(node, refusal), nil
	}
	return cclRoute{resolved: resolved, seq: seq, isSequence: isSequence}, nil
}

// wholeColumnRoute is the route of node computed over whole columns, for the
// reason refusal gives.
func wholeColumnRoute(node ccl.CCLNode, refusal error) cclRoute {
	return cclRoute{resolved: node, wholeColumns: true, reason: refusal.Error()}
}

// countedBatches is a file with no rows that counts how many times it is read,
// which is how many passes routeCCL makes for the whole-file parts. The count
// does not depend on the values: each part is read the same number of times
// whatever the rows hold, until one fails.
func countedBatches(n *int) ccl.Batches {
	return func(yield func(batch ccl.GlobalRowContext) error) error {
		*n++
		return nil
	}
}

// planNode plans node over a file of totalRows rows whose columns are names.
// It reports the route, the passes the whole-file parts take, and an error the
// call would return before reading any data.
func planNode(node ccl.CCLNode, totalRows int, names []string) (plan CCLPlan, route cclRoute, err error) {
	plan.NumRows = int64(totalRows)
	passes := 0
	route, err = routeCCL(node, totalRows, names, countedBatches(&passes))
	if err != nil {
		return CCLPlan{}, cclRoute{}, err
	}

	colNameMap := make(map[string]int, len(names))
	for i, name := range names {
		colNameMap[name] = i
	}
	bound, bindErr := ccl.Bind(node, colNameMap)
	var needed []int
	if bindErr == nil {
		needed, bindErr = ccl.ReferencedColumns(bound, len(names))
	}
	for _, c := range needed {
		plan.NeededColumns = append(plan.NeededColumns, names[c])
	}

	part, readsTable := ccl.TopSequenceReadsTable(node)
	switch {
	case readsTable && ccl.StreamRefusal(node) == nil:
		// Whether the stream takes the sequence depends on the value the data
		// gives the part, which a plan read from no rows does not have.
		plan.Mode = CCLStreamedSequence
		plan.MayReadMore = true
		plan.Reasons = append(plan.Reasons, fmt.Sprintf("%s is computed from the data; if the sequence cannot take its value, the call loads whole columns", part))
		plan.MinDataPasses = passes
	case route.wholeColumns:
		// The whole-column route binds the expression before it reads, so an
		// expression it cannot bind is the call's error here too.
		if bindErr != nil {
			return CCLPlan{}, cclRoute{}, bindErr
		}
		plan.Mode = CCLWholeColumns
		plan.Reasons = append(plan.Reasons, route.reason)
		plan.HeldColumns = slices.Clone(plan.NeededColumns)
		plan.MinDataPasses = passes + 1
		plan.HeldCells = int64(totalRows)*int64(len(plan.HeldColumns)) + int64(totalRows)
	case route.isSequence:
		plan.Mode = CCLStreamedSequence
		plan.MinDataPasses = passes
	default:
		plan.Mode = CCLStreamed
		plan.MinDataPasses = passes
	}
	if row, ok := ccl.FixedRowReadsTable(node); ok && !route.wholeColumns {
		// Which row it is comes from the data, and a plan read from no rows
		// does not know whether reading it takes a pass of its own.
		plan.MayReadMore = true
		plan.Reasons = append(plan.Reasons, fmt.Sprintf("the row of %s is computed from the data, which can take another pass to read", row))
	}
	if bindErr != nil && plan.Mode != CCLWholeColumns {
		plan.Reasons = append(plan.Reasons, fmt.Sprintf("%v; the call reports it when it evaluates the first row", bindErr))
	}
	return plan, route, nil
}

// ExplainFilterWithCCL returns the plan FilterWithCCL follows for filterExpr on
// the file at path, read from the file's metadata and the expression without
// evaluating any row. An expression FilterWithCCL refuses before reading, such
// as one that does not compile, is refused the same way; an accepted plan does
// not promise the filter succeeds on the data.
func ExplainFilterWithCCL(ctx context.Context, path string, filterExpr string) (CCLPlan, error) {
	if ctx == nil {
		return CCLPlan{}, errNilContext
	}
	if err := ctx.Err(); err != nil {
		return CCLPlan{}, err
	}
	compiled, err := ccl.CompileExpression(filterExpr)
	if err != nil {
		return CCLPlan{}, fmt.Errorf("failed to compile CCL expression: %w", err)
	}
	totalRows, colNames, _, err := cclFileInfo(path)
	if err != nil {
		return CCLPlan{}, err
	}
	plan, _, err := planNode(compiled, totalRows, colNames)
	if err != nil {
		return CCLPlan{}, fmt.Errorf("failed to evaluate CCL expression: %w", err)
	}
	// The pass that keeps the rows.
	plan.MinDataPasses++
	plan.HoldsResult = true
	plan.EstimatedHeldBytes = heldBytes(path, plan.HeldColumns)
	return plan, nil
}

// ExplainApplyCCL returns one plan per statement of cclScript on the file at
// path, in order, read from the file's metadata and the script without
// evaluating any row. Each plan counts the passes its statement adds before the
// file is written; ApplyCCL then reads the file once more to write it, and
// twice more again when a column it writes holds another type of value further
// on, which only the data shows. A statement that writes nothing is reported as
// skipped, since ApplyCCL skips it. The file can change between this call and
// ApplyCCL.
func ExplainApplyCCL(ctx context.Context, path string, cclScript string) ([]CCLPlan, error) {
	if ctx == nil {
		return nil, errNilContext
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	compiled, err := ccl.CompileMultiline(cclScript)
	if err != nil {
		return nil, fmt.Errorf("failed to compile CCL script: %w", err)
	}
	totalRows, colNames, _, err := cclFileInfo(path)
	if err != nil {
		return nil, err
	}
	names := slices.Clone(colNames)
	plans := make([]CCLPlan, 0, len(compiled))
	for _, node := range compiled {
		_, _, isNew := ccl.GetNewColInfo(node)
		_, isAssignment := ccl.GetAssignmentTarget(node)
		if !isNew && !isAssignment {
			plans = append(plans, CCLPlan{NumRows: int64(totalRows), Reasons: []string{"the statement writes nothing, so ApplyCCL skips it"}})
			continue
		}
		// ApplyCCL refuses an assignment to a column the file does not have
		// before it reads, whichever route the statement takes.
		if target, ok := ccl.GetAssignmentTarget(node); ok {
			if _, found := resolveAssignTarget(target, names); !found {
				return nil, fmt.Errorf("failed to apply CCL: %w", assignTargetError(target, names))
			}
		}
		plan, route, err := planNode(node, totalRows, names)
		if err != nil {
			return nil, fmt.Errorf("failed to apply CCL: %w", err)
		}
		if route.wholeColumns && plan.Mode == CCLWholeColumns {
			if totalRows == 0 {
				// A statement over no rows has nothing to load.
				plan.MinDataPasses--
			}
			// The column the statement computes is kept until the file is
			// written, beside the columns it loads.
			plan.HeldCells = int64(totalRows) * int64(len(plan.HeldColumns)+1)
		}
		plan.EstimatedHeldBytes = heldBytes(path, plan.HeldColumns)
		plans = append(plans, plan)
		if newName, _, isNew := ccl.GetNewColInfo(node); isNew {
			names = append(names, newName)
		}
	}
	return plans, nil
}

// heldBytes sums the uncompressed size the columns cols have in the file at
// path, over every row group. It is an estimate for CCLPlan, so a file whose
// metadata cannot be read gives 0 rather than an error.
func heldBytes(path string, cols []string) (total int64) {
	if len(cols) == 0 {
		return 0
	}
	defer func() {
		if recover() != nil {
			total = 0
		}
	}()
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer func() { _ = f.Close() }()
	r, err := file.NewParquetReader(f, file.WithReadProps(readerProperties()))
	if err != nil {
		return 0
	}
	defer func() { _ = r.Close() }()

	want := make(map[string]bool, len(cols))
	for _, c := range cols {
		want[c] = true
	}
	md := r.MetaData()
	schema := md.Schema
	for g := 0; g < r.NumRowGroups(); g++ {
		rg := md.RowGroup(g)
		for c := 0; c < schema.NumColumns(); c++ {
			// A nested column has several leaves, each under its top-level name.
			path := schema.Column(c).ColumnPath()
			if len(path) == 0 || !want[path[0]] {
				continue
			}
			chunk, err := rg.ColumnChunk(c)
			if err != nil {
				continue
			}
			total += chunk.TotalUncompressedSize()
		}
	}
	return total
}
