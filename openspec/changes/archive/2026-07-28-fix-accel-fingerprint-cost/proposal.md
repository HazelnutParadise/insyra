# Change: Make dataset fingerprinting proportional to the data, not to its text

## Why
`datasetFingerprint` renders every value in every column with `fmt.Sprintf("%v", buffer.Values)` and hashes the resulting string with FNV-64a, a byte-at-a-time hash (`accel/cache.go:332`). Both halves scale badly: a 4 Mi `float64` column becomes roughly 80 MB of decimal text, and then every one of those bytes goes through a multiply.

The cost is not theoretical. Measured on an Apple M3, `session.ProjectDataList` on a 4 Mi `float64` column takes 357 ms. The GPU work for the same column is 4.6 ms. Projection is roughly seventy times the device cost, which makes every GPU-versus-CPU number the accel runtime can report meaningless — `add-accel-gpu-execution` produced a cost table that measures the wrong thing until this is fixed.

## What Changes
- Hash the raw bytes of typed column values instead of their decimal rendering
- Replace FNV-64a with a bulk-oriented hash for the value payload
- Keep content addressing exactly as strict as it is today: every value still contributes to the fingerprint, and no sampling or length-only shortcut is introduced
- Add a benchmark so the cost of fingerprinting is a number in the repository rather than a claim

## Impact
- Affected specs: `accel-memory-cache`
- Affected code: `accel/cache.go`, `go.mod` (promotes `github.com/cespare/xxhash/v2` from indirect to direct)
