# Design: stats-discrete-time-hazard

## The model

A spell lasts `d` periods (a whole number, 1 or more) and either ended in period `d` (`observed` true) or was still going when observation stopped (censored). The discrete-time hazard is `h(t) = P(ends in period t | lasted to t)`. Allison's person-period method estimates it:

1. A spell of `d` periods becomes `d` rows. Row `t` has `y = 1` only for the last row of an observed spell.
2. Fit a binomial GLM of `y` on a function of `t` and the covariates: `g(h) = f(t) + x'b`, with `g` logit or cloglog.

Censoring needs no special handling: a censored spell simply has no `y = 1`. This is the exact likelihood of the discrete-time model, not an approximation.

## Proposed API

```go
func DiscreteTimeHazard(durations, observed insyra.IDataList, opts ...HazardOptions) (*HazardResult, error)

type HazardOptions struct {
    Covariates []insyra.IDataList // one value per spell, constant over its life
    Link       GLMLink            // LinkLogit (default) or LinkCLogLog
    Baseline   HazardBaseline     // how the hazard depends on the period; see below
    ConfidenceLevel float64
}

type HazardResult struct {
    Names []string                // baseline terms then covariates
    GLM   *GLMResult              // the fit, with coefficients, standard errors, p-values
    Hazard []float64              // fitted baseline hazard for period 1..max, covariates at their means
    Periods int
    Spells, Events int
}
```

`Baseline`: `BaselineLinear` (a term in `t`: "does ending get more likely as it lasts", the question in the issue, and the default), `BaselineLog` (a term in `log t`), `BaselineDummies` (one indicator per period, the flexible Allison form, which needs many events per period to be stable).

## Decisions for you

1. **Default baseline.** I recommend `BaselineLinear`: one interpretable coefficient answers the issue's question, and it stays stable with few spells. `BaselineDummies` is available for a full hazard curve.
2. **Include cloglog now?** `GLMLink` has no working cloglog (stub). Implementing it is a small, separately testable change (link, inverse link, derivative) and makes the model also a proportional-hazards model in continuous time. I recommend doing it in this change; the alternative is logit only and a follow-up.
3. **Covariates constant over a spell.** Time-varying covariates would need a long-format input. I recommend constant only now, since the issue's use (spell length alone, or with a few spell-level covariates) needs no more.
4. **Whole-number durations.** Non-integers, zero and negative durations are errors, with the row index. Rounding silently would change the model.

## Not proposed

Cox proportional hazards and parametric survival models are not in the issue and are a different estimator; they would be separate changes.

## Verification

R `glm(y ~ t + x, family = binomial("logit"))` and `binomial("cloglog")` on the same person-period data, through the existing cross-language test mechanism (`stats/testdata/crosslang_baseline.R`): coefficients, standard errors and deviance equal. Plus a simulation with a known constant hazard (coefficient on `t` near 0) and an increasing one.
