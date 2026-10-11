# Proposal: lingo-refuses-unread-statements

## Why

`lpgen-reports-errors` gave the LINGO parser an error return, but it still used it only for text it could not read at all. A statement it did not recognise was dropped without a word, so the model it returned was not the model in the file. Measured on 2026-09-30: `ParseLingo("MODEL:\nMIN= X1 + X2\nEND")`, whose only statement lacks its closing `;`, returned an empty model and a nil error, and `@FREE(X);`, `@BND(0, X, 10);` and `@GIN(Y);` were dropped, so a free variable became non-negative once the model was written as LP, a bound disappeared and a general-integer variable became continuous. The owner approved refusing such statements on 2026-10-01, the `AGENTS.md` follow-up recorded on 2026-09-30.

## What Changes

- `ParseLingo` and `ParseLingoFile` refuse a statement they do not recognise, a declaration whose variable they cannot read, a relation or objective that is not a linear expression, and a last statement left without its closing `;`. The error gives the line the statement starts on and the statement itself.
- They split a line into its statements, so a statement after a `!` comment on the same line is read. A comment runs to its `;`.
- They read three LINGO declarations they used to drop: `@GIN(x)` as a general integer (the LP `General` section, like `@INT`), `@FREE(x)` as the bound `x free`, and `@BND(l, x, u)` as the bound `l <= x <= u`.
- `ParseLingoModel_str` and `ParseLingoModel_txt`, Deprecated since `lpgen-reports-errors`, keep their old meaning and still drop what they do not recognise.

Whether LINGO's `Display Model` output uses `@FREE`, `@BND` or `@GIN` was not checked, because no copy of LINGO was available; the declarations are read because they are LINGO syntax a hand-written model uses.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `lpgen-model-files`: adds that the LINGO readers refuse what they do not read, and which declarations they read. Both requirements are added as new requirements, because `lpgen-model-files` is created by `lpgen-reports-errors` on this line.

## Impact

- `lpgen/lingo.go`, a new `lpgen/lingo_strict_test.go`, and an end-to-end test in `lp/` that solves a model with a free variable.
- `Docs/lpgen.md`, both changelogs.
- Not breaking. The change only affects `ParseLingo` and `ParseLingoFile`, which are added by `lpgen-reports-errors`. The deprecated names are unchanged.
