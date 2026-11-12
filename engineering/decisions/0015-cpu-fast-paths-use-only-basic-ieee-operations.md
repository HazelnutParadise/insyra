# 0015 — A CPU fast path behind correct rounding uses only IEEE 754's basic operations

**Status:** accepted
**Date:** 2026-09-26

## Context
Record 0014 makes `nn.Tanh` correctly rounded on every platform. An exhaustive check over all 2^32 float32 inputs on one machine proves a function only for platforms that compute the same bits. `math.Tanh` fuses multiply-adds on arm64, picks an FMA path at run time on amd64 (`math.Exp`) and is assembly on s390x. The adversarial review of the first version caught this.

## Decision
A CPU fast path behind a correctly rounded function uses only IEEE 754 `+ − × ÷`, with an explicit conversion around every product that feeds an addition or subtraction, and calls no `math` function. Go's specification forbids fusing a product across such a conversion, and gc performs no other floating-point fusion. A result the fast path cannot settle within a margin above its measured error comes from a table, with `math/big` behind the table. A CI test hashes the fast path's bits over its whole input range, so every platform CI runs confirms them.

`nn.Tanh` is the pattern. `|x| < 2^-13` and `|x| >= 9.5` are settled by proof. Between them `tanhFloat64` is accepted when its rounding lies more than 16 float64 steps from every float32 midpoint, and the four inputs inside that margin are a table. `Tape.Tanh`'s gradient rounds every step and fuses none.

## Options rejected
- `math.Tanh` as the fast path, for the platform reasons above.

## Consequences
- On darwin/arm64 `tanhFloat64`'s largest error over every input is 2.63 float64 steps. All 2^32 inputs were checked with 0 mismatches: the 271,581,184 between the analytic ranges against an independent `math/big` oracle anchored to 60-digit values from Python's `decimal`, the rest against their proven values.
- The hash test's first CI run gave the arm64 bits on linux/amd64 and windows/amd64 too. No forward result changed on those platforms or on arm64. The gradient changed 24,892 of 100,000 random pairs on arm64, which had fused `1 - y*y`; amd64 already produced these bits.
- Cost on one core, 2^20 inputs: normal(0, 2) activations 8.3 ns before, 7.5 ns after; random bit patterns 7.6 ns before, 5.5 ns after.
