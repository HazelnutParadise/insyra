# Proposal: fix-clear-defects-ccl-finance

## Why

Three more findings whose fix has one right answer: #365 (CCL-31), #366 (CCL-32) and the first part of #248 (FI-3). A fourth, #368 (CCL-36), turned out to be fixed already.

- **A number literal too large for a float64 became `+Inf`.** `strconv.ParseFloat`'s error was assigned to `_`, so a 400-digit literal compiled and every row carried an infinity nobody asked for.
- **`1e5` did not compile.** The tokenizer read the digits, stopped, and made `e5` an identifier, so the expression failed with `unexpected token`. `VALUE('1e3')` has always worked, and CCL's own string output uses exponent form below 0.000001 — so the notation is part of the language everywhere except where it is written directly.
- **`TOSTR` wrote Go's formatting complaints into the data.** `TOSTR(1.5, '%d')` produced the string `%!d(float64=1.5)`, and `TOSTR(1, '%')` produced `%!(NOVERB)%!(EXTRA float64=1)`. Both landed in a cell with no error.
- **`finance`'s `RoundUnnecessary` mode panicked.** It is documented to "panic with decimal.ErrRoundingNecessary", and it does: `NPV(0.03, [0, 1], Options{Scale: 2, Mode: RoundUnnecessary})` takes the program down. `error-philosophy` says no insyra package panics under the default configuration, and the point of the mode is to *learn* that rounding was needed.
- **#368 is already fixed.** It asked for rune-level positions in the tokenizer's error, which `ccl-error-reporting` delivered on 2026-09-09: `中文 + 1` now reports `at offset 0 (near "中")` rather than a byte from the middle of the character. A regression test is added and the issue closed.

## What Changes

- The parser checks `ParseFloat`'s error and reports a literal that is out of range for a float64.
- The tokenizer extends a number over an exponent suffix when there really is one — an `e` or `E`, an optional sign, and at least one digit. Without a digit the `e` stays an identifier, so a column called `E` or `E1` is untouched.
- `TOSTR`/`TEXT` inspect the formatted result for the markers `fmt` writes when the verb does not fit the value, and return an error naming the format and the type instead of handing the marker back as data.
- `finance` gains `Options.finish`, which applies the output scale and, under `RoundUnnecessary`, rounds half-up and compares: a value that changed is reported as an error. All 47 call sites go through it. The other rounding modes are unchanged.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `ccl-type-semantics`: a literal that cannot be represented, and a format that does not fit, are errors rather than values.
- `error-philosophy`: a rounding mode the caller chose does not panic.

## Impact

- `internal/ccl/ccl_tokenizer.go`, `internal/ccl/ccl_parser.go`, `internal/ccl/stdlib_typeconv.go`, `finance/options.go` and the 12 finance files that round a result, plus tests.
- User-visible, so both changelogs get entries. `1e5` becoming a number is additive. The other three turn a wrong value or a crash into an error, so an expression that used to produce `+Inf` or `%!d(...)` now fails — which is the point.
- `Docs/CCL.md` gains the exponent form in its number syntax and a line on what `TOSTR` does with a format that does not fit.
- The rest of FI-3 — `opts ...Options` as a variadic and `var Zero` being writable — belongs to the options-struct decision (D-8, #213) and the exported-var decision (#211), and is left alone.
