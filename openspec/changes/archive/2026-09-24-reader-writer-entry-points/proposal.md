## Why

Every read and write entry point in `insyra` and `parquet` takes a file path (#210: K-11, C-10, Q-8). Data that is not a file on disk — an HTTP response, a zip entry, an `embed.FS` file, bytes in memory, an S3 object — must be written to a temporary file, read back and deleted. CSV is also read whole into memory with no streaming entry point, which #294 has been waiting on.

The owner ruled on 2026-09-25: add reader and writer entry points, keep every existing one as a wrapper, and leave renaming to #213, which reshapes the same functions' parameters.

## What Changes

Core:
- `ReadCSV(r io.Reader, opts CSVReadOptions)`; `ReadCSV_FileWithOptions` and `ReadCSV_StringWithOptions` wrap it. Encoding auto-detection samples the reader's first bytes, so it works on any source.
- `StreamCSV(r io.Reader, opts CSVReadOptions, batchSize int) iter.Seq2[*DataTable, error]`. The header names every batch's columns; types are inferred per batch.
- `(*DataTable).WriteCSV(w io.Writer, opts CSVWriteOptions)`; `ToCSVWithOptions` keeps its write-then-rename.
- `ReadJSON` accepts an `io.Reader`; `(*DataTable).WriteJSON(w io.Writer, useColNames bool)`, which `ToJSON` wraps.
- `ReadExcel(r io.Reader, sheetName string, setFirstColToRowNames, setFirstRowToColNames bool)`, with the path version's unzip limit.

`parquet`:
- `ReadFrom(ctx, r io.ReaderAt, size int64, opt ReadOptions)`, `StreamFrom(ctx, r io.ReaderAt, size int64, opt ReadOptions, batchSize int)`, `WriteTo(dt, w io.Writer)`; `Read`, `Stream` and `Write` wrap them.

Nothing existing changes signature or behaviour.

## Capabilities

### New Capabilities
- `reader-writer-io`: that every format can be read from a reader and written to a writer, and that the path versions are the same code.

### Modified Capabilities
None.

## Impact

- `read.go`, `utils.go`, `datatable_csv.go`, the JSON writer, `parquet/api.go`; `interfaces.go` for the new methods.
- `Docs/DataTable.md`, `Docs/parquet.md`; both changelogs; `skills/insyra/`.
- `api-review.md` K-11, C-10, Q-8, SEC-16; `delivery-status.md`; issues #210 and #294.
