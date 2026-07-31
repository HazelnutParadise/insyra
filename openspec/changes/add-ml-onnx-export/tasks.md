# Tasks

## 1. Serialise
- [ ] 1.1 Generate the protobuf types from the ONNX schema, with no C dependency
- [ ] 1.2 Export linear regression as a linear regressor node
- [ ] 1.3 Export logistic regression as a linear classifier node, carrying its link
- [ ] 1.4 Export trees as tree ensemble nodes, preserving missing-value routing
- [ ] 1.5 Export the root package's scalers and encoders as their equivalents
- [ ] 1.6 Export a fitted pipeline as one graph

## 2. Refuse what cannot be expressed
- [ ] 2.1 Refuse a model with no equivalent, naming it
- [ ] 2.2 Write nothing on refusal
- [ ] 2.3 List in the documentation which models export and which do not, imputation among them, with the reason

## 3. Verify
- [ ] 3.1 Round-trip every exportable model through an independent runtime and compare predictions
- [ ] 3.2 Round-trip a pipeline on raw observations
- [ ] 3.3 Test that the exported file loads in a standard runtime unmodified
- [ ] 3.4 Skip the round-trip cleanly when the runtime is unavailable, and say so — a skipped verification must not read as a passing one

## 4. Record
- [ ] 4.1 Document export in `Docs/ml.md`, including the list of what exports
- [ ] 4.2 Changelog entry in `CHANGELOG.md` and `CHANGELOG_TW.md`
