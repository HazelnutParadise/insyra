# Proposal: cli-declared-arg-limits

## Why

cli/AGENTS.md says a command must refuse an argument it does not understand, because an ignored one makes a typo look like it worked. `cli-reject-ignored-args` fixed `accel` and the nine DataList statistics by hand and left the rest: an audit on 2026-09-11 found 53 more commands that read the arguments they need and drop the rest, so `iqr x junk` prints an answer and `find t 1 as found` stores nothing.

Fixing them one by one would mean fifty hand-written checks that the next command can forget. The owner chose on 2026-09-26 to declare each command's argument count where it is registered and check it in one place.

## What Changes

- `CommandHandler` gains a required `Args` field. `Register` wraps `Run` so an argument past the declared count is refused before the command runs, with an error that names it and shows the usage.
- Four ways to declare it: `MaxArgs(n)`; `.WithAlias()` for a trailing `as <var>` that is not counted; `FormArgs`/`FormArgsAt` when one argument picks a form with its own count (`ttest single|two|paired`, `clean <var> nan|outliers`); `OpenArgs()` for a command that checks every argument itself (a list of values, or a key/value option loop that already rejects unknown keys).
- All 117 registered commands declare `Args`. A test fails on a command that does not.
- **BREAKING**: more than fifty commands now refuse an argument they used to ignore. `as <var>` is refused by commands that store nothing.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `cli-argument-validation`: "Unknown arguments are refused" covers every command rather than a named list, and a new requirement says every command declares its argument count at registration.

## Impact

- `cli/commands/registry.go`, the new `cli/commands/arg_limit.go`, and every command file's `Register` call; new tests `arg_limit_test.go` and `arg_limit_dispatch_test.go`.
- `cli/AGENTS.md`, `Docs/cli-dsl.md`, `skills/use-insyra-cli/SKILL.md`, both changelogs, `AGENTS.md` (the 53-command follow-up is resolved; what the audit found beyond counting becomes its own follow-up), `delivery-status.md`.
- Not in this change: problems the audit found that a count cannot catch, such as `show` ignoring a range on a scaler variable, or `setrownames` dropping names past the row count. They are recorded in `AGENTS.md`.
