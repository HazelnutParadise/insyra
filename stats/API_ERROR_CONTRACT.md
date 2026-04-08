# stats API Error Contract

Updated: 2026-04-08

## Contract

1. Exported functions in `stats` now return `error` on invalid input or failed computation.
2. `stats` exported APIs no longer emit `LogWarning`/`LogFatal` as control flow.
3. Caller must handle returned `error` explicitly.

## Signature Pattern

- Result structs: `func X(...) (*ResultType, error)`
- Scalar values: `func X(...) (float64, error)`
- Multi-value APIs: append `error` as the last return value.
- Generic return (`Diag`): `func Diag(...) (any, error)`

## Updated Exported APIs

- T/Z tests: `SingleSampleTTest`, `TwoSampleTTest`, `PairedTTest`, `SingleSampleZTest`, `TwoSampleZTest`
- ANOVA/F tests: `OneWayANOVA`, `TwoWayANOVA`, `RepeatedMeasuresANOVA`, `FTestForVarianceEquality`, `LeveneTest`, `BartlettTest`, `FTestForRegression`, `FTestForNestedModels`
- Chi-square: `ChiSquareGoodnessOfFit`, `ChiSquareIndependenceTest`
- Correlation: `Correlation`, `CorrelationMatrix`, `CorrelationAnalysis`, `Covariance`, `BartlettSphericity`
- Regression: `LinearRegression`, `PolynomialRegression`, `ExponentialRegression`, `LogarithmicRegression`
- Other stats APIs: `PCA`, `Diag`, `CalculateMoment`, `Skewness`, `Kurtosis`

## Migration Example

```go
// before
res := stats.SingleSampleTTest(dl, 50)
if res == nil { /* failed */ }

// after
res, err := stats.SingleSampleTTest(dl, 50)
if err != nil { /* handle error */ }
```

## Validation Status

- `go test ./stats/...` passed
- `go test ./...` passed
