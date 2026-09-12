# Change: Keep orthogonal FA rotations orthogonal when Restarts > 1

## Why

`FactorAnalysis` orthogonal rotations (Varimax / Quartimax / GeominT / BentlerT) with `Rotation.Restarts > 1` can return a non-orthogonal rotation matrix. Communalities then change under a supposed orthogonal rotation, so the rotated loadings no longer represent the same model as the unrotated ones. Default `Restarts: 1` is fine; raising Restarts (documented as searching for a better local minimum) triggers the bug without error or warning.

Root cause: `FaRotations` adds Promax and TargetRot start matrices whenever `restarts > 1`. Those starts are oblique. GPA preserves orthogonality relative to the start, so a non-orthogonal start yields a non-orthogonal `rotmat`. When that start wins on the criterion, the caller gets a broken "orthogonal" solution. Heuristic starts also ignore the Restarts budget.

## What Changes

- For orthogonal methods, only identity, Varimax, and random orthonormal starts are used; Promax / TargetRot starts are reserved for oblique methods.
- `Restarts` caps the total number of starts (heuristics + random), not only the random ones.
- Orthogonal starts are QR-orthonormalized before use.
- Tests pin `R'R ≈ I` and stable communalities for orthogonal methods with Restarts > 1.
- Docs and both changelogs note the corrected Restarts behaviour.

## Impact

- Affected code: `stats/internal/fa/psych_faRotations.go`
- User-visible: orthogonal rotation numeric output when `Restarts > 1` (values that were wrong become correct)
- No API break; default Restarts path unchanged
