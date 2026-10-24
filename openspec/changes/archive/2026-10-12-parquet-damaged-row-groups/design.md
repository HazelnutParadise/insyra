# Design: parquet-damaged-row-groups

## Context

`file.serializedPageReader.Next` sets `p.err` when a page header cannot be decoded; `file.columnChunkReader.readNewPage` copies it into `c.err` and returns false. `file.recordReader.ReadRecords` loops `if !rr.HasNext() { break }` and returns the records read so far with a nil error, never consulting the reader's error. `pqarrow.leafReader.NextBatch` then moves past the row group, finds no more, and returns an empty batch; `pqarrow.recordReader.next` turns an empty batch into `io.EOF`. `fr.ReadRowGroups` takes the same path and returns a short table without an error.

## Decisions

### 1. Count rows against the metadata

The footer's row counts are read before any value and do not depend on the pages, so a short read is visible as a count that does not match. Every value path in this package goes through either `readTableFrom` or `streamAsArrowRecord`, so the check lives in those two places. When the count is short, each selected row group is read again on its own to find the ones that did not read in full, and the error names them. A damaged middle row group reads as empty while the ones after it read in full, so the total alone cannot say where the damage is; reading again costs only on a damaged file.

### 2. Not an Arrow upgrade

Correction, 2026-10-03: this was read from the loop alone. arrow-go v18.0.0 and later return `rr.Err()` at the end of `ReadRecords` (apache/arrow#43860), and a damaged file read on v18.0.0 through v18.8.0 fails with the page header error, so moving off `arrow/go/v17` fixes the upstream half. The count check stays useful either way.

## Risks

- A file whose metadata is itself wrong about its row counts now fails to read where it used to read short. Such a file is damaged too.
