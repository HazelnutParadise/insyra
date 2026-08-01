# Tasks

## 1. Implementation
- [x] 1.1 `GridSearch(x, y, candidates, k, metric, options...)` returning per-candidate results, the winner's index and name, the shared seed, and the winner refitted on the full data
- [x] 1.2 One fold assignment for all candidates: honour a supplied seed, draw and report one otherwise
- [x] 1.3 Rank with the declared direction, first candidate keeps ties, skip candidates whose mean is undefined and refuse a grid where every mean is
- [x] 1.4 Refuse: empty grid, unnamed or duplicate names, nil fit, directionless metric

## 2. Verification
- [x] 2.1 A loss metric picks the smaller mean and a gain metric the larger, on a grid where the winner is knowable
- [x] 2.2 Two identical candidates produce identical per-fold scores — the same-folds guarantee
- [x] 2.3 The refitted winner predicts and its features match the full table
- [x] 2.4 The reported seed reproduces the identical result
- [x] 2.5 Every refusal

## 3. Documentation
- [x] 3.1 `Docs/ml.md` grid-search section, recording the expanded-grid departure from scikit-learn
- [x] 3.2 `skills/insyra/` update
- [x] 3.3 Changelog entries in both languages
