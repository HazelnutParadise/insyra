# cli

Where things live in the command-line code. The rules for working here are in [AGENTS.md](AGENTS.md).

- `cli/commands/` — every DSL/CLI command (one file per command or per closely-related group)
- `cli/repl/` — interactive prompt, line editing, completion
- `cli/env/` — named environment persistence (`~/.insyra/envs/<name>/`)
- `cli/style/` — terminal styling primitives
