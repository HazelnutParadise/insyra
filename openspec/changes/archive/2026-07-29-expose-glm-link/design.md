## Context

`GLMResult` already publishes the link function used by a fitted generalized
linear model, while the specialized logistic and Poisson result structs keep
their chosen link private. That asymmetry prevents callers and serializers from
inspecting the response mapping without assuming the default.

The link is already selected and used internally during fitting and prediction.
This change only exposes that existing choice on the two specialized results.

## Goals / Non-Goals

**Goals:**

- Add an exported `Link GLMLink` field to logistic and Poisson results.
- Populate it from the exact link selected during fitting.
- Keep the existing internal fields and prediction behavior unchanged.
- Verify the published link reproduces the prediction transform and reports the
  expected specialized defaults.

**Non-Goals:**

- Changing supported link functions or fitting defaults.
- Adding a new prediction API or serializer.
- Removing the existing unexported implementation fields.
- Exposing internal optimizer state.

## Decisions

### Match `GLMResult.Link` exactly

The new field uses the existing exported `GLMLink` type and the same `Link`
name as `GLMResult`. Consumers can inspect all three result families through
one convention, and future serializers do not need a type-specific accessor.

Adding methods with different names was rejected because it would preserve the
inconsistency and force callers to know which result family they hold.

### Populate from the selected internal link

Fit construction copies the link value already chosen for logistic or Poisson
regression into the exported field. Prediction continues to use the existing
private field, so the public addition cannot alter numerical behavior or allow
callers to mutate a fitted model's active link.

### Verify behavior, not only field presence

Tests apply the published link to the fitted linear predictor and compare it to
`Predict`, then assert logistic's logit and Poisson's log defaults. This catches
the failure mode where a field exists but is left at its zero value or is
populated from a different option than the predictor uses.

## Risks / Trade-offs

- **[Risk] The exported value and private value diverge in a future fit path]**
  → Populate both at the same construction point and keep the behavior test
  next to the result tests.
- **[Risk] Callers assume the link can be changed after fitting]** → Document
  the field as fitted metadata; prediction remains driven by the internal
  state.
- **[Risk] Specialized defaults change later]** → Keep explicit tests for the
  reported defaults and require a separate proposal for changing them.

## Migration Plan

This is additive and source-compatible. Existing callers ignore the new fields;
new callers can inspect them for serialization or diagnostics. No persisted
result migration is needed, and rollback removes only the exported fields and
their tests.

## Open Questions

Whether other fitted result types should expose additional link metadata is a
future serialization/API consistency decision. This change covers only the two
specialized GLM results named in the proposal.
