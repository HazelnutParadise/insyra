# Tasks

## 1. Fix start selection
- [x] 1.1 Skip Promax / TargetRot starts for orthogonal methods (varimax, quartimax, geomint, bentlert)
- [x] 1.2 Cap total starts at `Restarts` (heuristics + random)
- [x] 1.3 QR-orthonormalize starts for orthogonal methods

## 2. Tests
- [x] 2.1 Add package test that fails before the fix for Quartimax/Varimax/GeominT/BentlerT with Restarts > 1
- [x] 2.2 Confirm communalities stay stable and `R'R ≈ I`

## 3. Docs
- [x] 3.1 Clarify Restarts start rules in `Docs/stats.md`
- [x] 3.2 Add Unreleased entries to `CHANGELOG.md` and `CHANGELOG_TW.md`
