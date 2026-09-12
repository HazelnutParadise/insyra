# stats

Every exported function in `stats` returns `(result, error)`. Nothing is
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
z-tests. Outside `(0, 1)` it falls back to 0.95.

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
direction out of a z-test's effect size.

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
and `GLM`. The `*WithOptions` forms take a struct; the plain ones use defaults.

## Clustering and dimension reduction

```go
km, err := stats.KMeans(dt, 3)                       // or KMeans(dt, 3, opts)
hc, err := stats.HierarchicalAgglomerative(dt, …)
labels := stats.CutTreeByK(hc, 3)                    // or CutTreeByHeight
db, err := stats.DBSCAN(dt, eps, minPts)
sil, err := stats.Silhouette(dt, labels)
p, err := stats.PCA(dt, …)
fa, err := stats.FactorAnalysis(dt, stats.DefaultFactorAnalysisOptions())
```

`KMeansOptions` has `NStart`, `IterMax` and `Seed` (a `*int64` — set it for a
reproducible run).

`Rotation.Restarts` is how many starting points the rotation is run from, best
criterion value winning. The default of 1 is fine for Varimax and Oblimin;
raise it for Geomin or Simplimax, whose criteria have local minima.
`RotationConverged` says whether the chosen solution converged, and `MaxIter`
governs extraction rather than rotation.

## Things that are easy to get wrong

- A blank or non-numeric cell is refused, not read as zero. `PairedTTest`,
  `OneWayANOVA` and the non-parametric tests reject the list; clean it first
  with `ClearNaNs`/`ClearNils`.
- `TwoWayANOVA` wants its cells in row-major order, `factorALevels ×
  factorBLevels` of them. There is no long-format entry point.
- `ChiSquareTestResult.ContingencyTable` stores each cell as a `[2]float64`
  (observed, expected) inside a DataTable, so the usual DataTable helpers do not
  read it usefully.
- `stats` never calls `LogFatal`; if a function returns an error, the result
  pointer is nil.
