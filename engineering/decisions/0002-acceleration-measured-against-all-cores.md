# 0002 — Acceleration is measured against every core, in the call site's direction

**Status:** accepted
**Date:** 2026-07-29

## Context
`accel` contained no parallelism, while the rest of the library parallelises its hot loops. Every speedup recorded before 2026-07-29 compared a GPU against one core of an eight-core machine.

## Decision
Every acceleration claim is measured against the host using all its cores, with upload, dispatch and readback counted, and in the direction the call site actually drives the kernel. When the CPU path is a serial loop, it is made parallel first and the device is measured against that.

## Options rejected
- Compare against the existing single-threaded path. Re-measured honestly, the best device case fell from 7.1x to 3.63x over 96 shapes.

## Consequences
- The KNN wiring was first measured with the dataset as the training side and won every shape up to 4.1x. Measured the way the call site runs it, with the test rows as the parallel side, the device loses below about 2,048 test rows (469 ms against the CPU's 324 ms at 1k rows). The floors in `/engineering/architecture.md` come from the second measurement.
- `nn`'s device milestone waited for a parallel CPU baseline (2026-08-02): MatMul was 98% of an encoder layer and Conv 98% of a CNN forward, both single-threaded loops. Output-element parallelism keeps each element's accumulation order, so that win is bit-identical; it brought the encoder layer from 3.35 s to about 0.9 s on the M3.
- `EdgeSum` was given the same treatment before its device measurement (2026-09-25): all cores against one, 2.1x to 3.8x, bound by gathering from scattered addresses.
