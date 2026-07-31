## Context

`PCAResult` already exposes loadings, eigenvalues, and explained variance, but
the fitted centering and scaling values were discarded after decomposition. A
caller could inspect the result but could not project a new table under the
same preprocessing, and the training scores had to be recomputed outside the
fit.

The existing PCA calculation and its R comparison are the source of truth. The
change adds the parameters and scores that the calculation already uses without
changing the decomposition or introducing a second projection algorithm.

## Goals / Non-Goals

**Goals:**

- Return per-column centering values used during fitting.
- Return per-column scaling values when the fit standardizes columns.
- Return the training observations' component scores.
- Preserve the distinction between covariance and correlation fits.
- Verify new-data projection and training scores against R's `prcomp` behavior.

**Non-Goals:**

- Changing the PCA decomposition, component ordering, or numerical method.
- Adding a public PCA prediction method in `stats`.
- Implementing GPU execution or serialization in this change.
- Retaining hidden mutable references to the input table.

## Decisions

### Store the exact fit parameters on `PCAResult`

`Center` and `Scale` are slices aligned with the input columns. `Scale` records
the per-column sample standard deviation when standardization was requested;
the result distinguishes the unstandardized case rather than manufacturing a
unit scale. `Scores` is a table containing the training rows projected through
the same center, scale, and component loadings.

These fields are additive and make the fitted result sufficient for a later
transformer. Recomputing means or scales from new data was rejected because it
would change the coordinate system and leak information from the data being
projected.

### Populate fields from the existing decomposition path

The PCA implementation keeps the current preprocessing and eigen decomposition
unchanged. Once the fit parameters and loadings are available, it materializes
the training scores through the same projection formula and stores them on the
result. No separate implementation is allowed to silently choose a different
normalization.

### Keep column alignment explicit

The fields are ordered by the fitted input columns. The `ml` PCA transformer
records those names and resolves them before applying center, scale, and
loadings to new data. A missing or reordered input is corrected by name at the
wrapper boundary rather than by recomputing PCA.

## Risks / Trade-offs

- **[Risk] A zero-variance column cannot be standardized]** → Preserve the
  existing PCA error behavior and do not emit a zero scale that would create
  infinities.
- **[Risk] Scores drift from the fit after a future normalization change]** →
  Test scores by projecting the training data through the returned fields and
  compare both to the fit's stored values.
- **[Risk] Center and scale slices are mutated by callers]** → Return copies at
  public boundaries where the surrounding result API permits, and document
  their fitted-column order.
- **[Risk] Sign indeterminacy makes component comparisons look different]** →
  Compare against the established R fixtures with the existing sign alignment
  convention rather than asserting raw signs without normalization.

## Migration Plan

This is additive. Existing code reading the original PCA fields continues to
work. New callers can use `Center`, `Scale`, and `Scores` for projection or
pipeline integration. No stored data migration is needed; rollback removes
the fields and related tests without changing the decomposition.

## Open Questions

The public `stats` API still does not expose a general `Transform` method;
pipeline integration owns that wrapper. ONNX serialization of PCA remains a
separate change because it needs an explicit graph mapping and runtime test.
