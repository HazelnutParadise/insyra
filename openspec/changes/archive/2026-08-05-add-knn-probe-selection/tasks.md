# Tasks: add-knn-probe-selection

## 1. Calibration measurements (no wiring until these exist)

- [x] 1.1 Reproduce the #190 matrix through `stats.KNearestNeighbors` on this host — unstructured and clustered regimes across the 5k/20k/50k × 16d/64d ladder, brute vs tree arms — and record the baseline table.
- [x] 1.2 Add a probe-only counting query to the tree searchers (hot `QueryKNN` untouched) and record the examined-candidate fraction for every cell of the matrix, establishing the fraction ↔ wall-clock-crossover relation the cutoff will be read from.
- [x] 1.3 Sweep `LeafSize` (at least 8/16/32/64) across the matrix before freezing any threshold; keep or fix the default 16 based on the numbers, and record the outcome either way.
- [x] 1.4 Measure the fraction's variance across probe sample positions and sizes to choose m, measure probe overhead on both regimes to choose the n-floor, and set the cutoff from the crossover rounded toward brute force. Record all three values with their measurements.
- [x] 1.5 Run the same two regimes at dims ≤ 8 where the rule proposes a kd-tree; record whether the same blindness appears and decide whether the probe extends to that branch.

## 2. Mechanism

- [x] 2.1 Thread a deterministic fixed-stride sample of the caller's test rows into `newSearcher` (internal signature change at all three call sites), with `min(m, len(test))` handling for small batches.
- [x] 2.2 Implement the probe-and-discard decision under auto only — build the proposed tree, run the sample through the counting query, return the brute searcher when the mean fraction exceeds the cutoff — with the measured m/cutoff/n-floor constants documented at their definitions.
- [x] 2.3 Extend the probe to the kd-tree branch if 1.5 showed the same failure mode; otherwise record the negative result in the change and leave the branch alone.

## 3. Verification

- [x] 3.1 Unit tests: an explicitly named algorithm is never probed or substituted; the same inputs select the same algorithm twice; small test batches complete correctly; auto's results equal brute's and the tree's on identical inputs regardless of which side of the cutoff the data falls.
- [x] 3.2 Benchmark check on the #190 ladder asserting auto lands within the recorded tolerance of the best manual `Algorithm` arm in every cell of both regimes — this is the change's verifiable output.
- [x] 3.3 Run the full `go test ./stats/...` suite and confirm the device-path tests are untouched by the seam change.

## 4. Docs, changelog, skills

- [x] 4.1 Update `Docs/stats.md`: what auto actually does now, the manual `Algorithm` escape hatch, and the `LeafSize` outcome from 1.3.
- [x] 4.2 Add the user-visible entry under `## Unreleased` in both `CHANGELOG.md` and `CHANGELOG_TW.md` under the `` ### `stats` `` heading.
- [x] 4.3 Check `skills/insyra/` references for KNN auto-selection claims and sync them if they describe the old rule.

## 5. Bookkeeping

- [x] 5.1 Record the decision delta and the measured thresholds in `delivery-status.md`; note the kd-tree outcome from 1.5.
- [x] 5.2 `openspec validate add-knn-probe-selection --strict` passes before handoff.
