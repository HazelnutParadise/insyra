# Proposal: cli-reject-ignored-args

## Why

cli/AGENTS.md says a command must reject an argument it does not understand, because an ignored argument makes a typo look like it worked. Two places still ignore them:

- The nine DataList statistics (`sum`, `mean`, `median`, `mode`, `stdev`, `var`, `min`, `max`, `range`) read their first argument and drop the rest. `mean x as m` prints the mean and stores nothing, so the next command that uses `m` fails far from the cause.
- `accel` parses `--precision` into a field nothing reads. It chose the precision of `accel run`, removed in v0.3.1. Batch 7 then registered it with Cobra and wrote it into the Usage, so `help accel` advertises it, while Cobra's handler never passes its value on. `accel devices foo` ignores `foo`.

The owner approved rejecting these on 2026-09-11.

## What Changes

- **BREAKING**: the nine statistics reject any argument after the variable, with an error that names the command and the argument.
- **BREAKING**: `accel` accepts only its action and `--mode`. `--precision` leaves the parser, the Cobra flags and the Usage, and any other argument is an error. In one-shot mode Cobra rejects `--precision` as an unknown flag, as it did in v0.3.2: batch 7's registration never shipped.
- The docs drop `--precision`, and the unreleased batch 7 changelog entry stops describing it.

## Capabilities

### New Capabilities

- `cli-argument-validation`: a command reports an argument it does not use instead of ignoring it.

### Modified Capabilities

(none)

## Impact

- `cli/commands/accel.go`, `registry.go`, `stats_dl.go`; a new test. `batch7_test.go` asserted that `--precision` is advertised and registered; it now asserts the opposite, because nothing reads the flag.
- `Docs/cli-dsl.md` and the three skill references that document `accel`; `CHANGELOG.md`, `CHANGELOG_TW.md`; `AGENTS.md`; `delivery-status.md`.
- Not in this change: an audit on 2026-09-11 found about fifty more commands that accept a trailing argument and ignore it (listed in the `AGENTS.md` follow-up). Fixing them needs a per-command rule for how many arguments each takes, which is a design decision of its own.
