"""Regenerates trend_baseline.json, the reference values quant/ta's trend tests compare against.

Aroon, TRIX, KAMA (fast 2, slow 30, TA-Lib's fixed constants) and the EMA of the TRIX signal come from TA-Lib. KAMA with
other fast and slow periods and Ichimoku are not in TA-Lib and are written here from their definitions, independently
of the Go code.

    python trend_baseline.py trend_baseline.json
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


def kama(p, n, fast, slow):
    out = np.full(len(p), np.nan)
    f, s = 2 / (fast + 1), 2 / (slow + 1)
    out[n - 1] = p[n - 1]
    for i in range(n, len(p)):
        path = np.abs(np.diff(p[i - n:i + 1])).sum()
        er = abs(p[i] - p[i - n]) / path if path else 0.0
        sc = (er * (f - s) + s) ** 2
        out[i] = out[i - 1] + sc * (p[i] - out[i - 1])
    return out


def mid(h, l, n):
    out = np.full(len(h), np.nan)
    for i in range(n - 1, len(h)):
        out[i] = (h[i - n + 1:i + 1].max() + l[i - n + 1:i + 1].min()) / 2
    return out


def ichimoku(h, l, tk, kj, sb, disp):
    t, k, b = mid(h, l, tk), mid(h, l, kj), mid(h, l, sb)
    a = np.full(len(h), np.nan)
    bb = np.full(len(h), np.nan)
    for i in range(disp, len(h)):
        j = i - disp
        if not (np.isnan(t[j]) or np.isnan(k[j])):
            a[i] = (t[j] + k[j]) / 2
        bb[i] = b[j]
    return t, k, a, bb


cases = []
for seed, n in [(1, 300), (2, 260), (3, 120)]:
    h, l, c = bars(seed, n)
    out = {}
    for p in (14, 5):
        down, up = talib.AROON(h, l, p)
        out[f'aroon{p}'] = dict(up=clean(up), down=clean(down), osc=clean(talib.AROONOSC(h, l, p)))
    for p, sg in ((15, 9), (5, 3)):
        trix = talib.TRIX(c, p)
        valid = np.flatnonzero(~np.isnan(trix))
        sig = np.full(len(c), np.nan)
        sig[valid[0]:] = talib.EMA(trix[valid[0]:], sg)
        out[f'trix{p}_{sg}'] = dict(trix=clean(trix), signal=clean(sig))
    out['kama10_2_30'] = clean(talib.KAMA(c, 10))
    out['kama30_2_30'] = clean(talib.KAMA(c, 30))
    out['kama6_3_20'] = clean(kama(c, 6, 3, 20))
    for tk, kj, sb, d in ((9, 26, 52, 26), (5, 10, 20, 10)):
        t, k, a, b = ichimoku(h, l, tk, kj, sb, d)
        out[f'ichimoku{tk}_{kj}_{sb}_{d}'] = dict(tenkan=clean(t), kijun=clean(k), senkouA=clean(a), senkouB=clean(b))
    cases.append(dict(high=h.tolist(), low=l.tolist(), close=c.tolist(), expected=out))
json.dump(dict(talib=talib.__version__, cases=cases), open(sys.argv[1], 'w'), separators=(',', ':'))
