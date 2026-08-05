# Tasks: add-accel-device-selection

## 1. Mechanism

- [x] 1.1 Apply `INSYRA_ACCEL_DEVICES` (IDs or zero-based indices, comma-separated) as a filter at the discovery boundary, so masked devices are invisible to all downstream layers.
- [x] 1.2 Add `Config.Devices` and compute the eligible set as env ∩ config; thread it through session construction so selection and planning only ever see eligible devices; `PreferredDevices` keeps ordering within the set.
- [x] 1.3 Empty-eligible-set behavior: error under strict modes naming the emptying bound; CPU fallback with a new dedicated `FallbackReason` under automatic modes. Unmatched entries appear in the session report.

## 2. Verification

- [x] 2.1 Stub-probe unit tests covering all five spec scenarios (mask hides, allowlist pins, strict×empty errors, auto×empty falls back with reason, unmatched entry surfaced), plus intersection and index-form parsing edges.
- [x] 2.2 Confirm defaults change nothing: with no env var and empty `Config.Devices`, the existing accel suite passes unmodified.
- [x] 2.3 Full `go test ./accel/...`.

## 3. Docs, changelog, bookkeeping

- [x] 3.1 `Docs/accel.md`: the three-axis table (mode × device bounds × preference) with the intersection rule and the ID-vs-index portability note. Changelog entries in both `CHANGELOG.md` and `CHANGELOG_TW.md`.
- [x] 3.2 `delivery-status.md` delta; `openspec validate add-accel-device-selection --strict` passes.
