# Proposal: cli-env-one-name

## Why

Two parts of #261 (CL-3) need no owner decision:

- Every one of the 26 package-level functions of `cli/env` (`env.Create`, `env.SaveState`, `env.SetBasePath`, …) only calls the same method on `Default()`. Each operation therefore has two public names, which the owner's ruling on #211 (one name per function) does not allow. Only tests and one line of `accel` call them.
- `BuildCobraCommands` adds the one-shot flags of `env` (`--keep-history`, `--force`) and `accel` (`--mode`) by comparing the command's name, so the builder has to know about individual commands, and a command an embedder registers cannot have a flag of its own.

The other two parts are left: `State.LastAccess string → time.Time` changes a field of `cli/env/state.go`, which a session on `dev` is changing now, and `NewAutoCompleter` returning readline's `AutoCompleter` would ripple through the REPL for little gain.

## What Changes

- The 26 wrappers stay for one release, each with `Deprecated: use Default().<Name> instead.`, keeping their meaning. An `AGENTS.md` follow-up records their removal. Every caller in the repository uses `Default()`.
- `CommandHandler` gains `Flags []CommandFlag`. A `CommandFlag` has a name, the help text, whether it takes a value, and optionally the form (first argument) it belongs to. `BuildCobraCommands` registers each declared flag with Cobra and appends each one that is set to the arguments `Run` receives, in declared order, as it did for `env` and `accel`. `env` and `accel` declare their flags; the builder names no command.
- Moving the callers found one in library code: `accel` read its default mode through the deprecated `LoadGlobalConfig`, that is from `Default()`, even in a session given its own `Manager`, against the rule on `ExecContext.Env` that commands go through the session's manager. It now reads the session's manager and falls back to `Default()` only when the session has none, as `fetch tw` already did.
- The `history` command's exemption from one-shot history recording is also decided by name in the builder. It is not a flag and the finding does not name it; it is left as it is and mentioned on #261.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `env-management`: the package-level functions are deprecated aliases of the `Default()` methods.
- `command-registry`: commands declare their one-shot flags at registration.

## Impact

- Code: `cli/env/manager.go`, `cli/env/config.go`, `cli/env/state.go` (doc comments only), `cli/commands/registry.go`, `cli/commands/env.go`, `cli/commands/accel.go` (its flag, and the manager its mode comes from).
- Tests: `cli/env/deprecated_wrappers_test.go`, `cli/commands/command_flags_test.go`, `TestAccelModeComesFromTheSessionManager` in `cli/commands/accel_test.go`; the callers in `cli/env/*_test.go`, `cli/env_roundtrip_test.go`, `cli/root_flags_test.go`, `cli/repl/api_test.go` and `engine/dsl/dsl_test.go` move to `Default()`.
- Docs: `Docs/cli-dsl.md`, `cli/AGENTS.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`, `api-review.md`, `delivery-status.md`, `AGENTS.md`.
