# Tasks

## 1. Declare
- [x] 1.1 Add the optional clusterer interface beside the package's other optional capabilities, embedding `Model`
- [x] 1.2 Have the fitted KMeans model implement it, reporting the number of clusters it converged on
- [x] 1.3 Document what implementing it asserts about the model's predictions

## 2. Refuse
- [x] 2.1 Refuse a regression metric against a clusterer, alongside the existing classifier arm
- [x] 2.2 Name the mismatch in the error, as the classifier arm does
- [x] 2.3 Leave every model that declares nothing behaving exactly as before

## 3. Verify
- [x] 3.1 Test that cross-validating KMeans with a regression metric is refused, and demonstrate the test fails without the change
- [x] 3.2 Test that a model declaring nothing is still scored as before
- [x] 3.3 Test that the clusterer reports the cluster count the fit converged on
- [x] 3.4 `go build ./...`, `go vet ./...` and `go test ./...` clean

## 4. Record
- [x] 4.1 Document the interface in `Docs/ml.md` beside the other optional capabilities
- [x] 4.2 Changelog entry in `CHANGELOG.md` and `CHANGELOG_TW.md`
