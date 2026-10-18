# Proposal: cli-number-variables

## Why

[#370](https://github.com/HazelnutParadise/insyra/issues/370): the CLI cannot carry a number from one command to the next. `mean x` prints a number that no later command can use, and `sample y frac f` fails when `f` is a name.

## What Changes

- Every numeric command argument accepts the name of a number variable (`int`/`float` held in the environment); a token that parses as a number is still a number.
- `sum`, `mean`, `median`, `mode`, `stdev`, `var`, `min`, `max` and `range` accept a trailing `as <var>` that stores the result; without `as` they only print, as before.
- `show <var>` prints a number variable.

## Capabilities

### Modified Capabilities

- `dsl-commands`: number variables.

## Impact

`cli/commands` (`helpers.go` and the commands that read numbers), `stats_dl.go`, `show.go`, a test in `batch4_test.go`, `Docs/cli-dsl.md`, both CHANGELOGs. Free-form value positions and name-or-number selectors are unchanged.
