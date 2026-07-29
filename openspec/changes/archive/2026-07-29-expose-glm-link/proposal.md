# Change: Expose the link function on logistic and Poisson results

## Why
`GLMResult` publishes the link it was fitted with — `Link GLMLink` at `stats/regression_glm.go:23`. The two specialised members of the same family do not: `logisticRegressionResult.link` and `poissonRegressionResult.link` are unexported (`stats/regression_logistic.go:54`, `stats/regression_poisson.go:47`), so nothing outside `stats` can tell how a fitted model maps its linear predictor to a response.

Today that is invisible, because the only code that needs the link lives inside `stats` and already has it. It stops being invisible the moment anything outside wants to serialise one of these models or reason about it — an ONNX exporter has to emit the right activation, and a wrapper has to say what `Predict` will return. The alternative is to assume: logistic is logit, Poisson is log. That happens to be true and it is still an assumption, made by code that cannot check it, about a field that already exists three feet away.

Three results in one family, one of which answers the question and two of which do not, is also just an inconsistency worth removing on its own terms.

## What Changes
- Export the link on the logistic and Poisson results, matching `GLMResult.Link` in name and type
- Keep the existing unexported field as the implementation, so nothing inside `stats` changes behaviour

## Impact
- Affected specs: `stats-regression`
- Affected code: `stats/regression_logistic.go`, `stats/regression_poisson.go`
- Additive: a new exported field on two structs, no signature changes
