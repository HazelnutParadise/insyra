"""Regenerates indicator_baseline.json, the reference values quant's indicator tests compare against.

The values come from TA-Lib (pip install TA-Lib), except the recursive KD, which TA-Lib does not have and which
is computed here from its definition (K and D start at 50 and move by 1/3 of the gap to RSV and to K).

    python indicator_baseline.py indicator_baseline.json
"""
import json
import sys

import numpy as np
import talib


def bars(seed, n):
    rng = np.random.default_rng(seed)
    r = rng.normal(0.0004, 0.015, n) + 0.01 * np.sin(np.arange(n) / 11.0)
    c = np.round(100 * np.exp(np.cumsum(r)), 4)
    o = np.r_[c[0], c[:-1]] * (1 + rng.normal(0, 0.004, n))
    h = np.round(np.maximum(o, c) * (1 + np.abs(rng.normal(0, 0.006, n))), 4)
    l = np.round(np.minimum(o, c) * (1 - np.abs(rng.normal(0, 0.006, n))), 4)
    return h, l, c


def recursive_kd(h, l, c, n, sk, sd):
    k = np.full(len(c), np.nan)
    d = np.full(len(c), np.nan)
    pk = pd = 50.0
    for i in range(n - 1, len(c)):
        hh, ll = h[i - n + 1:i + 1].max(), l[i - n + 1:i + 1].min()
        rsv = 50.0 if hh == ll else (c[i] - ll) / (hh - ll) * 100
        pk = pk * (sk - 1) / sk + rsv / sk
        pd = pd * (sd - 1) / sd + pk / sd
        k[i], d[i] = pk, pd
    return k, d


def clean(a):
    return [None if np.isnan(x) else float(x) for x in a]


cases = []
for seed, n in [(1, 300), (2, 260), (3, 120)]:
    h, l, c = bars(seed, n)
    out = {}
    for p in (1, 5, 20):
        out[f'sma{p}'] = clean(talib.SMA(c, p))
    for p in (3, 12, 26):
        out[f'ema{p}'] = clean(talib.EMA(c, p))
    for f, s, g in [(12, 26, 9), (5, 13, 4), (3, 8, 1)]:
        m, sg, hi = talib.MACD(c, f, s, g)
        out[f'macd{f}_{s}_{g}'] = dict(macd=clean(m), signal=clean(sg), hist=clean(hi))
    for p, k in [(20, 2.0), (10, 1.5), (5, 0.0)]:
        u, m, lo = talib.BBANDS(c, p, k, k, 0)
        out[f'boll{p}_{k}'] = dict(upper=clean(u), middle=clean(m), lower=clean(lo))
    for p in (1, 5, 14):
        out[f'atr{p}'] = clean(talib.ATR(h, l, c, p))
    out['trange'] = clean(talib.TRANGE(h, l, c))
    for n_, sk, sd in [(9, 3, 3), (5, 2, 4), (14, 1, 1)]:
        k, d = talib.STOCH(h, l, c, fastk_period=n_, slowk_period=sk, slowk_matype=0, slowd_period=sd, slowd_matype=0)
        out[f'stoch{n_}_{sk}_{sd}'] = dict(k=clean(k), d=clean(d))
        k, d = recursive_kd(h, l, c, n_, sk, sd)
        out[f'kd{n_}_{sk}_{sd}'] = dict(k=clean(k), d=clean(d))
    cases.append(dict(high=h.tolist(), low=l.tolist(), close=c.tolist(), expected=out))
json.dump(dict(talib=talib.__version__, cases=cases), open(sys.argv[1], 'w'), separators=(',', ':'))
