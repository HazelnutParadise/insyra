## Context

`vgQSimplimax(L, k)` in `stats/internal/fa/GPArotation_vgQ_simplimax.go` sorts the squared loadings, takes the `k`-th smallest as a threshold, and puts every squared loading at or below it into the criterion. Three places choose `k`, all as the number of rows: the `"simplimax"` arm of `obliqueCriterion` in `GPArotation_GPFoblq.go`, which the rotation evaluates at every trial; `criterion.go`, which the parity suite uses to compare solutions; and `FaRotations`, which passes it to the `Simplimax` wrapper, which ignores it. See proposal.md for what GPArotation 2026.8.2 does instead and what that changes.

GPArotation's `vgQ.simplimax(L, k)` builds a logical matrix, sets `Imat[order(L2)[1:k]] <- TRUE`, and returns `Gq = 2 * Imat * L` and `f = sum(L2[Imat])`. R's `order()` is stable, so equal values keep their column-major index order: `order(c(0.3, 0.1, 0.3, 0.1))` is `2 4 1 3`.

## Goals / Non-Goals

**Goals:**
- One criterion, GPArotation's, wherever simplimax is evaluated.
- The same criterion value and gradient as `vgQ.simplimax` to the last bit where the arithmetic allows, including on ties.

**Non-Goals:**
- A caller-facing `k`. GPArotation accepts one, but psych never passes it and `FactorRotationOptions` has no field for it; adding one is an API decision, not part of matching the reference.
- Changing how a multi-start search chooses among starts.

## Decisions

- **Select exactly `k` by a stable sort over column-major positions.** Collect the squared loadings in column-major order with their positions, sort the positions stably by value, and mark the first `k`. This is what `order(L2)[1:k]` does, ties included. A threshold with a tie-breaking count would reach the same set but restates the rule instead of following it.
- **Sum the criterion in column-major order.** `sum(L2[Imat])` visits the selected entries in column-major order. Matching it keeps the criterion bit-identical to R's where the inputs are, which the reference tests compare at 1e-15.
- **A NaN squared loading sorts last,** as `order()` puts `NA` last by default, so a NaN never displaces a real value from the criterion. Go's comparison would otherwise leave the sort order undefined.
- **One helper for the default `k`,** `nrow(L) * (ncol(L) - 1)`, used by `obliqueCriterion`, `criterion.go` and `FaRotations`. Three copies of the rule are how the port came to disagree with itself about what it was ignoring.
- **Remove the wrapper's `k` parameter.** It was accepted and never used, so leaving it would suggest a choice that does not exist.
- **Reference values pinned as literals from GPArotation 2026.8.2,** as in `gparotation_bb_test.go`: `vgQ.simplimax` on the four fixtures at the default `k` and at `k = nrow`, and the simplimax rotation cases regenerated through `GPArotation::simplimax(A, Tmat)`, the call psych makes.
- **Tests first.** The criterion test and the regenerated rotation cases fail on the current criterion before it changes.

## Risks / Trade-offs

- [Simplimax results change for three or more factors, and wherever squared loadings tie] → Marked **BREAKING** in both changelogs. With two factors and no ties the criterion is identical, and the existing two-factor reference cases stay as they are.
- [The existing `TestVgQSimplimax` asserts the old behaviour] → It is rechecked against GPArotation's values. A case it pins that GPArotation computes differently is corrected to GPArotation's value with the reason in the test, not removed.
- [The parity suite judged simplimax solutions with the old criterion] → Its simplimax leaves are measured before and after, and the change is described by what moved.
