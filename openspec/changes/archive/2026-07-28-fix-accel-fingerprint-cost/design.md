## Context
The fingerprint is the identity half of the accel cache. `cacheKey` is `fingerprint + lineage + index + buffer name` (`accel/cache.go:248`), and `ensureDatasetCached` uses the presence of the first key to decide whether a dataset has already been projected. Fingerprints are session-local: nothing writes one to disk, and `cli/env` persists only `accelMode`. That means the hash algorithm can change freely — there is no stored value to stay compatible with.

Two separate costs are in play, and both are on the same line:

- `fmt.Sprintf("%v", buffer.Values)` allocates one string holding the decimal rendering of the whole column. A 4 Mi `float64` column is 32 MB of data and roughly 80 MB of text.
- `fnv.New64a` hashes one byte at a time with a multiply per byte, so it then walks all 80 MB.

## Goals / Non-Goals
- Goals:
  - make fingerprint cost proportional to the bytes in the column
  - keep the fingerprint content-addressed over every value
  - leave the cache's observable behavior unchanged
- Non-Goals:
  - changing what the cache stores or when it evicts
  - changing `cacheKey`'s shape or the `Dataset.Fingerprint` field
  - making the fingerprint stable across processes, architectures, or releases; it never was, and nothing depends on it

## Decisions

- Decision: hash the raw bytes of typed value slices.
  - Rationale: the values are already contiguous typed slices (`[]float64`, `[]int64`, `[]bool`, `[]string`), so their bytes are available without rendering them to text. This removes the 80 MB allocation entirely and cuts what the hash has to walk from ~80 MB to 32 MB.
  - Side effect, accepted: raw bits distinguish `-0.0` from `0.0` and distinguish NaN payloads that `%v` renders identically as `NaN`. The fingerprint becomes stricter, never looser, so this cannot produce a false cache hit.

- Decision: encode into a fixed scratch buffer rather than aliasing the slice with `unsafe`.
  - Rationale: `unsafe.Slice` over the value slice would save roughly 4 ms on a 4 Mi column, which is not worth putting pointer aliasing into a core package for. `binary.LittleEndian.PutUint64` into a reused 32 KiB buffer, flushed to the hasher as it fills, keeps the code ordinary Go and allocates one buffer per call regardless of column size.

- Decision: use `github.com/cespare/xxhash/v2` for the value payload instead of FNV-64a.
  - Rationale: FNV-1a is a per-byte multiply, which is the wrong shape for tens of megabytes. xxhash processes a word at a time and is built for exactly this. It is already in the module graph as an indirect dependency, so this promotes an existing dependency rather than adding a new one, and there is no new download or supply-chain surface.
  - The metadata parts of the fingerprint (names, types, lengths, validity) move to the same hasher, so there is one hash per dataset rather than two.

- Decision: keep hashing every value; no sampling, no length-only shortcut.
  - Rationale: today a collision only mis-reports residency, because `CacheEntry` holds metadata and not values. That changes the moment device buffers are keyed off the fingerprint, which is the direction the runtime is already heading. Trading correctness for another few milliseconds would plant a defect that surfaces as wrong numbers later, which is precisely the class of problem the accel work has been digging itself out of.

## Value encoding

| Column type | Encoded as |
| --- | --- |
| `[]float64` | `math.Float64bits` per value, 8 bytes little-endian |
| `[]int64` | 8 bytes little-endian |
| `[]bool` | one byte, `0` or `1` |
| `[]string` | existing offsets-and-data path when present, otherwise length-prefixed bytes per value |
| `[]any` | per-element `%v`, unchanged — this is the untyped fallback and is not on the hot path |

Length-prefixing the string case matters: hashing concatenated strings without separators would give `["ab","c"]` and `["a","bc"]` the same fingerprint.

## Measured

Apple M3, 4 Mi `float64` column (32 MB).

| | before | after | change |
| --- | --- | --- | --- |
| `BenchmarkDatasetFingerprint` | 255 ms/op, 132 MB/s | 14.2 ms/op, 2355 MB/s | 17.9x faster |
| `BenchmarkProjectionOnly` (`ProjectDataList`) | 357 ms/op | 145 ms/op | 2.5x faster |
| `BenchmarkColumnSum/4Mi/gpu` end to end | 354 ms/op | 117 ms/op | 3.0x faster |

The fingerprint is no longer the problem, and the delivery plan's stated target — an order of magnitude off `BenchmarkProjectionOnly` — is **not** met by this change alone. `BenchmarkProjectValues` attributes the remainder: the typed-projection loop itself costs 94 ms of the 145 ms, against the fingerprint's 14 ms.

That remaining cost is structural rather than a defect. `insyra.DataList` stores `[]any`, so projecting 4 Mi values means unboxing 4 Mi interfaces and running `toFloat64` on each, plus building the nulls slice and validity bitmap. Removing it means either projecting from a typed representation the core does not currently keep, or making projection incremental. Recorded under `## Follow-ups` in `AGENTS.md`; it is a different capability slice and does not belong in this change.

## Risks / Trade-offs
- Risk: promoting `cespare/xxhash/v2` to a direct dependency.
  - Mitigation: it is already in `go.sum` and already compiled into the binary through an existing indirect path, so nothing new enters the build.
- Risk: fingerprints computed by the new code differ from the old ones.
  - Not a problem: they are session-local and never persisted or compared across processes. Tests that assert a specific hex value would need updating; tests that assert stability and difference will not.
- Trade-off: `[]any` columns keep the slow per-element formatting path.
  - Accepted: `projectValues` only produces `[]any` for genuinely mixed columns, which are not GPU-eligible anyway.
