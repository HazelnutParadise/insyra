# Proposal: cli-env-typed-state

## Why

A one-shot CLI command restores the environment's variables from `~/.insyra/envs/<name>/state.json` before it runs and saves them after, and that round trip changes the data. Measured on 2026-09-26 and again on 2026-09-27 with a locally built CLI: after `load c.csv as t` for a CSV whose columns are `zeta, alpha, when`, a separate `insyra types t` lists `alpha, when, zeta`, and the `3` that was a `float64` in memory comes back as `int64`. The columns come back alphabetical because a DataTable is written through `ToJSON_String` as one JSON object per row and read back with `ReadJSON`, and every whole float is typed `int64` on the way back because JSON cannot tell `3.0` from `3`. `parsedates` output comes back as strings, so a later `resample` fails. A fitted scaler is skipped on purpose and is gone at the next command (`variable not found: sc`), and an `hclust` tree comes back as a `map[string]interface {}`, so `cutree` refuses it. Anything addressed by column letter then points at a different column than it did one command earlier.

The REPL and `.isr` scripts keep variables in memory, which is why `Docs/cli-dsl.md` and the `use-insyra-cli` skill currently steer multi-step work away from one-shot commands.

## What Changes

- `state.json` stores every variable in a typed layout. A DataTable keeps its columns in order with their names, its row names, its name, and the Go type of every cell. A DataList keeps its name and the Go type of every cell. Top-level values and slices of values keep their Go type. The cell types covered are every built-in integer and float type, `bool`, `string`, `time.Time`, `time.Duration` (what CCL date subtraction produces), `[]byte`, `decimal.Decimal`, `json.Number`, and the nested maps and arrays `ReadJSON` produces. NaN and ±Inf are stored as values, and a string that is not valid UTF-8 is stored byte for byte.
- Fitted scalers (`scale fit`) and `hclust` trees persist and work in the next command.
- A variable the environment cannot store, such as a regression result, is left out of the file and reported at save time, once per variable in a REPL, script or DSL session. The command that created it still succeeds, and every other variable is saved. A stored variable this build cannot decode, such as one a newer release wrote, is kept as it is and written back unchanged.
- `state.json` files written by earlier releases are still read, through the existing reader. The next save rewrites them in the typed layout. Column order and types an earlier release already lost cannot be recovered.
- `env import` reads numbers exactly, so an exported integer above 2^53 imports unchanged.
- `env.Manager` gains `SaveVariables`, which saves like `SaveState` and also returns the variables it left out. `SaveState` keeps its contract: it returns an error only when the file was not written. `LoadState` keeps returning top-level scalars as typed values, and `RestoreVariables` returns every variable at its Go type. `dev` is the 0.3.x line, which takes no breaking changes, so no existing `cli/env` signature or error contract changes.
- `StandardScaler`, `MinMaxScaler`, `RobustScaler` and `MaxAbsScaler` implement `json.Marshaler` and `json.Unmarshaler`, so a fitted scaler can be saved and read back in Go too.
- `Docs/cli-dsl.md`, `Docs/DataTable.md`, the `use-insyra-cli` and `insyra` skills, and both CHANGELOGs describe the new behaviour.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `env-management`: the state serialization requirement describes the typed layout, and a new requirement covers variables the environment cannot store.
- `cli-session-robustness`: NaN persistence no longer requires the old writer layout, only that old files stay readable.
- `cli-env-scalar-roundtrip`: scalars keep their exact Go type through `RestoreVariables`, and `LoadState` types a scalar written in the new layout by the type it was saved with.
- `core-preprocessing`: a fitted scaler can be written to JSON and read back.

## Impact

- `cli/env/state.go` and new codec files in `cli/env`, `cli/env/manager.go` (`Export`, `Import`), `cli/root.go`, `cli/repl/repl.go`, `cli/repl/api.go`, and a new save helper in `cli/commands`.
- `datatable_scale.go` in the root package.
- `cli/env` starts importing `stats` for `HierarchicalResult` and `go-decimal` for decimal cells. Both are already in the module graph and in the CLI binary.
- A binary from before this change reading a new `state.json` sees the new variable kinds as plain maps rather than misreading them as empty tables.
