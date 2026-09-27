# Design: cli-env-typed-state

## Context

Every one-shot command runs `RestoreVariables` before it and `SaveState` after it (`cli/root.go`). The REPL saves after every line and on exit, and `DSLSession.Execute` saves after every successful line. All three go through `cli/env/state.go`, which today writes:

- a DataTable as `ToJSON_String(true)`, one JSON object per row, unless it holds NaN or ±Inf, in which case it writes the columns in order with `{"$float": "NaN"}` markers;
- a DataList as a JSON array of its cells;
- everything else as `"Raw"` JSON, except a scaler, which it skips.

`LoadState` decodes with `UseNumber` and then converts top-level scalars in place, so a stored `{"$float": "NaN"}` becomes a `float64` NaN inside the returned `State`. `Export` marshals that `State` again, which is why an environment holding a NaN scalar cannot be exported. `Import` decodes without `UseNumber`, so every number goes through `float64`.

The variables commands store are DataTables, DataLists, `float64`/`int` values, `[]int`/`[]float64`/`[]bool` slices (from `kmeans`, `dbscan`, `pca`), fitted scalers (`scale fit`), `*stats.HierarchicalResult` (`hclust`), and six regression result structs (`regression`). Later commands consume DataTables, DataLists, scalers and hierarchical results; the regression structs are only listed by `vars`.

## Goals / Non-Goals

**Goals:**
- What a command stored is what the next command restores: same Go type, same value, same column order, names and row names.
- A variable that cannot be stored is reported when it is saved, not discovered as missing later.
- Existing `state.json` files keep loading.

**Non-Goals:**
- Recovering order or types an earlier release already lost in its file.
- Persisting database connections, which are process resources.
- Making an older binary understand the new layout.
- Changing what happens when the whole `state.json` cannot be parsed (see Risks).

## Decisions

### 1. Typed JSON, one entry per column

Each variable is stored as `{"type": <kind>, "name": <name>, "data": <payload>}`. The kinds are new names, so they never collide with the old ones and the reader can tell the two layouts apart per variable:

| kind | holds | payload |
| --- | --- | --- |
| `table` | `*insyra.DataTable` | `{"columns": [column...], "rowNames": [...]}`; `name` is the table name |
| `list` | `*insyra.DataList` | one column without its name; `name` is the list name |
| `scalar` | one value of a cell type, or nil | `{"type": <tag>, "value": <value>}` |
| `slice` | a Go slice whose element type is a concrete cell type, such as `[]int` | one column; the element type is its `type` |
| `scaler` | the four fitted scalers | the scaler's own JSON (Decision 4) |
| `hclust` | `*stats.HierarchicalResult` | `encoding/json` of the struct |

A column is `{"name": ..., "type": <tag>, "values": [...]}` when every non-nil cell has the same type, and `{"name": ..., "types": [<tag or "">...], "values": [...]}` otherwise. A nil cell is `null` in either form, and `""` in `types`. A column with no non-nil cell has neither field.

A tag is the Go type name as `%T` prints it: `bool`, `string`, `int` to `int64`, `uint` to `uint64`, `float32`, `float64`, `time.Time`, `time.Duration`, `[]uint8`, `decimal.Decimal`, `json.Number`, `map[string]interface {}` and `[]interface {}`. `time.Duration` is on the list because CCL's date subtraction produces it, so a table with a date-difference column would otherwise be refused whole. Values:

| tag | JSON value |
| --- | --- |
| integers | JSON number, parsed back with the tag's bit size |
| `float32`, `float64` | JSON number (shortest form that parses back to the same bits), or the string `"NaN"`, `"+Inf"`, `"-Inf"` |
| `bool` | JSON bool |
| `string` | JSON string; a string that is not valid UTF-8, which `encoding/json` would rewrite with U+FFFD, is stored as `{"base64": ...}` of its bytes |
| `time.Duration` | JSON number of nanoseconds |
| `time.Time` | RFC 3339 with nanoseconds; years 0 to 9999 only. RFC 3339 writes the UTC offset in whole minutes, so a time whose offset has seconds (local mean time before a zone adopted standard time) is written in UTC, which keeps the instant |
| `[]uint8` | base64 string |
| `decimal.Decimal` | its `String()`, parsed back with `ParseExact`, so the scale survives; a negative scale is refused |
| `json.Number` | its literal text as a string |
| `map[string]interface {}`, `[]interface {}` | the value itself, when every leaf is a string, bool, nil or `json.Number`, which is what `ReadJSON` leaves inside nested JSON, and nesting is at most 64 levels, so a value that contains itself is refused instead of overflowing the stack; decoded with `UseNumber` |

Anything else is not a cell type, and the variable holding it is refused (Decision 3).

Alternatives considered:
- **Keep the rows layout and fix the reader.** A JSON object has no order the reader is obliged to keep, and a JSON number does not carry `int` versus `float64`, so no reader can recover either.
- **A tag on every cell.** Simpler, but a 100,000-row integer column would repeat the tag 100,000 times. Most columns are homogeneous, so the column-level `type` is the common case and per-cell `types` the fallback.
- **`encoding/gob`.** Keeps Go types natively, but needs every cell type registered, cannot be read by a person inspecting the file, and would make `state.json` no longer JSON while the old reader still has to exist next to it.

### 2. Restore returns Go values; LoadState keeps its contract

`dev` is the 0.3.x line and takes no breaking changes, so every existing `cli/env` method keeps its signature and its error contract.

`RestoreVariables` and `Export` read the file through one internal reader that decodes with `UseNumber` and converts nothing, so `Export` writes back what the file holds and a NaN is still the string `"NaN"`. Today `Export` goes through `LoadState`, whose in-place conversion turns a stored NaN into a `float64` NaN that `encoding/json` refuses, which is why an environment holding a NaN scalar cannot be exported.

