## Why

`VALUE('10%')` and `VALUE('1,234')` give an empty result, and a sheet's `50%` and `1,234` stay text when they are read, so the function meant to turn them into numbers silently cannot (#414).

## What Changes

- `VALUE` and `TONUM` read a percent sign and thousands commas, with or without decimals.
- Commas must group digits in threes. Currency signs stay unread.

## Impact

- `internal/ccl/stdlib_typeconv.go`, `Docs/CCL.md`, both changelogs. A formula that got an empty result from such text now gets a number.
