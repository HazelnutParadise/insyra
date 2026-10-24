# Proposal: parquet-arrow-go-v18

## Why

`parquet` reads and writes through `github.com/apache/arrow/go/v17` v17.0.0. In that version `(*recordReader).ReadRecords` returns a nil error after a page header it cannot decode, so a damaged row group reads as an empty one and the record reader reports `io.EOF`. Upstream fixed it as apache/arrow#43860 on 2024-08-28 and shipped the fix in arrow-go v18.0.0, the first release under the module path `github.com/apache/arrow-go/v18`, which this module never moved to. Measured on 2026-10-03 with a 3,000-row file whose third row group's first page header was overwritten: v17.0.0 returns 2,000 rows and a nil error from `ReadTable`, and every v18 release measured, v18.0.0 to v18.8.0, returns `parquet: deserializing page header failed`. `parquet-damaged-row-groups` guards against the short read by counting rows, but every other fix since v17 is missing too. The owner asked for the newest version on 2026-10-03.

## What Changes

- `parquet` imports `github.com/apache/arrow-go/v18` at v18.8.0, the newest release, instead of `github.com/apache/arrow/go/v17`. v18.8.0 needs Go 1.25, below this module's directive, and neither module path has a GitHub advisory.
- Names the new version marks Deprecated move to their replacements: `arrow.Record` and `array.NewRecord` become `arrow.RecordBatch` and `array.NewRecordBatch` in `parquet/ccl.go`.
- When the Arrow reader itself fails on a row group, which v18 now does for a page it cannot decode, the error names the file and the row groups that did not read in full, as the row-count check already does, and wraps the reader's error. A cancelled or expired context is returned as it is.
- A damaged Snappy page is an error. v17 decompressed pages in a goroutine of its own and panicked there with `snappy: corrupt input`, which no `recover` in this package can catch, so a read ended the program; v18 returns `could not decompress page`. Measured on 2026-10-03, and pinned by `TestReadRefusesADamagedSnappyPage` in `parquet/damaged_snappy_test.go`.
- The public API is unchanged: no exported name takes or returns an Arrow type.
- The page size limits are set to `1 << 40`, so a file v17 read is still read.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `parquet-damaged-files`, with a new requirement: an Arrow reader failure on a row group is named like one the row count finds. It is added rather than modified because the requirement it would modify is not in the main specs yet.

## Impact

- `go.mod`, `go.sum`, the `parquet/` files that import Arrow.
- `CHANGELOG.md`, `CHANGELOG_TW.md`.
