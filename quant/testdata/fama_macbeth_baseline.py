"""Regenerates fama_macbeth_baseline.json, the reference values quant's FamaMacBeth test compares against.

The values come from linearmodels.FamaMacBeth (pip install linearmodels) with the Bartlett kernel, an explicit
bandwidth and debiased=False, which is the variance FamaMacBeth computes. Lag 0 is cov_type="unadjusted".

    python fama_macbeth_baseline.py fama_macbeth_baseline.json
"""
import json
import sys

import numpy as np
import pandas as pd
from linearmodels import FamaMacBeth


def panel(seed, days, stocks):
    rng = np.random.default_rng(seed)
    beta = np.zeros((days, 3))
    level = np.array([0.001, 0.004, -0.002])
    for t in range(days):
        prev = beta[t - 1] - level if t else 0
        beta[t] = level + 0.6 * prev + rng.normal(0, 0.003, 3)
    rows = []
    for t in range(days):
        n = 2 if t == 17 else stocks  # day 17 has fewer stocks than regressors
        x = rng.normal(0, 1, (n, 2))
        r = beta[t, 0] + x @ beta[t, 1:] + rng.normal(0, 0.02, n)
        for i in range(n):
            rows.append([t, i, float(r[i]), float(x[i, 0]), float(x[i, 1])])
    rows[5][3] = None  # a missing characteristic drops that stock-day
    return rows


def main(out):
    rows = panel(478, 60, 40)
    df = pd.DataFrame(rows, columns=["date", "stock", "ret", "mom", "size"]).set_index(["stock", "date"])
    df["const"] = 1.0
    mod = FamaMacBeth(df["ret"], df[["const", "mom", "size"]])
    fits = {}
    for lag in (0, 3, 5):
        if lag == 0:
            res = mod.fit(cov_type="unadjusted", debiased=False)
        else:
            res = mod.fit(cov_type="kernel", kernel="bartlett", bandwidth=lag, debiased=False)
        fits[str(lag)] = {
            "params": [float(v) for v in res.params],
            "std_errors": [float(v) for v in res.std_errors],
            "tstats": [float(v) for v in res.tstats],
        }
    with open(out, "w") as f:
        json.dump({"rows": rows, "fits": fits}, f)


if __name__ == "__main__":
    main(sys.argv[1])
