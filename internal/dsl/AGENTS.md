# AGENTS.md (internal/dsl/)

Guidance for working on the command language: `internal/dsl/` and the `cli/` packages built on it. The top-level [AGENTS.md](../../AGENTS.md) covers the wider repo; this file focuses on **shared helpers and patterns inside `internal/dsl/commands/`** so you don't reinvent them.

> Rule of thumb: before writing a parser, lookup, or registration helper, check the lists below. If it's already here, use it. If you're tempted to write a near-duplicate, refactor the existing one instead.

## Where things live

- `internal/dsl/commands/` — every DSL/CLI command (one file per command or per closely-related group) and the registry
- `internal/dsl/env/` — named environment persistence (`~/.insyra/envs/<name>/`)
- `internal/dsl/style/` — terminal styling primitives
- `internal/dsl/` — the session `engine/dsl` exposes, and the tokenizer every mode shares
- `cli/` — the Cobra shell (`cli/commands/cobra.go` builds it from the registry), the REPL (`cli/repl/`), and the public names `cli/commands`, `cli/env` and `cli/style` give the internal packages

Nothing under `internal/dsl/` imports `cli/`, Cobra or readline; `TestEngineDSLDoesNotDependOnTheCLI` in `engine/dsl` fails when something does. Shell-only code goes in `cli/`.

## Always-reuse helpers (commands/helpers.go)

