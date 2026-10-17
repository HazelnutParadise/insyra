# Tasks: review-fixes-parquet-and-atomic-writes

## 1. Parquet durations, page limits and writer errors

- [x] 1.1 `parquet/internal.go`, `parquet/api.go`: a `time.Duration` column is written as `INT64` nanoseconds with the Arrow `Duration` type kept in the stored schema (`parquetWriteTable`), and read back as `time.Duration` scaled from the stored unit (`normalizeStoredDurationSchema`, `restoreStoredDuration`, `restoreDurationSlice`); test `TestDurationRoundTripsAcrossReadersAndWriter` in `parquet/foreign_types_test.go`
- [x] 1.2 `parquet/api.go`, `parquet/ccl.go`, `parquet/internal.go`: every reader is opened with `readerProperties()`, which sets both page limits to 1<<40; test `TestReaderPropertiesDisableArrowPageSizeLimits` in `parquet/api_test.go` checks the values only
- [x] 1.3 `parquet/api.go`: `ReadFrom` and `StreamFrom` wrap the caller's `io.ReaderAt` in `panicSafeReaderAt`; test `TestReaderAtPagePanicDoesNotCrashProcess` in `parquet/read_damaged_test.go`
- [x] 1.4 `parquet/api.go`: `WriteToContext` uses `NewRowGroupChecked`, and returns `ctx.Err()` after the footer; tests `TestWriteToReturnsNewRowGroupError` and `TestWriteToContextChecksCancellationAfterFooter` in `parquet/write_context_test.go`
- [x] 1.5 `Docs/parquet.md`: the `Duration` row of the type table and the paragraph on writing durations

## 2. Atomic writes

- [x] 2.1 `internal/utils/atomic_file.go`: `resolveFinalSymlinks` writes to the resolved target, existing modes are kept, new files get 0666 under the umask, the temporary file is `.insyra-<random>.tmp` in the target's directory, and a deferred cleanup removes it on failure or panic
- [x] 2.2 `internal/utils/atomic_file_test.go`: `TestWriteFileAtomicallyFollowsSymlinkAndKeepsMode`, `TestWriteFileAtomicallyLongNameAndPanicCleanup`
- [x] 2.3 `internal/utils/atomic_file_unix_test.go`: `TestWriteFileAtomicallyAppliesUmaskToNewFiles`, `TestWriteFileAtomicallyKeepsAnExistingFilesModeUnderUmask`
- [x] 2.4 `Docs/lpgen.md`: the `GenerateLPFile` description of the file mode, symbolic links and the temporary file

## 3. CSV formula guard

- [x] 3.1 `internal/csv/formula_guard.go`: the decision is a decimal-literal pattern instead of `strconv.ParseFloat`
- [x] 3.2 `internal/csv/formula_guard_test.go`: `TestGuardFormulaNumericSpellings`

## 4. Excel placeholder sheet

- [x] 4.1 `internal/excelsheet/replace.go`: the placeholder takes a name no sheet uses (`availableSheetName`), and a check that the sheet count rose
- [x] 4.2 `internal/excelsheet/replace_test.go`: `TestReplaceOnlySheetNamedPlaceholder`

## 5. ReadJSON

- [x] 5.1 `read.go`: a `json.Marshaler` case before the `io.Reader` case in `ReadJSON`
- [x] 5.2 `read_json_file_test.go`: `TestReadJSONPrefersMarshalerOverReader`

## 6. Changelogs

- [x] 6.1 `CHANGELOG.md` and `CHANGELOG_TW.md`: entries under `## Unreleased` for the atomic writes, `ReadJSON`, the Excel placeholder, the formula guard, the Parquet durations, page limits, `ReaderAt` panics and writer errors

## 7. Follow-through

- [x] 7.1 `datafetch/geocoding.go` `fileGeocodeCache.persist` writes through `internal/utils.WriteFileAtomically`, so the changelog's list of atomic writers is true and a file kept at `<path>.tmp` is no longer overwritten and deleted (`TestFileGeocodeCacheLeavesAFileAtDotTmpAlone`).
- [x] 7.2 Not marked **BREAKING**: following a symbolic link and keeping an existing file's permission bits is what `os.WriteFile` and v0.3.3 did, and the replace-the-link, always-0644 behaviour was only ever on the unreleased `0.4` line.
- [x] 7.3 `TestAPageAboveArrowsDefaultLimitReads` writes a 70 MiB value and reads it back through `Read` (skipped under `-short`); on `0.4` without the fix it fails with `compressed page size 73400324 exceeds configured limit 67108864`.
- [x] 7.4 `TestWriteThenReadKeepsADurationColumn` writes a `time.Duration` column with the public `Write` and reads it back with `Read`.
- [x] 7.5 `TestToExcelReplacesAWorkbooksOnlySheetNamedLikeThePlaceholder` drives `ToExcel` over a workbook whose only sheet has the placeholder's name and holds more cells than the new table.
- [x] 7.6 `TestWriteFileAtomicallyKeepsAnExistingFilesModeUnderUmask` pins that an existing file keeps its mode through the umask; the writer resolves a path's drive letter before following links.
