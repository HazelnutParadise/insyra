# 因素分析常數對齊說明

本文件記錄了因素分析演算法中所有常數與 R psych::fa 的對齊情況。

## 概述

所有常數已從硬編碼值改為具名常數,並對齊 R 的 psych 套件和 GPArotation 套件的預設值。

## 常數定義

### 收斂容差 (Convergence Tolerance)

```go
extractionTolerance = 1e-6  // 因子提取的一般收斂容差
```

**用途**: PAF, ML, MINRES 等提取方法的收斂判斷  
**R 對應**: psych::fa 內部使用的預設容差

### 旋轉相關常數 (Rotation Constants)

```go
rotationMaxIter      = 1000   // 旋轉最大迭代次數
rotationTolerance    = 1e-5   // 旋轉收斂容差
rotationGradientStep = 0.01   // 旋轉梯度下降步長
```

**來源**: R 的 GPArotation 套件預設值

- `maxit = 1000` (預設)
- `eps = 1e-5` (預設)

**變更前**: `rotMaxIter = 200`, `rotTol = 1e-6`
**影響**: 增加旋轉迭代次數和容差,更符合 R 的行為

### 數值穩定性常數 (Numerical Stability Constants)

```go
epsilonSmall  = 1e-10  // 矩陣求逆和奇異性檢查
epsilonTiny   = 1e-12  // 接近零的檢查 (如行範數)
epsilonMedium = 1e-6   // 共同性下界和總和檢查
```

**用途**:

- `epsilonSmall`: 矩陣求逆、因子符號反射檢測、Phi 正規化
- `epsilonTiny`: Kaiser 正規化中的行範數檢查
- `epsilonMedium`: PAF/ML 中的 communality 下界、Heywood case 處理

### MINRES 優化常數 (MINRES Optimization Constants)

```go
minresPsiLowerBound = 0.005  // Psi 下界
minresPsiUpperBound = 1.0    // Psi 上界
```

**R 對應**:

```r
optim(..., method = "L-BFGS-B", lower = 0.005, upper = 1)
```

**變更前**: 硬編碼 `lower = 0.005`
**影響**: 與 R 的 MINRES 實現完全一致

### Ridge 正規化 (Ridge Regularization)

```go
ridgeRegularization = 1e-6  // 矩陣求逆的數值穩定性
```

**用途**: safeInvert() 函數中用於改善矩陣條件數  
**R 對應**: R 在因子分數計算時也使用類似的正規化策略

### 相關矩陣對角線檢查

```go
corrDiagTolerance    = 1e-6  // 對角線偏離 1.0 的容差
corrDiagLogThreshold = 1e-8  // 記錄對角線偏離的閾值
uniquenessLowerBound = 1e-9  // uniqueness 值下界
```

### 機器精度與 Eigenvalue 閾值

```go
machineEpsilon         = 2.220446e-16  // R's .Machine$double.eps
eigenvalueMinThreshold = 2.220446e-14  // 100 * .Machine$double.eps
```

**R 對應**:

```r
eigens$values[eigens$values < .Machine$double.eps] <- 100 * .Machine$double.eps
```

**用途**: 防止非常小的 eigenvalue 導致數值不穩定

### 優化控制參數

```go
optimFnScale  = 1.0   // R: fnscale = 1
optimParScale = 0.01  // R: parscale = rep(0.01, ...)
```

**R 對應**:

```r
optim(..., control = c(list(fnscale = 1, parscale = rep(0.01, length(start)))))
```

## 修改摘要### 1. 旋轉演算法 (Rotation Algorithms)

- **Varimax, Quartimax, Promax, Oblimin**
  - 最大迭代次數: 200 → **1000**
  - 收斂容差: 1e-6 → **1e-5**
  - 目標函數改善閾值: 1e-10 → **epsilonSmall (1e-10)**

### 2. 因子提取 (Factor Extraction)

- **所有方法 (PCA, PAF, ML, MINRES)**
  - Eigenvalue 閾值: 0 → **eigenvalueMinThreshold (2.22e-14)**
  - 對應 R: `eigens$values[eigens$values < .Machine$double.eps] <- 100 * .Machine$double.eps`

- **PAF (Principal Axis Factoring)**
  - Communality 下界: 1e-6 → **epsilonMedium (1e-6)**
  - 最小誤差參數: 0.001 (與 R 一致)

- **ML (Maximum Likelihood)**
  - psMin: 1e-6 → **epsilonMedium (1e-6)**

- **MINRES (Minimum Residual)**
  - Psi 下界: 0.005 → **minresPsiLowerBound (0.005)**
  - 初始化 PAF 容差: 使用 extractionTolerance
  - Eigenvalue 下界: 也使用 eigenvalueMinThreshold

### 3. 矩陣操作 (Matrix Operations)

- **Kaiser 正規化**
  - 零行範數閾值: 1e-12 → **epsilonTiny (1e-12)**

- **相關矩陣正規化**
  - 對角線零檢測: 1e-10 → **epsilonSmall (1e-10)**

- **Promax 旋轉**
  - 矩陣求逆 ridge: 1e-10 → **epsilonSmall (1e-10)**
  - 縮放因子閾值: 1e-12 → **epsilonTiny (1e-12)**

- **Oblimin 旋轉**
  - 對角線值閾值: 1e-12 → **epsilonTiny (1e-12)**

### 4. 因子分數計算 (Factor Scores)

- **Regression, Bartlett, Anderson-Rubin**
  - 所有 safeInvert() 調用: 1e-6 → **ridgeRegularization (1e-6)**

### 5. 其他

- **因子符號反射檢測**
  - 反射判斷閾值: 1e-10 → **epsilonSmall (1e-10)**

- **Heywood case 處理**
  - 共同性上界調整: 1e-6 → **epsilonMedium (1e-6)**

## R psych::fa 預設參數對照表

| 參數 | R 預設值 | Go 實現 | 說明 |
|------|---------|---------|------|
| `min.err` | 0.001 | 0.001 | ✓ 一致 |
| `max.iter` | 50 | 50 | ✓ 一致 |
| `fm` | "minres" | FactorExtractionMINRES | ✓ 一致 |
| `rotate` | "oblimin" | FactorRotationOblimin | ✓ 一致 |
| `scores` | "regression" | FactorScoreRegression | ✓ 一致 |
| GPArotation `maxit` | 1000 | 1000 | ✓ 已對齊 |
| GPArotation `eps` | 1e-5 | 1e-5 | ✓ 已對齊 |

## 測試建議

在修改常數後,建議進行以下測試:

1. **收斂性測試**: 確保算法在合理迭代次數內收斂
2. **數值穩定性測試**: 測試病態矩陣和極端案例
3. **R 對照測試**: 使用相同資料集比較 Go 和 R 的結果
4. **旋轉一致性**: 驗證旋轉結果與 R GPArotation 套件一致

## 參考資料

1. R psych 套件: [https://personality-project.org/r/psych/](https://personality-project.org/r/psych/)
2. GPArotation 套件文件
3. Revelle, W. (2024). psych: Procedures for Psychological, Psychometric, and Personality Research

## 版本記錄

- 2025-10-03: 完成所有常數對齊,從硬編碼改為具名常數
