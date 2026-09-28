# Design: chisq-observed-expected

## Decisions

### Two tables, not one table with two columns

R's `chisq.test` returns `observed` and `expected` as two objects of the same shape, and so does SciPy (`chi2_contingency` returns the expected frequencies beside the input). Keeping the shape of the contingency table matters for independence, where the table is rows × columns: two columns per cell would give a table no one can read as a contingency table. Two tables of the same shape work for both tests, and each is a normal table of numbers that `Sum`, `Show` and the rest of the library can use.

For goodness of fit the single column is named after its table (`Observed`, `Expected`), so `res.Observed.GetCol(insyra.Name("Observed"))` and `res.Observed.GetCol("A")` both work.

### Probabilities keyed by category

A map cannot be misaligned: each probability says which category it belongs to. The alternative the issue offered, a slice in input order, still depends on an ordering the caller has to know. Matching is exact against the category label the test tabulates, which is the value's text with surrounding spaces removed, the same label that names the table's rows.

A key that matches no category is an error, as the issue asks. That keeps a typo (`"Blu"` for `"Blue"`) from silently dropping a category. It also means a category with a probability but no observations cannot be tested, which was already true of the slice form: the input is raw observations, so a category that never occurs has no label to match. The doc states this limit.

A category without a key is also an error, rather than a probability of zero, because a zero expected count makes the statistic undefined and the old code already refused it.

### The CLI names the category too

`chisq gof colors 0.5 0.3 0.2` had the same misalignment as the library. The tokens become `label=p`, split at the last `=` so a label may itself contain one. A bare number is refused with the new form in the message, so an old script fails instead of testing a different distribution.
