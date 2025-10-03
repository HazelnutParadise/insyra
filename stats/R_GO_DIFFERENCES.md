# R vs Go 因素分析實作差異分析

## 發現的關鍵差異

### 1. MINRES 目標函數 - fit.residuals

#### R 實作

```r
"fit.residuals" <- function(Psi, S, nf, S.inv = NULL, fm) {
    diag(S) <- 1 - Psi  # ❌ 注意:直接修改 S 的對角線!
    eigens <- eigen(S)
    eigens$values[eigens$values < .Machine$double.eps] <- 100 * .Machine$double.eps
    
    if (nf > 1) {
      loadings <- eigens$vectors[, 1:nf] %*% diag(sqrt(eigens$values[1:nf]))
    } else {
      loadings <- eigens$vectors[, 1] * sqrt(eigens$values[1])
    }
    
    model <- loadings %*% t(loadings)
    
    # 對於 MINRES
    residual <- (S - model)
    residual <- residual[lower.tri(residual)]  # 只取下三角
    residual <- residual^2
    
    error <- sum(residual)
}
```

#### Go 當前實作問題

1. ✅ **正確**: 建立新的 reducedCorr,不修改原始 corrMatrix
2. ✅ **正確**: 使用 `corrMatrix.At(i, j) - psi[i]` 修改對角線
3. ✅ **正確**: 只對下三角求和 (i+1 到 j)
4. ❌ **問題**: 特徵值處理不一致

### 2. MINRES 梯度函數 - FAgr.minres

#### R 實作

```r
FAgr.minres <- function(Psi, S, nf, fm) {
    Sstar <- S - diag(Psi)  # 減去 Psi,不是 1-Psi!
    E <- eigen(Sstar, symmetric = TRUE)
    L <- E$vectors[, 1:nf, drop = FALSE]
    load <- L %*% diag(sqrt(pmax(E$values[1:nf], 0)), nf)  # ⭐ pmax(..., 0)
    g <- load %*% t(load) + diag(Psi) - S
    diag(g)  # 只返回對角線元素
}
```

#### Go 當前實作問題

1. ✅ **正確**: `corrMatrix.At(i, j) - psi[i]` 建立 reducedCorr
2. ❌ **問題**: 使用 `math.Max(pairs[j].value, eigenvalueMinThreshold)`,R 使用 `pmax(..., 0)`
3. ❌ **問題**: 對角線計算可能不一致

### 3. 特徵值處理的關鍵差異

#### R 在 fit.residuals 中

```r
eigens$values[eigens$values < .Machine$double.eps] <- 100 * .Machine$double.eps
loadings <- eigens$vectors[, 1:nf] %*% diag(sqrt(eigens$values[1:nf]))
# 注意:這裡已經調整過,所以直接用 sqrt(eigens$values[1:nf])
```

#### R 在 FAgr.minres 中  

```r
load <- L %*% diag(sqrt(pmax(E$values[1:nf], 0)), nf)
# 使用 pmax(..., 0) 而不是閾值
```

#### Go 當前實作

```go
// 目標函數中
if pairs[j].value > eigenvalueMinThreshold {
    loadings.Set(i, j, pairs[j].vector[i]*math.Sqrt(pairs[j].value))
} else {
    loadings.Set(i, j, 0)
}

// 梯度函數中
if pairs[j].value > eigenvalueMinThreshold {
    loadings.Set(i, j, pairs[j].vector[i]*math.Sqrt(math.Max(pairs[j].value, eigenvalueMinThreshold)))
} else {
    loadings.Set(i, j, 0)
}
```

**問題**:

- R 在不同函數中使用不同的處理方式
- fit.residuals: 先調整特徵值,再直接開根號
- FAgr.minres: 使用 pmax(eigenvalue, 0)

### 4. SMC 計算差異

#### R 實作

```r
smc <- function (R, covar = FALSE) {
    R.inv <- Pinv(R)  # ⭐ 使用 pseudoinverse!
    smc <- 1 - 1/diag(R.inv)
    
    # 邊界處理
    if (max(smc, na.rm = TRUE) > 1) {
        message("In smc, smcs > 1 were set to 1.0")
        smc[smc > 1] <- 1
    }
    if (min(smc, na.rm = TRUE) < 0) {
        message("In smc, smcs < 0 were set to .0")
        smc[smc < 0] <- 0
    }
    
    return(smc)
}
```

#### Go 需要檢查

- 是否使用 pseudoinverse?
- 是否處理 smc > 1 的情況?
- 是否處理 smc < 0 的情況?

### 5. upper 界限計算差異

#### R 實作

```r
if (is.logical(SMC)) {
    S.smc <- smc(S, covar)
} else {
    S.smc <- SMC
}
upper <- max(S.smc, 1)  # ⭐ max(所有 SMC 值中的最大值, 1)
```

#### Go 當前實作

```go
h2 := computeSMC(corrMatrix, 0.001)
upper := minresPsiUpperBound // 初始為 1.0
for i := 0; i < p; i++ {
    if h2[i] > upper {
        upper = h2[i]  // ⭐ 逐個比較
    }
}
```

**問題**: 語義相同,但可能有細微數值差異

### 6. optim control 參數差異

#### R 實作

```r
res <- optim(start, fit.residuals, gr = FAgr.minres, 
            method = "L-BFGS-B", 
            lower = 0.005, 
            upper = upper, 
            control = c(list(fnscale = 1, 
                            parscale = rep(0.01, length(start)))),
            nf = nf, S = S, fm = fm)
```

**關鍵參數**:

- `fnscale = 1`: 目標函數縮放因子
- `parscale = rep(0.01, length(start))`: 參數縮放,所有參數縮放 0.01

#### Go 實作

gonum/optimize 不直接支持 fnscale 和 parscale,這些已註解掉

---

## 修正優先順序

### 高優先級 (影響數值結果)

1. **特徵值處理統一化**
   - fit.residuals: 調整特徵值到 100*eps,再開根號
   - FAgr.minres: 使用 pmax(eigenvalue, 0)

2. **SMC 邊界處理**
   - 加入 smc > 1 設為 1.0
   - 加入 smc < 0 設為 0.0
   - 考慮使用 pseudoinverse

3. **載荷計算一致性**
   - 確保單因子和多因子情況處理一致

### 中優先級

4. **upper 計算**
   - 改用 `max(max(h2), 1.0)` 而非循環

5. **符號翻轉邏輯**
   - 旋轉後調整載荷符號

### 低優先級 (可能無法在 gonum 中實現)

6. **optim control 參數**
   - fnscale 和 parscale 在 gonum 中沒有直接對應

---

## 測試計劃

1. 建立相同的測試數據
2. 比較 SMC 值
3. 比較初始 Psi
4. 比較 MINRES 目標函數值
5. 比較梯度值
6. 比較最終載荷
7. 比較 Phi 矩陣
8. 比較 uniquenesses
