"""Regenerates channels_baseline.json, the reference values quant/ta's channel and stop tests compare against.

Parabolic SAR comes from TA-Lib (SAR when start equals step, SAREXT otherwise; SAREXT marks a short trend with a
negative sign, which is dropped). ATR and EMA come from TA-Lib. Keltner, Donchian, SuperTrend and the chandelier
exit are not in TA-Lib and are written here from their definitions, independently of the Go code (SuperTrend follows
the TradingView script).

    python channels_baseline.py channels_baseline.json
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


def clean(a):
    return [None if np.isnan(x) else float(x) for x in a]


def sma(v, n):
    out = np.full(len(v), np.nan)
    for i in range(n - 1, len(v)):
        out[i] = v[i - n + 1:i + 1].mean()
    return out


def keltner(h, l, c, n, atr_n, k):
    mid = talib.EMA(c, n)
    atr = talib.ATR(h, l, c, atr_n)
    return mid + k * atr, mid, mid - k * atr


def keltner_classic(h, l, c, n):
    mid = sma((h + l + c) / 3, n)
    width = sma(h - l, n)
    return mid + width, mid, mid - width


def donchian(h, l, n):
    up = np.full(len(h), np.nan)
    lo = np.full(len(h), np.nan)
    for i in range(n, len(h)):
        up[i] = h[i - n:i].max()
        lo[i] = l[i - n:i].min()
    return up, (up + lo) / 2, lo


def supertrend(h, l, c, n, k):
    atr = talib.ATR(h, l, c, n)
    line = np.full(len(c), np.nan)
    direction = np.full(len(c), np.nan)
    pu = pl = None
    d = 0
    for i in range(len(c)):
        if np.isnan(atr[i]):
            continue
        mid = (h[i] + l[i]) / 2
        u, lo = mid + k * atr[i], mid - k * atr[i]
        if d == 0:
            d = -1
        else:
            u = u if (u < pu or c[i - 1] > pu) else pu
            lo = lo if (lo > pl or c[i - 1] < pl) else pl
            if d == -1:
                d = 1 if c[i] > u else -1
            else:
                d = -1 if c[i] < lo else 1
        direction[i] = d
        line[i] = lo if d == 1 else u
        pu, pl = u, lo
    return line, direction


def chandelier(h, l, c, n, k):
    atr = talib.ATR(h, l, c, n)
    long_ = np.full(len(c), np.nan)
    short = np.full(len(c), np.nan)
    for i in range(n, len(c)):
        long_[i] = h[i - n + 1:i + 1].max() - k * atr[i]
        short[i] = l[i - n + 1:i + 1].min() + k * atr[i]
    return long_, short


cases = []
for seed, n in [(1, 300), (2, 260), (3, 120)]:
    h, l, c = bars(seed, n)
    out = {}
    for name, (a, k, m) in {'20_10_2': (20, 10, 2.0), '10_5_1.5': (10, 5, 1.5)}.items():
        u, mid, lo = keltner(h, l, c, a, k, m)
        out['keltner' + name] = dict(upper=clean(u), middle=clean(mid), lower=clean(lo))
    for p in (20, 5):
        u, mid, lo = keltner_classic(h, l, c, p)
        out[f'keltnerClassic{p}'] = dict(upper=clean(u), middle=clean(mid), lower=clean(lo))
        u, mid, lo = donchian(h, l, p)
        out[f'donchian{p}'] = dict(upper=clean(u), middle=clean(mid), lower=clean(lo))
    for p, k in ((10, 3.0), (7, 2.0)):
        line, d = supertrend(h, l, c, p, k)
        out[f'supertrend{p}_{k}'] = dict(line=clean(line), direction=clean(d))
    for p, k in ((22, 3.0), (10, 2.0)):
        lg, sh = chandelier(h, l, c, p, k)
        out[f'chandelier{p}_{k}'] = dict(long=clean(lg), short=clean(sh))
    out['sar0.02_0.02_0.2'] = clean(talib.SAR(h, l, 0.02, 0.2))
    ext = talib.SAREXT(h, l, 0, 0, 0.01, 0.03, 0.15, 0.01, 0.03, 0.15)
    out['sar0.01_0.03_0.15'] = clean(np.abs(ext))
    cases.append(dict(high=h.tolist(), low=l.tolist(), close=c.tolist(), expected=out))
json.dump(dict(talib=talib.__version__, cases=cases), open(sys.argv[1], 'w'), separators=(',', ':'))
