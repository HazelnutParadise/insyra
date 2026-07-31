## Context

The `ml` package provides the shared model protocols used by fitted predictors:
`Model`, `Classifier`, `ProbaModel`, and `Importances`. Decision trees extend
those protocols with a new classifier and regressor while keeping feature names
and column ordering compatible with `DataTable`.

This change has no existing tree implementation or `float64` legacy to
preserve. It therefore follows the precision contract recorded in
`delivery-plan.md`: feature processing may use single precision, accumulated
quantities must not depend on summation order, and reported leaf values and
importances use double precision. Split finding is selection-shaped and is
eligible for later device acceleration, but this change does not implement a
device kernel or claim that acceleration is profitable.

## Goals / Non-Goals

**Goals:**

- Fit deterministic histogram decision trees for classification and regression.
- Support numeric and explicitly declared categorical features without requiring
  one-hot encoding.
- Learn a branch for missing values and define deterministic behaviour for
  missing values or unseen categories at scoring time.
- Honour maximum depth, maximum leaf count, and minimum leaf-size bounds.
- Expose predictions, class probabilities, leaf values, feature importances,
  and the fitted tree structure through the existing `ml` conventions.

**Non-Goals:**

- Implementing GPU execution, device fallback, or a benchmark that proves split
  finding is faster on a device.
- Exhaustively enumerating every categorical subset, which is exponential in
  the category count.
- Adding pruning, ensembles, feature subsampling, model persistence, or ONNX
  export in this change.
- Changing the existing `DataTable`, `DataList`, or estimator protocols.

## Decisions

### Use one shared fitting pipeline with two target modes

Expose `FitDecisionTreeClassifier` and `FitDecisionTreeRegressor`, backed by a
shared fitting and growth implementation. The public models embed the common
feature metadata and expose the methods required by the existing `ml` model
protocols. Classifiers retain a stable class list and probability vectors;
regressors return the double-precision value stored at the reached leaf.

This keeps classification and regression consistent without duplicating the
feature encoding, split search, traversal, or bound handling. A separate tree
package would duplicate the `ml` protocols and make model-selection integration
less direct.

### Encode features into deterministic bins before growing

Numeric features are converted to `float32`, sorted, and represented by
deterministic quantile edges. A configurable `MaxBins` limit bounds histogram
size, with a default of 32. Code `0` is reserved for missing values, so
missingness is not confused with an observed numeric bin.

Categorical features are selected by name in `CategoricalFeatures`. Each
observed category receives a stable code derived from its type and formatted
value. Codes are assigned in sorted key order, not input-row order. Candidate
categorical splits are deterministic subsets formed from categories ordered by
their target statistics, with code order breaking ties. This gives the tree a
categorical subset split without imposing an arbitrary numeric order or paying
the exponential cost of exhaustive subset enumeration.

Raw continuous thresholds were rejected because they make histogram size and
split search grow with the number of distinct values. One-hot encoding was
rejected because it loses the tree's native categorical split semantics and
requires a separate preprocessing step.

### Select splits with per-node statistics and deterministic tie-breaking

Each node builds statistics for its rows and evaluates numeric bin boundaries
or categorical subsets by impurity reduction. Classification uses class counts
and Gini impurity. Regression uses the mean-squared-error form derived from a
fixed-point target sum and sum of squares.

Regression targets are quantized once per fit into bounded `int64` values. The
scale is reduced when necessary so per-row squares and node totals stay within
the integer range. All node accumulation is then integer addition and
subtraction, making the result independent of input-row permutation. Reported
regression values divide by the scale and return `float64`.

Equal gains are resolved in this order: greater gain, lower feature index,
lower split rank/bin, preferred missing-left direction, then sorted category
code key. The rule is part of the implementation contract, so a candidate is
not selected merely because it was visited first.

For each candidate, missing training values are tried on both branches. The
direction with the better scored split is stored in the node. The same learned
direction is also the default for a missing value or unseen category at scoring
time, keeping traversal deterministic without imputation.

### Enforce growth bounds during recursive construction

The grower materializes a leaf first, then stops when the depth, leaf-count, or
minimum-leaf-size constraint prevents a valid split. A candidate is accepted
only when both resulting children satisfy `MinSamplesLeaf` and the gain is
positive. `MaxDepth` and `MaxLeaves` use zero as unlimited; `MinSamplesLeaf`
defaults to one and `MaxBins` defaults to 32.

Feature importance is the gain contributed by each accepted split, weighted by
the number of samples at that node, then normalized across features. Public
slice-returning methods clone their results so callers cannot mutate fitted
state accidentally.

### Keep scoring on the fitted schema

Scoring resolves columns by the feature names captured during fitting, rather
than by the input table's incidental column order. Numeric values use the same
`float32` conversion and stored edges as fitting. Categorical values use the
stored type-and-value code map. Traversal records the node's missing direction,
unseen-category default, and split membership, then returns the reached leaf.

Class labels are sorted by their stable type-and-value keys. Probability-table
columns use that class order, so `Classes()` and `PredictProba()` cannot drift
apart.

## Risks / Trade-offs

- [Risk] Quantile binning and `float32` conversion can merge nearby numeric
  values and produce different thresholds from an exact continuous tree.
  → [Mitigation] Bound the effect with `MaxBins`, use the same stored schema for
  scoring, and compare accuracy against scikit-learn while recording the
  difference rather than asserting algorithmic equality.
- [Risk] Fixed-point target quantization can lose more precision for unusually
  large or highly precise regression targets.
  → [Mitigation] Choose the largest safe scale for the observed target range,
  reduce it only to avoid `int64` overflow, and convert leaf results back to
  `float64`.
- [Risk] Target-statistic ordering only searches prefix subsets of categorical
  levels, so it may miss the globally optimal subset.
  → [Mitigation] Keep the search deterministic and linear in the number of
  observed category levels; revisit exhaustive or more advanced categorical
  search only if measured workloads justify its cost.
- [Risk] Missing and unseen categories may be routed to a branch that is
  surprising to callers.
  → [Mitigation] Learn and expose the direction on each node, document that the
  unseen-category default follows it, and test both fitting-time missingness
  and scoring-time missingness/unseen values.
- [Risk] Split finding is eligible for GPU execution but its performance is
  currently unmeasured.
  → [Mitigation] Keep the implementation CPU-complete and record GPU
  eligibility in `delivery-plan.md`; add a device path only after a benchmark
  establishes a win and an OpenSpec change defines the execution contract.

## Migration Plan

This is an additive `ml` API. Existing callers require no migration, and the
change adds no dependency or data migration. Consumers can opt into the new
fit functions when they need tree models. Rollback is a normal source revert,
which removes the new models and their documentation without changing existing
model behaviour.

## Open Questions

No question blocks this change. Future work must decide whether measured split
search justifies a device implementation and whether categorical search should
grow beyond deterministic target-statistic prefixes. Those decisions belong to
separate OpenSpec changes.
