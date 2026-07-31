# Tasks

## 1. The protocol
- [ ] 1.1 Create the `ml` package with `Transformer` and `Model` as declared in `design.md`, typed so no existing scaler or encoder needs an adapter
- [ ] 1.2 Add `InverseTransformer`, `ProbaModel`, `Importances` and `Exporter` as separate interfaces
- [ ] 1.3 Add `Step` and `Estimator` carrying a fit function rather than a configured object
- [ ] 1.4 Assert in the package that all four scalers and all three encoders satisfy `Transformer`, so a change to either side fails the build rather than the docs

## 2. Wrap what `stats` has
- [ ] 2.1 Wrap the seven regression families, recording the fitted column names on each
- [ ] 2.2 Wrap KMeans, with `Predict` delegating to `KMeansResult.Assign`
- [ ] 2.3 Wrap PCA as a `Transformer`, applying the fitted `Center`, `Scale` and `Components`
- [ ] 2.4 Wrap KNN classification and regression, the classifier as a `ProbaModel`
- [ ] 2.5 Reimplement no arithmetic — every wrapper calls the `stats` function and returns what it returns
- [ ] 2.6 Keep the underlying result reachable on each wrapper

## 3. Column matching
- [ ] 3.1 Match `Predict` columns by name against `Features`, not by position
- [ ] 3.2 Error on a missing column, naming it
- [ ] 3.3 Ignore extra columns rather than refusing them

## 4. Conformance
- [ ] 4.1 Add `ml/mltest` with `RunConformance`
- [ ] 4.2 Check `Features` is non-empty and free of duplicates
- [ ] 4.3 Check reordering the input columns does not change the prediction — the property positional binding silently violates
- [ ] 4.4 Check a `ProbaModel`'s probability columns match its classes in order and sum to one per row
- [ ] 4.5 Run it against every model this change wraps

## 5. Verify
- [ ] 5.1 Assert every wrapped model is bit-identical to calling the `stats` function directly, not merely close
- [ ] 5.2 Test that an error from the wrapped function comes back unchanged
- [ ] 5.3 Test that a scaler and an encoder work as pipeline steps with no adapter
- [ ] 5.4 `go test ./...` passes, `go vet ./...` clean

## 6. Record
- [ ] 6.1 Create `Docs/ml.md`, following an existing package page for structure
- [ ] 6.2 Add `ml` to the package tables in `README.md` and `README_TW.md`, and to the docs index `Docs/README.md`
- [ ] 6.3 Register `ml` in `allpkgs/allpkgs.go`
- [ ] 6.4 Add the Go API skill coverage in `skills/insyra/`
- [ ] 6.5 Changelog entry in `CHANGELOG.md` and `CHANGELOG_TW.md`
