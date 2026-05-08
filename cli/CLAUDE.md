# CLAUDE.md (cli/)

Guidance for working in `cli/`. The top-level [CLAUDE.md](../CLAUDE.md) covers the wider repo; this file focuses on **shared helpers and patterns inside `cli/`** so you don't reinvent them.

> Rule of thumb: before writing a parser, lookup, or registration helper, check the lists below. If it's already here, use it. If you're tempted to write a near-duplicate, refactor the existing one instead.

## Where things live

- `cli/commands/` — every DSL/CLI command (one file per command or per closely-related group)
- `cli/repl/` — interactive prompt, line editing, completion
- `cli/env/` — named environment persistence (`~/.insyra/envs/<name>/`)
- `cli/style/` — terminal styling primitives

## Always-reuse helpers (commands/helpers.go)

| Helper | Use for | Don't write your own |
|---|---|---|
| `parseAlias(args) -> (coreArgs, alias)` | extracting a trailing `as <var>`; defaults alias to `$result` | a copy that handles `as` differently — every transforming command must behave the same |
| `parseFlexBool(raw) -> (bool, err)` | any **option-value boolean** (`headers true\|false`, `rownames yes\|no`, `bom 1\|0`, …) | `strconv.ParseBool` (rejects `yes/no/on/off`); ad-hoc `== "true"` chains |
| `parseLiteral(raw) -> any` | converting a **data literal** (SQL `params`, single-cell `set` value): nil/true/false/int/float, else string | a separate type-coercion ladder; do not relax this with `yes/no` — it's about typed data, not flags |
| `getDataTableVar(ctx, name)` / `getDataListVar(ctx, name)` | every variable lookup that requires a specific type — produces the canonical "variable not found" / "is not a DataTable" error | manual `ctx.Vars[name]` + type assertions |
| `detectFileKind(path)` | path → `csv` / `json` / `excel` / `parquet` / `""` | duplicating the extension switch |
| `parseCSVTokens(raw)` | trimmed comma-separated string list | `strings.Split` + per-element trim by hand |
| `parseCSVInts(raw)` | comma-separated non-negative ints (e.g. parquet rowgroups) | a parallel int parser |

Tests:

| Helper | Use for |
|---|---|
| `newTestExecContext(t)` (in `db_test.go`) | every command test that needs an `*ExecContext` with `Vars` and a buffered `Output` |
| `mustConnectSQLite(t, ctx, name, dsn)` | DB-touching tests; auto-cleans up via `t.Cleanup` |

Don't construct an `ExecContext` literal in a new test file — re-export or import `newTestExecContext` instead. If you need a new shared test helper, put it in a `*_test.go` that the rest of the package can import.

## Special-purpose bool parsers — keep separate

Not every "true/false" string is an option flag. These are deliberately not unified with `parseFlexBool`:

- `filter.toBool` ([cli/commands/filter.go](commands/filter.go)) — value-level truthiness for CCL filtering. Accepts `""`, `nil`, `null`, numeric coercion, etc. Different domain (cell value), different rules.
- `rank` direction ([cli/commands/transform.go](commands/transform.go)) — domain enum (`asc`/`desc`/`ascending`/`descending`) with `true`/`false` as legacy aliases. Don't widen this to `yes/on/1` — direction words are first-class, the bool aliases are a courtesy.

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

Every command file has a `func init()` that calls `Register(&CommandHandler{...})` with `Name`, `Usage`, `Description`, `Run`. Don't bypass `Register` and don't mutate the registry from elsewhere.

The `Usage` string is what the user sees in `insyra help <command>`. Keep it accurate and tight; if it gets long, separate the major shapes with `|` (see `load`, `save`).

## Output and errors

- Success messages: `fmt.Fprintf(ctx.Output, ...)`. Don't print to stdout directly — REPL and `run` capture `ctx.Output`.
- Errors: return them, don't print. The dispatcher formats them.
- `_ = Register(...)` and `_, _ = fmt.Fprintf(...)` are intentional — `Register` returns an error only on duplicate names (programmer error, caught at compile-test) and the writer is in-memory.

## When you change a shared helper

Helpers in this list are imported across commands. A signature change to `parseAlias`, `parseFlexBool`, etc. means re-running `go test ./cli/...` and likely touching docs in:

- [Docs/cli-dsl.md](../Docs/cli-dsl.md)
- [skills/use-insyra-cli/SKILL.md](../skills/use-insyra-cli/SKILL.md)
- [skills/use-insyra-cli/references/](../skills/use-insyra-cli/references/)

Keep these in sync — the skill references are the source of truth for AI agents using the CLI.
