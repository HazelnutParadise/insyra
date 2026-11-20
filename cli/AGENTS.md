# AGENTS.md (cli/)

Rules for working in `cli/`. The top-level [AGENTS.md](/AGENTS.md) covers the wider repo.

## Known pitfalls

- Test a change to how the CLI opens an environment through `NewRootCommand` with `SetArgs` (as `TestRawArgCommandIgnoresABrokenDefaultEnvironment` does) or a built binary with its own `HOME`, not only through the `Manager`: commands with `DisableFlagParsing` (`newdl`, `addcol`, `addrow`, `show`, the hypothesis tests) take `--env` off in `wrapRawArgCommand`, after `PersistentPreRunE` has run.
