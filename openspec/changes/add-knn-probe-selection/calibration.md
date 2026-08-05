# Calibration record

All timings below were measured on this host on 2026-08-04 with the current
working tree, the public `stats.KNearestNeighbors` entry point, `k=5`,
`LeafSize=16`, and 1000 test rows. The generated data uses eight compact
clusters for the clustered regime and independent standard-normal coordinates
for the unstructured regime. Each cell was run once for each arm with the
same deterministic input seed. `GOMAXPROCS` was left at the host default.

## 1.1 Baseline

| regime | train rows | dims | brute | ball tree | ball / brute |
| --- | ---: | ---: | ---: | ---: | ---: |
| unstructured | 5,000 | 16 | 14.831 ms | 22.360 ms | 1.508 |
| unstructured | 5,000 | 64 | 51.922 ms | 115.545 ms | 2.225 |
| unstructured | 20,000 | 16 | 52.344 ms | 83.436 ms | 1.594 |
| unstructured | 20,000 | 64 | 206.986 ms | 440.458 ms | 2.128 |
| unstructured | 50,000 | 16 | 99.414 ms | 333.245 ms | 3.352 |
| unstructured | 50,000 | 64 | 481.250 ms | 1,854.247 ms | 3.853 |
| clustered | 5,000 | 16 | 10.350 ms | 5.702 ms | 0.551 |
| clustered | 5,000 | 64 | 48.521 ms | 15.483 ms | 0.319 |
| clustered | 20,000 | 16 | 36.326 ms | 17.162 ms | 0.472 |
| clustered | 20,000 | 64 | 196.495 ms | 83.345 ms | 0.424 |
| clustered | 50,000 | 16 | 104.826 ms | 57.899 ms | 0.552 |
| clustered | 50,000 | 64 | 473.832 ms | 347.140 ms | 0.733 |

## 1.2 Examined-candidate fractions

The counting query was run over the same 1000 test rows, `k=5`, and
`LeafSize=16`. A candidate is counted exactly when its training row's
squared distance is evaluated. Ball-tree means are:

| regime | train rows | dims | examined / all candidates |
| --- | ---: | ---: | ---: |
| unstructured | 5,000 | 16 | 0.996891 |
| unstructured | 5,000 | 64 | 0.999800 |
| unstructured | 20,000 | 16 | 0.986248 |
| unstructured | 20,000 | 64 | 0.999750 |
| unstructured | 50,000 | 16 | 0.957801 |
| unstructured | 50,000 | 64 | 0.999780 |
| clustered | 5,000 | 16 | 0.124960 |
| clustered | 5,000 | 64 | 0.125000 |
| clustered | 20,000 | 16 | 0.124867 |
| clustered | 20,000 | 64 | 0.124988 |
| clustered | 50,000 | 16 | 0.124562 |
| clustered | 50,000 | 64 | 0.124978 |

The measured crossover is a wide gap: every unstructured cell is above
0.95 and slower than brute force, while every clustered cell is below 0.13
and faster than brute force. The cutoff remains open until the LeafSize sweep
and sample-size/overhead measurements are complete.

## 1.3 LeafSize sweep

Each value is one `stats.KNearestNeighbors` ball-tree run over the same
1000-test-row matrix. Times are in milliseconds.

| regime | train rows | dims | leaf 8 | leaf 16 | leaf 32 | leaf 64 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| unstructured | 5,000 | 16 | 29.606 | 19.772 | 19.698 | 16.909 |
| unstructured | 5,000 | 64 | 81.554 | 69.494 | 68.359 | 64.227 |
| unstructured | 20,000 | 16 | 87.359 | 78.904 | 68.994 | 62.014 |
| unstructured | 20,000 | 64 | 401.931 | 346.737 | 292.317 | 282.674 |
| unstructured | 50,000 | 16 | 235.401 | 211.734 | 193.767 | 210.716 |
| unstructured | 50,000 | 64 | 1,576.600 | 1,396.449 | 2,878.846 | 1,387.696 |
| clustered | 5,000 | 16 | 4.759 | 4.662 | 4.057 | 4.041 |
| clustered | 5,000 | 64 | 17.600 | 15.519 | 14.773 | 13.588 |
| clustered | 20,000 | 16 | 25.806 | 21.523 | 16.284 | 16.469 |
| clustered | 20,000 | 64 | 85.868 | 79.582 | 67.361 | 68.648 |
| clustered | 50,000 | 16 | 65.190 | 48.666 | 48.037 | 53.198 |
| clustered | 50,000 | 64 | 293.199 | 268.845 | 268.345 | 307.465 |

No leaf size wins consistently. `LeafSize=16` stays the default because it is
within the measured best band on the large cells, avoids changing the
existing option default, and the probe cutoff will be calibrated against the
default rather than silently tuning another parameter.

## 1.4 Crossover batches

The crossover sweep uses the same deterministic eight-cluster generator while
varying its within-cluster noise. It is run in small `n`/dimension batches;
the tree and brute timings below are direct internal query timings over 1000
test rows, excluding construction. The first completed batch was `n=5,000`,
`dims=16` on 2026-08-05:

| noise | examined fraction | ball / brute | ball | brute |
| ---: | ---: | ---: | ---: | ---: |
| 0.15 | 0.124960 | 0.302 | 13.514 ms | 44.785 ms |
| 0.50 | 0.124960 | 0.289 | 10.303 ms | 35.638 ms |
| 1.00 | 0.124960 | 0.269 | 9.649 ms | 35.817 ms |
| 2.00 | 0.124960 | 0.258 | 9.537 ms | 36.904 ms |
| 4.00 | 0.142771 | 0.327 | 12.365 ms | 37.863 ms |
| 8.00 | 0.350071 | 0.720 | 27.192 ms | 37.785 ms |

