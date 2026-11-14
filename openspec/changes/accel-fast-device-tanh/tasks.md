## 1. Implementation

- [x] 1.1 `tanh_fast` in WGSL with its constants and a host model; `tanh_rounded` calls it first

## 2. Verification

- [x] 2.1 The model's error bound against a 600-bit oracle and its fallback count over every positive input
- [x] 2.2 The device equals `nn.Tanh` on all 2^32 inputs on the M3 and the GT 1030
- [x] 2.3 The device `tanh` and `TestRecurrentStepTiming` re-measured on the M3

## 3. Docs

- [x] 3.1 Both changelogs, the `AGENTS.md` follow-up resolved, the measurement in record 0030
