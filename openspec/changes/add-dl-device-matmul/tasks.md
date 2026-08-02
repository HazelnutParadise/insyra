# Tasks

## 1. The seam

- [x] 1.1 `dl`: package-level device-matmul hook (nil default), consulted from the 2-D matmul path only, at/above a floor constant; batched and sub-floor shapes never consult it
- [x] 1.2 `accel`: promote the prototype WGSL matmul to a production internal path, preserving k-serial per-output accumulation
- [x] 1.3 `accel/dlbridge`: opt-in blank-import package registering the device implementation, mirroring `accel/knnbridge`'s structure, with observable fallback and strict-mode failure

## 2. The floor

- [x] 2.1 Bisect the CPU/device crossover with the measurement harness between ~1M and ~268M MACs; record the floor as a measured, commented constant used by 1.1

## 3. Proof

- [x] 3.1 Hardware test: device result bit-equal (==) to CPU for shapes above the floor, gated on `INSYRA_ACCEL_GPU_TESTS=1`
- [x] 3.2 Fallback tests: nil hook, device-error, and sub-floor/batched shapes all produce the CPU result; strict GPU mode fails
- [x] 3.3 All dl suites (unit, parity, whole-model) pass with the bridge blank-imported on hardware
- [x] 3.4 End-to-end: measured encoder layer with bridge active recorded against the M19 CPU number in `delivery-status.md`

## 4. New-package contract and sync

- [x] 4.1 `Docs/dlbridge.md` (or section in Docs/accel.md + Docs/dl.md — follow how knnbridge documented itself), both README package tables if a new public package is added, docs index, allpkgs registration
- [x] 4.2 Changelog entries both languages; skills updated (insyra skill notes the opt-in device path)
