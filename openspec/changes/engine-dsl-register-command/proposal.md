# Proposal: engine-dsl-register-command

## Why

After `engine-dsl-manager`, a program running the command language through `engine/dsl` needed nothing under `cli/`, except to add a command of its own. The only way was `cli/commands.Register`, which no document mentioned, and `cli/commands` also builds the Cobra shell, so importing it pulled Cobra into the program. Idensyra, the one embedder found, already reaches into `cli/commands` for `ExecContext` and `Dispatch`. The owner chose on 2026-10-06 to give `engine/dsl` the registration API.

## What Changes

- `engine/dsl` gains `Register`, `CommandHandler`, `ExecContext`, `CommandFlag`, `ArgLimit`, `MaxArgs`, `FormArgs`, `FormArgsAt` and `OpenArgs`, aliases and wrappers of the registry the command language reads, so `Session.Context()` also returns a type named in `engine/dsl`.
- Under the one-name rule of #211, the same names in `cli/commands` are Deprecated for one release. `BuildCobraCommands`, `Registry`, `Dispatch`, `LookupCommand`, `SnapshotRegistry`, `DBConn`, `SanitizeHistoryLine`, `CloseAllDBConns` and `SaveEnvState` have no counterpart in `engine/dsl` and stay. `cli/root.go` and `cli/repl` name `ExecContext` through `engine/dsl`.
- `Docs/cli-dsl.md` gets a section with a runnable example and every field of `CommandHandler`. Writing it found that `Aliases` reach only the one-shot shell, because `Dispatch` finds a command by `Name`; the field's row says so. The one built-in alias, `exit`'s `quit`, is handled by the REPL before dispatch, so nothing a user does changes.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `dsl-layering`: a program registers its own command through `engine/dsl`.

## Impact

- Code: new `engine/dsl/commands.go`; `cli/commands/commands.go` (Deprecated notices), `cli/root.go`, `cli/repl/repl.go`, `cli/repl/completer.go`.
- Tests: `engine/dsl/commands_test.go`, `cli/commands/deprecated_names_test.go`; `cli/repl` tests and `engine/dsl/api_test.go` (keeps the deprecated spelling on purpose).
- Docs: `Docs/cli-dsl.md`, `engine/README.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`, `AGENTS.md`, `delivery-status.md`.
