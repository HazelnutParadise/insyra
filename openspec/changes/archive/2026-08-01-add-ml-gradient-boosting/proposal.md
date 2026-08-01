# Change: Gradient boosting — the other half of the ensemble family

## Why
A forest reduces variance by averaging independent trees; boosting reduces bias by fitting each tree to what the previous ones left unexplained. They are the two standard ensemble answers, they fail differently, and applied work reaches for whichever the data rewards. With the forest landed, boosting is the remaining half.

The regression case is exact and small: the negative gradient of squared loss is the residual, and a regression tree's leaf mean is already the optimal squared-loss leaf value, so a stage is literally "fit a tree to the residuals". The classification case is where implementations differ in rigour: the trees are fitted to probability residuals, but their leaf means minimise the wrong loss. The standard fix — replacing each leaf's value with the Newton step Σ(y−p)/Σp(1−p) over the training rows in that leaf — is what makes the additive log-odds model converge on the loss actually being boosted, and it is implemented rather than approximated.

Binary classification only in this version. Multiclass boosting fits one tree per class per stage and is a different amount of machinery; a target with more classes is refused with the limit named, not silently approximated.

## What Changes
- Add `FitGradientBoostingRegressor` — squared loss, residual fitting, scikit-learn's defaults (100 stages, learning rate 0.1, depth-3 trees)
- Add `FitGradientBoostingClassifier` — binary logistic loss, prior log-odds base, Newton-step leaf values
- Stop early when the residuals reach zero, reporting how many stages actually ran
- Refuse: non-positive stages or learning rate, a single-class target, a multiclass target
- Verify: determinism with no seed involved, training fit improving with stages, probability validity, conformance, refusals, and accuracy agreement with scikit-learn

## Impact
- Affected specs: `ml-trees`
- Affected code: `ml/gradient_boosting.go`, docs, changelogs, `skills/insyra/`
- Additive; nothing existing changes behaviour
