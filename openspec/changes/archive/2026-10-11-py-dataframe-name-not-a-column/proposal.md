# Proposal: py-dataframe-name-not-a-column

## Why

`insyra.Return` gives a pandas DataFrame's table the DataFrame's `name`, read with `getattr(result, "name", None)`. pandas returns a column as an attribute, so for a DataFrame with a column called `name` the `getattr` finds the column, and its printed form became the table's name. Measured on 2026-10-01: `pd.DataFrame({'name': ['x', 'y'], 'v': [1, 2]})` came back as a table named `"0    x\n1    y\nName: name, dtype: str"`. Measured again on 2026-10-03 with pandas 3.0.6: a DataFrame whose column index has `name` as a first-level label returns a sub-DataFrame the same way. A column called `name` is common in ordinary data.

## What Changes

- The pandas DataFrame branch of `insyra._normalize_result` reads a column called `name` from the frame's own `__dict__`, not through `getattr`, so a column called `name` is never the table's name. A name set with `df.name = "scores"`, which pandas stores in that `__dict__` when there is no such column, still becomes the table's name.
- polars is unaffected: measured with polars 1.44.2, a polars DataFrame with a column `name` has no `name` attribute.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `py-run-api`: what a pandas DataFrame's table is named.

## Impact

- Code: `insyra._normalize_result` in `py/builtin.go`.
- Tests: `py/dataframe_name_test.go`, with the fake Python standing in for the runner, since the change is in the Python the runner generates.
- Docs: `Docs/py.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`.
