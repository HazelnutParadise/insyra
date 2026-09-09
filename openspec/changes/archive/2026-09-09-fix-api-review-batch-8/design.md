# Design: fix-api-review-batch-8

## Context

`internal/ccl` evaluates a compiled AST once per row. Two node shapes carry binary operators: `cclBinaryOpNode` for a single operator and `cclFoldChainNode` for a run of two or more at the same precedence (added when the parser was flattened). Both currently evaluate every operand before calling `applyOperator`, and a comment in the fold path says so deliberately, to stay identical to the binary path. Making one short-circuit without the other would reintroduce exactly the divergence that comment guards against, so both change together and the existing equivalence corpus keeps proving they agree.

`AND`, `OR`, `IF` and the sequence and aggregate functions are special-cased in `evaluateWithCallDepth` before the generic "evaluate every argument, then call" path. The registered `stdlib.go` implementations of `AND`/`OR` are therefore unreachable, which is why their argument checks never ran. `CASE` is not special-cased, which is why it does not short-circuit.

## Decisions

### Boolean coercion stays; the documentation was wrong

`Docs/CCL.md` says `1 && 0` and `'yes' && true` are errors. They are not: `toBool` accepts numbers and the words true/yes/1/false/no/0/empty. Two ways to close the gap — make the evaluator strict, or write down what it does.

Strict would be worse. `IF`, `AND()`, `OR()` and `CASE` all read their condition through the same `toBool`, so a strict `&&` would mean `IF(1, …)` works and `1 && true` does not. And a 0/1 indicator column is one of the most ordinary things in a data table; erroring on `A && B` there buys nothing. So the coercion is kept and the docs gain the table that was missing. This is the only item in this batch where the fix is a documentation change, and it is deliberate.

### Comparison: numbers first, then text, then an error

The order matters, because CCL is a spreadsheet language where `"45.5" > 40` is documented to be true. So: if both operands convert to numbers, compare numerically (unchanged — `'10' > '9'` stays true). Otherwise, if both are strings, compare with Go's string ordering. Otherwise it is a comparison between things that have no common order, and that is an error — which is what the docs already promised for `"hello" > 5`. `nil` keeps its documented exemption and stays false, because `nil` means "no value" rather than "a value of another type".

### A range is not a value

`ColumnRange` and `RowRange` are how the evaluator passes `A:B` and `1:5` to the operators that consume them — aggregate arguments and row access. They are not results. Rather than exporting them so callers can type-assert (which would make an internal representation part of the API), an expression that ends with one is rejected at the top level, in `ccl.go` where the value is about to become a cell, and in `Evaluate` so `engine/ccl` callers get the same answer. The message names the operator, because "A:B is not a value on its own" is more useful than a type name.

The same rule covers `@` in a sequence function: `LAG(@, 1)` currently flattens the whole table and shifts it, giving every cell the same nonsense slice. A sequence function operates on one column, so `@` is refused with that reason.

### Whole numbers, checked where they are converted

Three places turn a float into an index: row access (`evaluateRowAccess`), range bounds (`evaluateRange`), and `scalarInt` in the sequence library. `scalarInt` already refuses NaN, ±Inf and anything outside int32 — batch 4 added that — but still truncates `2.9` to `2`. The other two do a bare `int(f)`, which on arm64 turns NaN into 0 and on amd64 into `MinInt64`: the same expression gives different answers on different machines. One shared helper, `wholeIndex`, does the check in all three, so a future fourth caller cannot miss it.

### Precedence

`&` moves to its own level between comparison and additive, matching Excel. Everything below it shifts up by one; the relative order of every other operator is unchanged, so only expressions that mix `&` with `+`/`-` behave differently. Those expressions were either an error (`'a' & 1 + 2`) or were silently doing arithmetic on a concatenated string (`'1' & '2' + 3` → `15`), so nothing that was both working and intentional changes.

`-2^2 = 4` and `2^3^2 = 64` are left alone. Unary minus binding tighter than `^` matches Excel, and the review found no evidence anyone was surprised in practice — the fix is the precedence table in the docs, which did not exist.

## Risks

- **Short-circuiting changes when an error surfaces.** `false && (1/0 > 0)` used to fail and now returns false. Any test pinning the old behaviour is pinning the defect; `TestFoldChain_NoShortCircuit` is one, and is rewritten to prove the two node shapes still agree, which was its actual purpose.
- **Comma enforcement rejects expressions in the wild.** Anything it rejects was a typo that silently changed the result, so the failure is the point, but it will surface on upgrade rather than at write time.
- **String ordering changes filters.** A filter comparing text columns returned nothing before and returns rows now. That is the fix, but a caller who tuned around the empty result will see different output.
