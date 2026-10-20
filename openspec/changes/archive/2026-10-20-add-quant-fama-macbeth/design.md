# Design: add-quant-fama-macbeth

## Daily regression

Each date's rows with a finite return and every characteristic finite form an n × (k+1) matrix with a column of ones. The regression is solved with gonum's thin SVD. The rank uses NumPy's `matrix_rank` tolerance, `max(n, k+1) · ε · σ_max`; a rank below k+1 means the cross-section is collinear (including a characteristic that is constant that day), and the day is skipped with that reason, the way `linearmodels` drops the day. R² is `1 − SSE/TSS` with the return demeaned; a day whose returns are all equal has R² NaN and is left out of `MeanR2`.

`Standardize` replaces each characteristic by `(x − mean)/sd` over that day's used rows, with the sample standard deviation (n−1), the pandas default. A zero standard deviation is a collinear day.

## Time-series test

Each coefficient's series over the estimated days goes to `stats.NeweyWestMeanTest`. The option's sign is translated: 0 → −1 (automatic, `stats.NeweyWestLag(T)`), negative → 0, positive unchanged. The automatic lag depends only on T, so every coefficient shares it, as in `linearmodels`; its own automatic bandwidth is the Newey-West (1994) data-driven one, so the baseline passes an explicit lag.

The variance is `S/T` with Bartlett weights `1 − j/(lag+1)` and no small-sample factor, which equals `linearmodels.FamaMacBeth.fit(cov_type="kernel", kernel="bartlett", bandwidth=lag, debiased=False)`; `debiased=True` divides by T−1 instead. The p value is two-sided from t with T−1 degrees of freedom, as `NeweyWestMeanTest` reports it. `linearmodels` reads it from the normal when `debiased=False` and from t with N−k−1 (all stock-days, not T) when `debiased=True`, so the p values are compared against the t statistic, not against `linearmodels`.

Fewer than two estimated days is an error: no time-series test exists.

## Baseline

`quant/testdata/fama_macbeth_baseline.py` builds a seeded panel (including a missing cell and a thin day), runs `linearmodels` with lags 0, 3 and 5, and writes `fama_macbeth_baseline.json`, which the Go test reads; CI needs no Python.
