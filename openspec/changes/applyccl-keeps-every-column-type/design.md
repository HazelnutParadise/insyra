# Design: applyccl-keeps-every-column-type

## Decisions

### 1. Carry the file's arrays, not rebuild them

Every cell is also read into a Go value, because statements evaluate on Go values. A list, a struct or a map reads as `nil`, so no builder could rebuild one from its Go value, and a decimal's exact scale would need its own round trip. A run therefore carries, beside its Go values, the Arrow array each column of the file came from. A stage that writes a column drops that column's array; a stage that splits or regroups rows slices the arrays with the rows (`array.NewSlice`); the writer writes an unwritten column's array as it is. The values never pass through Go for such a column, so they come back bit for bit, whatever the type.

### 2. Arrays are retained while a run holds them

A record's arrays are released when its batch is done, so a run retains each array it keeps and slices retain their parent. The writer retains what it puts into a record. The default allocator is Go's, so a reference not released is reclaimed by the garbage collector rather than leaked, but the stages release what they drop where the code makes that plain.

### 3. Narrower types for a written column

`holdsEveryValue` already keeps a written column's type when every value fits. It learns `int8`, `int16`, `int32` and the unsigned widths (a whole number in range, from any Go integer or a whole `float64`), `float32` (a value that survives a round trip through `float32`), `Date32` (a time at midnight UTC) and `Date64` (a time at a whole millisecond of a day at midnight UTC), and the builder builds them. A decimal column that is assigned takes the type `Write` gives, because CCL computes in `float64`.
