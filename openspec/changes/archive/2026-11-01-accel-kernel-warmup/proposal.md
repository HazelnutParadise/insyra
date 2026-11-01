# Proposal: accel-kernel-warmup

## Why

[#584](https://github.com/HazelnutParadise/insyra/issues/584): each built-in GPU kernel was compiled on the call that first needed it, so the first `nn` step on a device waited for every kernel it touched (40 ms to 1.4 s each on an M3, up to 1.8 s on a GT 1030 with a cold driver cache).

## What Changes

- The first resident buffer made on a device starts a goroutine that compiles the 28 built-in kernels, most-used first.
- It compiles without `submitMu`, so computations, uploads and readbacks do not queue behind it, and takes `submitMu` only to install a result through the kernel's `sync.Once`; a result a computation beat it to is released.
- A kernel compiled but not installed when the device is lost is released with the device's other children, and a warmup whose device was released compiles nothing more.
- Caller kernels, results and public names are unchanged.
