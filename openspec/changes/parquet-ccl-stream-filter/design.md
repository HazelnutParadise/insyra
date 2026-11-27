# Design: parquet-ccl-stream-filter

Draft for #415 part 3. Nothing is implemented yet.

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
	// means every column. The filter may read any column of the file.
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

- `FilterWithCCL` becomes the collect of `StreamFilterWithCCL`: the three paths (streamed, sequence, whole columns) yield each batch of kept rows instead of appending it to one table, and `FilterWithCCL` appends the batches. One evaluator, so the stream and the table give the same values, order, column names and missing values, and an empty result is a table with the columns and no rows in both.
- Each yielded batch holds the kept rows of one 1,000-row read (the sequence path yields what its stage releases). A batch with no kept rows is not yielded.
- Memory: the streamed and sequence paths hold one batch plus the rows a sequence needs from a later batch. The whole-column path still holds the needed columns and one decision per row, as `CCLPlan` reports; `RefuseWholeColumns` refuses it.
- `MaxRows` counts the rows yielded; the row past it ends the sequence with `ErrCCLRowLimit` after the rows within the limit were yielded. `FilterWithCCL` keeps returning `nil` and no rows for it.
- Breaking out of the loop stops the read and closes the file, as `Stream` does; cancelling `ctx` ends the sequence with the context's error.
- `Columns` drops the other columns from each yielded batch after the filter is evaluated, so the filter can read a column the result leaves out. A name the file does not have is an error before anything is read. It applies to `FilterWithCCL` too.

## Out of scope

- A Parquet file sink (`WriteTo` for a stream of tables), aggregates over the stream, joins, several files, and reading only the needed columns in each batch.
- Lazy versions of the eager `DataTable` methods.

## Both lines

The function and the option are new, so they ship on `0.4` and `dev` with the same names; `dev`'s `Stream` returns channels, so `dev`'s `StreamFilterWithCCL` returns the same `(<-chan *insyra.DataTable, <-chan error)` pair its `Stream` does, keeping one iteration style per line.

## Testing

On a 2,500-row file: the stream's batches appended equal `FilterWithCCL`'s table for `A > AVG(A)`, `A > MEDIAN(A)`, a condition keeping every row, `@` and `LAG` across batch boundaries; `Columns` order and missing values; `MaxRows` yielding exactly the rows within the limit; breaking out early and cancelling leave no goroutine and the file closed; and a benchmark of peak memory against `FilterWithCCL` on a filter that keeps every row.
