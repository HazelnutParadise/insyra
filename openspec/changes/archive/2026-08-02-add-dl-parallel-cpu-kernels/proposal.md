# Change: Parallelize dl's MatMul and Conv on the CPU, bit-identically

## Why

Measured on an 8-core M3 (best of 5, realistic shapes), the two kernels that
dominate every dl workload are single-threaded naive loops:

- Encoder layer (batch 32, seq 128, hidden 256, heads 4, FFN 1024): 3.37 s
  total, of which plain MatMul is 89.7% (QKV/output projections 29.5%, FFN up
  29.9%, FFN down 30.3%) and batched attention MatMul another 8.4%. Everything
  else — softmax, gelu, layernorm, residuals — is 2% combined.
- MNIST-class CNN forward (batch 64): 530 ms total, of which Conv is 98.4%
  (the 16→32 3×3 layer alone is 95.3%).

M17 (device inference) may not be scoped from this baseline. The project has
already withdrawn one round of acceleration claims for comparing a device
against one core of an eight-core machine; a GPU ticket cut against a
single-threaded naive loop would repeat that mistake with a larger multiplier.
The honest CPU baseline comes first, and it is a large win on its own: the
work is embarrassingly parallel over independent output elements.

## What Changes

- `MatMul` (the 2-D fast path and the batched path) and `Conv` compute their
  independent output elements in parallel across CPU cores.
- The accumulation order **within each output element is unchanged**, so every
  output is bit-identical to the serial result. Parallelism partitions outputs;
  it must not restructure any inner reduction. K-blocking, SIMD-lane
  reduction, and any other transform that reorders a sum is out of scope.
- Small inputs stay on the serial path behind a measured size threshold, so
  tiny graphs do not pay goroutine overhead.
- No API change. No new dependency. The rest of the operator family is
  untouched — measurement says it does not matter yet.

## Non-Goals

- No device kernels (that is M17, scoped after this lands).
- No assembly, no SIMD intrinsics, no cgo.
- No parallelism in the other kernels; below ~2% of wall time each, the
  complexity is not earned.

## Impact

- Affected specs: `dl-inference`
- Affected code: `dl/kernels.go` (MatMul, matMul2D, Conv), tests.
- Expected outcome on 8 cores: ≥4x wall-time drop on the measured encoder
  layer and CNN forward, with byte-for-byte identical outputs.
