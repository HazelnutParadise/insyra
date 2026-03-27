## 1. Implementation
- [x] 1.1 Define backend discovery requirements and scenarios in `specs/accel-device-discovery/spec.md`
- [x] 1.2 Write `design.md` for probe order, normalized device metadata, and default policy
- [x] 1.3 Validate the change with `openspec validate add-accel-backend-discovery --strict`
- [x] 1.4 Add a discoverer registry and session-level auto-discovery hook in `accel`
- [x] 1.5 Add normalized device scoring and primary-device selection helpers
- [x] 1.6 Add package tests covering open-time discovery, CPU-mode bypass, and primary-device selection
- [x] 1.7 Implement builtin CUDA, Metal, and WebGPU discoverer stubs behind the new discovery contract
- [x] 1.8 Honor `DiscoveryTimeout` across session-level discoverers and native probe commands
- [x] 1.9 Add shared-memory budget fallback and cross-backend dedupe so portable/native inventory stays credible when native budget data is missing
