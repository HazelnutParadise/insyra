# Tasks

## 1. Export
- [x] 1.1 Export the class-label and probability marker interfaces, named the way the package names its other optional capabilities
- [x] 1.2 Export their methods, and update the built-in metrics that implement them
- [x] 1.3 Leave the dispatch logic unchanged, so no built-in metric changes behaviour

## 2. Document
- [x] 2.1 State on `Prediction` which fields are populated under which conditions
- [x] 2.2 State on the marker interfaces what implementing one obliges the model to supply

## 3. Verify
- [x] 3.1 Define a metric in an external test package that declares it needs probabilities, and assert it receives them
- [x] 3.2 Assert a metric declaring nothing still receives predictions as before
- [x] 3.3 Assert a probability-requiring metric against a model with no probabilities is refused with an error
- [x] 3.4 Assert every built-in metric produces the same score it did before

## 4. Record
- [x] 4.1 Document custom metrics in `Docs/ml.md`
- [x] 4.2 Changelog entry in `CHANGELOG.md` and `CHANGELOG_TW.md`
