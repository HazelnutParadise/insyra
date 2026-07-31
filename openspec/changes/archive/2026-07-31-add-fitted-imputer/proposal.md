# Change: An imputer that remembers what it was fitted on

## Why
Insyra can fill missing values — `FillWithMean`, `FillWithMedian`, `FillWithMode`, `FillForward`, `FillBackward`, `FillByInterpolation`, on both `DataList` and `DataTable`. Every one of them computes its statistic from whatever data it is called on and mutates that data in place.

That is the right shape for cleaning a table by hand. It is the wrong shape for a model, and the difference is not stylistic.

A model fitted on training data must impute new observations with the **training** statistic. Imputing them with their own mean is leakage: the held-out data has informed the transformation applied to it, so the score comes out better than the model deserves and no error is ever raised. This is the same class of mistake the pipeline exists to prevent for scaling, and scaling is already safe here — `insyra.StandardScaler` fits, remembers, and transforms. Imputation is the one preprocessing operation with no fitted form.

The consequence is concrete: there is no way to put imputation in a pipeline at all. `FillWithMean(cols ...string) *DataTable` returns no error and is a method on the table rather than an operation over one, so it does not satisfy the transformer protocol. Pipelines therefore have no answer for missing data, which real data has.

The shape to follow is already here. `insyra.Scaler` is `Fit` / `Transform` / `FitTransform` / `InverseTransform` / `Params` / `Kind`, and the four scalers implement it. An imputer is the same interface with a simpler parameter.

## What Changes
- Add an imputer that fits a replacement value per column and applies it to any table afterwards
- Support the strategies that already exist as in-place operations: mean, median, mode, and a caller-supplied constant
- Follow the `Scaler` interface shape, so it is a transformer wherever a scaler is one, with no adapter
- Expose the fitted values, so they can be inspected and later serialised
- Leave the existing in-place methods alone — they are correct for what they do, and this adds the form a model needs

## Impact
- Affected specs: `core-preprocessing`
- Affected code: new file in the root package alongside `datatable_scale.go`
- Additive; nothing existing changes behaviour
- Unblocks imputation in `insyra/ml` pipelines, and imputer export in `add-ml-onnx-export`
