# Design: parquet-ccl-stream-filter

Draft for #415 part 3, revised after an independent review.

## Problem

`FilterWithCCL` returns every kept row as one `DataTable`, so a filter that keeps most of a large file holds most of it, and `MaxRows` can only refuse it. #415 asks for a limited scan → select/filter → output path that reuses `Stream` and the CCL batch semantics instead of a second evaluator, gives the rows as an iterator or to a sink, and collects into a `DataTable` only when asked.

## Proposed API

```go
// StreamFilterWithCCL gives the rows of path that filterExpr keeps, in file
// order, batch by batch, without holding them.
func StreamFilterWithCCL(ctx context.Context, path string, filterExpr string, opts ...FilterWithCCLOptions) iter.Seq2[*insyra.DataTable, error]

type FilterWithCCLOptions struct {
	MaxRows            int64
	RefuseWholeColumns bool
	// Columns are the columns of the result, by name, in the order given; empty
	// means every column. They are names, as in ReadOptions.Columns. The
	// filter may read any column of the file, and every pass still reads every
	// column: Columns saves the result's memory only.
	Columns []string
}
```

Example:

```go
for batch, err := range parquet.StreamFilterWithCCL(ctx, "sales.parquet", "['amount'] > 100",
	parquet.FilterWithCCLOptions{Columns: []string{"id", "amount"}}) {
	if err != nil {
		return err
	}
	if err := w.Write(batch); err != nil { // any sink: a CSV writer, a database, a channel
		return err
	}
}
```

## How it works

- `FilterWithCCL` and `StreamFilterWithCCL` share one internal producer of kept rows: the three paths (streamed, sequence, whole columns) hand each run of kept rows to a callback instead of appending to one table. `FilterWithCCL` appends the raw rows and builds one table, as now; the stream builds a table per run. One evaluator, so both give the same values, order, column names and missing values.
- Each yielded batch holds the kept rows of one read of up to 1,000 rows (the sequence path yields what its stage releases); batch sizes are not promised. A run with no kept rows is not yielded, and when nothing at all is kept the stream yields one table with the columns and no rows, so a sink still gets its header.
- Compiling, reading the file's shape and deciding the route happen inside the iterator, so their errors arrive as `(nil, err)`; the clock is frozen once per range. A nil context yields an error.
- Memory: the streamed and sequence paths hold one batch plus the rows a sequence needs from a later batch. The whole-column path still holds the needed columns and one decision per row, as `CCLPlan` reports; `RefuseWholeColumns` refuses it.
- `MaxRows` counts the rows yielded; the row past it ends the sequence with `ErrCCLRowLimit` after the rows within the limit were yielded. `FilterWithCCL` keeps returning `nil` and no rows for it.
- Any error after the first batch (an evaluation error at a later row, cancellation, a file that changed while read) arrives after earlier batches were yielded; what a sink already wrote is the caller's to keep or discard. The source is a file, so no snapshot is taken.
- Breaking out of the loop stops the read and closes the file, as `Stream` does; cancelling `ctx` ends the sequence with the context's error.
- `ExplainFilterWithCCL`'s plan describes the stream too, except `HoldsResult`, which is false for it; the comments and docs say so instead of adding a function.
- `Columns` drops the other columns from each yielded batch after the filter is evaluated, so the filter can read a column the result leaves out. A name the file does not have, or one given twice, is an error before anything is read. It applies to `FilterWithCCL` too.

## Out of scope

- A Parquet file sink (`WriteTo` for a stream of tables), aggregates over the stream, joins, several files, and reading only the needed columns in each batch.
- Lazy versions of the eager `DataTable` methods.

## Both lines

The function and the option are new, so they ship on `0.4` and `dev` with the same names and the same `iter.Seq2` return type. `dev`'s `Stream` returns channels, but `dev` is on Go 1.26 too, and one signature per name lets a caller upgrade without rewriting the loop.

## Testing

On a 2,500-row file: the stream's batches appended, and `FilterWithCCL`'s table, both equal the loaded table's own CCL filter for `A > AVG(A)`, `A > MEDIAN(A)`, a condition keeping every row, `@` and `LAG` across batch boundaries; `Columns` order and missing values; `MaxRows` yielding exactly the rows within the limit; breaking out early and cancelling leave no goroutine and the file closed; a CSV sink written per batch and read back; and a benchmark of peak memory against `FilterWithCCL` on a filter that keeps every row.
