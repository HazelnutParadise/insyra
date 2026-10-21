"""Regenerates romano_wolf_baseline.json, the reference values quant's RomanoWolf tests compare against.

The stepdown is written here from its definition, not taken from a library: Romano & Wolf (2005),
"Stepwise Multiple Testing as Formalized Data Snooping", Econometrica 73(4), with the studentization
re-estimated on every resampled series. Each case stores its own bootstrap time positions, drawn with
NumPy, so the test can feed the same positions to the Go code and compare statistics, critical values
and the step at which every column is rejected.

The long-run variance is Hansen's (2005) stationary-bootstrap kernel estimate,
kappa(n, i) = ((n-i)/n)(1-q)^i + (i/n)(1-q)^(n-i), q = 1/block, summed directly over every lag;
the Go code obtains the same autocovariances by FFT.

    python romano_wolf_baseline.py romano_wolf_baseline.json
"""
import json
import math
from fractions import Fraction
import sys

import numpy as np


def stationary_positions(rng, n, block):
    out = []
    while len(out) < n:
        start = int(rng.integers(n))
        length = int(rng.geometric(1.0 / block))
        out.extend((start + j) % n for j in range(length))
    return out[:n]


def moving_positions(rng, n, block):
    out = []
    while len(out) < n:
        start = int(rng.integers(n - block + 1))
        out.extend(range(start, start + block))
    return out[:n]


def long_run_variance(x, mean, block):
    n = len(x)
    q = 1.0 / block
    c = x - mean
    total = np.dot(c, c) / n
    for i in range(1, n):
        kappa = (n - i) / n * (1 - q) ** i + i / n * (1 - q) ** (n - i)
        total += 2 * kappa * np.dot(c[:-i], c[i:]) / n
    return total


def sigmas(d, means, block):
    return np.sqrt(np.array([long_run_variance(d[:, j], means[j], block) for j in range(d.shape[1])]))


def case(name, seed, n, k, block, benchmark, scheme, reps, alpha, edges):
    rng = np.random.default_rng(seed)
    common = rng.normal(0, 0.01, n)
    perf = np.empty((n, k))
    for j in range(k):
        e = rng.normal(0, 0.01, n)
        ar = np.empty(n)
        prev = 0.0
        for t in range(n):
            prev = 0.3 * prev + e[t]
            ar[t] = prev
        perf[:, j] = 0.0 + 0.6 * common + ar
    for j, edge in edges.items():
        perf[:, j] += edge
    bench = rng.normal(0.0001, 0.008, n) + 0.5 * common if benchmark else np.zeros(n)
    d = perf - bench[:, None]

    root = math.sqrt(n)
    dbar = d.mean(axis=0)
    sigma = sigmas(d, dbar, block)
    w = root * dbar / sigma

    draw = stationary_positions if scheme == "stationary" else moving_positions
    positions = [draw(rng, n, block) for _ in range(reps)]
    w_star = np.empty((reps, k))
    for b, pos in enumerate(positions):
        y = d[pos]
        ybar = y.mean(axis=0)
        s = sigmas(y, ybar, block)
        w_star[b] = root * (ybar - dbar) / s

    # the ceil(alpha*B)-th largest maximum, with alpha taken as its shortest decimal form
    j = min(max(1, math.ceil(Fraction(repr(alpha)) * reps)), reps)
    rejected_at = [0] * k
    critical = []
    active = list(range(k))
    step = 0
    while active:
        step += 1
        maxima = np.sort(w_star[:, active].max(axis=1))[::-1]
        c = float(maxima[j - 1])
        critical.append(c)
        hit = [c_ for c_ in active if w[c_] > c]
        if not hit:
            break
        for c_ in hit:
            rejected_at[c_] = step
        active = [c_ for c_ in active if c_ not in hit]

    return {
        "name": name,
        "perf": perf.tolist(),
        "benchmark": bench.tolist() if benchmark else None,
        "block_size": block,
        "alpha": alpha,
        "positions": positions,
        "sigma": sigma.tolist(),
        "statistics": w.tolist(),
        "critical_values": critical,
        "rejected_at": rejected_at,
    }


def main(path):
    cases = [
        case("stationary-two-steps", 65, 120, 6, 5, True, "stationary", 200, 0.1, {1: 0.011, 4: 0.0045, 2: 0.004}),
        case("moving-no-benchmark", 12, 100, 5, 4, False, "moving", 200, 0.2, {0: 0.01, 3: 0.004}),
        case("stationary-block-1", 13, 90, 4, 1, True, "stationary", 200, 0.1, {2: 0.02}),
        case("stationary-null", 21, 80, 5, 8, True, "stationary", 200, 0.1, {}),
    ]
    with open(path, "w") as f:
        json.dump(cases, f)


if __name__ == "__main__":
    main(sys.argv[1])
