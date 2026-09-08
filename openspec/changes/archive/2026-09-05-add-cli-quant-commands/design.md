# Design: add-cli-quant-commands

## Context

Multi-form commands (`ttest`, `regression`, `fetch`) register one `CommandHandler` with `Forms` and `Examples` and dispatch on the first argument. `corr` prints `name=value` and stores the scalar under the alias; `regression` stores a result table. `parseAlias`, `getDataListVar`, `getDataTableVar` exist in `helpers.go`. `quant` functions are error-first with per-period conventions that must be passed explicitly.

## Goals / Non-Goals

**Goals:** every `quant` function reachable with the same conventions and no CLI-side defaults the library refuses.
**Non-Goals:** `WalkForward`, `BlockBootstrap`, `PBO`, `DeflatedSharpeRatio` (they take callbacks, matrices, or scalar inputs that need their own command shapes — a later change); option chains in bulk; plotting.

## Decisions

- **One `quant` command, form per function.** Matches `ttest single|two|paired`; `help quant` enumerates them. Form names are short (`maxdd`, `annret`, `ir`, `bs`, `iv`) because they are typed interactively; the Description line names the Go function for each.
- **Required positionals for `periods`, `days`, `confidence`.** The library validates rather than defaults; the CLI must not invent 252. `rf`/`mar`/`q` default to 0 because 0 is the library's own "ignore" value.
- **Scalar → print and store float; struct → print and store one-row `DataTable`.** Tables let `.isr` scripts `save` or `show` them; `quant factor` returns a row per factor so the exposure list is a normal table, with alpha in a sibling `<var>_alpha` table the way `corr` stores `<var>_p`.
- **Errors wrap the library error** so row-numbered refusals reach the user verbatim.

## Risks / Trade-offs

- [Many forms in one command] → `Forms` and `Examples` keep `help quant` navigable; tests cover every form.
- [Users pass prices instead of returns] → the same footgun as the Go API; the `Docs/cli-dsl.md` quickstart shows `pctchange` + `clean nil` before `quant sharpe`.
