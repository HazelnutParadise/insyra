# stats 包開發準則

統計學由多個基礎概念疊加組成。**這個包的內部架構也必須如此**。

在新增任何統計方法之前，先確認你需要的計算是否已有現成的建構塊。
**複用現有建構塊，不要重新實作。**

---

## 分層架構

### gonum 優先原則

實作任何層級時，**如果 gonum 有符合需求的實作，優先使用 gonum。** 直接引用或建立輕量包裝都可以，無需重新實作。

常用的 gonum 模組：

- `gonum/mat` — 矩陣運算（LU 分解、求逆、行列式）
- `gonum/stat` — 樣本統計（均值、方差、相關係數）
- `gonum/stat/distuv` — 機率分佈（t、F、χ²、常態）

---

```
Layer 0 — 數學原語（其他所有層的地基）
  distutil.go              統計分佈：t/z/F/χ² 的 CDF、quantile、p 值
  internal/linalg/linalg.go  矩陣運算：高斯消去、矩陣求逆、行列式

Layer 1 — 樣本統計原語
  sampleutil.go 樣本 SE、合併方差、Welch df
  moments.go    一般化矩（CalculateMoment，已供 skewness/kurtosis 使用）

Layer 2 — 統計推斷原語
  mathutil.go   信賴區間、信賴水準、效果量、eta²、F-ratio、
                相關係數轉換（correlationToT、Fisher z-transform）

Layer 3 — 回歸原語
  olsutil.go    OLS 係數求解、殘差/R²、SE/t/p 推斷、CI 建構

Layer 4 — 對外統計方法（公開 API）
  ttest.go  ztest.go  anova.go  ftest.go  chi_square.go
  correlation.go  regression.go  pca.go
  moments.go  skewness.go  kurtosis.go
```

**Layer N 只能呼叫 Layer 0 ~ N-1 的函數，不得向上呼叫。**

---

## 新增統計方法前的檢查清單

### 需要 p 值？先查 distutil.go

| 你需要的 | 使用函數 |
|---|---|
| Student's t 雙尾 p 值 | `tTwoTailedPValue(t, df float64)` |
| Student's t 單尾 CDF | `tCDF(t, df float64)` |
| Student's t 臨界值 | `tQuantile(p, df float64)` |
| 標準常態 p 值（含方向） | `zPValue(z float64, alt AlternativeHypothesis)` |
| 標準常態 CDF | `zCDF(z float64)` |
| F 分佈右尾 p 值 | `fOneTailedPValue(f, df1, df2 float64)` |
| F 分佈雙尾 p 值 | `fTwoTailedPValue(f, df1, df2 float64)` |
| 卡方右尾 p 值 | `chiSquaredPValue(chi2, df float64)` |

### 需要樣本統計量？先查 sampleutil.go

| 你需要的 | 使用函數 |
|---|---|
| 樣本平均數的標準誤 s/√n | `sampleSE(s, n float64)` |
| 不等方差雙樣本 SE | `twoSampleSE(var1, var2, n1, n2 float64)` |
| 合併方差（等方差假設） | `pooledVariance(var1, var2, n1, n2 float64)` |
| 等方差雙樣本 SE + 合併方差 | `pooledSE(var1, var2, n1, n2 float64) (se, pVar float64)` |
| Welch-Satterthwaite 自由度 | `welchDF(var1, var2, n1, n2 float64)` |
| 高階矩（偏度/峰度基礎） | `CalculateMoment(dl, n, central)` |

### 需要推斷工具？先查 mathutil.go

| 你需要的 | 使用函數 |
|---|---|
| 驗證並取得信賴水準 | `resolveConfidenceLevel(cl float64)` |
| 對稱信賴區間 | `symmetricCI(center, margin float64)` |
| 兩端 NaN 的 CI | `nanCI()` |
| t 誤差邊界 critT × SE | `tMarginOfError(cl, df, se float64)` |
| z 誤差邊界 critZ × SE | `zMarginOfError(cl, se float64)` |
| Cohen's d 效果量切片 | `cohenDEffectSizes(d float64)` |
| Eta-squared η² | `etaSquared(ssEffect, ssError float64)` |
| ANOVA F 值 | `fRatio(ssBetween float64, dfBetween int, ssWithin float64, dfWithin int)` |
| 相關係數 → t 統計量 | `correlationToT(r, n float64)` |
| Fisher's z-transform | `fisherZTransform(r float64)` |
| Fisher's z 反函數 | `fisherZInverse(z float64)` |
| Pearson/Spearman CI（z-space） | `pearsonFisherCI(r, n, cl float64)` |

