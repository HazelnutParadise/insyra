# Proposal: dsl-outside-cli

## Why

`engine/dsl` is how a Go program runs the insyra command language, and it imported `cli/env` and `cli/repl` (#260, EN-2). Measured on 2026-10-03, `go list -deps ./engine/dsl` listed `cli/env`, `cli/style`, `cli/commands`, `cli/repl`, `github.com/spf13/cobra` and `github.com/ergochat/readline` with its internal packages. A library entry point depended on the command-line front end, so a program that only executes DSL lines built the Cobra shell and the REPL's line editor, and anything in the CLI layer could reach into the engine's dependency graph.

The command implementations are not CLI code: the REPL, `.isr` scripts, one-shot commands and `engine/dsl` all dispatch to the same registry. Only `BuildCobraCommands` (the Cobra shell), the REPL loop and its completer belong to the front end.

## What Changes

- Move, with `git mv` so history and pending edits follow the files: `cli/commands` to `internal/dsl/commands`, `cli/env` to `internal/dsl/env`, `cli/style` to `internal/dsl/style`, and the DSL session (`cli/repl/api.go`) to `internal/dsl` as `Session`/`NewSession`, with the tokenizer every mode shares as `dsl.Tokenize`.
- Take Cobra out of the moved registry. `BuildCobraCommands` and the code that registers and forwards declared flags stay in `cli/commands`, built over an injectable dispatcher, so its tests use their own handlers instead of swapping the registry.
- `engine/dsl` imports `internal/dsl`; its exported API is unchanged. `NewSession` still takes `*env.Manager` and `Context` still returns `*commands.ExecContext`, because `cli/env.Manager` and `cli/commands.ExecContext` are aliases of the moved types.
- `cli/commands`, `cli/env` and `cli/style` keep every exported name as an alias or a wrapper. `cli/env` keeps the 26 functions `cli-env-one-name` deprecated. `cli/commands.Registry` holds the registry's map, so assigning another map to it no longer replaces the registry; that is the one behaviour change, recorded as BREAKING.
- `cli/repl.DSLSession` and `NewDSLSession` become Deprecated aliases of `engine/dsl`'s names, under the one-name rule of #211, with an `AGENTS.md` follow-up for their removal.
- `TestEngineDSLDoesNotDependOnTheCLI` runs `go list -deps` on `engine/dsl` and fails on any package under `cli/`, Cobra or readline.
- Two tests that asked Cobra whether a command's flags were known now check the command's declared `Flags`, which is what Cobra is built from.
- The command-authoring guide moves from `cli/AGENTS.md` to `internal/dsl/AGENTS.md`; `ENG.md` records the dependency direction.

## Capabilities

### New Capabilities

- `dsl-layering`: which packages the command language may depend on, and which public names it keeps.

### Modified Capabilities

(none)

## Impact

- Code: every file of `cli/commands`, `cli/env` and `cli/style` moves to `internal/dsl/`; `cli/repl/api.go` becomes `internal/dsl/session.go`; new `internal/dsl/tokenize.go`, `cli/commands/commands.go`, `cli/commands/cobra.go`, `cli/env/env.go`, `cli/style/style.go`, `cli/repl/session.go`; `engine/dsl/dsl.go`, `cli/root.go`, `cli/repl/repl.go`.
- Tests: `engine/dsl/layering_test.go`, `engine/dsl/api_test.go`, `cli/commands/cobra_test.go` (the tests that drive Cobra, moved out of the internal package), `cli/repl/session_test.go`, `internal/dsl/tokenize_test.go`.
- Docs: `Docs/cli-dsl.md`, `engine/README.md`, `skills/use-insyra-cli/SKILL.md` (where a command's source lives), `internal/dsl/AGENTS.md`, `AGENTS.md`, `ENG.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`, `api-review.md`, `delivery-status.md`.
- Coordination: a change on `dev` to `cli/env/state.go` and `cli/env/manager.go` lands on `internal/dsl/env/` at the next `dev` to `0.4` merge. The files moved without other edits beyond import paths and the removed wrappers, so Git's rename detection carries the change across.
