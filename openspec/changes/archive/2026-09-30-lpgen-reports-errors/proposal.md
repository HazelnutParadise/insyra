# Proposal: lpgen-reports-errors

## Why

LP-3 of the API review ([#258](https://github.com/HazelnutParadise/insyra/issues/258)) found that `lpgen` reports none of its failures to the caller:

- `GenerateLPFile(filename)` returns nothing. A path it cannot create and an objective type it does not know are logged as warnings, and the caller carries on as if the file were written. Worse, the file is opened with `os.Create` before the objective type is checked, so saving a model with an unknown objective type over an existing LP file truncates that file and leaves only the two banner lines in it.
- `ParseLingoModel_txt(path)` and `ParseLingoModel_str(text)` return `nil` with no error when the file cannot be opened or read, so the only signal is a `nil` model and a log line. Their names use underscores, which the owner's ruling on #211 and the naming half of #212 retire, and the two carry identical copies of the whole parser.

## What Changes

- **BREAKING** `(*LPModel).GenerateLPFile(filename string) error` returns the failure instead of logging it: a file that cannot be created or written, or an objective type other than minimize/maximize. It writes through a temporary file in the same directory that is renamed into place only after everything was written, so a failed call leaves no file behind and an existing file at `filename` untouched. A call used as a statement still compiles; only code that took the method as a value of type `func(string)` breaks.
- New `ParseLingo(model string) (*LPModel, error)` and `ParseLingoFile(path string) (*LPModel, error)`. They read what the old functions read and return the same model; a file that cannot be opened or read is an error instead of `nil`. Both run one parser over an `io.Reader`, so the duplicate copy goes.
- `ParseLingoModel_str` and `ParseLingoModel_txt` stay for one release, **Deprecated**, keeping their old meaning: the model, or `nil` and a warning.
- `GenerateLPFile` writes through `internal/utils.WriteFileAtomically`, the temporary-file-and-rename helper `ToCSV`, `ToJSON` and `ToExcel` use, which `parquet-write-options` moved out of the core package, rather than keeping a second copy.
- Removing the deprecated names is recorded as an `AGENTS.md` follow-up.

Structured LP modelling (building constraints from values rather than strings) is a feature, not part of this fix.

## Capabilities

### New Capabilities

- `lpgen-model-files`: how `lpgen` saves a model to a file and reads a LINGO model, and what a failure looks like.

### Modified Capabilities

None.

## Impact

- `lpgen/lpgen.go`, `lpgen/lingo.go` and their tests. Depends on `parquet-write-options` for `internal/utils/atomic_file.go`.
- `Docs/lpgen.md`, `Docs/tutorials/capacity-planning-with-lp-and-lpgen.md`, both changelogs, `AGENTS.md` (removal follow-up), `api-review.md` (LP-3), `delivery-status.md`.
