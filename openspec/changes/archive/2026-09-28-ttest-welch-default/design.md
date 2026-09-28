# Design: ttest-welch-default

## Context

`TwoSampleTTest` already computes both variants: with `equalVariance` it uses `pooledSE` and `n1 + n2 − 2` degrees of freedom and Cohen's d on the pooled standard deviation, otherwise `twoSampleSE`, `welchDF` and d on the average variance. Only where the choice comes from changes.

## Decisions

### A bool field whose zero value is Welch

The field is `EqualVariance bool`, matching the name of the parameter it replaces and R's `var.equal`, and `false` is Welch, so the zero value of `TTestOptions` is the recommended test. A field named for Welch (`Welch bool`) would make the zero value Student's test, the default the owner did not choose.

### The positional bool is removed, not kept beside the field

Keeping `equalVariance` and adding the field would give two places for one setting and leave the question of which wins. Removing it is safe: a bool cannot be passed where `TTestOptions` is expected, so every old call fails to compile instead of changing meaning.

### The field is ignored by the one-sample and paired tests

One options type serves all three t-tests. `EqualVariance` has no meaning where there is one sample or one column of differences, so those tests ignore it rather than refuse it; refusing would make the shared type harder to reuse across a report's tests for no protection.

### The CLI follows the library

`ttest two a b` calls the library without the field, so it runs Welch's test, and the Forms line states the default. `equal` and `unequal` are unchanged, so scripts that spell the choice out print what they printed before.

## Risks / Trade-offs

- A CLI script that relied on `ttest two a b` meaning equal variances now gets Welch's statistic, degrees of freedom and p-value. The changelog marks it BREAKING under `### CLI`.
