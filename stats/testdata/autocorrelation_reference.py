"""Ljung-Box reference values from statsmodels 0.14.6 for stats/autocorrelation.go.

Run from stats/testdata after autocorrelation_reference.R, whose output holds
the clustered series:
    python autocorrelation_reference.py > autocorrelation_reference_py.txt
"""
import re

import numpy as np
from statsmodels.stats.diagnostic import acorr_ljungbox

t = np.arange(1, 31)
short = np.sin(t) * np.cos(t**2 / 7) + t % 3

clustered = {}
with open("autocorrelation_reference.txt") as f:
    for line in f:
        m = re.match(r"clustered\.x\[(\d+)\]=\s*(\S+)", line)
        if m:
            clustered[int(m.group(1))] = float(m.group(2))
clustered = np.array([clustered[i] for i in sorted(clustered)])

for prefix, x, max_lag in (("short", short, 4), ("clustered", clustered, 10)):
    res = acorr_ljungbox(x, lags=max_lag)
    for m in range(1, max_lag + 1):
        print(f"{prefix}.Q[{m}]={float(res['lb_stat'][m])!r}")
        print(f"{prefix}.PQ[{m}]={float(res['lb_pvalue'][m])!r}")
