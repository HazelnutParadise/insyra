# Proposal: engine-dsl-register-command

## Why

After `engine-dsl-manager`, a program running the command language through `engine/dsl` needed nothing under `cli/`, except to add a command of its own. The only way was `cli/commands.Register`, which no document mentioned, and `cli/commands` also builds the Cobra shell, so importing it pulled Cobra into the program. Idensyra, the one embedder found, already reaches into `cli/commands` for `ExecContext` and `Dispatch`. The owner chose on 2026-10-06 to give `engine/dsl` the registration API.

## What Changes

- `engine/dsl` gains `Register`, `CommandHandler`, `ExecContext`, `CommandFlag`, `ArgLimit`, `MaxArgs`, `FormArgs`, `FormArgsAt` and `OpenArgs`, aliases and wrappers of the registry the command language reads, so a program names every type it needs in `engine/dsl`.
- `Register` checks the name before it wraps the handler and keeps its own copy, so a refused duplicate leaves the registered command and the caller's handler untouched. It also refuses a handler without `Run`, and a negative argument limit or form position.
- `ArgLimit` checks only the argument count. A command must check its own required arguments; the custom-command example in `Docs/cli-dsl.md` does.
- `Docs/cli-dsl.md` gets a "Register a command" section with an example.

Left on 0.4:

- The same nine names in `cli/commands` are not marked `Deprecated:` on this line, and `cli/root.go` and `cli/repl` still name `ExecContext` through `cli/commands`. 0.4 does both under the one-name rule (#211).

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `dsl-layering` (created by `engine-dsl-manager`): a program registers its own command through `engine/dsl`.

## Impact

- Code: new `engine/dsl/commands.go`; `cli/commands/registry.go` (name check before wrapping, handler copy); `cli/commands/arg_limit.go` (`validate`).
- Tests: `engine/dsl/commands_test.go`, `cli/commands/registry_test.go`.
- Docs: `Docs/cli-dsl.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`.
