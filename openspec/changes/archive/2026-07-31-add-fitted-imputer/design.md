## Context

The root package currently offers `FillWithMean`, `FillWithMedian`, and
`FillWithMode` as in-place operations. They calculate from the table being
mutated, so they are suitable for one-off cleaning but unsafe as a model
preprocessing step: transforming validation or production data with a newly
calculated statistic leaks information from that data.

The fitted form belongs beside `datatable_scale.go` and must use the existing
`DataTable` column-resolution, actor, cloning, and missing-value conventions.
The `ml.Transformer` protocol already accepts any root transformer with
`Transform(*DataTable) (*DataTable, error)`. The existing `Scaler` interface is
also the established fit/transform/inspect shape that this additive API should
follow.

## Goals / Non-Goals

**Goals:**

- Fit one replacement value per selected column from the fitting table only.
- Support mean, median, mode, and a caller-supplied constant.
- Return a new table from `Transform`, leaving both the fitted state and input
  table unchanged.
- Pass through columns that were not selected at fit time.
- Expose the fitted replacement values and make the type usable as an
  `insyra/ml` pipeline transformer without an adapter.
- Keep the existing in-place fill methods unchanged.

**Non-Goals:**

- Fitting forward-fill, backward-fill, or interpolation state.
- Imputing `DataList` values through a second public API in this change.
- Preserving which cells were missing so an inverse operation can restore them.
- Changing the behavior, warnings, or column-selection rules of existing
  `FillWith*` methods.

## Decisions

1. **Use a fitted `SimpleImputer` with a strategy value.**

   Add an `ImputationStrategy` enum-like string type with `mean`, `median`,
   `mode`, and `constant` values. `NewSimpleImputer` accepts the strategy and
   a constant value only for the constant strategy. This follows the existing
   `NewStandardScaler` naming and makes the strategy visible in `Kind()`.

   The alternative was four unrelated constructors. That would mirror the
   old convenience methods but make it harder to configure a pipeline and
   harder to add a serialisable fitted-state representation.

2. **Reuse the existing `Scaler` surface and parameter record.**

   `SimpleImputer` implements `Fit`, `Transform`, `FitTransform`, `Params`,
   and `Kind` with the same signatures as `Scaler`, so its type assertion into
   a pipeline is compile-time checked. Extend `ScalerParams` with a
   `Replacement any` field and return one record per fitted column. The
   existing scaler records remain unchanged in meaning; the new field is
   populated only for imputers.

   Imputation is lossy, so `InverseTransform` returns a clear unsupported
   error rather than pretending that missing cells can be reconstructed. The
   pipeline only needs `Transform`. This is preferable to silently returning
   an unchanged table, which would make a caller believe inversion succeeded.

3. **Store column references and resolved names separately.**

   Fit resolves each requested name or Excel-style reference using the same
   rules as `Scaler.Fit`, then stores the original reference, resolved output
   name, and replacement. Transform resolves by the stable output name so
   columns can be reordered while preserving the fitted binding. Unfitted
   columns are cloned and passed through.

4. **Derive values from a snapshot while holding the table actor.**

   Fit reads each selected column under `DataTable.AtomicDo`, ignores values
   recognized by the existing `isMissing` helper, and stores only the derived
   replacement. Transform clones the table's columns under its actor and
   replaces missing cells in fitted columns. The fitted replacement is never
   recomputed during Transform.

5. **Match existing strategy semantics exactly.**

   Mean and median require every observed value in the column to be numeric.
   If an observed value is non-numeric, the fitted column is marked as
   pass-through, matching the existing table-level methods instead of
   injecting a numeric value into a mixed column. Mode uses deep equality and
   first occurrence as the tie-breaker. Mean, median, and mode refuse a
   column with no observed values, returning an error that names the column.
   A constant strategy stores the caller's value and may fill a column whose
   observed values are non-numeric.

6. **Keep the core implementation CPU-only.**

   Imputation is a memory-bound preprocessing operation. It does not enter the
   GPGPU strategy or add an accelerator kernel. The fitted state is ordinary
   Go data and remains available to later serialisation work, including ONNX
   export.

## Risks / Trade-offs

- **[Inverse transformation cannot restore missing cells]** → Return an
  explicit error and document that imputation is one-way; pipeline use relies
  only on `Transform`.
- **[Mixed columns are intentionally left unchanged for numeric strategies]**
  → Record a pass-through parameter and test it, matching existing
  `FillWithMean` and `FillWithMedian` behavior rather than silently changing
  data types.
- **[A caller can pass an unsuitable constant]** → Preserve the supplied value
  without coercion, as the existing data model permits heterogeneous cells;
  later model fitting remains responsible for rejecting non-numeric input.
- **[Column references can become invalid between fit and transform]** → Return
  the same named missing-column error used by scalers instead of silently
  applying the replacement to a different column.

## Migration Plan

This is additive. Add the fitted imputer and tests, update the DataTable
documentation and both changelogs, then use it directly as an `ml.Step`:

1. Fit the imputer on the training table.
2. Apply the fitted imputer to training, validation, and production tables.
3. Keep existing `FillWith*` calls for one-off in-place cleaning.

There is no data migration or rollback step. Removing the new type leaves the
existing in-place methods unchanged.

## Open Questions

None. `NewSimpleImputer` stores constructor arguments and `Fit` rejects a
constant supplied to a non-constant strategy, a missing constant for
`ImputeConstant`, or more than one constant value.
