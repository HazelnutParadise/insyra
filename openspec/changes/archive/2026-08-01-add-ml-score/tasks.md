# Tasks

## 1. The verb
- [x] 1.1 Add `Score(model, x, y, metric)` returning the metric's result
- [x] 1.2 Route it through the same model/metric compatibility check and the same prediction assembly cross-validation uses, rather than a second copy
- [x] 1.3 Validate the nil and length cases on the same terms as the rest of the package

## 2. Tests
- [x] 2.1 Test that a model scored on the data it was fitted on returns the metric's own value for that data, computed directly
- [x] 2.2 Test that a metric needing probabilities is refused for a model without them, before prediction
- [x] 2.3 Test that a label metric over a probability-reporting model derives labels identically to cross-validation over the same fold

## 3. Documentation
- [x] 3.1 Document `Score` in `Docs/ml.md`, including that the metric is an argument and why
- [x] 3.2 Update `skills/insyra/` so an agent writing Go reaches for `Score` rather than a bare metric function
- [x] 3.3 Add the entry to `CHANGELOG.md` and `CHANGELOG_TW.md`
