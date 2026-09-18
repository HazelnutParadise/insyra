# stats

The tests, fits and models in `stats` return their result with an error last;
some return several values before it (`CorrelationMatrix`, `CorrelationAnalysis`,
`BartlettSphericity`). A few helpers return no error at all, such as `NormCDF`,
`DefaultFactorAnalysisOptions` and `RegisterKNNDeviceSearcher`. Nothing is
reported through `Err()` here, and nothing logs a warning in place of failing —
check the error.

Inputs are `insyra.IDataList` or `insyra.IDataTable`, which in practice means
`*insyra.DataList` and `*insyra.DataTable`: the interfaces carry an unexported
method, so a third party cannot implement them.

## Picking a test

| Question | Function |
| --- | --- |
| Does this sample's mean differ from a known value? | `SingleSampleTTest(data, mu, confidenceLevel...)` |
| Do two independent samples differ? | `TwoSampleTTest(a, b, equalVariance, confidenceLevel...)` |
| Do two measurements of the same subjects differ? | `PairedTTest(before, after, confidenceLevel...)` |
| The same, with the population σ known | `SingleSampleZTest(data, mu, sigma, alternative, confidenceLevel)` / `TwoSampleZTest(a, b, sigma1, sigma2, alternative, confidenceLevel)` |
| Do three or more groups differ? | `OneWayANOVA(groups...)` |
| Two factors at once | `TwoWayANOVA(factorALevels, factorBLevels, cells...)` |
| Repeated measures | `RepeatedMeasuresANOVA` |
| The same questions without assuming normality | `SingleSampleWilcoxon`, `PairedWilcoxon`, `MannWhitneyU`, `KruskalWallis`, `FriedmanTest` |
| Are two categorical variables related? | `ChiSquareIndependenceTest` |
| Does a distribution match expected counts? | `ChiSquareGoodnessOfFit` |
| Do groups have equal variance? | `FTestForVarianceEquality`, `BartlettTest`, `LeveneTest` |

`confidenceLevel` is variadic on the t-tests and a required parameter on the
z-tests. Leaving it out of a t-test uses 0.95. A value given outside `(0, 1)` is
an error on both; the z-tests have no fallback.

`alternative` on the z-tests is an `AlternativeHypothesis`; there is no default,
so pass one explicitly.

## Reading a result

Every test result embeds the same base:

```go
type testResultBase struct {
    Statistic   float64      // t, z, F, χ², …
    PValue      float64
    DF          *float64     // nil when the test has no degrees of freedom
    CI          *[2]float64  // nil when the test produces no interval
    EffectSizes []EffectSizeEntry
}
```

`EffectSizeEntry` is `{Type string; Value float64}` with `Type` one of
`"cohen_d"`, `"hedges_g"`, `"glass_delta"` and so on — read the slice, do not
assume a position.

`TTestResult` adds `Mean`, `Mean2`, `MeanDiff`, `N`, `N2`; the pointers are nil
when the test has only one group.

**The t-tests keep the sign of the effect size. The z-tests report its absolute
value**, to match the R output their reference tests are pinned to. Do not read
direction out of a z-test's effect size. Which package and function each
method is checked against, and how closely, is the "Reference Implementations"
table in `Docs/stats.md`.

**Constant data has no variance**, so a t statistic comes out ±Inf (or NaN when
the mean equals `mu`) with a p-value of 0 or NaN, and no error. Check the
statistic before reporting it.

## Correlation and regression

```go
r, err := stats.Correlation(x, y, stats.PearsonCorrelation)
corr, pvals, err := stats.CorrelationMatrix(dt, stats.SpearmanCorrelation)
fit, err := stats.LinearRegression(y, x1, x2)   // y first, then each predictor
```

`CorrelationMatrix` returns two DataTables — the coefficients and the p-values —
before the error.

The regression family is `LinearRegression`, `WeightedLinearRegression`,
`PolynomialRegression`, `ExponentialRegression`, `LogarithmicRegression`,
`LogisticRegression`, `PoissonRegression`, `RidgeRegression`, `LassoRegression`
and `GLM`. Only `LogisticRegression` and `PoissonRegression` have a
`*WithOptions` form taking a struct. `GLM` requires `GLMOptions` as its first
argument, `LassoRegression` takes optional `LassoOptions` after its predictors,
and the others take no options.

## Clustering and dimension reduction

```go
km, err := stats.KMeans(dt, 3)                       // or KMeans(dt, 3, opts)
hc, err := stats.HierarchicalAgglomerative(dt, …)
labels, err := stats.CutTreeByK(hc, 3)               // or CutTreeByHeight
db, err := stats.DBSCAN(dt, eps, minPts)
sil, err := stats.Silhouette(dt, insyra.NewDataList(labels)) // labels is a []int
p, err := stats.PCA(dt, …)
fa, err := stats.FactorAnalysis(dt, stats.DefaultFactorAnalysisOptions())
```

`KMeansOptions` has `NStart`, `IterMax` and `Seed` (a `*int64` — set it for a
reproducible run).

**Factor analysis: leave `Rotation.Restarts` at its default of 1 when the
rotation is orthogonal.** More than one restart currently returns a
non-orthogonal rotation matrix, so the rotated loadings stop describing the same
model. Tracked as issue #373. `RotationConverged` says whether the returned
rotation converged, and `MaxIter` governs extraction rather than rotation.

## Things that are easy to get wrong

- A blank or non-numeric cell is refused, not read as zero — but how much the
  error tells you, and whether NaN and ±Inf count as unreadable, differs by
  function. Clean the list first with `ClearNaNs`/`ClearNils` and none of this
  matters.
  - `SingleSampleTTest`, `TwoSampleTTest`, `SingleSampleZTest`, `TwoSampleZTest`,
    `FTestForVarianceEquality`, `BartlettTest`, `LeveneTest`, `CalculateMoment`,
    `Skewness` and `Kurtosis` name the series and the **one-based** row, and
    refuse NaN and ±Inf as well: `data contains a non-numeric value at row 3:
    <nil>`, `data1 contains a non-finite value at row 3: NaN`, `group 0 contains
    a non-numeric value at row 3: <nil>`.
  - `PairedTTest` and `MannWhitneyU` name only the series: `invalid numeric
    value in data1`, with no position.
  - `OneWayANOVA`, `KruskalWallis` and `FriedmanTest` name a **zero-based**
    position: `invalid data at group 0 index 2`, `invalid numeric value at
    group 0 index 2`, `invalid numeric value at subject 0 condition 2`.
  - None of the five in the previous two bullets refuses NaN or ±Inf. A NaN
    reaches the arithmetic: `PairedTTest` returns a NaN statistic and a NaN
    p-value with a nil error, `OneWayANOVA` returns F and P as NaN with a nil
    error, and `KruskalWallis`, `MannWhitneyU` and `FriedmanTest` rank the NaN
    and return an ordinary-looking result. Check the statistic before reporting
    it.
- `TwoWayANOVA` wants its cells in row-major order, `factorALevels ×
  factorBLevels` of them. There is no long-format entry point.
- `ChiSquareTestResult.ContingencyTable` stores each cell as a `[2]float64`
  (observed, expected) inside a DataTable, so the usual DataTable helpers do not
  read it usefully.
- `stats` never calls `LogFatal`; if a function returns an error, the result
  pointer is nil.