### 需要矩陣運算？先查 `internal/linalg/linalg.go`

> 這是 `stats` 包中唯一放進 `stats/internal/` 的模組。  
> 原因：純 `float64` 矩陣數學，**不依賴** `EffectSizeEntry`、`AlternativeHypothesis`、`defaultConfidenceLevel`、`norm` 等 stats 型別，不會造成 circular import。  
> 其他工具檔案（`distutil.go`、`sampleutil.go`、`mathutil.go`、`olsutil.go`）留在 `stats` 包，因為它們使用 stats 包型別。

| 你需要的 | 使用函數（`import ".../stats/internal/linalg"`） |
|---|---|
| 解線性方程組 Ax=b | `linalg.GaussianElimination(A, b)` |
| 矩陣求逆 | `linalg.InvertMatrix(A)` |
| 方陣行列式 | `linalg.DeterminantGauss(matrix)` |

### 需要 OLS 迴歸計算？先查 olsutil.go

| 你需要的 | 使用函數 |
|---|---|
| 二參數 OLS（y = a + bx） | `simpleOLSCoeffs(xs, ys)` |
| Σ(x-x̄)² | `simpleOLSSxx(xs)` |
| 殘差、SSE、R²、Adjusted R² | `computeGoodnessOfFit(ys, yHatFn, df)` |
| 係數的 SE / t / p（矩陣版） | `computeCoeffInference(coeffs, XTXInv, mse, df)` |
| 兩係數 CI（截距 + 斜率） | `buildTwoCoeffCIs(a, b, seA, seB, df)` |
| 多係數 CI 切片 | `buildMultiCoeffCIs(coeffs, standardErrors, df)` |

---

## 禁止事項

**以下行為在此包中明確禁止**，因為對應的共用實作已存在：

```go
// ❌ 禁止：直接建立分佈物件
distuv.StudentsT{Mu: 0, Sigma: 1, Nu: df}.CDF(t)
distuv.F{D1: df1, D2: df2}.CDF(f)
distuv.ChiSquared{K: df}.CDF(x)

// ✅ 應該使用
tCDF(t, df)
fOneTailedPValue(f, df1, df2)
chiSquaredPValue(chi2, df)

// ❌ 禁止：手算 SE
stddev / math.Sqrt(float64(n))

// ✅ 應該使用
sampleSE(stddev, float64(n))

// ❌ 禁止：hardcode 1.96
z - 1.96*se
z + 1.96*se

// ✅ 應該使用（支援任意信賴水準）
pearsonFisherCI(r, n, confidenceLevel)

// ❌ 禁止：inline eta-squared
ssEffect / (ssEffect + ssError)

// ✅ 應該使用
etaSquared(ssEffect, ssError)

// ❌ 禁止：inline F-ratio
(ssBetween / float64(dfBetween)) / (ssWithin / float64(dfWithin))

// ✅ 應該使用
fRatio(ssBetween, dfBetween, ssWithin, dfWithin)

// ❌ 禁止：重複 Cohen's d 切片
[]EffectSizeEntry{{Type: "cohen_d", Value: d}}

// ✅ 應該使用
cohenDEffectSizes(d)

// ❌ 禁止：inline 信賴水準驗證
if cl <= 0 || cl >= 1 { cl = defaultConfidenceLevel }

// ✅ 應該使用
cl = resolveConfidenceLevel(cl)

// ❌ 禁止：用 math.Pow 算整數次方（特別是 2、3）
math.Pow(x, 2)
math.Pow(x, 3)

// ✅ 應該使用（更快、最後 1~2 ULP 也更準）
x * x
x * x * x
```

---

## 新增統計方法的步驟

1. **識別所需的統計學概念層**（p 值？SE？CI？效果量？）
2. **逐層查閱上方清單**，確認現有建構塊是否已覆蓋
3. 若現有建構塊不足，**先在對應 Layer 的檔案中新增共用函數**，再使用它
4. 對外公開的結果型別放在 `structs.go` 或對應的方法檔案中
5. 任何新的私有計算函數**不得放在 Layer 4 的方法檔案中**，應放在 Layer 0-3

---

## 詳細重構規劃

見 [REFACTOR_PLAN.md](REFACTOR_PLAN.md)，記錄了所有現有的重複點及完整的修改規格。

---

## 2026-04-08 API Update (Error-First)

- Exported functions now return `error` for invalid input / failed computation.
- Exported `stats` APIs do not use `LogWarning`/`LogFatal` for failure signaling.
- Callers must handle `err` explicitly.
- See `Docs/stats.md` and `Docs/tutorials/*` for updated error-return usage examples.
