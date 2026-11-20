# Design: parquet-ccl-explain-and-budget

Draft for #415 part 2, revised after an independent review. Nothing is implemented yet.

## Problem

`FilterWithCCL` reads a file 1,000 rows at a time, yet three things it can do are invisible until it runs:

- An aggregate such as `A > AVG(A)` reads the file again before any row is evaluated, once per aggregate source and twice for `VAR` and `STDEV`.
- An expression the batches cannot answer, such as `A > MEDIAN(A)` or `@`, loads every column it needs into memory, the whole file for `@`.
- Every kept row is held in memory, so `A > 0` on a 50-million-row file returns a 50-million-row table.

A caller cannot ask beforehand which of these will happen, and cannot cap any of them. `ApplyCCL` writes to a file, so only the first two apply to it.

## Proposed API

### Explain: metadata only, no row evaluated

```go
// CCLPlan says how FilterWithCCL or one statement of ApplyCCL would run on a
// file. It is read from the file's metadata and the expression alone.
type CCLPlan struct {
	Mode    CCLMode // how the rows are evaluated
	Reasons []string // why Mode is not CCLStreamed, or why it may change at run time

	NeededColumns []string // the columns the expression reads, in file order
	HeldColumns   []string // the columns loaded whole into memory; empty unless CCLWholeColumns
	NumRows       int64    // the file's row count, from its metadata

	// Exact counts for a run that succeeds on this file.
	MinDataPasses int   // full reads of the file's data, the final one included
	HeldCells     int64 // values held in memory besides the result: NumRows × len(HeldColumns), plus one decision per row
	HoldsResult   bool  // every kept row is held in memory (FilterWithCCL)

	// MayReadMore is true when the run can read the file more times than
	// MinDataPasses, or switch to CCLWholeColumns, for reasons only the data
	// decides; Reasons says which.
	MayReadMore bool

	// EstimatedHeldBytes is an estimate: the uncompressed on-disk size of
	// HeldColumns, summed over the row groups. Memory in Go is usually larger.
	EstimatedHeldBytes int64
}

// CCLMode is how the rows are evaluated.
type CCLMode int

const (
	CCLStreamed         CCLMode = iota // batch by batch
	CCLStreamedSequence                // a top-level LAG, CUMSUM, ROLLING_*: batch by batch, holding the rows a later batch is needed for
	CCLWholeColumns                    // the needed columns are loaded whole, then evaluated as the table does
)

func (m CCLMode) String() string // "CCLStreamed", …; an unknown value prints CCLMode(n)

func ExplainFilterWithCCL(ctx context.Context, path string, filterExpr string) (CCLPlan, error)
func ExplainApplyCCL(ctx context.Context, path string, cclScript string) ([]CCLPlan, error)
```

How each field is known:

- `Mode`, `HeldColumns`: from the checks the real call runs first (`StreamRefusal`, the structural part of `NewTopSequence`). One case depends on the data: a sequence function whose argument is itself whole-file, such as `LAG(A, COUNT(A))`, is streamed only if that value is accepted. Explain then reports `CCLStreamedSequence` with `MayReadMore` and a reason, rather than reading data.
- `MinDataPasses`: the passes `ResolveWholeTable` takes (one per round for fixed rows, one per round per aggregate source, two for `VAR`/`STDEV`), plus the column load, plus the final read. A zero-row file reads nothing. `ApplyCCL` may read twice more when a written column changes type, which `MayReadMore` reports.
- `NeededColumns`: the expression bound against the file's column names, as the loaded table binds it. Each pass still reads every column of each batch today; narrowing that is a separate change.
- Explain takes a context because it opens the file and reads its footer.

Explain answers in one direction only: a plan Explain refuses (a syntax error, a missing column) is a call that fails the same way, but a plan Explain accepts can still fail on the data, such as `STDEV` over one value.

`FilterWithCCL` and `ApplyCCL` will build the same plan and run from it, so Explain cannot drift from the call. Tests count the passes the call really makes, with a counter like the existing `wholeColumnsLoaded`, and compare them with `MinDataPasses`.

`ExplainApplyCCL` returns one plan per statement, in order. A statement that writes nothing is reported with `CCLStreamed`, zero passes and a reason saying it is skipped, since `ApplyCCL` skips it.

