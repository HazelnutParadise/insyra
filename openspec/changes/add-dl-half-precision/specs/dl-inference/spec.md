## ADDED Requirements

### Requirement: Half precision loads as exact f32 widening

`dl` SHALL load `F16` and `BF16` tensors from SafeTensors files and
FLOAT16/BFLOAT16 initializers from ONNX files by widening every value
exactly into f32 — preserving subnormals, signed infinities, and NaN —
because every half-precision value is exactly representable in f32.
Compute remains f32; half-precision arithmetic SHALL NOT be claimed.
Quantized dtypes SHALL remain refused by name.

#### Scenario: SafeTensors halves widen bit-exactly

- **WHEN** f16 and bf16 files covering normals, rounded values,
  subnormals, ±inf, and NaN are loaded
- **THEN** every f32 value SHALL equal the reference widening
  bit-exactly

#### Scenario: ONNX half initializers decode

- **WHEN** a graph carries FLOAT16 or BFLOAT16 initializers
- **THEN** they SHALL load as exact f32 widenings and the graph SHALL
  execute in f32, matching `onnxruntime` within f32 tolerance on a
  one-op parity row

#### Scenario: Quantized stays refused

- **WHEN** a file carries a quantized dtype
- **THEN** loading SHALL refuse naming the tensor and dtype
