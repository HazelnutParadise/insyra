## Context

`runRBaseline` calls `Rscript stats/testdata/crosslang_baseline.R factor_analysis <json>` and caches the output under a key built from the script's content, the toolchain signature, the method and the payload. `factor_analysis_stats` calls `psych::principal` for PCA and `psych::fa` otherwise, each once rotated and once unrotated. Only `psych::fa` with a rotation draws random numbers (`n.rotations = 20`); `kmeans_stats` in the same script already seeds explicitly. See proposal.md for the measured run-to-run differences.

## Goals / Non-Goals

**Goals:**
- A factor analysis baseline is a function of its payload and toolchain only.

**Non-Goals:**
- Working around psych's `faRotations` tie-break. Patching psych inside the script would make the reference something other than what psych users get; the defect is recorded instead.
- Changing how the parity suite compares factor-frame fields. That is a separate change, and it will be measured against the seeded baselines this change produces.

## Decisions

- **One fixed seed, set at the top of `factor_analysis_stats`.** Every call then starts from the same generator state regardless of what the session ran before. A seed derived from the payload would also be reproducible, but adds nothing: the payload is already part of the cache key.
- **Test by running the script twice without the cache.** Comparing stdout byte for byte is the property the cache relies on. It is gated like the other factor analysis R tests, because it needs psych.

## Risks / Trade-offs

- [Every R baseline regenerates once] → The cache key includes the script's content, so editing the script invalidates all of it; a full regeneration took about twelve minutes on 2026-09-13.
- [The seeded reference may land at a different point of a flat minimum than today's cache] → Expected and measured: the strict suite is rerun and its change reported.
