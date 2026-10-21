"""Regenerates abnormal_returns_baseline.json, the reference values quant's AbnormalReturns test compares against.

A fixed-seed synthetic panel of three stocks and a market index, with events planted at known rows. The reference
is written independently of the Go code with numpy and scipy: simple returns, scipy.stats.linregress for the
market model, a plain mean and t statistic across events for each relative day, and scipy.stats.t for the
two-tailed p-value. One event sits on a NaN price (dropped as a whole), one runs off the end of the table, and two
events of one stock sit 30 rows apart, so the later event's estimation window overlaps the earlier event's
event window and those rows are left out of the fit.

    python abnormal_returns_baseline.py abnormal_returns_baseline.json
"""
import json
import sys

import numpy as np
from scipy import stats

ROWS = 400
EVENTS = {0: [60, 150, 180, 300], 1: [80, 200, 390], 2: [100, 250]}
NAN_PRICE = (1, 205)  # stock 1, row 205: inside the window of its event on row 200
EVENT_WINDOW = (-5, 10)
ESTIMATION_WINDOW = (-60, -11)
MIN_ESTIMATION = 20


def panel(seed):
    rng = np.random.default_rng(seed)
    rm = rng.normal(0.0004, 0.01, ROWS)
    levels = 1000 * np.cumprod(1 + rm)
    prices = []
    betas = [0.8, 1.3, 1.0]
    for i in range(3):
        r = 0.0002 * i + betas[i] * rm + rng.normal(0, 0.012, ROWS)
        for s in EVENTS[i]:
            for d in range(-5, 11):
                if 0 <= s + d < ROWS:
                    r[s + d] += 0.03 * np.exp(-abs(d) / 3) * (1 if d >= 0 else 0.3)
        prices.append(100 * np.cumprod(1 + r))
    return levels, prices


def simple(x):
    out = np.full(len(x), np.nan)
    for t in range(1, len(x)):
        if not (np.isnan(x[t]) or np.isnan(x[t - 1])):
            out[t] = x[t] / x[t - 1] - 1
    return out


def study(prices, levels, model, min_gap):
    rm = simple(levels)
    w0, w1 = EVENT_WINDOW
    ars, kept, missing = [], [], 0
    for i, p in enumerate(prices):
        rs = simple(p)
        flagged = EVENTS[i]
        last = None
        for s in flagged:
            if min_gap and last is not None and s - last < min_gap:
                continue
            last = s
            if s + w0 < 1 or s + w1 >= ROWS or np.isnan(rs[s + w0:s + w1 + 1]).any() or np.isnan(rm[s + w0:s + w1 + 1]).any():
                missing += 1
                continue
            alpha, beta = 0.0, 1.0
            if model == "model":
                xs, ys = [], []
                for t in range(s + ESTIMATION_WINDOW[0], s + ESTIMATION_WINDOW[1] + 1):
                    if t < 1 or t >= ROWS or np.isnan(rs[t]) or np.isnan(rm[t]):
                        continue
                    if any(e != s and e + w0 <= t <= e + w1 for e in flagged):
                        continue
                    xs.append(rm[t])
                    ys.append(rs[t])
                if len(xs) < MIN_ESTIMATION:
                    missing += 1
                    continue
                fit = stats.linregress(xs, ys)
                alpha, beta = fit.intercept, fit.slope
            ar = rs[s + w0:s + w1 + 1] - (alpha + beta * rm[s + w0:s + w1 + 1])
            ars.append(ar)
            kept.append((i, s))
    ar = np.array(ars)
    car = np.cumsum(ar, axis=1)

    def tp(m):
        n = m.shape[0]
        mean = m.mean(axis=0)
        t = mean / (m.std(axis=0, ddof=1) / np.sqrt(n))
        p = 2 * stats.t.sf(np.abs(t), n - 1)
        return mean, t, p

    mar, art, arp = tp(ar)
    mcar, cart, carp = tp(car)
    return {
        "days": list(range(w0, w1 + 1)),
        "mean_ar": mar.tolist(), "ar_t": art.tolist(), "ar_p": arp.tolist(),
        "mean_car": mcar.tolist(), "car_t": cart.tolist(), "car_p": carp.tolist(),
        "events": len(kept), "missing": missing,
        "kept": [[a, b] for a, b in kept],
        "cars": car.tolist(),
    }


def main(out):
    levels, prices = panel(478)
    prices[NAN_PRICE[0]][NAN_PRICE[1]] = np.nan
    cfg = {
        "market_adjusted": study(prices, levels, "adjusted", 0),
        "market_model": study(prices, levels, "model", 0),
        "market_model_mingap": study(prices, levels, "model", 50),
    }
    data = {
        "prices": [[None if np.isnan(v) else float(v) for v in p] for p in prices],
        "levels": [float(v) for v in levels],
        "events": {str(k): v for k, v in EVENTS.items()},
        "event_window": list(EVENT_WINDOW),
        "estimation_window": list(ESTIMATION_WINDOW),
        "min_estimation": MIN_ESTIMATION,
        "cases": cfg,
    }
    with open(out, "w") as f:
        json.dump(data, f)


if __name__ == "__main__":
    main(sys.argv[1])
