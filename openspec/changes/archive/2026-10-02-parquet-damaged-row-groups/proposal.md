# Proposal: parquet-damaged-row-groups

## Why

A Parquet file whose page header in a later row group is damaged reads as if that row group were empty: Arrow takes the page for the end of the row group and goes on with the next. Measured on 2026-10-02 on a 3,000-row file written with `RowGroupSize: 1000` and eight bytes of its third row group's first page header overwritten: `Read` returns 2,000 rows with a nil error, and `ApplyCCL` writes those 2,000 rows back over the file, so the last 1,000 rows, still in the damaged file, are gone. The cause is upstream: Arrow's page reader records the failure (`parquet: deserializing page header failed`), but `file.recordReader.ReadRecords` treats `HasNext()` returning false as the end of the row group without asking for that error, the leaf reader gets zero records, and `pqarrow`'s record reader reports `io.EOF`, which callers take for the end of the file. arrow-go v18.0.0 and later report the error (apache/arrow#43860, corrected on 2026-10-03).

## What Changes

- Every reader checks the rows it read against the row count the file's metadata gives for the row groups it read, and fails when they differ, naming the file, both counts and the row groups that did not read in full: `Read`, `ReadFrom`, `ReadColumn`, `Stream`, `StreamFrom`, `FilterWithCCL` and `ApplyCCL`.
- `ApplyCCL` therefore leaves a damaged file as it was instead of replacing it with its first rows.
- Reading only the intact row groups of a damaged file, through `ReadOptions.RowGroups`, still works.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `parquet-damaged-files`: a damaged row group is an error, never a short read.

## Impact

- `parquet/api.go` (`readTableFrom`), `parquet/internal.go` (`streamAsArrowRecord`).
- `Docs/parquet.md`, both changelogs, `AGENTS.md` (the follow-up is resolved; reporting the upstream defect waits for the owner), `delivery-status.md`.