| Helper | Use for | Don't write your own |
| --- | --- | --- |
| `parseAlias(args) -> (coreArgs, alias)` | extracting a trailing `as <var>`; defaults alias to `$result` | a copy that handles `as` differently — every transforming command must behave the same |
| `parseFlexBool(raw) -> (bool, err)` | any **option-value boolean** (`headers true\|false`, `rownames yes\|no`, `bom 1\|0`, …) | `strconv.ParseBool` (rejects `yes/no/on/off`); ad-hoc `== "true"` chains |
| `parseLiteral(raw) -> any` | converting a **data literal** (SQL `params`, single-cell `set` value): nil/true/false/int/float, else string | a separate type-coercion ladder; do not relax this with `yes/no` — it's about typed data, not flags |
| `getDataTableVar(ctx, name)` / `getDataListVar(ctx, name)` | every variable lookup that requires a specific type — produces the canonical "variable not found" / "is not a DataTable" error | manual `ctx.Vars[name]` + type assertions |
| `resolveColumnToken(cmd, table, token)` / `resolveRowToken(...)` / `resolveColumnTokens(...)` | every token that picks a column or a row: reads it as a number, as letters (columns) and as a name, refuses when two readings disagree, and honours `number:` / `index:` / `name:` (#315) | `strconv.Atoi` then `GetColByName`, or any other order: each order picks the wrong column on some table |
| `colSelector(cmd, table, token)` / `colSelectors(...)` | the same, returned as a library selector (the column's name when it picks it alone, else its position) to pass to `GroupBy`, `Pivot`, `Resample`, the scalers and encoders | passing the raw token, which the library reads as an Excel index |
| `detectFileKind(path)` | path → `csv` / `json` / `excel` / `parquet` / `""` | duplicating the extension switch |
| `parseCSVTokens(raw)` | trimmed comma-separated string list | `strings.Split` + per-element trim by hand |
| `parseCSVInts(raw)` | comma-separated non-negative ints (e.g. parquet rowgroups) | a parallel int parser |

Tests:

| Helper | Use for |
| --- | --- |
| `newTestExecContext(t)` (in `db_test.go`) | every command test that needs an `*ExecContext` with `Vars` and a buffered `Output` |
| `mustConnectSQLite(t, ctx, name, dsn)` | DB-touching tests; auto-cleans up via `t.Cleanup` |

Don't construct an `ExecContext` literal in a new test file — re-export or import `newTestExecContext` instead. If you need a new shared test helper, put it in a `*_test.go` that the rest of the package can import.

## Special-purpose bool parsers — keep separate

Not every "true/false" string is an option flag. These are deliberately not unified with `parseFlexBool`:

- `filter.toBool` ([internal/dsl/commands/filter.go](commands/filter.go)) — value-level truthiness for CCL filtering. Accepts `""`, `nil`, `null`, numeric coercion, etc. Different domain (cell value), different rules.
- `rank` direction ([internal/dsl/commands/transform.go](commands/transform.go)) — domain enum (`asc`/`desc`/`ascending`/`descending`) with `true`/`false` as legacy aliases. Don't widen this to `yes/on/1` — direction words are first-class, the bool aliases are a courtesy.

If you add a new "looks like a bool" argument, ask: is it a CLI option flag (use `parseFlexBool`), a typed data literal (use `parseLiteral`), or a domain enum (write a small focused switch)?

## Option-parsing pattern (key/value loop)

All multi-option commands use the same shape. Match it.

```go
func parseFooOptions(args []string) (FooOptions, error) {
    var opts FooOptions
    for i := 0; i < len(args); {
        key := strings.ToLower(args[i])
        next := func() (string, error) {
            if i+1 >= len(args) {
                return "", fmt.Errorf("foo: option %q requires a value", args[i])
            }
            return args[i+1], nil
        }
        switch key {
        case "headers":
            v, err := next()
            if err != nil {
                return opts, err
            }
            b, err := parseFlexBool(v)
            if err != nil {
                return opts, fmt.Errorf("foo: invalid value for headers: %w", err)
            }
            opts.Headers = b
            i += 2
        // ... more cases ...
        default:
            return opts, fmt.Errorf("foo: unknown option %q (supported: headers, ...)", args[i])
        }
    }
    return opts, nil
}
```

Conventions:

- **Reject unknown options** with the supported list in the message — silent ignore makes typos invisible.
- **`*Set` flags** on the options struct when format-specific code needs to reject "this option doesn't apply here" (see `fileLoadOptions.HeadersSet` in `load.go`).
- **Bare-flag back-compat**: when adding a value form to an existing bare flag (e.g. `save sql ... rownames` → optionally `rownames true|false`), peek at `args[i+1]`, only consume it if `parseFlexBool` accepts it; otherwise fall back to the bare-flag default. See `db_save.go` for the canonical pattern.

## Command registration

Every command file has a `func init()` that calls `Register(&CommandHandler{...})` with `Name`, `Usage`, `Description`, `Args`, `Run`. Don't bypass `Register` and don't mutate the registry from elsewhere.

### Args: how many arguments the command takes

`Args` is required; `TestEveryCommandDeclaresItsArguments` fails on a command without it. `Register` wraps `Run` so an argument past the declared count is refused before the command runs, with an error naming it and the Usage. Don't write a per-command "too many arguments" check.

- `MaxArgs(n)` — at most `n` arguments. `iqr <var>` is `MaxArgs(1)`; an optional trailing argument counts, so `sort <var> <col> [asc|desc]` is `MaxArgs(3)`.
- `.WithAlias()` — a trailing `as <var>` is allowed on top of the count. Add it only when the command stores its result; a command that stores nothing must refuse `as`.
- `FormArgs(map[string]int{...})` — the first argument picks a form with its own count, which includes the form word: `ttest single <var> <mu>` is 3 and `ttest two <var1> <var2> [equal|unequal]` is 4. `FormArgsAt(i, ...)` when the form word is at position `i` (`clean <var> nan|outliers`). An unknown form is left for the command to report.
- `OpenArgs()` — the command checks every argument itself: a list of values (`newdl`, `dropcol`) or a key/value option loop that already rejects an unknown key. Don't use it to skip counting a fixed-shape command.

The `Usage` string is what the user sees in `insyra help <command>`. Keep it accurate and tight; if it gets long, separate the major shapes with `|` (see `load`, `save`).

### Flags: what the one-shot form takes on top of its arguments

A command whose one-shot form takes a flag (`env import ... --force`, `accel ... --mode gpu`) declares it in `Flags` with a `CommandFlag`. `BuildCobraCommands` in `cli/commands` registers it with Cobra and hands it to `Run` as arguments, the way the REPL and scripts pass it, so `Run` parses it in one place. Set `Form` when the flag belongs to some forms only (`"import|delete"` for several), and `TakesValue` when it takes a value. Don't special-case a command by name in `BuildCobraCommands`.

### Forms and Examples (optional but expected for complex commands)

`CommandHandler` has two optional `[]string` fields rendered by `help <cmd>` under "Forms:" and "Examples:" headers. Use them — don't stuff everything into `Usage` or write your own help-printing path.

- **`Forms`** — one entry per major shape, with a short tail-aligned note. Use a blank-string entry to insert a visual blank line (e.g. before a "DSN forms:" sub-block in `db`).
- **`Examples`** — complete `insyra ...` lines that copy-paste cleanly into a shell.

Add them whenever the command has subcommands, multiple sub-shapes, or non-obvious option semantics. Simple commands (`mean x`, `shape x`) leave both empty and `help` falls back to the two-line format. The current "complex enough to deserve Forms/Examples" set is: `ttest`, `ztest`, `anova`, `ftest`, `chisq`, `regression`, `fetch`, `plot`, `db`, `groupby`, `load`, `save`. Match the existing tone (concise descriptors, lowercase verbs in tail notes) when adding more.

Don't introduce a parallel `LongHelp string` or print extra help from inside `Run` — the structured fields exist so `help` output stays uniform and so we can later generate skill-reference docs from the registry.

## Output and errors

- Success messages: `fmt.Fprintf(ctx.Output, ...)`. Don't print to stdout directly — REPL and `run` capture `ctx.Output`.
- Errors: return them, don't print. The dispatcher formats them.
- `_ = Register(...)` and `_, _ = fmt.Fprintf(...)` are intentional — `Register` returns an error only on duplicate names (programmer error, caught at compile-test) and the writer is in-memory.

## When you change a shared helper

Helpers in this list are imported across commands. A signature change to `parseAlias`, `parseFlexBool`, etc. means re-running `go test ./internal/dsl/... ./cli/...` and likely touching docs in:

- [Docs/cli-dsl.md](../../Docs/cli-dsl.md)
- the command's own `Usage`, `Forms` and `Examples`, which `insyra help <command>` prints

Keep these in sync. `Docs/cli-dsl.md` and `help` are what AI agents read: [skills/use-insyra-cli/SKILL.md](../../skills/use-insyra-cli/SKILL.md) teaches them to look there and lists no commands itself.
