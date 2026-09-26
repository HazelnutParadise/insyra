# accel-exact-arithmetic Specification

## Purpose
Exact arithmetic for device kernels. Float32 values cross into and out of a kernel as bit patterns, products and sums are formed exactly in unsigned integers, and the result is rounded once to the nearest float32. A device result therefore equals the CPU's bit for bit whatever the hardware, the shader compiler or the way the work is split.

## Requirements

### Requirement: A device sum of products is the CPU's sum

The device library SHALL compute, for any list of float32 operand pairs, the exact sum of their products rounded once to the nearest float32 with ties to even, under the rules `nn-edge-sum` gives for `EdgeSum`. Those rules cover subnormal results, overflow to infinity, a NaN operand, `0·∞`, infinities of both signs, `+0` for an exact zero, and the sign of a nonzero total that rounds to zero. Every NaN result SHALL be the bit pattern `0x7fc00000`.

#### Scenario: Random rows
- **WHEN** the library sums rows of random operand pairs on a device, drawn both from moderate magnitudes and from every float32 bit pattern
- **THEN** every output bit pattern equals `nn.EdgeSum` on the same products and the `math/big` exact sum rounded to float32

#### Scenario: Rows built to break an implementation
- **WHEN** a row cancels to a value far below its terms, lies exactly half-way between two float32 values, straddles the smallest normal or the largest finite float32, holds NaN, infinities, `0·∞` or signed zeros, carries through every digit of the register, or is long enough to force many carry passes
- **THEN** every output bit pattern equals `nn.EdgeSum` and the oracle

### Requirement: A sum split across registers merges to the same bits

Products accumulated in separate registers and merged SHALL round to the same bits as the same products accumulated in one register, whatever the split and the order.

#### Scenario: Two registers per row
- **WHEN** each row's products are divided between two registers in interleaved order, the registers are merged and the result is rounded
- **THEN** the result equals the single-register result and the oracle for every row

### Requirement: The library holds no floating-point value

The library SHALL use no floating-point type and no signed integer type. Float32 inputs and outputs SHALL cross it as `u32` bit patterns, and every digit's true value SHALL stay inside the range its bounds are stated for, so the arithmetic never relies on wrapping.

#### Scenario: The source is inspected
- **WHEN** the library source is scanned for `f16`, `f32`, `f64` and `i32`
- **THEN** none appears
