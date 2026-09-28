# GLM for Binary Outcomes and Counts

This tutorial shows how to choose between ordinary linear regression, logistic regression, Poisson regression, and generic GLM.

## Choose the model

Use `stats.LinearRegression` when the response is continuous and residuals are roughly symmetric. Use `stats.LogisticRegression` when the response has two classes such as `0/1`, `true/false`, or `"no"/"yes"`. Use `stats.PoissonRegression` when the response is a count, especially events per user, incidents per day, or arrivals per interval. Use `stats.GLM` when you need the family/link interface directly, prior weights, or a shared path for multiple GLM families.

For rates, pass an offset equal to `log(exposure)`. If Pearson chi-square divided by residual degrees of freedom is much larger than 1, the Poisson variance assumption may be too tight; `PoissonRegressionOptions{DispersionCheck: true}` marks `OverDispersed` when that statistic is greater than 1.5.

## Logistic regression

```go
package main

import (
    "fmt"
    "log"

    "github.com/HazelnutParadise/insyra"
    "github.com/HazelnutParadise/insyra/stats"
)

func main() {
    converted := insyra.NewDataList("no", "no", "yes", "no", "yes", "yes", "no", "yes")
    visits := insyra.NewDataList(1, 2, 3, 4, 5, 6, 7, 8)

    fit, err := stats.LogisticRegressionWithOptions(stats.LogisticRegressionOptions{
        PositiveClass: "yes",
    }, converted, visits)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("odds ratio = %.3f\n", fit.OddsRatios[1])
}
```

If the classes are nearly perfectly separated, use `SepError` to fail fast or `SepRidge` to apply a small L2 penalty:

```go
fit, err := stats.LogisticRegressionWithOptions(stats.LogisticRegressionOptions{
    SeparationPolicy: stats.SepRidge,
    Ridge:            1e-4,
}, converted, visits)
```

## Poisson regression with exposure

```go
counts := insyra.NewDataList(1, 2, 3, 5, 8, 13)
spend := insyra.NewDataList(0.2, 0.5, 0.9, 1.4, 2.0, 2.7)
exposure := insyra.NewDataList(
    math.Log(1.0), math.Log(1.2), math.Log(0.9),
    math.Log(1.3), math.Log(1.5), math.Log(1.8),
)

fit, err := stats.PoissonRegressionWithOptions(stats.PoissonRegressionOptions{
    Offset:          exposure,
    DispersionCheck: true,
}, counts, spend)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("IRR = %.3f, dispersion = %.3f\n", fit.IncidenceRateRatios[1], fit.DispersionStatistic)
```

## Generic GLM

```go
weights := insyra.NewDataList(1, 2, 1, 1, 2, 1)

fit, err := stats.GLM(stats.GLMOptions{
    Family:  stats.Poisson,
    Link:    stats.Log,
    Offset:  exposure,
    Weights: weights,
}, counts, spend)
if err != nil {
    log.Fatal(err)
}

// The model was fit with an offset, so the offset must be supplied for
// prediction too. Predict (without an offset) would return an error here;
// use PredictWithOffset and pass the offset for the new rows.
predictedRates, err := fit.PredictWithOffset(stats.PredictResponse, exposure, spend)
if err != nil {
    log.Fatal(err)
}
predictedRates.Show()
```

The implementation is tested against R `stats::glm()` and Python `statsmodels.GLM` for coefficients, standard errors, Wald statistics, deviance, log-likelihood, AIC/BIC, fitted values, offsets, and prior weights.
