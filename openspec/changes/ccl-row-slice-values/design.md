# Design: ccl-row-slice-values

## Context

Batch 8 rejected `A:B` and `LAG(@, 1)` because both were writing an internal representation into cells. Rejecting them was over-correction: the review found a broken representation and I treated it as a meaningless question. `(A:B).#` and `LAG(@.#, 1)` already produced exactly the values a reader would expect from the shorter forms, so the language could express both meanings — just not with the spellings people reach for first.

## Decisions

### The cell holds a slice, not a joined string

`@` and `(A:B).#` already put `[]any` into a cell. A joined string (`"10, 1"`) would read better in `Show`, but it would lose the types, and `SUM((A:B).#)` — which works today — would stop working. Anyone who wants text writes `A & ', ' & B`. Consistency with the two forms that already exist wins.

### Only LAG and LEAD take a row

A sequence function either moves values around or does arithmetic on them. `LAG` and `LEAD` only move, so a row is a perfectly good element. Everything else — `CUMSUM`, `DIFF`, `PCT_CHANGE`, the `ROLLING_*` family — calls `toFloat64` on each element, and a row will never be a number.

Letting them through would have produced a column of `nil`, which is what a text column already does there. That is a real gap, but it is CCL-28's gap and it is about values whose type is only known at run time. A row is different: it is a *static* property of the expression that it can never be a number, so the evaluator can say so before running. The allowlist lives next to the registrations in `stdlib_sequences.go`, so adding a sequence function means deciding which of the two kinds it is.

### Where the row-dependence rule was narrowed

`IsRowDependent` decides whether an expression is evaluated once and broadcast, or evaluated per row. Making `A:B` mean "the current row's A and B" makes it row-dependent, and the first attempt said so for every `:` node. That was wrong: `:` is two operators sharing a symbol. Between two column references it is a column range; between two numbers it is a row range, and a row range names the same rows whatever row is being evaluated.

The blanket rule made `A.(0:1)` row-dependent, so `SUM(A.(0:1))` counted the same two cells once per row — 30 became 90 on a three-row table. The rule is now keyed on the operands: a `:` whose sides are both column references is row-dependent; everything else falls through to the operand check exactly as before. `TestCCLRangeConsumersUnchanged` pins the three shapes that regression touched.

The aggregate path needed one more change: `evaluateToColumn` used to ask `IsRowDependent` and then expand the range in its row-independent branch. It now expands a column range before asking, so `SUM(A:C)` reads every cell in those columns regardless of what the row-dependence rule says.

## Risks

- **Two readings of `A:B` now depend on position** — inside an aggregate it is every cell in those columns, on its own it is the current row's cells. `@` has had exactly this split since it was added (`SUM(@)` is the whole table, `@` alone is the row), and the docs describe it; the same paragraph now covers ranges.
- **A cell holding a slice renders as `[10 1]`** in `Show` and in `ToCSV`. That is pre-existing for `@` and unchanged, but it is now reachable through a second spelling.
