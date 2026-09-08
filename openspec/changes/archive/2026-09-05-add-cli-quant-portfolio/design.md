# Design: add-cli-quant-portfolio

## Context

`cli/commands/quant.go` drives dispatch, `help quant` Forms, and usage errors from one `quantForms` table; each form has a `run` function using shared helpers (`parseQuantOptions` for `key value` float options, `quantTable`, `quantFloat`, `quantOneRowTable`, `quantLibraryError`). `quant.OptimizePortfolio` takes a `DataTable` and a `PortfolioConfig`; `EfficientFrontier` returns `[]PortfolioResult`.

## Decisions

- **Two forms in the same table**, so `help quant` and the unknown-form message stay complete without new registration.
- **Objective keyword before options**: `minvar|target <r>|maxsharpe` is positional (the `target` keyword consumes the next token), then the existing `key value` loop handles `rf`; `min`/`max` are parsed as comma lists by a small helper because `parseQuantOptions` only handles floats.
- **Weights as a two-column table** rather than one wide row: an `.isr` script can `show` it, `sort` it, or `save` it; the wide layout is reserved for the frontier where each row is a portfolio.
- **Bounds validated in the CLI** for length and numeric parse (so the error names the column count), everything else left to the library so its messages are the single source of truth.

## Risks / Trade-offs

- [Asset names that collide with the frontier's fixed columns] → an asset literally named `Variance` would clash; the frontier uses the fixed columns first and reports a clear error on a collision rather than renaming silently.
