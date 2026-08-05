# Design: add-knn-probe-selection

## Context

`stats/internal/knn/knn.go` builds a fresh searcher on every public call — `Classify`, `Regress`, and `Neighbors` each receive train and test together, call `newSearcher(train, opts)`, then query it once per test row in parallel. `resolveAlgorithm` decides brute/kd/ball from `(n, dims)` alone. Issue #190 measured that rule choosing a ball tree up to 3.28x slower than parallel brute force on unstructured data while the same tree wins 2x on clustered data at identical `(n, dims)` — the deciding property is pruning effectiveness, which `(n, dims)` cannot see.

Two facts shape the design. First, because the searcher is rebuilt per call, the caller's actual test rows are in hand at construction time — a probe can use real queries without any two-phase restructuring of the query loops and without assuming test ≈ train. Second, all three search paths are exact: a wrong selection costs time, never correctness, so the mechanism needs no verification half.

The device path is unaffected: `deviceBatch` answers before any searcher is built, and only under auto.

## Goals / Non-Goals

**Goals:**

- On the shapes measured in #190, auto is never materially slower than the best manual `Algorithm` choice, in both data regimes — asserted by benchmark, not assumed.
- The selection decision is driven by the examined-candidate fraction, a hardware-independent quantity, with every threshold calibrated by recorded measurement.
- Results stay bit-identical to today on every path; the change is confined to `stats/internal/knn` with no public API surface.

**Non-Goals:**

- No persistent index or caching across calls; each call still builds and decides independently.
- No `LeafSize` auto-tuning — only a sensitivity sweep so the cutoff is not calibrated against a mistuned tree.
- No change to the device bridge, its gates, or explicit `Algorithm` choices.
- No kd-tree probe unless this change's own measurement shows the dims ≤ 8 branch failing the same way.

## Decisions

1. **Probe with the caller's own test rows, inside the construction seam.** `newSearcher` (internal signature, free to change) additionally receives a probe sample drawn from `test`. After building a tree under auto, it runs the sample through the tree counting the fraction of training points examined per query; if the mean fraction exceeds the calibrated cutoff, it discards the tree and returns the brute searcher. Alternatives considered and rejected: probing with sampled *train* rows (assumes test ≈ train — the one assumption the per-call construction makes unnecessary); a two-phase escape valve in the query loops (same criterion, but restructures three public paths for no additional information); intrinsic-dimensionality estimation (an unverified proxy whose estimator itself performs neighbor searches); build-time ball-radius statistics (a second unverified proxy replacing the first); retuning the static `(n, dims)` rule (the unstructured penalty rises with n while the clustered win shrinks — one static threshold cannot track two opposing curves).
2. **The metric is examined-candidate fraction, not time.** Tree searchers expose a probe-only counting query (the hot `QueryKNN` path stays untouched); the probe runs single-threaded before parallel dispatch, so the counter needs no synchronization. A fraction transfers across hosts in a way a millisecond threshold cannot, which is what the single-machine measurement base demands.
3. **Deterministic probe sample.** The sample is `min(m, len(test))` rows taken at a fixed stride across the test batch — no RNG, so the same inputs always produce the same selection. Stride rather than prefix, because callers may pass ordered test sets whose first rows are unrepresentative.
4. **Probe answers are discarded, not cached.** The chosen searcher re-answers those rows in the main loop. Worst-case waste is m tree queries plus m brute re-answers — bounded, measured, and small against the 2x–3.28x swing being decided. Caching by slice identity was rejected as complexity without measured payoff.
5. **Cutoff biased toward brute force when uncertain.** The measured asymmetry decides ties: wrongly keeping the tree costs up to 3.28x and grows with n; wrongly discarding it costs at most the measured 2x clustered win. The calibration tasks pick the cutoff from the wall-clock crossover, then round in brute's favor.
6. **Calibration before wiring, numbers recorded.** m, the cutoff, an n-floor below which probing is skipped, and the `LeafSize` interaction are measured on both data regimes across the #190 size ladder before the mechanism lands; the chosen values and their measurement go in the change and the decision delta in `delivery-status.md`.

## Risks / Trade-offs

- [Probe sample unrepresentative of the batch] → fixed-stride sampling across the whole batch; calibration measures the fraction's variance across sample positions to size m.
- [Cutoff calibrated on one machine] → the metric is a fraction, not time; the calibration task also checks the fraction↔crossover relation on a second host when one is available, and the brute-biased rounding bounds the cost of drift.
- [`LeafSize` = 16 was never calibrated; a mistuned tree inflates the fraction] → the sweep task runs before the cutoff is frozen; if 16 is materially wrong, fix the default in the same change rather than calibrating around it.
- [Probe adds overhead where the tree was winning] → on prunable data the probe queries are the fast kind; the n-floor skips probing where totals are too small to matter; both measured.
- [Small test batches] → probe covers `min(m, len(test))` rows; with tiny batches total wall-clock is small either way, and determinism holds.
- [Calibration fails to find a stable cutoff] → fallback is decided in advance: keep the current rule, document the behavior and the manual `Algorithm` escape in `Docs/stats.md`, and record the negative result — the mechanism does not ship on guessed numbers.

## Open Questions

- The concrete values of m, the cutoff, and the n-floor — deliberately open until the calibration tasks produce them.
- Whether the kd-tree branch joins the probe: answered by this change's dims ≤ 8 measurement, not assumed either way.
