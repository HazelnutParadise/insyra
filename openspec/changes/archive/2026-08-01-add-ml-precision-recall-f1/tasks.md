# Tasks

## 1. The metrics
- [x] 1.1 Add `ClassAverage` with macro as the zero value, plus micro, weighted and binary
- [x] 1.2 Add the shared per-class computation on top of the label handling `ConfusionMatrix` already has
- [x] 1.3 Add `PrecisionMetric`, `RecallMetric`, `F1Metric` — each `ClassLabelMetric`, classification kind, higher-is-better
- [x] 1.4 Add the `Precision`, `Recall`, `F1` direct helpers for the macro default

## 2. Refusals
- [x] 2.1 Binary averaging without a positive class
- [x] 2.2 A positive class under a non-binary average
- [x] 2.3 A named positive class that is not among the observed labels, listing what was observed
- [x] 2.4 Binary averaging over more than two observed labels

## 3. Verification
- [x] 3.1 Hand-worked multiclass example pinning every averaging mode for all three metrics
- [x] 3.2 A binary case where the two choices of positive class give different numbers — the property that made the ROC AUC option pointless is absent here, and the test proves it
- [x] 3.3 Zero-division: a never-predicted class contributes zero
- [x] 3.4 scikit-learn parity for all averaging modes, behind the reference-toolchain gate, and wired into the reference-verification workflow
- [x] 3.5 The metrics work through `CrossValidate` and `Score`, and the result carries the direction

## 4. Documentation
- [x] 4.1 Metrics table and an averaging section in `Docs/ml.md`, recording the departure from scikit-learn's binary default and why
- [x] 4.2 Update `skills/insyra/`
- [x] 4.3 Add the entry to `CHANGELOG.md` and `CHANGELOG_TW.md`
