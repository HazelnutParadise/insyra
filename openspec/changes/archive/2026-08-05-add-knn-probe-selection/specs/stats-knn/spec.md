# stats-knn Delta: add-knn-probe-selection

## ADDED Requirements

### Requirement: Auto algorithm selection is decided by observed pruning
Under the auto algorithm, when the static rule proposes a tree, the system SHALL probe the built tree with a deterministic fixed-stride sample of the caller's own test rows, measure the fraction of training points examined per probe query, and SHALL answer the whole call with brute force when that fraction exceeds the calibrated cutoff. An explicitly named algorithm SHALL never be probed or substituted, and selection SHALL change only which exact search runs — never the results.

#### Scenario: Unstructured high-dimensional data escapes the tree

- **WHEN** auto proposes a tree and the probe's examined fraction exceeds the cutoff
- **THEN** brute force answers every test row
- **AND** the results are identical to the tree's, because every path is exact

#### Scenario: Clustered data keeps the tree

- **WHEN** auto proposes a tree and the probe's examined fraction clears the cutoff
- **THEN** the tree answers the call, preserving its measured advantage

#### Scenario: An explicit algorithm is honored

- **WHEN** the caller names brute force, kd-tree, or ball tree
- **THEN** no probe runs and the named machine answers

#### Scenario: Selection is deterministic

- **WHEN** the same train, test, k, and options are supplied twice
- **THEN** the same algorithm answers both calls

#### Scenario: Small batches stay correct

- **WHEN** the test batch holds fewer rows than the probe sample size
- **THEN** the probe covers the rows that exist and the call completes with exact results

### Requirement: Probe thresholds are measured, not guessed
The probe sample size, the examined-fraction cutoff, and the n-floor below which probing is skipped SHALL come from recorded measurements spanning both data regimes across the issue #190 size ladder, calibrated jointly with `LeafSize` sensitivity. Ambiguous cutoff placement SHALL round toward brute force, because the measured cost of wrongly keeping a tree (up to 3.28x, growing with n) exceeds the measured cost of wrongly discarding one (at most the 2x clustered win).

#### Scenario: The benchmark holds on the issue's shapes

- **WHEN** the #190 size ladder runs on unstructured and clustered data
- **THEN** auto's wall clock is within the recorded tolerance of the best manual `Algorithm` choice in every cell

#### Scenario: The kd-tree branch is measured before it is touched

- **WHEN** the same regimes run at dims ≤ 8 where the static rule proposes a kd-tree
- **THEN** the outcome is recorded, and the probe extends to that branch only if the same failure mode appears
