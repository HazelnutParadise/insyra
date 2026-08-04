# Tasks

## 1. Reader

- [x] 1.1 `dl.LoadSafeTensors(io.Reader)`: header-length bounds, JSON header parsing, offset/length/shape/dtype validation, named refusals, unsupported dtypes listed together
- [x] 1.2 Dtype handling per the Tensor contract: F32 native; other dtypes either load into the dtypes Tensor already carries (documented) or refuse naming the dtype — no silent widening

## 2. Proof

- [x] 2.1 Pure-Go structural tests (ungated): truncated header, oversized header length, bad JSON, overlapping regions, out-of-range offsets, element-count mismatch, `__metadata__` tolerated
- [x] 2.2 Reference round-trip (gated via `internal/reftest`): fixtures written by the venv's Python `safetensors`, every value compared exactly, mixed shapes and at least two dtypes

## 3. Sync

- [x] 3.1 Docs/dl.md section; changelog entries both languages; skills note SafeTensors loading
