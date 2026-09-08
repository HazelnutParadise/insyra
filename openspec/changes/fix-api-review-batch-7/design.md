# Design: fix-api-review-batch-7

## Context

Every item was reproduced by a failing test before any code changed. The CLI already had the right pattern in places (`groupby` checks `table.Err()`, `rank` rejects a bad direction); this change makes it the rule.

## Decisions

1. **Check the target before calling, then check `Err()` after.** The pre-check produces a message that names what was missing and lists what is available, which a recorded library error cannot. The post-check catches everything else. Shared helpers live in `cli/commands/targets.go`.
2. **The pre-checks read `ColNames()`/`RowNames()`/`NumCols()`/`NumRows()`, never the `Get*` lookups.** On this branch a missed lookup records an error on the table, and a *check* must not leave a mark on the user's data.
3. **`checkTableErr` uses `PopErr`, not `Err`.** `Err()` is sticky, so reading without clearing would make the next command in a REPL session fail for the previous command's reason.
4. **Closed switches for option values, with the documented spellings only.** `parseSortDirection` and `parseEqualVariance` are new; `parseAlternativeHypothesis` changes from returning a default to returning an error. *Alternative*: accept a prefix match — rejected, `d` would be ambiguous between `desc` and nothing, and silent coercion is the bug being fixed.
5. **`config` refuses an unknown key rather than ignoring it.** The `switch` had no `default`, so `config bogus-key 123` reported the whole config back as if it had worked. Valid values for the three enumerated settings live next to the parser so the error message can list them.
6. **`accel --precision` is registered rather than removed.** The flag is parsed and does something; the bug was that Cobra never learned about it, so a one-shot `insyra accel plan --precision float32` was rejected before the command ran.
7. **`plot` takes exactly `[save <file>]`.** Its options-loop scanned for `save` anywhere and ignored everything else; it now reads position 0 and refuses anything that is not `save`.
8. **`sample` checks the count in the CLI**, because the library's own guard returns an empty result rather than an error. `setcolnames` requires exactly one name per column, since fewer silently blanks the rest and more silently adds empty columns (the library behaviour is #227, still open).

## Risks / Trade-offs

- [A script that has been passing with an ignored argument now fails] → deliberate; the changelog says so, and the failure names the argument.
- [`setcolnames` with the wrong count used to "work"] → it produced a table whose later columns had no names, or extra empty columns; both were silent data damage.
