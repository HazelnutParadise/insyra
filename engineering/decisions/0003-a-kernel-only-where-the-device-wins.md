# 0003 — A device kernel exists only where measurement shows the device wins

**Status:** accepted
**Date:** 2026-07-29

## Context
Four device operations existed. Measured against all cores (record 0002), three did not pay: a column sum at 0.7x, a distance matrix whose readback grew with the answer, and an `f32` nearest query no `float64` caller could use. None had shipped.

## Decision
Every accelerated operation except exact-nearest was removed, and a new kernel is proposed only after a measurement shows the device winning at the shapes a caller uses. Memory-bound work, such as column sums, means and simple scans, stays on the CPU permanently.

## Options rejected
- Keep the losing operations. Each meant a kernel to maintain against a backend with essentially one maintainer, a method to document, and a changelog entry claiming a speedup measured against one core.

## Consequences
The measurement is recorded before the kernel is proposed, in a decision record or in the change that proposes the kernel. A kernel that exists for a reason other than speed, such as keeping a resident computation on the device, says so and claims no speedup (records 0016 and 0019).
