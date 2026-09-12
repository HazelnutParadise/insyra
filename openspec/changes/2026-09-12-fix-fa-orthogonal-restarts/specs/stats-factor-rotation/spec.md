## ADDED Requirements

### Requirement: Orthogonal rotations keep an orthogonal rotation matrix under restarts

When the caller requests an orthogonal factor rotation (Varimax, Quartimax, GeominT, BentlerT), the system SHALL return a rotation matrix R satisfying R'R ≈ I, including when `Rotation.Restarts` is greater than 1.

Orthogonal methods SHALL NOT use oblique heuristic starts (Promax, TargetRot). Starts for orthogonal methods SHALL be orthonormal (identity, Varimax, or random orthonormal), and `Restarts` SHALL bound the total number of starts.

#### Scenario: Quartimax with Restarts greater than 1

- **WHEN** FactorAnalysis (or FaRotations) runs Quartimax with Restarts ≥ 2 on a multi-factor loading matrix
- **THEN** the returned rotation matrix satisfies max|R'R − I| at machine precision
- **AND** per-variable communalities match the unrotated loadings (max |h²_rotated − h²_unrotated| at machine precision)

#### Scenario: Restarts equals 1 stays identity-only

- **WHEN** Restarts is 1 (the FactorAnalysis default)
- **THEN** only the identity start is used
- **AND** orthogonal methods still return an orthogonal R

#### Scenario: Oblique methods may still use Promax / Target starts

- **WHEN** an oblique rotation supporting restarts runs with Restarts > 1
- **THEN** Promax and TargetRot heuristic starts remain allowed within the Restarts budget
