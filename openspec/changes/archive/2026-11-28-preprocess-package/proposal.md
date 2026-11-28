# Proposal: preprocess-package (0.3.x)

## Why

On `0.4` the scalers, the categorical encoders and the fitted imputer leave the root package for a package of their own, `preprocess`, the way scikit-learn keeps them in `sklearn.preprocessing` and `sklearn.impute`. The owner's rule is that a name `0.4` replaces ships on `0.3.x` too, with the old one deprecated, so `0.3.6` users change their code once and need no second change for `0.4`. The design, confirmed by the owner on 2026-11-28, is in `/mnt/project-files/453/split-design.md`.

Milestone: #453 root-package split, slice 2 of 3, the `dev` half. Waits for `excel-package` only because the branch stacks on it; the `0.4` half is its own change on `0.4`.

## What Changes

- New package `preprocess`, ported from `0.4` so it behaves as `0.4` does: `StandardScale`, `MinMaxScale`, `RobustScale`, `MaxAbsScale` and their `New…Scaler` constructors, `OneHotEncode`, `LabelEncode`, `OrdinalEncode` with `OneHotOptions`, `LabelOptions`, `OrdinalOptions`, `SimpleImputer`, and the policy types and constants. Columns are column selectors (a bare string is an Excel-style index, `insyra.Name` a name), and a one-hot column name writes a float category as a plain decimal.
- The package is a port, not aliases of the core types, because the core reads a bare string as a name first; aliases would give `0.3.6` code a meaning that changes on `0.4`.
- `internal/prep` holds the selector rule and missing-value helpers the package uses; `internal/utils` gains `FloatText` and `ValueText`, as on `0.4`.
- A scaler's JSON written by the core (`"ref"` a plain string, 0.3.4 onward) decodes into the `preprocess` scaler, resolving the reference name first and then as an index, as the core did, and is written back the same way.
- `ml` accepts both kinds in pipelines, `ColumnTransformer` and ONNX export, and exports the same graph for both.
- Deprecated, behaviour unchanged: the `DataTable` scaling and encoding methods (and on `IDataTable`), the scaler, encoder and imputer types, their constructors and options, and the policy types and constants. The CLI keeps using them until `0.4`.

## Capabilities

### New Capabilities

- `preprocess-package`: where fitted preprocessing lives and how it reads columns.

## Impact

- Code: `preprocess/`, `internal/prep/`, `internal/utils/text.go`, `datatable_scale.go`, `datatable_encode.go`, `datatable_simple_imputer.go`, `interfaces.go`, `ml/`, `allpkgs/`, `//nolint:staticcheck` on remaining deprecated uses.
- Docs: `Docs/preprocess.md` (new), `Docs/DataTable.md`, `Docs/ml.md`, `Docs/README.md`, `README.md`, `README_TW.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`.
