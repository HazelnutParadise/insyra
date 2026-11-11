# 0010 — The performance gap to ONNX Runtime is stated, not chased

**Status:** accepted
**Date:** 2026-08-03

## Context
Measured at the validated real-model shapes on the M3, best of 5, identical inputs: MobileNetV2 batch 1 takes 170 ms in `nn` against 7.3 ms in ONNX Runtime (about 23x), MiniLM batch 8 by sequence 128 takes 3.34 s against 63 ms (about 53x). Disabling the device path shows the gap is CPU kernel throughput: ONNX Runtime's MLAS runs hand-written per-architecture assembly at about 250 GFLOP/s, pure Go reaches about 4 to 8, and the Go compiler does not vectorise automatically.

## Decision
`nn` stays pure Go. `Docs/nn.md` states the measured numbers and what they imply.

## Options rejected
- An assembly GEMM microkernel. It is the one lever that would move every model, and it would give up the portability and auditability the package exists for. It stays unbuilt until a real workload demands it.

## Consequences
A user who needs ONNX Runtime's speed is told so in the documentation rather than discovering it.
