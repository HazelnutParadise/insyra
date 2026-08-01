# Change: Grid search — pick the best estimator by cross-validation, fairly

## Why
The pieces exist — `CrossValidate` scores one estimator, `Better` ranks two results — but a caller comparing configurations still writes the loop themselves, and the loop has two traps the pieces do not guard on their own. Folds are drawn per `CrossValidate` call, so two candidates evaluated with unseeded sampling are scored on *different* folds and their means are not comparable. And the winner has to be refitted on the full training data afterwards, which is easy to forget and quietly evaluates a model fitted on k−1 folds' worth of data.

scikit-learn's `GridSearchCV` takes an estimator plus a parameter grid and expands the candidates itself, by reflecting over constructor parameters. That machinery is the same one `clone()` needed and is the protocol's recorded departure: configuration lives in closures here, so the grid arrives already expanded — a list of named estimators — and what remains is exactly the part worth centralising: identical folds for every candidate, direction-aware ranking, and the refit.

## What Changes
- Add `GridSearch`, taking the feature table, target, named candidate estimators, fold count and a metric, returning every candidate's cross-validation result, the winner, and the winner refitted on the full data
- Guarantee every candidate is scored on identical folds: an explicit seed is honoured, and when none is given one is drawn once, applied to all candidates, and reported on the result so the run can be reproduced
- Rank by the metric's declared direction; ties keep the earliest candidate; a metric with no direction is refused up front
- Require every candidate to be named and the names unique, because a result nobody can attribute to a configuration is unusable
- Fail fast, naming the candidate, when any candidate's fit fails

## Impact
- Affected specs: `ml-model-selection`
- Affected code: new `ml/grid_search.go`, `Docs/ml.md`, `skills/insyra/`, changelogs
- Additive; nothing existing changes behaviour
