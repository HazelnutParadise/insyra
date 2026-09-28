## ADDED Requirements

### Requirement: Diagonal and identity matrices have typed functions

`stats` SHALL 提供：

- `DiagOf(m mat.Matrix) ([]float64, error)`：回傳 `m` 的主對角線 `m[i][i]`，`i` 從 0 到 `min(列數, 欄數)-1`。`m` 為 nil 介面或 nil 指標時 SHALL 回傳錯誤。
- `DiagMatrix(v []float64) (*mat.Dense, error)`：回傳 `len(v)`×`len(v)`、對角線為 `v`、其餘為 0 的矩陣。`v` 為空時 SHALL 回傳錯誤。
- `DiagMatrixSize(v []float64, nrow, ncol int) (*mat.Dense, error)`：回傳 `nrow`×`ncol`、對角線前 `len(v)` 格為 `v`、其餘為 0 的矩陣。`nrow` 或 `ncol` 小於 1 時 SHALL 回傳錯誤；`len(v)` 大於 `min(nrow, ncol)` 時 SHALL 回傳錯誤，SHALL NOT 截斷 `v`。
- `IdentityMatrix(n int) (*mat.Dense, error)`：回傳 `n`×`n` 單位矩陣。`n` 小於 1 時 SHALL 回傳錯誤。

這些函式 SHALL NOT panic，也 SHALL NOT 修改傳入的 `v` 或 `m`。

#### Scenario: Extracting a diagonal
- **WHEN** 呼叫 `DiagOf(mat.NewDense(2, 3, []float64{1, 2, 3, 4, 5, 6}))`
- **THEN** 回傳 `[1 5]`

#### Scenario: A rectangular identity
- **WHEN** 呼叫 `DiagMatrixSize([]float64{1, 1}, 2, 3)`
- **THEN** 回傳 2×3 矩陣 `[[1 0 0] [0 1 0]]`

#### Scenario: Too many diagonal values
- **WHEN** 呼叫 `DiagMatrixSize([]float64{1, 2, 3}, 2, 2)`
- **THEN** 回傳錯誤，矩陣為 nil

#### Scenario: A size of zero
- **WHEN** 呼叫 `IdentityMatrix(0)` 或 `DiagMatrix(nil)`
- **THEN** 回傳錯誤，不 panic

### Requirement: Diag is deprecated and no longer panics

`Diag(x any, dims ...int) (any, error)` SHALL 保留一個版本，標為 Deprecated，doc comment SHALL 列出每種用法對應的新函式。它 SHALL 保留原本的意義，但會產生小於 1 的列數或欄數時（例如 `Diag(0)`、`Diag([]float64{})`、`Diag(-1)`）SHALL 回傳錯誤，SHALL NOT panic。

#### Scenario: The old sizes that panicked
- **WHEN** 呼叫 `Diag(0)`
- **THEN** 回傳錯誤，不 panic

#### Scenario: The old meaning is kept
- **WHEN** 呼叫 `Diag([]float64{1, 2, 3})`、`Diag(3)` 與 `Diag(nil, 2, 3)`
- **THEN** 結果分別與 `DiagMatrix([]float64{1, 2, 3})`、`IdentityMatrix(3)` 與 `DiagMatrixSize([]float64{1, 1}, 2, 3)` 相同
