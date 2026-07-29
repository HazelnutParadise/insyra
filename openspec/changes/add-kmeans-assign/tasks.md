# Tasks

## 1. Assign
- [x] 1.1 Add an assignment method to the fitted KMeans result, returning the nearest centre index and its squared distance per observation
- [x] 1.2 Break ties by the lowest centre index
- [x] 1.3 Refuse a column count that does not match the fitted centres, with an error naming the mismatch
- [x] 1.4 Reuse the package's existing squared-distance primitive rather than writing a second one
- [x] 1.5 Keep the loop shaped so a device path can be added later without changing the signature

## 2. Verify
- [x] 2.1 Test that assigning the training data reproduces the fit's own cluster assignment
- [x] 2.2 Test the tie rule with observations placed exactly between two centres
- [x] 2.3 Test the column-count refusal
- [x] 2.4 Compare against R for a fitted model and held-out observations

## 3. Record
- [x] 3.1 Document the method in `Docs/stats.md`
- [x] 3.2 Changelog entry in `CHANGELOG.md` and `CHANGELOG_TW.md`
