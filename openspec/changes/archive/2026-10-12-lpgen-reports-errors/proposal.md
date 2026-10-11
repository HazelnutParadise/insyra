# Proposal: lpgen-reports-errors

## Why

LP-3 of the API review ([#258](https://github.com/HazelnutParadise/insyra/issues/258)) found that `lpgen` reports none of its failures to the caller:

- `GenerateLPFile(filename)` returns nothing. A path it cannot create and an objective type it does not know are logged as warnings, and the caller carries on as if the file were written. Worse, the file is opened with `os.Create` before the objective type is checked, so saving a model with an unknown objective type over an existing LP file truncates that file and leaves only the two banner lines in it.
- `ParseLingoModel_txt(path)` and `ParseLingoModel_str(text)` return `nil` with no error when the file cannot be opened or read, so the only signal is a `nil` model and a log line. Their names use underscores, which the owner's ruling on #211 and the naming half of #212 retire, and the two carry identical copies of the whole parser.

## What Changes

- New `(*LPModel).WriteLP(w io.Writer) error`, which writes the LP text to any `io.Writer`. `GenerateLPFile` writes the same bytes as before, and `Solve` in the `lp-pure-go-default` change uses `WriteLP` for its model text.
- `GenerateLPFile(filename string)` keeps its signature and its output. Its failures are still logged as warnings, as before.
- New `ParseLingo(model string) (*LPModel, error)` and `ParseLingoFile(path string) (*LPModel, error)`. They read what the old functions read and return the same model. A file that cannot be opened or read is an error instead of `nil`. Both run one parser over an `io.Reader`, so the duplicate copy goes.
- `ParseLingoModel_str` and `ParseLingoModel_txt` are **Deprecated**, keep their old behaviour, and are removed in 0.4.

Left on 0.4:
- `GenerateLPFile` returning an error, and writing through a temporary file that is renamed into place. The `WriteFileAtomically` helper is not on this line.
- Removing the deprecated names in 0.4.

`GenerateLPFile` and `WriteLP` check the objective type before writing anything, so an unknown type no longer cuts an existing file down to its banner (the review of this port; 0.4 fixed the same defect).

Structured LP modelling (building constraints from values rather than strings) is a feature, not part of this fix.

## Capabilities

### New Capabilities

- `lpgen-model-files`: how `lpgen` writes a model as LP text and reads a LINGO model, and what a failure looks like.

### Modified Capabilities

None.

## Impact

- `lpgen/write_lp.go`, `lpgen/lingo.go` and their tests.
- `Docs/lpgen.md`, both changelogs.
- Not breaking. `GenerateLPFile`'s signature and output are unchanged, and the LINGO names that return `nil` on failure stay as they are.
