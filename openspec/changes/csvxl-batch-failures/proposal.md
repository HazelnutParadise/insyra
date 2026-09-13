## Why

`CsvToExcel` and `AppendCsvToExcel` keep going when one CSV in a batch fails, but they lose the reason and leave damage behind. The error says only "1 files failed to convert". The output workbook gets an empty sheet named after the failed file. `AppendCsvToExcel` empties an existing sheet of the same name before it finds out the CSV cannot be read, then saves the workbook: measured on 2026-09-13, appending a missing CSV to sheet `data` erased the cell holding "keep me". The owner chose batch tolerance for #267 (C-1): convert every file that works, and say exactly which ones did not.

## What Changes

- Each CSV is read and decoded in full before its sheet is created or replaced. A CSV that fails leaves no sheet behind in `CsvToExcel`, and leaves an existing sheet of the same name as it was in `AppendCsvToExcel`.
- A sheet that cannot be created for one file, such as a name Excel does not allow, is that file's failure rather than the end of the batch.
- The returned error joins one error per failed file with `errors.Join`. Each names the CSV and wraps its cause, so `errors.Is(err, os.ErrNotExist)` works.
- The workbook is saved when at least one CSV succeeded. When none did, `CsvToExcel` writes no file and `AppendCsvToExcel` leaves the workbook untouched.
- `EachCsvToOneExcel` goes through `CsvToExcel` and follows the same rules.
- **BREAKING**: a failed file no longer produces an empty sheet, a bad sheet name no longer stops the batch before anything is saved, and the error lists each file instead of a count.

## Capabilities

### New Capabilities
- `csvxl-batch-conversion`: how the CSV-to-workbook functions treat a batch in which some files fail.

### Modified Capabilities
None.

## Impact

- `csvxl/convert.go`, new tests in `csvxl/batch_failure_test.go`.
- `Docs/csvxl.md`, the `csvxl` example in `skills/insyra/SKILL.md`, both CHANGELOGs.
- `api-review.md` C-1, `delivery-status.md`, issue #267.
