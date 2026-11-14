# Proposal: py-results-keep-nulls

## Why

`insyra.Return` sends a polars or pandas DataFrame through `to_numpy().tolist()`. NumPy has no missing integer, so both libraries hand it a float column instead. Measured on 2026-11-14 with polars 1.44.2, pandas 3.0.6 and NumPy 2.5.3, the versions 0.4 pins, and with polars 2.0.0 (#415). On this line the whole DataFrame goes through `to_numpy()` at once, which loses the same values:

- A polars integer column with a null left Python as floats: `[2**53 + 1, None]` became `[9007199254740992.0, NaN]`, and the null came back as `NaN`. A polars `UInt64` column holding `2**64 - 1` and a null became `1.8446744073709552e+19`.
- A polars null in any numeric column came back as `NaN`, while the same Series returned on its own, which already uses `to_list()`, comes back as `nil`. `Docs/py.md` says `None` comes back as `nil`.
- A polars list column made `insyra.Return` raise `Object of type ndarray is not JSON serializable`, and a struct column came back as a list of floats (`{"x": 1}` as `[1.0]`, a null struct as `[NaN]`).
- With polars 2.0.0, an `Int128` column, which 2.0 gives for `Int64 + UInt64`, made `to_numpy` panic (`not implemented for dtype Int128`).
- A pandas `Int64` column with `pd.NA` became floats, as above. A pandas `boolean` or `string` column holding `pd.NA`, and an `Int64` Series holding one, made `insyra.Return` raise `Object of type NAType is not JSON serializable`.

## What Changes

- A pandas and a polars DataFrame are sent column by column, a polars column with `to_list()`, the conversion a polars Series already uses. An integer column with nulls stays integers, a null is `None`, a list cell is a list and a struct cell is a dict. A NaN polars holds as a value stays NaN.
- A pandas column or Series of a nullable extension dtype (`Int64`, `boolean`, `string`, and the other extension dtypes) is sent as its own values, with `pd.NA` as `None`. A NumPy-backed column is sent as before, so its missing value is still NaN, as in pandas.
- The rest of a result is unchanged. Numbers still come back as float64 on this line, as before; dates, times and decimals, which no path sends today, are outside this change.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `py-run-api`: how a DataFrame's columns are sent, and what a missing value in one comes back as.

## Impact

- Code: the new `insyra._pandas_rows`, `insyra._pandas_values` and `insyra._polars_rows` in `py/builtin.go`.
- Tests: the new gated test `py/missing_values_e2e_test.go`, run with `INSYRA_PY_E2E=1`, since the change is in the Python the runner generates.
- Docs: `Docs/py.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`.
- Behaviour: a polars null, and a pandas `pd.NA`, come back as `nil` where they came back as `NaN`, so `ClearNils` rather than `ClearNaNs` removes them.
