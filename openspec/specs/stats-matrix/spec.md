# stats-matrix Specification

## Purpose
How `stats` builds and reads diagonal and identity matrices with typed functions, and what the deprecated `Diag` keeps doing on this line.

## Requirements

### Requirement: DiagOf extracts matrix diagonal

`stats.DiagOf(m mat.Matrix) ([]float64, error)` SHALL 回傳 `m` 的主對角線元素 `m[i][i]`（`i` 從 0 到 `min(列數, 欄數)-1`），且在 `m` 為 nil 介面或 nil 指標時回傳錯誤。

#### Scenario: Extracting a diagonal
- **WHEN** 呼叫 `DiagOf(mat.NewDense(2, 3, []float64{1, 2, 3, 4, 5, 6}))`
- **THEN** 回傳 `[1 5]`

#### Scenario: Nil matrix input
- **WHEN** 呼叫 `DiagOf(nil)`
- **THEN** 回傳錯誤，不 panic

### Requirement: DiagMatrix creates a square diagonal matrix

`stats.DiagMatrix(v []float64) (*mat.Dense, error)` SHALL 回傳以 `v` 為對角線、其餘為 0 的 `len(v)`×`len(v)` 方陣，且在 `v` 為空時回傳錯誤。

#### Scenario: Creating a square diagonal matrix
- **WHEN** 呼叫 `DiagMatrix([]float64{1, 2, 3})`
- **THEN** 回傳 3×3 對角線為 `1, 2, 3` 的矩陣

#### Scenario: Empty slice input
- **WHEN** 呼叫 `DiagMatrix(nil)` 或 `DiagMatrix([]float64{})`
- **THEN** 回傳錯誤，不 panic

### Requirement: DiagMatrixSize creates a rectangular diagonal matrix

`stats.DiagMatrixSize(v []float64, nrow, ncol int) (*mat.Dense, error)` SHALL 回傳對角線前 `len(v)` 格為 `v`、其餘為 0 的 `nrow`×`ncol` 矩陣，且在維度無效或 `len(v) > min(nrow, ncol)` 時回傳錯誤而非截斷。

#### Scenario: A rectangular identity
- **WHEN** 呼叫 `DiagMatrixSize([]float64{1, 1}, 2, 3)`
- **THEN** 回傳 2×3 矩陣 `[[1 0 0] [0 1 0]]`

#### Scenario: Too many diagonal values
- **WHEN** 呼叫 `DiagMatrixSize([]float64{1, 2, 3}, 2, 2)`
- **THEN** 回傳錯誤，矩陣為 nil

### Requirement: IdentityMatrix creates an identity matrix

`stats.IdentityMatrix(n int) (*mat.Dense, error)` SHALL 回傳 `n`×`n` 單位矩陣，且在 `n < 1` 時回傳錯誤。

#### Scenario: A size of zero
- **WHEN** 呼叫 `IdentityMatrix(0)`
- **THEN** 回傳錯誤，不 panic

#### Scenario: Valid identity matrix
- **WHEN** 呼叫 `IdentityMatrix(3)`
- **THEN** 回傳 3×3 單位矩陣

### Requirement: Diag is deprecated and keeps its behavior

`Diag(x any, dims ...int) (any, error)` SHALL 標為 Deprecated 並註明在 0.4 移除，doc comment SHALL 列出每種用法對應的新函式。它 SHALL 保留原本的行為。

#### Scenario: The old meaning is kept
- **WHEN** 呼叫 `Diag([]float64{1, 2, 3})`、`Diag(3)` 與 `Diag(nil, 2, 3)`
- **THEN** 結果分別與 `DiagMatrix([]float64{1, 2, 3})`、`IdentityMatrix(3)` 與 `DiagMatrixSize([]float64{1, 1}, 2, 3)` 相同
