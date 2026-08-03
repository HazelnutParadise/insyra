# Change: Half-precision checkpoints load, value-exactly

## Why

Industry checkpoints increasingly ship f16 or bf16. dl refuses both
today — the honest placeholder until a storage-vs-compute design was
decided. The design is now decided (M23): **half precision is a
storage dtype; compute stays f32.** The deciding fact is that
f16→f32 and bf16→f32 widening is value-exact — every half-precision
value is exactly representable in f32 — so loading requires no
tolerance decision at all, and the SafeTensors rule against silent
widening becomes deliberate, documented, exact widening.

## What Changes

- `LoadSafeTensors` accepts `F16` and `BF16` entries, decoding them
  into f32 tensors: f16 per IEEE 754 binary16 (sign/5-bit
  exponent/10-bit mantissa, subnormals, ±inf, NaN preserved), bf16 as
  the top 16 bits of a binary32. The conversion is documented on the
  loader as exact widening; the refusal list shrinks accordingly, and
  quantized dtypes remain refused by name.
- ONNX initializers of FLOAT16 (10) and BFLOAT16 (16) decode the same
  way, and `Cast` accepts those dtype targets by producing the f32
  widening (graphs that compute in half run in f32 — documented; full
  half-precision execution is out of scope and stays refused where it
  cannot be represented).
- The proof: a fixture (torch in the venv) writes f16 and bf16
  safetensors covering normals, values that round at half precision,
  subnormals, ±inf, and NaN; the Go loader's f32 values are compared
  BIT-EXACT against torch's own float() widening. An ONNX one-op
  parity row with an f16 initializer pins the decoder path against
  onnxruntime.
- Docs (dtype table and the widening contract), changelogs both
  languages, skills — same change.

## Non-Goals

- No half-precision arithmetic, storage, or outputs; no quantized
  dtypes (GGUF track); no fp16 whole-model execution parity (a real
  fp16 graph runs in f32 here, which is a different numeric claim than
  ORT-in-fp16 and is documented as such).

## Impact

- Affected specs: `dl-inference`
- Affected code: dl safetensors and ONNX decoders, Cast, tests, docs,
  changelogs, skills.
