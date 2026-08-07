## 1. Implementation
- [x] 1.1 Define typed projection and cache requirements in `specs/accel-memory-cache/spec.md`
- [x] 1.2 Write `design.md` for null handling, string transport, residency, and budget rules
- [x] 1.3 Validate the change with `openspec validate add-accel-columnar-layout-cache --strict`
- [x] 1.4 Add lineage-aware session-local cache identity and aggregate budget enforcement without pretending projection buffers already have per-device residency
