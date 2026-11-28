# Proposal: preprocess-package

## Why

The scalers, encoders and the fitted imputer are not `DataTable` methods by nature: each fits state on one table and applies it to others, the way scikit-learn keeps them in `sklearn.preprocessing` and `sklearn.impute` (https://scikit-learn.org/stable/api/sklearn.preprocessing.html). They make up about 2,300 lines of the root package. This is the second slice of #453 (moving functions that need not be `DataTable` methods out of the root directory). The design, reviewed independently and confirmed by the owner on 2026-11-28, is in `/mnt/project-files/453/split-design.md`; this change carries its `preprocess` part on `0.4`.

Milestone: #453 root-package split, slice 2 of 3 (`excel`, then `preprocess`, then `sqlio`). Waits for nothing; it is stacked on `excel-package` only to keep the changelogs from conflicting.

## What Changes

- **BREAKING** New package `preprocess`: `NewStandardScaler`, `NewMinMaxScaler`, `NewDefaultMinMaxScaler`, `NewRobustScaler`, `NewMaxAbsScaler`, the four scaler types, `Scaler`, `DataListScaler`, `ScalerParams`; `StandardScale(dt, cols...)`, `MinMaxScale(dt, min, max, cols...)`, `RobustScale(dt, cols...)`, `MaxAbsScale(dt, cols...)`; `OneHotEncode(dt, OneHotOptions)`, `LabelEncode(dt, LabelOptions)`, `OrdinalEncode(dt, OrdinalOptions)`, the three encoder types, `Encoder`, `NaNPolicy`, `UnknownPolicy`, `LabelSort`; `NewSimpleImputer`, `SimpleImputer`, `SimpleImputerOptions`, `ImputationStrategy`.
- **BREAKING** Removed from the root package: all of the above, and the seven methods from `DataTable` and `IDataTable`. `LabelEncodeOptions` and `OrdinalEncodeOptions` become `LabelOptions` and `OrdinalOptions`.
- Results, error text and the scaler JSON format are unchanged; JSON and CLI environment files saved before still load. The in-place fills (`FillForward`, `FillWithMean`, ...) and `DataList.Normalize`/`Standardize` stay in the core.
- The column selector rule, the type-7 quantile and the missing-value tests move to `internal/prep`, shared by the core and `preprocess`.
- `ml`, `nn`, the CLI (`scale`, `encode`, `show`, environment files) and `allpkgs` use `preprocess`. CLI commands are unchanged.

## Capabilities

### New Capabilities

- `preprocess-package`: where fitted preprocessing lives and how it is called.

## Impact

- Code: `preprocess/`, `internal/prep/`, `column_selector.go`, `interfaces.go`, `enum_names.go`, `ml/`, `internal/dsl/`, `allpkgs/`; `datatable_scale.go`, `datatable_encode.go`, `datatable_simple_imputer.go`, `datatable_preprocess.go` deleted.
- Tests: the root tests of the moved code move to `preprocess/`.
- Docs: `Docs/preprocess.md` (new), `Docs/DataTable.md`, `Docs/DataList.md`, `Docs/ml.md`, `Docs/nn.md`, `Docs/README.md`, `README.md`, `README_TW.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`.
- `dev` gets the same package with the old names marked Deprecated, in a change of its own.
