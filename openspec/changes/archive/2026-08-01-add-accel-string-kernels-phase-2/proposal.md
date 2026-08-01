# Change: Separate full string kernels into Phase 2 — WITHDRAWN

## Status
Withdrawn on 2026-08-01, never implemented. Kept as a record of the boundary it drew and why that boundary no longer means anything.

## What it proposed
A planning boundary: declare full GPU string kernels a Phase 2 track so they could not derail Phase 1 runtime convergence, while Phase 1 kept encoded-string transport and key-based eligibility.

## Why it is withdrawn
Both halves of the premise are gone.

**Phase 2 no longer exists as a deferral target.** The boundary was drawn to protect a Phase 1 that would hand off to a Phase 2 doing the harder work. Phase 1 finished, and the conclusion was that no call site in the library was measured to profit from a device — the accelerated operations that could be measured were removed, and the one that survives has no caller clearing its threshold. There is nothing for a Phase 2 to receive.

**And the rule that emerged would exclude string kernels anyway.** Whether an operation may be accelerated is now decided by the shape of its result: a selection may, because a device can propose and the host decide; values the device holds exactly may; new values may not, because nothing verifies them more cheaply than recomputing them. A string kernel produces new strings or new values. It falls in the third category regardless of how well it is written, so deferring it to a later phase describes a phase that could not accept it.

Deferring something to a future that has been ruled out is not a plan, and leaving the change open would report pending work that nobody intends to do.

## What survives
The encoded-string transport itself is still in the tree — `ProjectDataList` builds string buffers with offsets and data, and the cache accounts for their bytes. Nothing consumes them: the sole remaining operation requires numeric columns. That is dead weight rather than a defect, and removing it is a separate question from withdrawing this change, since it would be needed again by any future string operation.

## Impact
- No specs added; the `accel-string-kernels` capability this would have created does not exist and will not
- No code was written
