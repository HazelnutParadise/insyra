## Context

The regression fit functions already return coefficients, but linear,
polynomial, exponential, and logarithmic results did not expose a way to score
new observations. The GLM family did have prediction methods, but those paths
had never been compared with an external `predict()` reference. The missing
validation matters because `insyra/ml` wraps these results behind a prediction-
shaped protocol.

The design centralizes predictor-shape validation and coefficient evaluation in
the existing regression helpers. It adds point predictions and cross-language
checks without changing how any regression is fitted.

## Goals / Non-Goals

**Goals:**

- Add `Predict` to the four regression result families that lack it.
- Reuse the existing GLM-style prediction convention and response types.
- Refuse a predictor count mismatch with an actionable error.
- Undo the fitting transform for exponential and logarithmic predictions so
  callers receive the response on the original scale.
- Validate all seven regression prediction paths against R fixtures.
- Document the point-estimate scope and the uncertainty results R also returns.

**Non-Goals:**

- Changing fit coefficients, optimizer behavior, or existing GLM predictions.
- Adding standard errors, confidence intervals, or prediction intervals.
- Supporting formula parsing, named predictors, or automatic column selection.
- Replacing the existing cross-language test harness.

## Decisions

### Share input gathering and shape validation

All regression predictions accept the same variadic predictor-list shape used by
the GLM family. A shared helper gathers each `IDataList`, checks that the
predictor count matches the fit, checks equal row lengths, and builds the
numeric design matrix. This keeps error wording and row handling consistent
across seven result types.

Writing four independent validation loops was rejected because one family
would eventually accept a different shape or silently truncate a column.

### Evaluate each model's fitted form, then restore its response scale

Linear regression evaluates the intercept and coefficients directly.
Polynomial regression expands the same powers used during fitting.
Exponential regression evaluates the transformed linear predictor and applies
the inverse exponential response transform. Logarithmic regression evaluates
the transformed predictor and applies its corresponding inverse response form.
The model-specific method owns only this formula; shared helpers own input
validation and result construction.

Returning predictions in transformed space was rejected because callers expect
the response variable represented by the fitted model, and `ml` wrappers would
otherwise expose a different semantic scale from direct `stats` use.

### Validate existing GLM predictions as well as new methods

The R fixtures call `predict()` for linear, polynomial, exponential,
logarithmic, logistic, Poisson, and GLM cases. The Go tests compare point
predictions using the repository's existing tolerance and cross-language
helpers. This validates the five existing methods rather than treating their
prior existence as evidence.

R's fitted values, standard errors, and intervals remain outside the point
prediction contract and are recorded as a known gap instead of being silently
discarded from the documentation.

## Risks / Trade-offs

- **[Risk] Predictor order is wrong]** → Preserve the existing positional
  convention, reject only count and row-shape violations, and make the order
  explicit in method documentation.
- **[Risk] Inverse transforms overflow or produce invalid values]** → Use the
  same floating-point functions as fitting, retain existing error/NaN
  semantics, and cover representative transformed-scale fixtures.
- **[Risk] A shared helper changes an existing GLM behavior]** → Keep the
  helper's accepted input shape aligned with current GLM tests and compare all
  existing methods against the external fixtures.
- **[Risk] R is unavailable in a developer environment]** → Follow the existing
  cross-language harness behavior and report an unavailable reference runtime
  explicitly rather than treating a skipped comparison as a pass.

## Migration Plan

This is additive. Existing fit calls and the five existing prediction methods
keep their signatures. Callers can invoke `Predict` on the four newly capable
results, and `ml` wrappers can use all seven families. No model file migration
is needed; removing the methods is a normal source revert.

## Open Questions

Prediction uncertainty, intervals, and richer R result objects remain future
work. Named-column binding belongs to the `ml` protocol and should not be
introduced into the positional `stats` methods by this change.
