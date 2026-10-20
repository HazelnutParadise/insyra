"""Regenerates data_snooping_baseline.json, the reference values quant's RealityCheck and SPATest tests compare against.

Everything is computed here from the formulas of Hansen (2005), "A Test for Superior Predictive Ability",
Journal of Business & Economic Statistics 23(4), and White (2000), not taken from a library: Python's
arch.bootstrap.SPA differs from the paper (see Docs/quant.md). Each case stores its own bootstrap time
positions, drawn here with NumPy, so the test can feed the same positions to the Go code and compare the
p-values exactly.

    python data_snooping_baseline.py data_snooping_baseline.json
"""
import json
import math
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


def omega2(x, block):
    n = len(x)
    q = 1.0 / block
    c = x - x.mean()
    total = np.dot(c, c) / n
    for i in range(1, n):
        kappa = (n - i) / n * (1 - q) ** i + i / n * (1 - q) ** (n - i)
        total += 2 * kappa * np.dot(c[:-i], c[i:]) / n
    return total


def case(seed, n, k, block, benchmark, scheme, reps, edge=None, poor=None):
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
        perf[:, j] = 0.0002 + 0.6 * common + ar
    if edge is not None:
        perf[:, edge] += 0.006
    if poor is not None:
        perf[:, poor] -= 0.02
    bench = rng.normal(0.0001, 0.008, n) + 0.5 * common if benchmark else np.zeros(n)
    d = perf - bench[:, None]

    dbar = d.mean(axis=0)
    omega = np.sqrt(np.array([omega2(d[:, j], block) for j in range(k)]))
    root = math.sqrt(n)
    t_k = root * dbar / omega
    t_spa = max(t_k.max(), 0.0)
    v_rc = max(root * dbar.max(), 0.0)

    lower = np.maximum(dbar, 0.0)
    threshold = -np.sqrt(omega ** 2 / n * 2 * math.log(math.log(n)))
    consistent = np.where(dbar >= threshold, dbar, 0.0)

    draw = stationary_positions if scheme == "stationary" else moving_positions
    positions = [draw(rng, n, block) for _ in range(reps)]
    counts = {"rc": 0, "lower": 0, "consistent": 0, "upper": 0}
    for pos in positions:
        star = d[pos].mean(axis=0)
        if max(0.0, (root * (star - dbar)).max()) >= v_rc:
            counts["rc"] += 1
        for name, center in (("lower", lower), ("consistent", consistent), ("upper", dbar)):
            if max(0.0, (root * (star - center) / omega).max()) >= t_spa:
                counts[name] += 1

    return {
        "name": f"seed{seed}",
        "perf": perf.tolist(),
        "benchmark": bench.tolist() if benchmark else None,
        "block_size": block,
        "positions": positions,
        "dbar": dbar.tolist(),
        "omega": omega.tolist(),
        "spa_statistic": t_spa,
        "spa_best": int(np.argmax(t_k)),
        "rc_statistic": v_rc,
        "rc_best": int(np.argmax(dbar)),
        "p_rc": counts["rc"] / reps,
        "p_lower": counts["lower"] / reps,
        "p_consistent": counts["consistent"] / reps,
        "p_upper": counts["upper"] / reps,
    }


def main(path):
    cases = [
        case(1, 60, 5, 5, True, "stationary", 120),
        case(2, 80, 8, 1, False, "stationary", 120, poor=3),
        case(3, 50, 3, 50, True, "moving", 120),
        case(4, 100, 6, 10, True, "moving", 120, edge=2, poor=5),
    ]
    with open(path, "w") as f:
        json.dump(cases, f)


if __name__ == "__main__":
    main(sys.argv[1])
