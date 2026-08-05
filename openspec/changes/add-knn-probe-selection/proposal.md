# Proposal: add-knn-probe-selection

## Why

`resolveAlgorithm` picks the KNN search algorithm from row and dimension counts alone, and issue #190 measured what that costs: on unstructured data the auto-selected ball tree is 1.84x–3.28x slower than parallel brute force, with the penalty growing with n — while on clustered data the same tree wins by 2x. The same (n, dims) produces opposite outcomes, so no static rule over those inputs can choose correctly; the deciding property is pruning effectiveness, and it can only be observed, not inferred from shape.

## What Changes

- Auto selection (and only auto — explicit `Algorithm` choices keep meaning what they say) gains a construction-time probe: after a tree is built, a small sample of the caller's own test rows is run through it, counting the fraction of training points actually examined. When pruning fails to clear the calibrated cutoff, the tree is discarded and brute force answers the whole call. The probe uses real test rows because every public entry point receives train and test together and builds its searcher per call — no distribution assumption is needed.
- The cutoff, the probe sample size, and their interaction with `LeafSize` are calibrated by measurement before the mechanism is wired, and the measured numbers are recorded. The decision metric is the examined-candidate fraction — a hardware-independent quantity — not a time threshold.
- The kd-tree branch (dims ≤ 8) is measured for the same blindness. The probe extends to it only if measurement shows the same failure mode; otherwise the finding is recorded and the branch left alone.
- The device path is untouched: the probe runs only when no registered device searcher has already answered the batch.
- `Docs/stats.md` documents auto's actual selection behavior, and both changelogs carry the user-visible entry.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `stats-knn`: auto algorithm selection SHALL decide from observed pruning on the caller's own queries rather than from row and dimension counts alone, and SHALL never change results — only which exact search runs.

## Impact

- **Code**: `stats/internal/knn/knn.go` (`newSearcher` seam, examined-candidate counting in the tree searchers); benchmark and calibration tests in `stats/internal/knn/`.
- **API**: none — `Options{Algorithm, LeafSize, Weighting}` unchanged; internal package signatures may change freely.
- **Behavior**: results are bit-identical (all paths are exact); only wall-clock changes. Worst added cost is one probe sample re-answered by brute force after a discard.
- **Dependencies**: none added.
- **Docs**: `Docs/stats.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`.
