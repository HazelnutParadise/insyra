# 因素分析對齊 R 實作 - 修改記錄

## 日期: 2025-10-03

### 已完成的修改

#### 1. MINRES 目標函數 - 特徵值處理 ✅

**R 行為**:

```r
eigens$values[eigens$values < .Machine$double.eps] <- 100 * .Machine$double.eps
loadings <- eigens$vectors[, 1:nf] %*% diag(sqrt(eigens$values[1:nf]))
```

**Go 修改**:

```go
// 調整小特徵值到 100 * machineEpsilon
adjustedEigenvalues := make([]float64, len(pairs))
for i := range pairs {
    if pairs[i].value < machineEpsilon {
        adjustedEigenvalues[i] = eigenvalueMinThreshold // 100 * machineEpsilon
    } else {
        adjustedEigenvalues[i] = pairs[i].value
    }
}

// 使用調整後的特徵值計算載荷
for i := 0; i < p; i++ {
    for j := 0; j < numFactors; j++ {
        loadings.Set(i, j, pairs[j].vector[i]*math.Sqrt(adjustedEigenvalues[j]))
    }
}
```

**影響**: 確保小特徵值不會導致數值不穩定

#### 2. MINRES 梯度函數 - pmax(eigenvalues, 0) ✅

**R 行為**:

```r
load <- L %*% diag(sqrt(pmax(E$values[1:nf], 0)), nf)
```

**Go 修改**:

```go
// 使用 pmax(eigenvalue, 0) 如 R 所示
for i := 0; i < p; i++ {
    for j := 0; j < numFactors; j++ {
        ev := math.Max(pairs[j].value, 0)
        loadings.Set(i, j, pairs[j].vector[i]*math.Sqrt(ev))
    }
}
```

**影響**: 確保負特徵值被設為 0,防止 sqrt(負數) 錯誤

#### 3. MINRES upper 界限計算 ✅

**R 行為**:

```r
S.smc <- smc(S, covar)
upper <- max(S.smc, 1)  # max(所有 SMC 值, 1.0)
```

**Go 修改前**:

```go
upper := minresPsiUpperBound // 1.0
for i := 0; i < p; i++ {
    if h2[i] > upper {
        upper = h2[i]
    }
}
if upper < minresPsiUpperBound {
    upper = minresPsiUpperBound
}
```

**Go 修改後**:

```go
maxH2 := 0.0
for i := 0; i < p; i++ {
    if h2[i] > maxH2 {
        maxH2 = h2[i]
    }
}
upper := math.Max(maxH2, minresPsiUpperBound)
```

**影響**: 更清晰地表達 R 的邏輯,語義相同但代碼更簡潔

---

### 待驗證的差異

#### 1. SMC 計算邊界處理

**R 行為**:

```r
if (max(smc, na.rm = TRUE) > 1) {
    smc[smc > 1] <- 1
}
if (min(smc, na.rm = TRUE) < 0) {
    smc[smc < 0] <- 0  # 設為 0.0
}
```

**Go 當前**:

```go
if smc < minErr {
    smc = minErr  // 使用 minErr (通常 0.001)
}
if smc > 1.0 {
    smc = 1.0
}
```

**潛在問題**: Go 使用 `minErr` 作為下界,而 R 使用 `0.0`。這可能導致 SMC 值的差異。

**建議**: 考慮添加一個標誌,允許使用 0.0 作為下界以精確對齊 R

#### 2. 其他提取方法的特徵值處理

需要檢查:

- **PAF**: 是否也需要調整特徵值?
- **ML**: 是否也需要調整特徵值?
- **PCA**: 是否也需要調整特徵值?

根據 R 源碼,所有方法在計算載荷時都應該處理小/負特徵值。

---

### 下一步行動

1. ✅ **MINRES 核心邏輯** - 已完成
2. ⏳ **PAF/ML/PCA 特徵值處理** - 需要檢查
3. ⏳ **旋轉算法對齊** - 需要分析 GPArotation
4. ⏳ **符號翻轉邏輯** - 需要添加
5. ⏳ **測試驗證** - 需要建立測試案例

---

### 編譯狀態

✅ 所有修改已通過編譯驗證

---

### 參考文件

- `R_FUNCTIONS_NEEDED.md` - 完整的 R 源碼
- `R_GO_DIFFERENCES.md` - 詳細的差異分析
- `FACTOR_ANALYSIS_CONSTANTS.md` - 常數定義文檔
