# Tasks

## 1. Expose
- [ ] 1.1 Add an exported `Link` to the logistic result, matching `GLMResult.Link` in name and type
- [ ] 1.2 Add the same to the Poisson result
- [ ] 1.3 Populate both at fit time from the link already chosen internally

## 2. Verify
- [ ] 2.1 Test that the published link, applied to the linear predictor outside the package, reproduces what `Predict` returns
- [ ] 2.2 Test that logistic reports the logit link and Poisson the log link

## 3. Record
- [ ] 3.1 Document the field in `Docs/stats.md`
- [ ] 3.2 Changelog entry in `CHANGELOG.md` and `CHANGELOG_TW.md`
