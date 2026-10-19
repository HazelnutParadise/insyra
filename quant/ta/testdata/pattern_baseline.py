"""Regenerates pattern_baseline.json, the reference values quant's pattern, fan and gap tests compare against.

The values come from an independent implementation (scripts/stock_pertest/pattern_lib.py in
TimLai666/investment-research, written for the study behind issue #471). It is not part of this repository.

    python pattern_baseline.py /path/to/pattern_lib.py pattern_baseline.json

numba is only a compiler for that file; a stand-in that leaves the functions as plain Python gives the same
numbers, so numba need not be installed. The inputs are synthetic and seeded.
"""
import importlib.util
import json
import sys
import types

import numpy as np

nb = types.ModuleType('numba')
nb.njit = lambda *a, **k: (a[0] if a and callable(a[0]) else (lambda f: f))
sys.modules['numba'] = nb
spec = importlib.util.spec_from_file_location('pattern_lib', sys.argv[1])
pl = importlib.util.module_from_spec(spec)
spec.loader.exec_module(pl)


def since_of(n, breaks):
    since = np.zeros(n, np.int64)
    s = 0
    for i in range(n):
        s = 0 if (i == 0 or i in breaks) else s + 1
        since[i] = s
    return since


def ohlcv(rng, n, gap_scale, spike):
    r = rng.normal(0.0003, 0.015, n) + 0.012 * np.sin(np.arange(n) / 9.0)
    r[n // 3:n // 2] *= 2.0
    c = 100 * np.exp(np.cumsum(r))
    prev = np.r_[c[0], c[:-1]]
    gap = rng.standard_t(3, n) * gap_scale
    gap[rng.random(n) < 0.6] = 0.0
    o = prev * (1 + gap)
    h = np.maximum(o, c) * (1 + np.abs(rng.normal(0, 0.006, n)))
    l = np.minimum(o, c) * (1 - np.abs(rng.normal(0, 0.006, n)))
    v = np.exp(rng.normal(10, 0.35, n))
    v[rng.random(n) < spike] *= 3.5
    return tuple(np.round(x, 4) for x in (o, h, l, c, v))


out = {'cases': []}
for name, seed, n, breaks, mult, gap_scale, spike in [
    ('a', 3, 1400, [500, 1000], 1.5, 0.008, 0.08), ('b', 5, 1800, [], 1.2, 0.008, 0.08),
    ('c', 8, 1500, [700], 2.0, 0.008, 0.08), ('d', 21, 2500, [900, 1700], 1.5, 0.02, 0.2),
    ('e', 22, 2500, [], 1.5, 0.03, 0.3)]:
    rng = np.random.default_rng(seed)
    o, h, l, c, v = ohlcv(rng, n, gap_scale, spike)
    since = since_of(n, set(breaks))
    th = pl.theta_series(c, since, mult)
    piv = pl.zigzag_online(c, since, th)
    pidx, ppr, pty, pcf, psg = piv
    ones = np.ones(n)
    E, rec, _ = pl.detect(c, ones, np.zeros(n), since, th)
    ct, cdir, cday, cvol, ctr, cfrom = rec
    th60 = pl.std60(c, since)
    g = pl.gap_features(o, h, l, c, v, since, th60)
    gaps = []
    for t in np.flatnonzero(g['gu'] | g['gd']):
        up = bool(g['gu'][t])
        labels = [k for k in pl.GAPS if g[k][t]]
        assert len(labels) <= 1
        f5 = g['fillu5' if up else 'filld5'][t]
        f20 = g['fillu20' if up else 'filld20'][t]
        gaps.append(dict(row=int(t), up=up, label=labels[0] if labels else '',
                         relativeVolume=round(float(g['rv'][t]), 10), trendZ=round(float(g['z'][t]), 10),
                         fill5=None if np.isnan(f5) else bool(f5), fill20=None if np.isnan(f20) else bool(f20)))
    case = dict(
        name=name, multiple=mult, breaks=breaks,
        open=o.tolist(), high=h.tolist(), low=l.tolist(), close=c.tolist(), volume=v.tolist(),
        pivots=[dict(kind=int(pty[i]), index=int(pidx[i]), price=float(ppr[i]), confirmed=int(pcf[i]), segment=int(psg[i])) for i in range(len(pidx))],
        patterns=[dict(kind=int(ct[i]), formed=int(cfrom[i]), breakout=int(cday[i]), direction=int(cdir[i]), pretrend=int(ctr[i])) for i in range(len(ct))],
        fans=[dict(direction=-1 if row < 26 else 1, number=(row - 23) % 3 + 1, at=int(t)) for row in range(23, 29) for t in np.flatnonzero(E[row])],
        gaps=gaps,
    )
    out['cases'].append(case)
    brk = sum(1 for d in cdir if d != 0)
    print(name, 'pivots', len(pidx), 'patterns', len(ct), 'broken', brk,
          'kinds', sorted(set(ct.tolist())), 'fans', len(case['fans']), 'gaps', {k: sum(1 for x in gaps if x['label'] == k) for k in pl.GAPS}, 'unclassified', sum(1 for x in gaps if not x['label']))
json.dump(out, open(sys.argv[2], 'w'), separators=(',', ':'))
