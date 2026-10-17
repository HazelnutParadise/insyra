# Proposal: review-fixes-parquet-and-atomic-writes

## Why

Commit 5d757782 fixes defects in the Parquet reader and writer, the atomic file writers, the CSV formula guard, the Excel placeholder sheet and `ReadJSON`. Each one can be reached by a caller of the public API, and each is covered by a test added in the same commit:

- A `time.Duration` column could not be written and read back as `time.Duration`. Before, a Duration column read as `nil` cells, with a warning (`parquet/foreign_types_test.go`, `TestDurationRoundTripsAcrossReadersAndWriter`).
- A file with a page above arrow-go v18's default limits (64 MiB compressed, 256 MiB uncompressed) was refused. The reader now sets both limits to 1<<40 (`parquet/api.go`, `readerProperties`). Only the limit values are tested (`parquet/api_test.go`, `TestReaderPropertiesDisableArrowPageSizeLimits`).
- A panic inside the `io.ReaderAt` given to `ReadFrom` or `StreamFrom` ended the process, because Arrow reads columns in goroutines (`parquet/read_damaged_test.go`, `TestReaderAtPagePanicDoesNotCrashProcess`).
- `WriteTo` and `WriteToContext` panicked when a row group could not be started, and `WriteToContext` returned `nil` when the context was cancelled while the footer was written (`parquet/write_context_test.go`).
- The shared atomic writer (`internal/utils/atomic_file.go` at e643a5ff) renamed its temporary file over the path, so a symbolic link was replaced by a regular file. It also chmod-ed every file to 0644, ignored the umask, named the temporary file `.<name>.*.tmp` so a name near the 255-byte limit failed, and left the temporary file behind when the write callback panicked (`internal/utils/atomic_file_test.go`, `internal/utils/atomic_file_unix_test.go`).
- The CSV formula guard used `strconv.ParseFloat` to decide whether text was a number (`internal/csv/formula_guard.go` at e643a5ff). So `+Inf`, `-NaN` and `+0x1p2` were written unguarded, and `-1e309`, which overflows, was guarded (`internal/csv/formula_guard_test.go`, `TestGuardFormulaNumericSpellings`).
- A workbook whose only sheet was named `__insyra_placeholder__` kept its old content when that sheet was replaced, because the placeholder had the same name (`internal/excelsheet/replace_test.go`, `TestReplaceOnlySheetNamedPlaceholder`).
- `ReadJSON` read a value that implements both `json.Marshaler` and `io.Reader` as a stream (`read_json_file_test.go`, `TestReadJSONPrefersMarshalerOverReader`).

## What Changes

- **Behaviour change**: an atomic write follows a symbolic link at the path and writes the file it resolves to. Before, the link was replaced by a regular file. A link to a file that does not exist yet creates that file and stays a link.
- **Behaviour change**: an atomic write keeps the permission bits of an existing file. A new file is created with mode 0666 under the process umask. Before, every file was 0644.
- The temporary file of an atomic write is named `.insyra-<random>.tmp` in the target's directory. It is removed if the write callback panics, and the panic continues to the caller.
- The atomic writer is shared by `ToCSV`, `ToJSON`, `ToExcel`, Parquet `Write`, Parquet `ApplyCCL` and `GenerateLPFile`. The file geocode cache does not use it (see Impact).
- **Behaviour change**: the CSV formula guard prefixes text that is not a decimal number: `+Inf`, `-NaN` and hexadecimal floats such as `+0x1p2` now get the leading apostrophe. `-1e309` is written as it is, without the apostrophe it used to get.
- **Behaviour change**: `csvxl.AppendCSVToExcel` and `DataTable.ToExcel` with `SheetExistsReplace` replace a workbook whose only sheet is named `__insyra_placeholder__`, and the old content no longer stays. The placeholder sheet takes a free name such as `__insyra_placeholder___1`.
- **Behaviour change**: `ReadJSON` reads a value that is both `json.Marshaler` and `io.Reader` as the JSON its `MarshalJSON` returns.
- Parquet `Duration` columns round-trip as `time.Duration`. `Write` stores them as `INT64` nanoseconds with the Arrow `Duration` type kept in the file's Arrow schema. The readers restore `time.Duration`, scaled from the stored unit, when the file stores its Arrow schema.
- `Read`, `Stream`, `Inspect`, `ApplyCCL` and `FilterWithCCL` read a file with a page above arrow-go v18's default limits.
- `ReadFrom` and `StreamFrom` return an error when the caller's `io.ReaderAt` panics.
- `WriteTo` and `WriteToContext` return the error of a row group that cannot be started or written, and do not panic. `WriteToContext` returns `ctx.Err()` when the context is cancelled while the footer is written.
- `Docs/parquet.md` (Duration row and paragraph) and `Docs/lpgen.md` (`GenerateLPFile` description) are updated. The old text said `GenerateLPFile` made a new 0644 file, replaced a symbolic link and did not keep permissions.
- `CHANGELOG.md` and `CHANGELOG_TW.md` gained entries under `## Unreleased` for the items above.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `core-atomic-file-output`: the `ToCSV`/`ToJSON` requirement now names the resolved target directory, and four requirements are added for symbolic links, permission bits, the umask and the temporary file.
- `parquet-write`: the temporary-file requirement names the resolved target directory; two requirements are added for row group errors and footer cancellation.
- `lpgen-model-files`: the temporary-file requirement names the resolved target directory.
- `parquet-ccl-output`: the temporary-file requirement names the resolved target directory.
- `parquet-column-types`: two requirements are added for `Duration` columns.
- `parquet-damaged-files`: two requirements are added, for a panicking `ReaderAt` and for large pages. The large-page requirement is the nearest fit among the existing capabilities; a separate capability for reader limits is the alternative if the owner prefers it.
- `csv-formula-guard`: the requirement defines a number by the decimal literal shape, and a scenario for `+Inf`, `-NaN`, `+0x1p2` and `-1e309` is added.
- `excel-write`: one requirement is added for a replacement over a sheet named as the placeholder.
- `reader-writer-io`: one requirement is added for `ReadJSON` reading a `json.Marshaler` before an `io.Reader`.

## Impact

- `internal/utils/atomic_file.go`, `internal/utils/atomic_file_test.go`, `internal/utils/atomic_file_unix_test.go`.
- `internal/csv/formula_guard.go` and its test; `internal/excelsheet/replace.go` and its test.
- `parquet/api.go`, `parquet/ccl.go`, `parquet/internal.go`, and the parquet tests named above.
- `read.go` and `read_json_file_test.go`.
- `Docs/parquet.md`, `Docs/lpgen.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`.
- Output changes: files written by the atomic writers now have their permission bits or the umask's mode; a symbolic link at a written path stays a link; `+Inf`, `-NaN` and hexadecimal-float text in CSV output gets the apostrophe; `-1e309` loses it.
- Not covered: the file geocode cache (`datafetch/geocoding.go`, `fileGeocodeCache.persist`) writes `path + ".tmp"` with `os.WriteFile(..., 0o644)` and renames it over the path. It still replaces a symbolic link and forces 0644. The CHANGELOG entries of this batch name the file geocode cache among the writers that now follow symbolic links, which is not true of this code. The entry must be corrected, or the cache must be moved to the shared writer, in a separate change.
- The changes are not marked **BREAKING** in the CHANGELOG. The owner decides whether the symbolic link, permission and CSV output changes need that mark.
