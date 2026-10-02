# Proposal: py-placeholders-one-pass

## Why

`RunCodef`, `RunFilef`, their `Context` forms and `Run` put Go values into Python source by replacing `$v1`, `$v2`, … in the template. Three ways of doing that let an argument's content become code. Measured on 2026-10-02 against `replacePlaceholders` in `py/py.go`:

- Each placeholder was replaced with `strings.ReplaceAll` over the whole text in turn, so a later pass rewrote what an earlier one had inserted. `replacePlaceholders("title = $v1\nlabel = $v2", "$v2", "+__import__('os').system('id')+")` gave `title = ""+__import__('os').system('id')+""`, valid Python that runs a command. A program that passes two pieces of user text as arguments can be made to run code. The adversarial review of `py-typed-run` found this on 2026-09-30 and recorded it as an `AGENTS.md` follow-up; PY-2 in `api-review.md` had called the substitution injection-safe.
- `$v1` also matched the start of `$v10`, so with ten or more arguments `$v10` became the first value followed by `0`.
- A value JSON cannot write fell back to `fmt.Sprintf("%v", v)`, raw text in the script. `[]any{"__import__('os').system('id'),", math.NaN()}` gave `x = [__import__('os').system('id'), NaN]`: valid Python that runs the command before the `NaN` fails.

## What Changes

- Placeholders are found in one pass over the template, each by its whole number (`\$v([1-9][0-9]*)`), and replaced there. A value is never searched for placeholders, `$v10` is the tenth argument, and a placeholder past the last argument, or with a leading zero, is left as written, as before.
- Only the values the template uses are converted, each once.
- A value that cannot be written as a Python literal is an error naming its placeholder, before anything runs, instead of being written with `%v`. Such a value never made a script that worked: `math.NaN()` became `NaN`, a name Python does not know. A `[]float64` holding a NaN or an infinity, which was written as `NaN` or `+Inf` the same way, is refused too.
- Every other conversion is unchanged: strings, bools, `[]int`, `[]float64`, `[]string`, `IDataList`, `IDataTable` and values JSON can write produce the same Python text.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `py-run-api`: how `$v1`, `$v2`, … are replaced.

## Impact

- Code: `py/py.go`, `replacePlaceholders` and the conversion it does, which moves into a function of its own.
- Tests: `py/placeholders_test.go`, with the three measured cases and the output of every kind of value as it is today.
- Docs: `Docs/py.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`, `delivery-status.md`, `api-review.md` (the PY-2 row), and `AGENTS.md`, where the follow-up is deleted.
