# Tasks

## 1. The protocol
- [x] 1.1 Create the `ml` package with `Transformer` and `Model` as declared in `design.md`, typed so no existing scaler or encoder needs an adapter
- [x] 1.2 Add `InverseTransformer`, `ProbaModel`, `Importances` and `Exporter` as separate interfaces
- [x] 1.3 Add `Step` and `Estimator` carrying a fit function rather than a configured object
- [x] 1.4 Assert in the package that all four scalers and all three encoders satisfy `Transformer`, so a change to either side fails the build rather than the docs

## 2. Wrap what `stats` has
- [x] 2.1 Wrap the seven regression families, recording the fitted column names on each
- [x] 2.2 Wrap KMeans, with `Predict` delegating to `KMeansResult.Assign`
- [x] 2.3 Wrap PCA as a `Transformer`, applying the fitted `Center`, `Scale` and `Components`
- [x] 2.4 Wrap KNN classification and regression, the classifier as a `ProbaModel`
- [x] 2.5 Reimplement no arithmetic — every wrapper calls the `stats` function and returns what it returns
- [x] 2.6 Keep the underlying result reachable on each wrapper

## 3. Column matching
- [x] 3.1 Match `Predict` columns by name against `Features`, not by position
- [x] 3.2 Error on a missing column, naming it
- [x] 3.3 Ignore extra columns rather than refusing them

## 4. Conformance
- [x] 4.1 Add `ml/mltest` with `RunConformance`
- [x] 4.2 Check `Features` is non-empty and free of duplicates
- [x] 4.3 Check reordering the input columns does not change the prediction — the property positional binding silently violates
- [x] 4.4 Check a `ProbaModel`'s probability columns match its classes in order and sum to one per row
- [x] 4.5 Run it against every model this change wraps

## 5. Verify
- [x] 5.1 Assert every wrapped model is bit-identical to calling the `stats` function directly, not merely close
- [x] 5.2 Test that an error from the wrapped function comes back unchanged
- [x] 5.3 Test that a scaler and an encoder work as pipeline steps with no adapter
- [x] 5.4 `go test ./...` passes, `go vet ./...` clean

## 6. Record
- [x] 6.1 Create `Docs/ml.md`, following an existing package page for structure
- [x] 6.2 Add `ml` to the package tables in `README.md` and `README_TW.md`, and to the docs index `Docs/README.md`
- [x] 6.3 Register `ml` in `allpkgs/allpkgs.go`
- [x] 6.4 Add the Go API skill coverage in `skills/insyra/`
- [x] 6.5 Changelog entry in `CHANGELOG.md` and `CHANGELOG_TW.md`

## 7. Review remediation
- [x] 7.1 Reject unnamed or duplicate fitted feature columns instead of creating an unusable name-based model
- [x] 7.2 Make logistic `Predict` return class labels while `PredictProba` returns response probabilities
- [x] 7.3 Reject Poisson and GLM offsets that the current `Model` protocol cannot supply during prediction
