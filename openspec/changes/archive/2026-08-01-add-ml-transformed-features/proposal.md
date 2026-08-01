# Change: A pipeline says which columns its estimator actually saw

## Why
A pipeline reports the columns it was fitted on. Its estimator reports one importance per column it saw. When a step changes the column count — one-hot encoding is the ordinary case — those two lists are different lengths, and the pipeline currently offers no way to line them up.

Measured on a pipeline with one numeric and one categorical column, encoded into three: the fitted pipeline's feature list holds 2 names, and its importances hold 4 numbers. A caller reading them together attributes the first importance to the wrong column and has no signal that anything is wrong. Feature importances exist to be read next to names; unnamed, they are four anonymous numbers.

scikit-learn answers this with `get_feature_names_out()`, which a pipeline forwards through each step. The information is already here and does not need to be requested from anyone: fitting a pipeline runs every transform, so the final table's column names are in hand at the moment fitting ends. They are simply discarded.

## What Changes
- Record the column names the final estimator was fitted on, captured from the fully transformed table during fitting
- Expose them as an optional capability, discovered the way this package's other optional capabilities are
- State that a pipeline's importances are indexed by those names and not by the pipeline's input columns
- Extend the conformance helper to require that a model reporting importances reports as many as it has feature names — the check that would have caught this
- Add a test that a pipeline inside cross-validation fits its preprocessing on training rows only, so the leakage-free property is asserted rather than assumed

## Impact
- Affected specs: `ml-pipeline`
- Affected code: `ml/pipeline.go`, `ml/interfaces.go`, `ml/mltest/`, `Docs/ml.md`, `skills/insyra/`
- Additive; nothing existing changes behaviour
