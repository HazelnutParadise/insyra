# stats-matrix Specification

## Purpose
The typed helpers in `stats` that read a matrix's diagonal and build diagonal and identity matrices, one function per job. None of them panics: a size below 1, an empty slice and a nil matrix are errors.

## Requirements

### Requirement: Diagonal and identity matrices have typed functions

`stats` SHALL 提供：

- `DiagOf(m mat.Matrix) ([]float64, error)`：回傳 `m` 的主對角線 `m[i][i]`，`i` 從 0 到 `min(列數, 欄數)-1`。`m` 為 nil 介面或 nil 指標時 SHALL 回傳錯誤。
- `DiagMatrix(v []float64) (*mat.Dense, error)`：回傳 `len(v)`×`len(v)`、對角線為 `v`、其餘為 0 的矩陣。`v` 為空時 SHALL 回傳錯誤。
- `IdentityMatrix(n int) (*mat.Dense, error)`：回傳 `n`×`n` 單位矩陣。`n` 小於 1 時 SHALL 回傳錯誤。

這些函式與 `DiagMatrixSize` SHALL NOT panic，也 SHALL NOT 修改傳入的 `v` 或 `m`。

#### Scenario: Extracting a diagonal
- **WHEN** 呼叫 `DiagOf(mat.NewDense(2, 3, []float64{1, 2, 3, 4, 5, 6}))`
- **THEN** 回傳 `[1 5]`

#### Scenario: A size of zero
- **WHEN** 呼叫 `IdentityMatrix(0)` 或 `DiagMatrix(nil)`
- **THEN** 回傳錯誤，不 panic

### Requirement: A diagonal matrix of a given size

`DiagMatrixSize(v []float64, nrow, ncol int) (*mat.Dense, error)` SHALL 回傳 `nrow`×`ncol`、對角線前 `len(v)` 格為 `v`、其餘為 0 的矩陣。`nrow` 或 `ncol` 小於 1 時 SHALL 回傳錯誤；`len(v)` 大於 `min(nrow, ncol)` 時 SHALL 回傳錯誤，SHALL NOT 截斷 `v`。

#### Scenario: A rectangular identity
- **WHEN** 呼叫 `DiagMatrixSize([]float64{1, 1}, 2, 3)`
- **THEN** 回傳 2×3 矩陣 `[[1 0 0] [0 1 0]]`

#### Scenario: Too many diagonal values
- **WHEN** 呼叫 `DiagMatrixSize([]float64{1, 2, 3}, 2, 2)`
- **THEN** 回傳錯誤，矩陣為 nil
