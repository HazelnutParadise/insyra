# Change: Decision trees, the model family `stats` does not have

## Why
Everything else in v1 wraps something. A package that only wraps is a shim, and nobody installs a shim. Decision trees are the one family `stats` has no answer to, and they are the right one to add rather than any other:

They are what people reach for on tabular data, and they are what the classical ONNX corpus is full of — so the tree scorer written here serves both training and any later import. They have no `float64` legacy to preserve, so unlike everything wrapped, they can follow the precision contract from the first line. And split finding is selection-shaped, which under the acceleration rules is the one shape a device may accelerate by default.

The precision arrangement is not ours to invent. scikit-learn's tree module holds features in `float32` and targets and accumulators in `float64`, stating the roles in the declarations themselves. XGBoost removed its single-precision histogram option in 1.7 as "dangerous to use" and replaced it with fixed-point integer gradients. Two implementations arrived independently at the same answer, and it is the answer the precision contract in `delivery-plan.md` records.

## What Changes
- Add a decision tree for classification and for regression, built on a histogram of binned features
- Hold features in `float32`, accumulate in fixed-point integers, and report in `float64`, as the precision contract requires
- Handle missing values by learning which way they go at each split, rather than by imputing or refusing them
- Support categorical splits, since a tree that requires one-hot encoding first is a tree that cannot use its own strength
- Make the tree deterministic — the same data and the same options produce the same tree, on any machine
- Report feature importances
- Write no device kernel. Split finding is eligible; whether it pays is a measurement this change does not make

## Impact
- Affected specs: `ml-trees`
- Affected code: `ml/`
- Depends on `add-ml-estimator-protocol`