Example:

```go
// sales.parquet has a column named amount; CCL names a column ['amount'].
plan, err := parquet.ExplainFilterWithCCL(ctx, "sales.parquet", "['amount'] > MEDIAN(['amount'])")
// plan.Mode == parquet.CCLWholeColumns
// plan.Reasons == []string{"MEDIAN cannot be computed batch by batch"}
// plan.HeldColumns == []string{"amount"}
// plan.MinDataPasses == 2, plan.HoldsResult == true
```

### Budgets on FilterWithCCL

```go
// FilterWithCCLOptions limits what FilterWithCCL may hold in memory. The zero
// value is today's behaviour: no limit.
type FilterWithCCLOptions struct {
	MaxRows            int64 // 0 = no limit; more kept rows than this is an error. Negative is an error.
	RefuseWholeColumns bool  // an expression that would load whole columns is an error before any data is read
}

var (
	ErrCCLRowLimit           = errors.New("parquet: the filter kept more rows than MaxRows")
	ErrCCLWholeColumnsRefused = errors.New("parquet: the expression needs whole columns, which RefuseWholeColumns forbids")
)

func FilterWithCCL(ctx context.Context, path string, filterExpr string, opts ...FilterWithCCLOptions) (*insyra.DataTable, error)
```

- `MaxRows` is a hard limit checked as each row is kept, so it cannot be checked before reading the way `ReadColumnOptions.MaxValues` is. The call stops at the first row past the limit, closes the file and returns `nil` with an error that matches `errors.Is(err, ErrCCLRowLimit)` and names the limit. No partial table is returned, as with `MaxValues`.
- `MaxRows` limits the result only. The memory of `HeldColumns` and of a sequence function's held rows is what `RefuseWholeColumns` and Explain are for.
- `RefuseWholeColumns` is checked before any data pass, and its error carries the plan's reasons. A plan with `MayReadMore` that would switch to whole columns at run time fails with the same error at that point.
- More than one `FilterWithCCLOptions` is an error, as the settings rule requires.

### ApplyCCL keeps its signature

`ApplyCCL(ctx, path, script, opts ...WriteOptions)` shipped in 0.3.x, so it cannot gain a second options type: Go allows one trailing variadic, and reusing `WriteOptions` is not possible because passing one replaces the source file's layout. A caller who wants to refuse whole columns calls `ExplainApplyCCL` first. The file can change between the two calls; that gap is documented.

## Both lines

The change ships on `0.4` and on `dev` (0.3.6) with the same names and behaviour. Both lines have the same streaming machinery (`StreamRefusal`, `ResolveWholeTable`, `NewTopSequence`); `git diff origin/dev origin/0.4` touches 53 lines of `parquet/ccl.go` and `internal/ccl/stream_resolve.go`, so the work is done on `0.4` and moved to `dev` by cherry-pick with those hunks adjusted, not by a merge.

`ApplyCCL` keeps `opts ...WriteOptions` on both lines for the same reason: it shipped in 0.3.x. On `dev` a breaking signature change is not acceptable at all, so `ExplainApplyCCL` followed by `ApplyCCL` is the only way there, with the same check-then-act gap documented. If 0.4 later wants `RefuseWholeColumns` inside `ApplyCCL`, it needs a new function, which `dev` would then get under the same name.

## Compatibility

- `FilterWithCCL` gains a trailing variadic, so every existing call compiles and behaves as before. Code that stores it in a variable of the old function type would stop compiling; nothing in the module does.
- Everything else is new. Nothing changes without an option.

## Out of scope

- Reading only `NeededColumns` in each batch.
- Byte budgets and spilling to disk.
- Part 3 of #415 (a scan, select, filter and sink pipeline), which would reuse `CCLPlan`.

## Decided in this revision

- Names follow the functions they explain (`ExplainFilterWithCCL`, `FilterWithCCLOptions`) and `FileInfo.NumRows`.
- Two sentinel errors, because a refusal before reading and a limit hit mid-read call for different handling.
- Pass counts are a minimum plus a flag, and the byte figure is labelled an estimate, because only those can be stated without reading data.
