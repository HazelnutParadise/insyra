# Change: KNN reaches the device — the first wiring the measurement supports

## Why
The dormancy decision demanded a measurement before any operation was wired to the accelerator, and KNN now has one: every KNN-shaped workload benchmarked on the M3 wins on-device against the all-core CPU path, up to 4.1x, through the exact-nearest operation whose answers are recomputed in float64 on the host. The operation already returns exactly what KNN needs — each row's k nearest counterparts, sorted, exact — so the wiring is roles, not kernels: the test set is the dataset, the training set is the queries.

The wiring must not cost anyone who does not use it. `stats` cannot import `accel` — the device stack is 41 packages and 1.9s of cold build that `stats` users did not ask for — so the path is a registration seam: `stats` exposes a socket, a new leaf package `accel/knnbridge` plugs the accelerator into it, and a caller opts in with one blank import. No import, no dependency, no behaviour change.

## What Changes
- `stats/internal/knn` gains a batch device-searcher socket, consulted only when the algorithm is `auto`; explicit `brute`, `kd_tree` and `ball_tree` never touch it
- The socket's answer is validated defensively — shape, index range, k — and anything malformed falls back to the CPU path rather than panicking `stats`
- `stats.RegisterKNNDeviceSearcher` forwards the registration through the internal boundary
- New `accel/knnbridge`: blank-import to activate; pre-gates on device presence, k ≤ 7 (the shortlist cap minus the verification slot), and the measured per-row work floor; discards any answer the runtime did not accelerate
- A true-direction benchmark arm: the earlier measurement ran with the roles transposed, which is the same arithmetic but not necessarily the same device efficiency — the wiring direction is measured before the claim is repeated
- Device parity is asserted exactly: classify, regress and neighbors through the bridge equal the brute-force CPU results, index for index

## Impact
- Affected specs: `stats-knn` (new)
- Affected code: `stats/internal/knn/`, `stats/knn.go`, new `accel/knnbridge/`, `accel/knnbench_test.go`, docs, changelogs, `skills/insyra/`
- Additive; without the blank import nothing changes, with it only `auto` changes and only in speed
