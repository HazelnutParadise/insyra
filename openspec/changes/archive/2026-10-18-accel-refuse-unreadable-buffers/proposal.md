# Proposal: accel-refuse-unreadable-buffers

## Why

[#441](https://github.com/HazelnutParadise/insyra/issues/441): `accel.Buffer.Values` is `any`, and `ExecuteNearestExact` reads only `[]float64` and `[]int64`. A hand-built `Buffer` holding `[]float32` or `[]int` with a numeric `Type` made it return an empty result with a nil error (reason `dtype-not-eligible`), so two rows produced zero indices. That breaks the rule that every operation returns its answer. Measured on `dev` before this change: both `[]float32{1, 2}` and `[]int{1, 2}` gave `err = <nil>`, `Index = []`.

## What Changes

- `ExecuteNearestExact` returns an error naming the column when a buffer is declared numeric (`float64`, `int64`, unknown or empty `Type`) but holds a `Values` it cannot read, when `Type` and `Values` disagree, or when columns differ in length.
- Unchanged: a column whose `Type` is `bool`, `string` or `any` is still a dataset the operation is not eligible for, and still reports `dtype-not-eligible` with a nil error outside strict mode.
- Not done here: typed constructors and an unexported or typed `Values`. Those are the owner's call because they break `Dataset`/`Buffer` as documented user types; they stay on #441 for `0.4`.

## Capabilities

### Modified Capabilities

- `accel-gpu-execution`: a malformed buffer is an error, not an empty answer.

## Impact

`accel/exact.go`, a test in `accel/`, `CHANGELOG.md`, `CHANGELOG_TW.md`.
