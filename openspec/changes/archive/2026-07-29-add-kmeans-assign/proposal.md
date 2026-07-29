# Change: Let a fitted KMeans assign rows it was not fitted on

## Why
`KMeansResult` returns the cluster of every training row and the centres it converged on (`stats/clustering.go:19`), and no way to use the second on anything but the first. There is no method on the result at all — a caller holding a fitted model and a new observation has to reimplement the distance loop.

That is a gap on its own terms: assigning new points to fitted centres is what people do with a clustering model after they have one, and it is one line of arithmetic away from what the result already holds.

It also happens to be the only place in `stats` where a device may run by default. The acceleration rules admit an operation when its result is a selection — which row, which index, what order — because the device can rank in single precision and the host settle it in `float64`. Assigning a row to the nearest centre is exactly that shape, and `accel.ExecuteNearestExact` already implements it, returning the `float64` answer whether or not a device took part. Every other algorithm in `stats` either produces new `float64` values, which cannot be accelerated without changing its numbers, or has already been reduced by an algorithm a device cannot beat.

This change does not wire the device up. It adds the operation the device would serve, on the CPU, with the same result either way — so that when a device is worth using there is something for it to accelerate, and until then nothing behaves differently.

## What Changes
- Add an assignment method to the fitted KMeans result, taking observations and returning the nearest centre for each
- Return the distance alongside the index, since a caller judging whether a point belongs to any cluster needs it
- Break ties by the lowest centre index, matching every other selection in the codebase
- Refuse observations whose column count does not match the fitted centres
- Compute on the CPU in `float64`, and shape the implementation so a device path can be added later without changing the answer or the signature

## Impact
- Affected specs: `stats-clustering`
- Affected code: `stats/clustering.go`
- Additive: a new method on an existing result type, no signature changes
