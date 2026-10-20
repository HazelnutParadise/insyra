"""Regenerates volume_baseline.json, the reference values quant/ta's money-flow and volume-profile tests compare against.

MFI, the accumulation/distribution line and the Chaikin oscillator come from TA-Lib. Chaikin money flow, ease of
movement, the negative and positive volume indexes and the volume profile are not in TA-Lib and are written here from
their definitions, independently of the Go code.

    python volume_baseline.py volume_baseline.json
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
    v = np.round(np.exp(rng.normal(14, 0.4, n)))
    return h, l, c, v


def clean(a):
    return [None if np.isnan(x) else float(x) for x in a]


def mfv(h, l, c, v):
    return np.where(h != l, ((c - l) - (h - c)) / np.where(h != l, h - l, 1) * v, 0.0)


def cmf(h, l, c, v, n):
    flow = mfv(h, l, c, v)
    out = np.full(len(c), np.nan)
    for i in range(n - 1, len(c)):
        s = slice(i - n + 1, i + 1)
        out[i] = flow[s].sum() / v[s].sum()
    return out


def emv(h, l, v, n, scale):
    raw = np.full(len(h), np.nan)
    for i in range(1, len(h)):
        if v[i] == 0 or h[i] == l[i]:
            continue
        dist = (h[i] + l[i]) / 2 - (h[i - 1] + l[i - 1]) / 2
        raw[i] = dist / ((v[i] / scale) / (h[i] - l[i]))
    out = np.full(len(h), np.nan)
    for i in range(n, len(h)):
        w = raw[i - n + 1:i + 1]
        if not np.isnan(w).any():
            out[i] = w.mean()
    return out


def vindex(c, v, up):
    out = np.zeros(len(c))
    out[0] = 1000
    for i in range(1, len(c)):
        move = v[i] > v[i - 1] if up else v[i] < v[i - 1]
        out[i] = out[i - 1] * c[i] / c[i - 1] if move else out[i - 1]
    return out


def profile(h, l, v, n, bins, va):
    poc = np.full(len(h), np.nan)
    vah = np.full(len(h), np.nan)
    val = np.full(len(h), np.nan)
    for i in range(n - 1, len(h)):
        s = slice(i - n + 1, i + 1)
        lo, hi = l[s].min(), h[s].max()
        if hi == lo:
            continue
        width = (hi - lo) / bins
        vol = np.zeros(bins)
        for hh, ll, vv in zip(h[s], l[s], v[s]):
            if hh == ll:
                vol[min(int((hh - lo) / width), bins - 1)] += vv
                continue
            for k in range(bins):
                ov = min(hh, lo + (k + 1) * width) - max(ll, lo + k * width)
                if ov > 0:
                    vol[k] += vv * ov / (hh - ll)
        total = v[s].sum()
        if total == 0:
            continue
        best = int(np.argmax(vol))
        a = b = best
        held = vol[best]
        while held < va * total and (a > 0 or b < bins - 1):
            below = vol[a - 1] if a > 0 else -1
            above = vol[b + 1] if b < bins - 1 else -1
            if above > below:
                b += 1
                held += vol[b]
            else:
                a -= 1
                held += vol[a]
        poc[i] = lo + (best + 0.5) * width
        val[i], vah[i] = lo + a * width, lo + (b + 1) * width
    return poc, vah, val


cases = []
for seed, n in [(1, 300), (2, 260), (3, 120)]:
    h, l, c, v = bars(seed, n)
    out = {}
    for p in (14, 5):
        out[f'mfi{p}'] = clean(talib.MFI(h, l, c, v, p))
    out['ad'] = clean(talib.AD(h, l, c, v))
    for f, s in ((3, 10), (2, 5)):
        out[f'adosc{f}_{s}'] = clean(talib.ADOSC(h, l, c, v, f, s))
    for p in (20, 5):
        out[f'cmf{p}'] = clean(cmf(h, l, c, v, p))
    for p in (14, 3):
        out[f'emv{p}'] = clean(emv(h, l, v, p, 1e8))
    out['emv14_scale1e6'] = clean(emv(h, l, v, 14, 1e6))
    out['nvi'] = clean(vindex(c, v, False))
    out['pvi'] = clean(vindex(c, v, True))
    for p, b, va in ((20, 24, 0.7), (10, 12, 0.5)):
        poc, vah, val = profile(h, l, v, p, b, va)
        out[f'profile{p}_{b}_{va}'] = dict(poc=clean(poc), vah=clean(vah), val=clean(val))
    cases.append(dict(high=h.tolist(), low=l.tolist(), close=c.tolist(), volume=v.tolist(), expected=out))
json.dump(dict(talib=talib.__version__, cases=cases), open(sys.argv[1], 'w'), separators=(',', ':'))
