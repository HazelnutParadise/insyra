# 0001 — Whether a device may be used follows the shape of the result

**Status:** accepted
**Date:** 2026-07-28

## Context
Acceleration is meant to be on by default and to change no numbers. The hottest operations in the library, CCL value expressions among them, produce new `float64` values, and WebGPU has no `f64`.

## Decision
Whether an operation may run on a device by default is decided by the shape of its result, not by how hot the operation is. A selection (which row, which index, what order) may: the device ranks in `f32` and proposes, and the host recomputes the shortlist in `float64` and decides, so the answer is exact. Values in a type the device holds exactly may. New `float64` values are opt-in only, behind an explicit precision request. The table and the two rules that follow from it are in `AGENTS.md` under "How the CPU and the GPU Divide the Work".

## Options rejected
- Accelerate what is hottest. CCL value expressions were measured at 11x and 146x, but nothing verifies a new value more cheaply than recomputing it, so a device result would change numbers.
- Defer string kernels to a later phase. Withdrawn on 2026-08-01: a string kernel produces new values, which this rule places behind an opt-in in any phase, and the phase it was deferred to no longer exists.

## Consequences
- CCL sits on both sides of the line: its predicates are selections and qualify, its value expressions do not.
- A missing or broken device is a performance event, never a correctness one, because the host half of a selection is a complete implementation.
- The 11x CCL comparison was a single-core interpreter result measured with goroutine-ID guards that `thread-ccl-eval-depth` later removed (2026-08-05: 15.5 ms to 3.4 ms at 10k rows, 143.7 ms to 23.9 ms at 100k), so it is no baseline for any future question about CCL on a device.
