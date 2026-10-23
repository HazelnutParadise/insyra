"""Regenerates oscillators_baseline.json, the reference values quant/ta's oscillator tests compare against.

CCI, Momentum, ROC (a percentage), the ultimate oscillator and the stochastic RSI come from TA-Lib. AR, BR, CR with
each midpoint, the force index and Elder-ray are not in TA-Lib and are written here from their definitions,
independently of the Go code.

    python oscillators_baseline.py oscillators_baseline.json
"""
import json
import sys

import numpy as np
import talib


def bars(seed, n):
    rng = np.random.default_rng(seed)
    r = rng.normal(0.0004, 0.015, n) + 0.01 * np.sin(np.arange(n) / 11.0)
    c = np.round(100 * np.exp(np.cumsum(r)), 4)
    o = np.round(np.r_[c[0], c[:-1]] * (1 + rng.normal(0, 0.004, n)), 4)
    h = np.round(np.maximum(o, c) * (1 + np.abs(rng.normal(0, 0.006, n))), 4)
    l = np.round(np.minimum(o, c) * (1 - np.abs(rng.normal(0, 0.006, n))), 4)
    v = np.round(np.exp(rng.normal(10, 0.35, n)))
    return o, h, l, c, v


def clean(a):
    return [None if np.isnan(x) else float(x) for x in a]


def arbr(o, h, l, c, n):
    ar = np.full(len(c), np.nan)
    br = np.full(len(c), np.nan)
    for i in range(n - 1, len(c)):
        s = slice(i - n + 1, i + 1)
        ar[i] = 100 * (h[s] - o[s]).sum() / (o[s] - l[s]).sum()
    for i in range(n, len(c)):
        s = slice(i - n + 1, i + 1)
        pc = c[i - n:i]
        br[i] = 100 * np.maximum(0, h[s] - pc).sum() / np.maximum(0, pc - l[s]).sum()
    return ar, br


def cr(o, h, l, c, n, kind):
    mid = {'hl': (h + l) / 2, 'wc': (2 * c + h + l) / 4, 'typ': (h + l + c) / 3, 'ohlc': (o + h + l + c) / 4}[kind]
    out = np.full(len(c), np.nan)
    for i in range(n, len(c)):
        s = slice(i - n + 1, i + 1)
        pm = mid[i - n:i]
        out[i] = 100 * np.maximum(0, h[s] - pm).sum() / np.maximum(0, pm - l[s]).sum()
    return out


def force(c, v, n):
    raw = np.full(len(c), np.nan)
    raw[1:] = np.diff(c) * v[1:]
    out = np.full(len(c), np.nan)
    out[1:] = talib.EMA(raw[1:], n)
    return out


cases = []
for seed, n in [(1, 300), (2, 260), (3, 120)]:
    o, h, l, c, v = bars(seed, n)
    out = {}
    for p in (20, 5):
        out[f'cci{p}'] = clean(talib.CCI(h, l, c, p))
        out[f'mom{p}'] = clean(talib.MOM(c, p))
        out[f'roc{p}'] = clean(talib.ROC(c, p))
    out['ultosc7_14_28'] = clean(talib.ULTOSC(h, l, c, 7, 14, 28))
    out['ultosc3_5_8'] = clean(talib.ULTOSC(h, l, c, 3, 5, 8))
    for p, st in ((14, 14), (6, 5)):
        k, d = talib.STOCHRSI(c, p, st, 3, 0)
        out[f'stochrsi{p}_{st}'] = dict(k=clean(k), d=clean(d))
    for p in (26, 6):
        ar, br = arbr(o, h, l, c, p)
        out[f'arbr{p}'] = dict(ar=clean(ar), br=clean(br))
    for kind in ('hl', 'wc', 'typ', 'ohlc'):
        out[f'cr26_{kind}'] = clean(cr(o, h, l, c, 26, kind))
    for p in (13, 1):
        out[f'force{p}'] = clean(force(c, v, p))
    for p in (13, 5):
        ema = talib.EMA(c, p)
        out[f'elder{p}'] = dict(bull=clean(h - ema), bear=clean(l - ema))
    cases.append(dict(open=o.tolist(), high=h.tolist(), low=l.tolist(), close=c.tolist(), volume=v.tolist(), expected=out))
json.dump(dict(talib=talib.__version__, cases=cases), open(sys.argv[1], 'w'), separators=(',', ':'))
