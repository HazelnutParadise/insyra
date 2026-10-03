# Proposal: py-dataframe-name-not-a-column

## Why

`insyra.Return` gives a pandas DataFrame's table the DataFrame's `name`, read with `getattr(result, "name", None)`. pandas returns a column as an attribute, so for a DataFrame with a column called `name` the `getattr` finds the column, and its printed form became the table's name. Measured on 2026-10-01 with the pinned environment: `pd.DataFrame({'name': ['x', 'y'], 'v': [1, 2]})` came back as a table named `"0    x\n1    y\nName: name, dtype: str"`. Measured again on 2026-10-03 with pandas 3.0.6: a DataFrame whose column index has `name` as a first-level label returns a sub-DataFrame the same way. A column called `name` is common in ordinary data. The adversarial review of `py-nested-table-results` found it and recorded it as an `AGENTS.md` follow-up.

## What Changes

- The pandas DataFrame branch of `insyra._normalize_result` reads the name only when `name` is not a column label, so such a DataFrame comes back with no table name. A name set with `df.name = "scores"`, which pandas stores as a plain attribute when there is no such column, still becomes the table's name.
- polars is unaffected: measured with polars 1.44.2, a polars DataFrame with a column `name` has no `name` attribute.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `py-run-api`: what a pandas DataFrame's table is named.

## Impact

- Code: `insyra._normalize_result` in `py/builtin.go`.
- Tests: the gated end-to-end test in `py/environment_setup_test.go`, since the change is in the Python the runner generates.
- Docs: `Docs/py.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`, `delivery-status.md`, and `AGENTS.md`, where the follow-up is deleted.
