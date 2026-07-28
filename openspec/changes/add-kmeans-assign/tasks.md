# Tasks

## 1. Assign
- [ ] 1.1 Add an assignment method to the fitted KMeans result, returning the nearest centre index and its squared distance per observation
- [ ] 1.2 Break ties by the lowest centre index
- [ ] 1.3 Refuse a column count that does not match the fitted centres, with an error naming the mismatch
- [ ] 1.4 Reuse the package's existing squared-distance primitive rather than writing a second one
- [ ] 1.5 Keep the loop shaped so a device path can be added later without changing the signature

## 2. Verify
- [ ] 2.1 Test that assigning the training data reproduces the fit's own cluster assignment
- [ ] 2.2 Test the tie rule with observations placed exactly between two centres
- [ ] 2.3 Test the column-count refusal
- [ ] 2.4 Compare against R for a fitted model and held-out observations

## 3. Record
- [ ] 3.1 Document the method in `Docs/stats.md`
- [ ] 3.2 Changelog entry in `CHANGELOG.md` and `CHANGELOG_TW.md`
