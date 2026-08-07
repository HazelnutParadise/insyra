# Tasks

## 1. The declaration
- [x] 1.1 Add a direction type with three values: larger is better, smaller is better, no direction
- [x] 1.2 Add the declaring method to the metric protocol
- [x] 1.3 Declare it on all seven built-in metrics
- [x] 1.4 Refuse, at the point a metric is used, a scalar-scoring metric that declares no direction

## 2. Comparison
- [x] 2.1 Add a comparison over two cross-validation results that uses the declared direction
- [x] 2.2 Refuse a comparison across different metrics, and a comparison of directionless results
- [x] 2.3 Carry the direction on the cross-validation result

## 3. Tests
- [x] 3.1 Test that a loss metric ranks the smaller mean better and a gain metric the larger — the case that silently picks the wrong model today
- [x] 3.2 Test both refusals
- [x] 3.3 Extend the conformance helper so a third-party metric's direction is checked for consistency with the scores it returns

## 4. Documentation
- [x] 4.1 Document the direction and the comparison in `Docs/ml.md`, with the table of built-in directions
- [x] 4.2 Update `skills/insyra/`
- [x] 4.3 Add the entry to `CHANGELOG.md` and `CHANGELOG_TW.md`
