# Design: stats-discrete-time-hazard

## The model

A spell lasts `d` periods (a whole number, 1 or more) and either ended in period `d` (`observed` true) or was still going when observation stopped (censored). The discrete-time hazard is `h(t) = P(ends in period t | lasted to t)`. Allison's person-period method estimates it:

1. A spell of `d` periods becomes `d` rows. Row `t` has `y = 1` only for the last row of an observed spell.
2. Fit a binomial GLM of `y` on a function of `t` and the covariates: `g(h) = f(t) + x'b`, with `g` logit or cloglog.

Censoring needs no special handling: a censored spell simply has no `y = 1`. This is the exact likelihood of the discrete-time model, not an approximation.

## API (revised after a review of the design)

```go
// stats
func PersonPeriod(durations, observed insyra.IDataList, covariates ...insyra.IDataList) (*insyra.DataTable, error) // columns Spell, Period, Event, then the covariates
func DiscreteTimeHazard(durations, observed insyra.IDataList, opts ...DiscreteTimeHazardOptions) (*DiscreteTimeHazardResult, error)

type DiscreteTimeHazardOptions struct {
    Covariates      []insyra.IDataList // one value per spell, constant over its life
    Link            GLMLink            // Logit (default) or Cloglog, the existing constants
    Baseline        HazardBaseline     // HazardBaselineLog (default), HazardBaselineLinear, HazardBaselineDummies
    ConfidenceLevel float64
}

type HazardTerm struct {
    Name                                string
    Coefficient, StdErr, ZValue, PValue float64
    CI                                  [2]float64
}

type DiscreteTimeHazardResult struct {
    Duration *HazardTerm  // the term in the period for the log and linear baselines; nil for dummies
    Terms    []HazardTerm // intercept, baseline terms, covariates, by name
    GLM      *GLMResult   // the full fit; its rows are person-periods, not spells
    Hazard   []float64    // fitted hazard for period 1..max with covariates at their means
    N, Events int         // spells and how many ended
}
func (r *DiscreteTimeHazardResult) Term(name string) (HazardTerm, bool)
```

`PersonPeriod` is public so a user who wants an interaction, a quadratic term or a stratified model can build the table and pass it to `GLM` themselves; `DiscreteTimeHazard` is that expansion plus the baseline columns plus `GLM`.

Usage:

```go
res, _ := stats.DiscreteTimeHazard(durations, ended)
res.Duration.Coefficient // > 0: the longer it has lasted, the more likely it ends
res.Duration.PValue
```

## What changed from the first version, and why

- **Names.** The options and result follow the function name, as every other `stats` function does (`KaplanMeier`, `KaplanMeierResult`). The baseline constants carry the `Hazard` prefix. The link constants are the existing `stats.Logit` and `stats.Cloglog`; the first draft named `LinkLogit` and `LinkCLogLog`, which do not exist.
- **Named terms.** `res.GLM.Coefficients[1]` depends on the order of the design matrix and moves when the baseline changes; `res.Duration` and `res.Term(name)` do not.
- **Default baseline is `log t`.** The coefficient of `t` changes with the unit of the period (days to weeks multiplies it by 5); the coefficient of `log t` does not. With cloglog, the `log t` coefficient is the discrete Weibull shape minus 1, so above 0 means a rising hazard, and cloglog is the exact discretisation of the continuous-time proportional-hazards model.
- **A free check.** With no covariates and one indicator per period, the fitted hazard of period `t` equals `Events/AtRisk` of `KaplanMeier`.

## Decisions for you

1. **Include cloglog now?** `GLM` has no working cloglog (a stub marked TODO in `stats/glm_links.go`). Implementing the link, its inverse and derivative is small and separately testable. I recommend doing it in this change.
2. **Covariates constant over a spell.** Time-varying covariates need a long-format input; with `PersonPeriod` public, a user can already build that table and call `GLM`. I recommend constant only.
3. **Whole-number durations.** Non-integers, zero and negative durations are errors, with the row index.

## Spells from `quant.Regimes`

`quant.Regimes` does not report the stretch after the last turning point, but that stretch, still going, is exactly the censored spell the model needs. Reviewed API:

```go
type RegimeOptions struct {
    MinPhase, MinCycle int
    SeriesRows int // number of rows of the price series; when > 0 the last, unfinished stretch is reported
}
type Regime struct {
    Kind        RegimeKind
    Start, End  int
    Length      int
    Change      float64
    ConfirmedAt int
    Ended       bool // true when a confirmed turning point closed it; false for the last stretch
}
func RegimeDurations(regimes []Regime, kind RegimeKind) (durations, observed *insyra.DataList, err error) // in quant; feeds KaplanMeier and DiscreteTimeHazard
```

`RegimeDurations` lives in `quant`, since `stats` must not depend on `quant`. Its two lists have the shape `KaplanMeier(durations, observed)` and `DiscreteTimeHazard` take, so a bull market is one row: how long it lasted and whether it ended.

```go
pivots, _ := quant.ZigZag(close, 0.10)
regimes, _ := quant.Regimes(pivots, quant.RegimeOptions{SeriesRows: n})
durations, ended, _ := quant.RegimeDurations(regimes, quant.RegimeBull)
res, _ := stats.DiscreteTimeHazard(durations, ended)
```

## Not proposed

Cox proportional hazards and parametric survival models are not in the issue and are a different estimator; they would be separate changes. A left-truncated spell (one already running when the data starts) must be excluded by the caller; the documentation says so.

## Verification

R `glm(y ~ log(t) + x, family = binomial("logit"))` and `binomial("cloglog")` on the same person-period data, through the existing cross-language test mechanism (`stats/testdata/crosslang_baseline.R`): coefficients, standard errors and deviance equal. The Kaplan-Meier agreement above. A simulation with a known constant hazard (coefficient on `log t` near 0) and an increasing one. A period with no event under `HazardBaselineDummies` is an error that names the periods, not a failure to converge.
