# Proposal: parquet-arrow-go-v18

## Why

`parquet` reads and writes through `github.com/apache/arrow/go/v17` v17.0.0. In that version `(*recordReader).ReadRecords` returns a nil error after a page header it cannot decode, so a damaged row group reads as an empty one and the record reader reports `io.EOF`. Upstream fixed it as apache/arrow#43860 on 2024-08-28 and shipped the fix in arrow-go v18.0.0, the first release under the module path `github.com/apache/arrow-go/v18`, which this module never moved to. Measured on 2026-10-03 with a 3,000-row file whose third row group's first page header was overwritten: v17.0.0 returns 2,000 rows and a nil error from `ReadTable`, and every v18 release measured, v18.0.0 to v18.8.0, returns `parquet: deserializing page header failed`. `parquet-damaged-row-groups` guards against the short read by counting rows, but every other fix since v17 is missing too. The owner asked for the newest version on 2026-10-03.

## What Changes

- `parquet` imports `github.com/apache/arrow-go/v18` at v18.8.0, the newest release, instead of `github.com/apache/arrow/go/v17`. v18.8.0 needs Go 1.25, below this module's directive, and neither module path has a GitHub advisory.
- Code the new version marks Deprecated moves to the replacement its deprecation names.
- When the Arrow reader itself fails on a row group, which v18 now does for a page it cannot decode, the error names the file and the row groups that did not read in full, as the row-count check already does, and wraps the reader's error. A cancelled or expired context is still returned as it is.
- A damaged Snappy page is an error. v17 decompressed pages in a goroutine of its own and panicked there with `snappy: corrupt input`, which no `recover` in this package can catch, so a read ended the program; v18 returns `could not decompress page`. Measured on 2026-10-03, and pinned by a test that ends the test binary on v17.
- The public API is unchanged: no exported name takes or returns an Arrow type.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `parquet-damaged-files`: a damaged row group the Arrow reader reports is named like one the row count finds.

## Impact

- `go.mod`, `go.sum`, the 11 files of `parquet/` that import Arrow.
- `Docs/parquet.md` if it names the Arrow module, both changelogs (`parquet`), `AGENTS.md` (the follow-ups on moving to v18 and on the damaged Snappy page are resolved), `delivery-status.md`.