`LoadState` keeps returning top-level scalars as typed values, as the `cli-env-scalar-roundtrip` spec requires: an old `"Raw"` scalar by the old rule (an integer literal as `int64`, anything else as `float64`), and a `"scalar"` in the new layout at the type it was saved with. Everything else is returned as stored.

`RestoreVariables` decodes each variable: the new kinds through the typed decoder, `DataTable`, `DataList` and `Raw` through the existing legacy code unchanged. `Import` decodes with `UseNumber`, so the numbers it writes back are the text it read.

A variable in the new layout that this build cannot decode, such as one written by a newer release with a cell type this build does not know, or one edited by hand, is restored as an unexported placeholder that holds its stored entry. `SaveState` writes that entry back unchanged, kind and name included, so running an older binary once does not destroy it. Commands see a value of an unknown type and refuse it the way they refuse any other wrong type.

### 3. Refusing a variable is a report, not a failure

The save encodes each variable on its own. A variable that fails is left out and the file is written with the rest. `Manager.SaveVariables` returns the left-out variables, each with its name, Go type and reason, sorted by name, and an error only when the file was not written. `SaveState` is `SaveVariables` without the list, so its contract does not change.

`commands.SaveEnvState(ctx)` is the one place the CLI saves: `cli/root.go` after a one-shot command, the REPL after each line and on exit, and `DSLSession.Execute`. It calls `SaveVariables` and prints one warning per left-out variable to `ctx.Output`, where the CLI's other warnings go, so the command that created the variable still succeeds. `ExecContext` remembers the name and type it has already warned about, so a REPL, script or DSL session warns once per variable, and again only if the name comes to hold a different unsaveable type. The warning says the variable is gone when the process ends or another environment is opened, because `env open` replaces the session's variables with the opened environment's.

Alternatives considered:
- **`SaveState` returns an error listing the left-out variables after writing the file.** A caller that returns on any error would then treat a written state as a failed one. That changes an error contract, which the 0.3.x line does not take.
- **Fail the command.** A regression result the user can see on screen would make the command that printed it exit non-zero, and a script would stop.
- **Persist the regression structs through `encoding/json`.** The logistic and Poisson results carry unexported fitted state and `any` fields, so they would come back different, and nothing reads them after the command that prints them.

### 4. Scalers serialize themselves

The scaler's fitted state is unexported, and a column's `ref` (the name or letter it was fitted by, which `Transform` resolves again) is not in `Params()`. Only the root package can write it, so the scalers implement `json.Marshaler` on the shared `*scaler` and `json.Unmarshaler` on each concrete type, which knows its own kind and refuses JSON of another kind:

```json
{"kind": "minmax", "featureMin": 0, "featureMax": 1, "fitted": true,
 "columns": [{"ref": "A", "name": "zeta", "mean": 0, "std": 0, "min": 1.5, "max": 3,
              "median": 0, "q1": 0, "q3": 0, "iqr": 0, "maxAbs": 0,
              "outputMin": 0, "outputMax": 1}]}
```

Floats are numbers, or `"NaN"`, `"+Inf"`, `"-Inf"`, because a column fitted with no values has NaN parameters. The affine coefficients `Transform` uses are not stored: `computeColumn` and the decoder both derive them from the parameters through one function, so a decoded scaler computes the same bits as the original and a hand-edited file cannot hold coefficients that disagree with its parameters. Decoding into a temporary value first leaves the receiver unchanged on error.

This is a public addition, `json.Marshal(scaler)`, which previously produced `{}`. It is the idiomatic Go way to persist a value, and it gives Go users the same train-now, transform-later workflow across processes that the CLI now has.

### 5. Hierarchical trees through encoding/json

Every field of `stats.HierarchicalResult` is exported and plain, so `encoding/json` round-trips it into the typed struct. If a height were NaN, `json.Marshal` fails and the tree is refused under Decision 3 rather than written wrong.

## Risks / Trade-offs

- [An unreadable `state.json` is overwritten] `cli/root.go` treats any `RestoreVariables` error as an empty environment and the save after the command replaces the file. This change does not make that more likely, since files it writes decode by construction, but it does not fix it either → recorded as a follow-up in `AGENTS.md`.
- [An older binary loses the new kinds' types] It reads a new kind as a plain map and saves it back as `Raw`. The data survives as a map, not as a table → accepted; a downgrade is not a supported path, and the distinct kind names are what prevent the worse outcome of misreading the payload as an empty table.
- [Time zones] The instant is always kept. The zone name is not stored: a time comes back in `Local` when its offset is not zero and matches the local zone at that instant, in UTC when its offset is zero or has seconds, and at a fixed offset otherwise. Go's own `MarshalJSON` and `MarshalBinary` keep no zone name either → accepted.
- [Two commands at once] Two one-shot commands on one environment both write `state.json.tmp` and rename it, so the later rename wins and the other command's changes are lost. This predates the change and is recorded as a follow-up in `AGENTS.md`.
- [File size and speed] Per-cell tags only appear in mixed columns, and values are written without the column name repeated per row, so the file is expected to be no larger than the rows layout. The save/restore time of a 100,000-row table is measured before and after and recorded in the tasks.
- [Warnings on stdout] A one-shot `regression ... as r` prints its warning to stdout, like the CLI's other warnings, so a caller parsing that command's output sees one more line → accepted for consistency; later commands print nothing, because the variable is not in the environment they restore.

## Migration Plan

None needed by users. An existing environment is read with the legacy reader and rewritten in the new layout by the next command that saves it. Rolling back to an older binary reads new-kind variables as maps (see Risks).
