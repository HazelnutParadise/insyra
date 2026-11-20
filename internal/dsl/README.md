# internal/dsl

Where things live in the command-line code. The rules for working here are in [AGENTS.md](AGENTS.md).

- `internal/dsl/commands/` — every DSL/CLI command (one file per command or per closely-related group) and the registry
- `internal/dsl/env/` — named environment persistence (`~/.insyra/envs/<name>/`)
- `internal/dsl/style/` — terminal styling primitives
- `internal/dsl/` — the session `engine/dsl` exposes, and the tokenizer every mode shares
- `cli/` — the Cobra shell (`cli/commands/cobra.go` builds it from the registry), the REPL (`cli/repl/`), and the public names `cli/commands`, `cli/env` and `cli/style` give the internal packages
