# Proposal: cli-exit-ends-script

## Why

`exit` and `quit` are registered commands, but only the REPL acts on them, by matching the line's text before it dispatches it. Everywhere else the command does nothing (#328, api-review CLI-20). Measured on 2026-10-03 with the CLI built from `0.4` at 61e5b0e9:

- `insyra run s.isr` on a script whose second line is `exit` runs every line after it and prints `script complete`.
- `quit` in a script fails with `unknown command: quit`, because `Dispatch` does not look up a command's aliases; only Cobra does, for one-shot use.
- `insyra exit` and `insyra quit` succeed silently with status 0, although there is nothing for them to end.
- The Go `Session.ExecuteFile` runs past an `exit` line the same way `run` does.

## What Changes

- `exit` and `quit` end the script they appear in, at that line: the lines after it do not run. `run` prints `script ended by exit at line N` in place of `script complete` and succeeds. When the script was started by another script's `run` line, every script in that chain stops, and control returns to whoever ran the outermost `run`: the shell for a one-shot command, the prompt for the REPL, the caller for the Go API.
- `Session.ExecuteFile` stops at an `exit` or `quit` line and returns nil.
- In the REPL, `exit` and `quit` go through the command like any other line instead of being matched as text first. What the user sees does not change.
- Run anywhere else (one-shot `insyra exit`, a single `Session.Execute("exit")`), `exit` fails with an error saying it only ends the REPL or a script. A one-shot `insyra exit` therefore exits with status 1.
- `Dispatch` and `LookupCommand` find a command by any of its aliases, so `quit` works in a script and in the Go API as it already did in one-shot use, and `help quit` describes it. `exit` is the only built-in command with an alias. So that one word never names two commands, `Register` refuses a name that is another command's alias and an alias that is already a name or an alias; before, two commands could share an alias and `Dispatch` would pick one in map order.
- `Session.ExecuteFile` counts as one level of running script, so an `exit` in a script the file starts with `run` ends the whole file, as under `insyra run`.
- `ErrExit` is exported from `engine/dsl`, and every error `exit` returns wraps it, so a program embedding the DSL can tell `exit` apart with `errors.Is`.
- `Docs/cli-dsl.md`, both CHANGELOGs, `api-review.md` and `delivery-status.md` are updated. The `cli-command-guide.md` example the finding cites was removed with the skills' reference files, and no current document gives `insyra exit` as an example.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `script-runner`: a script stops at an `exit` or `quit` line.
- `command-registry`: a command answers to its aliases everywhere, and an alias is never shared.

## Impact

- Code: `internal/dsl/commands/exit.go`, `internal/dsl/commands/run.go`, `internal/dsl/commands/registry.go` (`Dispatch`), `internal/dsl/session.go`, `cli/repl/repl.go`, `engine/dsl/commands.go` (`ErrExit`).
- Tests: `internal/dsl/commands/exit_test.go`, `internal/dsl/session_test.go`.
- Docs: `Docs/cli-dsl.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`, `api-review.md`, `delivery-status.md`.
- One-shot `insyra exit` used to exit 0 and now exits 1. No library change, no new dependency.
