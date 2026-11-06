## Why

A `state.json` whose `lastAccess` was a number, boolean, array or object failed to load, and the command that followed started from no variables and saved over the file, so `"lastAccess": 123` erased every variable. The same happened to any `state.json` that could not be read, because this line never had the stop `dev` added. The custom JSON methods that read the timestamp sat on the public `State`, so a caller's struct embedding `State` lost its own fields when encoded.

## What Changes

- A `lastAccess` that is not readable RFC3339 text, whatever its JSON type, loads as the zero time and the variables load as usual.
- A `state.json` that exists but cannot be read stops the command, `env open`, `env import` into the open environment, the REPL and `NewSession`, with an error naming the file, and leaves the file unchanged. A missing or empty file starts the environment empty. `Manager.OpenVariables` does this, as on `dev`.
- `State` has no JSON methods of its own. One private type describes the stored layout and is used by reading, `SaveVariables`, `Export` and `Import`; `ExportPayload` reads and writes through it.

## Impact

- `internal/dsl/env` (`state.go`, `state_file.go`, `state_open.go`, `manager.go`), `cli/root.go`, `cli/repl/repl.go`, `internal/dsl/session.go`, `internal/dsl/commands/env.go`.
- The `state.json` layout is unchanged and 0.3.6 still reads it.
