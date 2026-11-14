# Handoff

Unfinished work that has no ticket. Delete a section when its work is done, and the file when it is empty.

## Kernel warmup on a cold shader cache
- **Asked:** `accel-kernel-warmup` (#624) compiles the 28 built-in kernels in the background when the first resident buffer is created, so the first training step does not wait for them.
- **Done and checked:** with a warm driver cache, first steps equal `dev`'s on the M3 (12 runs each) and the GT 1030 (10 runs each). Caller kernels are not warmed.
- **Tried and failed:** nothing yet.
- **Next:** clear the GT 1030 driver's shader cache and time the first step with and without warmup. A shorter first step with warmup confirms the change; an equal one means the driver cache was doing the work.
- **Sources:** [accel-kernel-warmup](/openspec/changes/archive/2026-11-01-accel-kernel-warmup/proposal.md).
