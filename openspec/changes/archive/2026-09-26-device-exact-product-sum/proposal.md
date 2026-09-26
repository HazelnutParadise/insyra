# Proposal: device-exact-product-sum

## Why

M35 of #379. `ENG.md` defines a device result as the correctly rounded value of the exact operation and requires a device kernel to compute it in integers, because WGSL lets an implementation flush subnormals, ignore the sign of zero, reassociate and contract floating-point arithmetic, while its integer operations are exact. `EdgeSum` (M33) is now the exact sum of its products rounded once. Before a production kernel can be built on that (M36), the device needs the arithmetic itself: decoding float32 bit patterns, forming exact products, accumulating them exactly and rounding once, returning exactly the bits the CPU returns.

This is the device half of the Kulisch long accumulator that `nn` already uses on the CPU, the technique ExBLAS uses for reproducible sums on GPUs. It is built and verified on its own, so M36 builds on arithmetic already shown to match the CPU and is left to deal only with how the work is split and when it pays.

## What Changes

- A WGSL library in `accel/internal/wgpu` providing an exact register and four functions: clear it, add the exact product of two float32 bit patterns, merge another register into it, and round it once to a float32 bit pattern. It follows the CPU accumulator's design, adapted to 32-bit integers:
  - the register counts in units of 2^-298, as the CPU's does, and holds 27 digits of 24 bits in `u32`, so a product (48 bits at any offset) always spans three digits;
  - carries are deferred and propagated every 64 additions, which bounds every digit's true value below 2^31;
  - all arithmetic is unsigned. Unsigned arithmetic means the same in WGSL and in every language a WGSL compiler emits. Signed overflow and the right shift of a negative number are undefined or implementation-defined in C++, the base of Metal's shading language. naga guards the first and not necessarily the second, and the library should not depend on a translator's care. Negative contributions are stored in two's complement and carries are sign-extended by hand;
  - no floating-point type appears anywhere in the library. Inputs and outputs cross the boundary as `u32` bit patterns.
  - the library uses no `select()`. gogpu/naga v0.19 writes a scalar `select()` that is an operand of another expression as an unparenthesized ternary in Metal. The first device run caught it as a wrong carry, and the source test now refuses any `select(`.
- Special values follow `nn-edge-sum` exactly. A NaN operand, `0·∞`, or infinities of both signs make the result the NaN `0x7fc00000`. Otherwise an infinite product makes the result that infinity. An exact zero is `+0`, and a nonzero total that rounds to zero keeps its sign.
- A gated parity suite on the device compares every output bit pattern with `nn.EdgeSum` and with an independent `math/big` oracle. It covers random rows, the full float32 bit-pattern range, and adversarial rows: cancellation, ties, the subnormal and overflow boundaries, NaN, infinities, signed zeros, carries rippling through every digit, and rows long enough to force many carry passes. Rows are also split across two registers and merged, to show that a split sum returns the same bits.
- The library's throughput is measured once against the all-core CPU `EdgeSum` on the same products and recorded in `delivery-status.md`. This is not a gate. It is the number M36 starts from.

## Capabilities

### New Capabilities

- `accel-exact-arithmetic`: device-side exact arithmetic on float32 bit patterns, correctly rounded once, returning the CPU's bits.

### Modified Capabilities

(none)

## Impact

- A new `accel/internal/wgpu/exact_sum.go` (the library source) and test files beside it. No production path calls the library yet; M36 wires it.
- `ENG.md` records the register layout and the unsigned-only rule under "How a device kernel computes it".
- `delivery-status.md`: M35 done, the throughput data point, Next Ticket M36.
- Nothing user-visible changes, so no CHANGELOG entry.
