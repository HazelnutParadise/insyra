# Proposal: cli-env-one-name

## Why

Two parts of #261 (CL-3) need no owner decision:

- Every one of the 26 package-level functions of `cli/env` (`env.Create`, `env.SaveState`, `env.SetBasePath`, …) only calls the same method on `Default()`. Each operation therefore has two public names, which the owner's ruling on #211 (one name per function) does not allow. Only tests and one line of `accel` call them.
- `BuildCobraCommands` adds the one-shot flags of `env` (`--keep-history`, `--force`) and `accel` (`--mode`) by comparing the command's name, so the builder has to know about individual commands, and a command an embedder registers cannot have a flag of its own.

The other two parts are left: `State.LastAccess string → time.Time` changes a field of `cli/env/state.go`, which a session on `dev` is changing now, and `NewAutoCompleter` returning readline's `AutoCompleter` would ripple through the REPL for little gain.

## What Changes

- The 26 wrappers stay for one release, each with `Deprecated: use Default().<Name> instead. Removed in 0.4.`, keeping their meaning. Every caller in the repository outside tests uses `Default()`.
- `CommandHandler` gains `Flags []CommandFlag`. A `CommandFlag` has a name, the help text, whether it takes a value, and optionally the form (first argument) it belongs to. `BuildCobraCommands` registers each declared flag with Cobra and appends each one that is set to the arguments `Run` receives, in declared order, as it did for `env` and `accel`. `env` and `accel` declare their flags. `Register` refuses a flag with no name or one declared twice.
- `accel` reads its default mode from the session's manager (`ExecContext.Env`) and falls back to `Default()` only when the session has none, instead of always reading `Default()`. This is the one caller the move found in library code that did not follow the session's manager.

Left on 0.4:

- `State.LastAccess` stays a `string` on this line.
- `NewAutoCompleter` is not changed by this port.
- The `history` command's exemption from one-shot history recording is still decided by name in `cli/commands/registry.go`. It is not a flag, so it is left as it is and mentioned on #261.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `env-management`: the package-level functions are deprecated aliases of the `Default()` methods.
- `command-registry`: commands declare their one-shot flags at registration.

## Impact

- Code: `cli/env/manager.go`, `cli/env/config.go`, `cli/env/state.go` (`Deprecated:` notices); `cli/commands/registry.go` (`CommandFlag`, flag checks in `Register`, flag handling in `BuildCobraCommands`); `cli/commands/env.go`, `cli/commands/accel.go` (flag declarations; accel's manager).
- Tests: `cli/env/deprecated_wrappers_test.go`, `cli/commands/command_flags_test.go`.
- Docs: `CHANGELOG.md`, `CHANGELOG_TW.md`.
