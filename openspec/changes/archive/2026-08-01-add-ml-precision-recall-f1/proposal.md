# Change: Precision, recall and F1 — the three metrics every classification report leads with

## Why
The metric set stops at accuracy, log loss, ROC AUC and the confusion matrix. Precision, recall and F1 are the three numbers a classification result is ordinarily reported with — in papers and in production dashboards alike — and today a caller has to derive them from `ConfusionMatrixResult` by hand, redoing the per-class bookkeeping and the averaging conventions themselves. Everything they need is already computed; it is only not exposed.

Two conventions have to be settled rather than inherited blindly:

**The default averaging cannot be scikit-learn's.** sklearn defaults to `average='binary'` with `pos_label=1`, which only works because its labels are usually the integers 0 and 1. This package's labels are arbitrary values, so that default degenerates into guessing which label the caller means. Macro averaging is the one mode that is well defined over any label set without naming a class, so it is the default, and the departure is recorded.

**The positive class is load-bearing here, unlike ROC AUC.** The withdrawn positive-class change measured that AUC is invariant under swapping the class, because swapping the label swaps the score column with it. Precision and recall have no such symmetry — precision of "churned" and precision of "retained" are different numbers about different mistakes. So binary averaging does not default the positive class; it refuses to run without one named.

## What Changes
- Add `PrecisionMetric`, `RecallMetric` and `F1Metric`, each a `ClassLabelMetric` with `HigherIsBetter` direction, computed from the same label handling `ConfusionMatrix` uses
- Add `ClassAverage` with four modes: macro (the default), micro, weighted, and binary with a required named positive class
- Refuse a positive class combined with a non-binary average, rather than carrying an option that does nothing; refuse binary averaging over more than two observed labels, as scikit-learn does
- Follow scikit-learn's zero-division convention: a class never predicted contributes precision 0 rather than an error, documented
- Add direct helpers `Precision`, `Recall`, `F1` for the macro default, matching the existing helper table
- Verify the numbers against scikit-learn's `precision_recall_fscore_support` under the reference-toolchain gate, and against hand-worked examples unconditionally

## Impact
- Affected specs: `ml-model-selection`
- Affected code: `ml/model_selection.go`, `Docs/ml.md`, `skills/insyra/`, `.github/workflows/reference-verification.yml`
- Additive; nothing existing changes behaviour