The batch has not yet reached the wall-clock crossover, so no cutoff is
chosen from it.

The second completed batch was `n=5,000`, `dims=64` on 2026-08-05:

| noise | examined fraction | ball / brute | ball | brute |
| ---: | ---: | ---: | ---: | ---: |
| 0.15 | 0.125000 | 0.198 | 39.280 ms | 198.073 ms |
| 0.50 | 0.125000 | 0.197 | 38.649 ms | 196.675 ms |
| 1.00 | 0.125000 | 0.195 | 38.132 ms | 196.050 ms |
| 2.00 | 0.125000 | 0.188 | 37.150 ms | 197.281 ms |
| 4.00 | 0.152911 | 0.237 | 46.333 ms | 195.803 ms |
| 8.00 | 0.375764 | 0.552 | 108.161 ms | 195.988 ms |

The second batch also remains below the crossover at the largest measured
fraction.

The `n=20,000` crossover batches were completed on 2026-08-05:

| dims | noise | examined fraction | ball / brute |
| ---: | ---: | ---: | ---: |
| 16 | 8.00 | 0.321022 | 0.725 |
| 16 | 12.00 | 0.446715 | 0.915 |
| 16 | 14.00 | 0.502985 | 1.033 |
| 16 | 16.00 | 0.548886 | 1.187 |
| 64 | 8.00 | 0.360264 | 0.558 |

The `n=50,000`, `dims=16` batches were completed on 2026-08-05:

| noise | examined fraction | ball / brute |
| ---: | ---: | ---: |
| 12.00 | 0.415746 | 0.891 |
| 13.00 | 0.448558 | 1.050 |
| 14.00 | 0.474833 | 1.021 |

The observed wall-clock crossover is bracketed between fractions 0.416 and
0.449 on the largest cell, and between 0.447 and 0.503 at 20k/16d. The
cutoff is therefore frozen at **0.44**, the measured crossover rounded toward
brute force. It is below the first measured slower point in both brackets,
while all issue-ladder unstructured cells are at least 0.957801 and all
clustered cells are at most 0.125000.

### Sample-size variance and overhead

Eight fixed-stride offsets were measured on every ladder cell. The worst
standard deviation for `m=16` was 0.005815, on unstructured 50k/16d; the
corresponding values for `m=32`, `m=64`, and `m=128` were 0.005526, 0.004913,
and 0.002569. On clustered 50k/16d they were 0.000069, 0.000064, 0.000039,
and 0.000027. The extra cost grew linearly: on unstructured 50k/16d, eight
probes took 117.253 ms, 224.471 ms, 461.866 ms, and 902.380 ms respectively.
`m=16` is selected because it cuts the observed `m=8` worst variation while
avoiding the materially larger probe cost of `m>=32`; the measured regime
gap is more than 0.8 fraction units.

The n-floor batch measured probe overhead at the first auto-tree size and
larger sizes. At `n=64`, `m=16` cost 51.5 µs (16d) and 185.2 µs (64d) on
unstructured data, against 2.176 ms and 6.520 ms for brute; clustered cost
9.6 µs and 21.1 µs, against 0.955 ms and 3.399 ms. The existing static rule
already chooses brute for `n<64`. The measured n-floor is therefore **64**:
there is no auto-tree case below it, and the first eligible case has probe
overhead below 2.4% of the brute query time.

## 1.5 KD-tree branch check

The same 1000-test-row measurement was run at 4d and 8d, where the static
rule proposes a kd-tree, on 2026-08-05. Timings are direct internal query
timings and exclude construction.

| regime | train rows | dims | examined fraction | kd / brute |
| --- | ---: | ---: | ---: | ---: |
| unstructured | 5,000 | 4 | 0.044671 | 0.174 |
| unstructured | 5,000 | 8 | 0.500644 | 0.994 |
| unstructured | 20,000 | 4 | 0.012286 | 0.037 |
| unstructured | 20,000 | 8 | 0.236270 | 0.474 |
| unstructured | 50,000 | 4 | 0.005537 | 0.017 |
| unstructured | 50,000 | 8 | 0.139633 | 0.298 |
| clustered | 5,000 | 4 | 0.031494 | 0.135 |
| clustered | 5,000 | 8 | 0.113388 | 0.251 |
| clustered | 20,000 | 4 | 0.010459 | 0.036 |
| clustered | 20,000 | 8 | 0.081430 | 0.163 |
| clustered | 50,000 | 4 | 0.005039 | 0.016 |
| clustered | 50,000 | 8 | 0.058267 | 0.112 |

The kd-tree does not show the ball-tree blindness: it is never materially
slower than brute on this matrix, including the borderline unstructured 5k/8d
cell. The probe therefore remains ball-tree-only; no kd-tree wiring is added.

## Verification ladder tolerance

After wiring, the public `stats.KNearestNeighbors` issue-190 ladder was run
once per arm with `k=5`, `LeafSize=16`, and 1000 test rows on 2026-08-05.
Auto/best-manual ratios were:

| regime | 5k/16d | 5k/64d | 20k/16d | 20k/64d | 50k/16d | 50k/64d |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| unstructured | 1.203 | 1.054 | 1.185 | 1.136 | 1.375 | 1.386 |
| clustered | 1.145 | 1.011 | 1.079 | 1.169 | 1.084 | 1.097 |

The first full run peaked at 1.386. A repeat of the slowest unstructured
50k/16d cell measured 1.404 at the high-noise end, with three repeated runs
at 1.200, 1.342, and 1.304. A later full unstructured ladder run reached
1.414 at 20k/16d. The benchmark assertion uses the recorded tolerance
**1.45**, rounded above the observed 1.414 ceiling for host noise.
