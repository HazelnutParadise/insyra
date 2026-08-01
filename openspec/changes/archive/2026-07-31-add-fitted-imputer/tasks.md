# Tasks

## 1. The imputer
- [x] 1.1 Add an imputer following the `Scaler` interface shape — fit, transform, fit-and-transform, inspect its parameters, report its kind
- [x] 1.2 Support mean, median, mode and a caller-supplied constant
- [x] 1.3 Store one replacement per column, derived at fitting time only
- [x] 1.4 Leave a column alone when a numeric strategy meets non-numeric observed values, matching the existing in-place behaviour
- [x] 1.5 Refuse to fit a column with no observed values, naming it
- [x] 1.6 Pass through columns it was not fitted on
- [x] 1.7 Reuse the existing missing-value detection rather than writing a second one

## 2. Verify
- [x] 2.1 Test that transforming a second table uses the first table's values — construct a case where using the second table's own statistic would give a different answer
- [x] 2.2 Test that fit-and-transform equals the existing in-place method on the same table
- [x] 2.3 Test that the fitted values are readable and correct
- [x] 2.4 Test the all-missing column refusal and the non-numeric column pass-through
- [x] 2.5 Assert the imputer satisfies the transformer shape a pipeline needs, and assert it does NOT carry `InverseTransform` — imputation is not reversible, and an always-erroring method would tell a type assertion the capability is present

## 3. Record
- [x] 3.1 Document it in the DataTable docs beside scaling and encoding, stating plainly when to use it rather than the in-place methods
- [x] 3.2 Changelog entry in `CHANGELOG.md` and `CHANGELOG_TW.md`
