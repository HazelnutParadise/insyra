# Tasks

## 1. Decoders

- [ ] 1.1 f16 and bf16 → f32 widening (subnormals, ±inf, NaN); SafeTensors F16/BF16 entries load; refusal list updated, quantized still named
- [ ] 1.2 ONNX FLOAT16/BFLOAT16 initializers decode; Cast to those targets produces the f32 widening; docs state graphs run in f32

## 2. Proof

- [ ] 2.1 Gated fixture: torch writes f16 and bf16 safetensors with normals, rounding cases, subnormals, ±inf, NaN; Go f32 values bit-exact vs torch's float() widening
- [ ] 2.2 Ungated unit tests: hand-built half bit patterns (including 0x7C00/0xFC00 inf, quiet NaN, smallest subnormal) widen to the exact expected f32 bits
- [ ] 2.3 One-op ONNX parity row with an f16 initializer vs onnxruntime; existing suites pass unchanged

## 3. Sync

- [ ] 3.1 Docs dtype table and widening contract; changelogs both languages; skills
