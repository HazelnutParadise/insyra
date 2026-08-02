# Change: dl reads SafeTensors files

## Why

M18's verification signal is first-step gradients matching PyTorch under
fixed initial weights loaded via SafeTensors. That makes the SafeTensors
reader the first slice of the training milestone: without it, no weight can
be fixed on both sides of the comparison. It is also independently useful —
SafeTensors is the dominant weight-exchange format on model hubs, and the
decided GGUF/LLM future track benefits from dl reading real checkpoints.

The format is small and fully specified: an 8-byte little-endian header
length, a JSON header mapping tensor names to dtype/shape/byte-offsets, and
a single raw data region. Like the ONNX loader, it is untrusted input: a
malformed file must produce an error naming what is wrong, never a panic,
and unsupported dtypes must be reported all at once.

## What Changes

- `dl.LoadSafeTensors(io.Reader)` returning named tensors. F32 loads
  natively. F64, F16, BF16, and integer dtypes the Tensor type already
  carries load with explicit, documented conversion or are refused naming
  the dtype — following the Tensor dtype contract, not silently widening.
- Validation at load time: header length bounds, JSON well-formedness,
  offset/length consistency against the data region, dtype/shape element
  count agreement, overlapping or out-of-range regions refused. Every
  refusal names the tensor.
- A reference round-trip test: fixtures written by the Python `safetensors`
  library (present in the crosslang venv) are read by dl and every value
  compared exactly; gated through `internal/reftest` like the other
  reference toolchains, with the pure-Go structural tests ungated.
- Docs, changelogs both languages, skills — same change.

## Non-Goals

- No autodiff, no optimizers, no training loop (the next M18 slice).
- No writing SafeTensors (read-only until something needs to write).
- No quantized dtypes beyond refusal-with-name (the GGUF track owns those).
- No mmap or lazy loading; whole-file reads are enough for the harness.

## Impact

- Affected specs: `dl-inference`
- Affected code: new `dl/safetensors.go` (+ tests), `dl/testdata` fixture
  generator, docs, changelogs, skills.
