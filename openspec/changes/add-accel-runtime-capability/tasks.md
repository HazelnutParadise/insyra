## 1. Implementation
- [x] 1.1 Define the public runtime API shape in `specs/accel-runtime/spec.md`
- [x] 1.2 Write `design.md` covering package boundaries, ownership, and non-goals
- [x] 1.3 Validate the change with `openspec validate add-accel-runtime-capability --strict`
- [x] 1.4 Add execution wrappers and execution-result surface so projected datasets can produce truthful residency/report events before backend-native kernels exist
- [x] 1.5 Add a backend allocator registry and ledger fallback seam so future backend allocators can plug into runtime execution without rewriting the execution surface
