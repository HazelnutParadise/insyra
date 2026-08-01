"""Generate window_fixtures.json from canonical pandas/numpy outputs.

Run this whenever the fixture set needs to be refreshed. CI does NOT run this
script; the committed JSON is the source of truth for Go tests.

Usage:
    python testdata/gen_window_fixtures.py
"""

from __future__ import annotations

import json
import math
import os
from typing import Any

import pandas as pd  # type: ignore


HERE = os.path.dirname(os.path.abspath(__file__))
OUTPUT = os.path.join(HERE, "window_fixtures.json")


def _clean(values: list[Any]) -> list[Any]:
    """Convert pandas NaN/None to JSON null, keep finite floats as float."""
    out: list[Any] = []
    for v in values:
        if v is None:
            out.append(None)
            continue
        try:
            if pd.isna(v):
                out.append(None)
                continue
        except (TypeError, ValueError):
            pass
        if isinstance(v, float):
            if math.isnan(v) or math.isinf(v):
                out.append(None)
            else:
                out.append(float(v))
        elif isinstance(v, (int,)):
            out.append(int(v))
        else:
            out.append(v)
    return out


def _series(values: list[Any]) -> pd.Series:
    # pandas auto-promotes to float; explicit dtype keeps NaN handling consistent.
    return pd.Series(values, dtype="float64")


CASES: dict[str, list[dict[str, Any]]] = {
    "shift": [
        {"input": [10, 20, 30, 40, 50], "periods": 1},
        {"input": [10, 20, 30, 40, 50], "periods": -2},
        {"input": [10, 20, 30, 40, 50], "periods": 0},
        {"input": [10, 20, 30], "periods": 5},
        {"input": [10, None, 30, 40], "periods": 1},
    ],
    "diff": [
        {"input": [10, 13, 18, 17, 25], "periods": 1},
        {"input": [10, 13, 18, 17, 25], "periods": 2},
        {"input": [10, None, 18, 17], "periods": 1},
    ],
    "pct_change": [
        {"input": [100, 110, 99, 99], "periods": 1},
        {"input": [10, 20, 40, 80], "periods": 1},
        {"input": [5, 0, 10], "periods": 1},
    ],
    "cumsum": [
        {"input": [1, 2, 3, 4, 5]},
        {"input": [1, None, 3, 4]},
        {"input": [10, -5, 3, -2]},
    ],
    "cumprod": [
        {"input": [2, 3, 4]},
        {"input": [1, 2, None, 3]},
    ],
    "cummax": [
        {"input": [3, 1, 4, 1, 5, 9, 2]},
        {"input": [None, 3, 1]},
    ],
    "cummin": [
        {"input": [3, 1, 4, 1, 5, 9, 2]},
        {"input": [None, 3, 1, 2]},
    ],
    "rolling_sum": [
        {"input": [1, 2, 3, 4, 5], "window": 2},
        {"input": [1, 2, 3, 4, 5], "window": 3},
        {"input": [1, 2, 3, 4, 5], "window": 3, "min_obs": 1},
    ],
    "rolling_mean": [
        {"input": [1, 2, 3, 4, 5], "window": 3},
        {"input": [1, 2, 3, 4, 5], "window": 3, "min_obs": 1},
        {"input": [1, None, 3, 4, 5], "window": 3, "min_obs": 2},
    ],
    "rolling_min": [
        {"input": [5, 1, 4, 2, 3], "window": 3},
    ],
    "rolling_max": [
        {"input": [5, 1, 4, 2, 3], "window": 3},
    ],
    "rolling_median": [
        {"input": [1, 3, 2, 4, 5], "window": 3},
    ],
    "rolling_std": [
        {"input": [1, 2, 3, 4, 5], "window": 3},
    ],
    "rolling_var": [
        {"input": [1, 2, 3, 4, 5], "window": 3},
    ],
    "expanding_mean": [
        {"input": [1, 2, 3, 4], "min_obs": 1},
        {"input": [1, 2, 3, 4], "min_obs": 3},
    ],
    "expanding_sum": [
        {"input": [1, 2, 3, 4], "min_obs": 1},
    ],
    "expanding_std": [
        {"input": [1, 2, 3, 4, 5], "min_obs": 1},
    ],
    "expanding_var": [
        {"input": [1, 2, 3, 4, 5], "min_obs": 1},
    ],
}


def run_case(op: str, case: dict[str, Any]) -> list[Any]:
    s = _series(case["input"])
    if op == "shift":
        return _clean(s.shift(case["periods"]).tolist())
    if op == "diff":
        return _clean(s.diff(case["periods"]).tolist())
    if op == "pct_change":
        return _clean(s.pct_change(case["periods"]).replace([float("inf"), float("-inf")], float("nan")).tolist())
    if op == "cumsum":
        return _clean(s.cumsum().tolist())
    if op == "cumprod":
        return _clean(s.cumprod().tolist())
    if op == "cummax":
        return _clean(s.cummax().tolist())
    if op == "cummin":
        return _clean(s.cummin().tolist())
    if op.startswith("rolling_"):
        method = op[len("rolling_"):]
        w = case["window"]
        m = case.get("min_obs", w)
        r = s.rolling(window=w, min_periods=m)
        result = getattr(r, method)()
        return _clean(result.tolist())
    if op.startswith("expanding_"):
        method = op[len("expanding_"):]
        m = case.get("min_obs", 1)
        e = s.expanding(min_periods=m)
        result = getattr(e, method)()
        return _clean(result.tolist())
    raise ValueError(f"unknown op: {op}")


def main() -> None:
    fixtures: dict[str, list[dict[str, Any]]] = {}
    for op, cases in CASES.items():
        fixtures[op] = []
        for case in cases:
            expected = run_case(op, case)
            fixtures[op].append({**case, "expected": expected})

    with open(OUTPUT, "w", encoding="utf-8") as f:
        json.dump(fixtures, f, indent=2, ensure_ascii=False)
        f.write("\n")
    print(f"wrote {OUTPUT}")


if __name__ == "__main__":
    main()
