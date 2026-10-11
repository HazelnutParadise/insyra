## Why

Every read and write entry point in `insyra` takes a file path (#210: K-11, C-10). Data that is not a file on disk — an HTTP response, a zip entry, an `embed.FS` file, bytes in memory, an S3 object — must be written to a temporary file, read back and deleted. CSV is also read whole into memory with no streaming entry point, which #294 has been waiting on.

The owner ruled on 2026-09-25: add reader and writer entry points, keep every existing one as a wrapper, and leave renaming to 0.4 (#213), which reshapes the same functions' parameters.

## What Changes

Core:
- `ReadCSV(r io.Reader, opts CSVReadOptions)`; `ReadCSV_FileWithOptions` and `ReadCSV_StringWithOptions` wrap it. Encoding auto-detection samples the reader's first bytes, so it works on any source.
- `StreamCSV(r io.Reader, opts CSVReadOptions, batchSize int) iter.Seq2[*DataTable, error]`. The header names every batch's columns; types are inferred per batch.
- `(*DataTable).WriteCSV(w io.Writer, opts CSVWriteOptions)`; `ToCSVWithOptions` keeps its write-then-rename.
- `ReadJSON` accepts an `io.Reader`; `(*DataTable).WriteJSON(w io.Writer, useColNames bool)`, which `ToJSON` wraps. A value that implements both `json.Marshaler` and `io.Reader` is marshaled first as before.
- `ReadExcel(r io.Reader, sheetName string, setFirstColToRowNames, setFirstRowToColNames bool)`.
- File path readers share the new shared paths, read the same tables as before, and retain their original error text when the file cannot be read during auto-detection.

`parquet`'s `ReadFrom` and `WriteTo` come with the parquet port; `parquet.StreamFrom` stays on 0.4 with its iterator `Stream`. Left on 0.4: read-write-names (renames, `NoHeaderRow`, `HasRowNames`), one value per optional parameter, CSV formula guard on by default, and exported function variables becoming functions.

Nothing existing changes signature or behaviour.

## Capabilities

### New Capabilities
- `reader-writer-io`: that every core format can be read from a reader and written to a writer, and that the path versions are the same code.

### Modified Capabilities
None.

## Impact

- `read.go`, `utils.go`, `datatable_csv.go`, `datatable_json.go`; `interfaces.go` for the new methods.
- `Docs/DataTable.md`; both changelogs.
