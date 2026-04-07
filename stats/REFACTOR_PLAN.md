# stats 包重構計劃：抽取共用統計原語

> 日期：2026-04-07（v2 完整版，補入 OLS 迴歸層重複點）  
> 分支：enhance/stats  
> 目標：消除 stats 包內所有重複的統計計算邏輯，統一為單一實作，**不改變對外 API**。

---

## 一、現況分析：重複造輪子完整清單

### 1. Student's t 分佈

| 重複點 | 出現位置 | 次數 |
|---|---|---|
| `distuv.StudentsT{Mu:0, Sigma:1, Nu:df}.CDF(t)` 建立分佈物件後呼叫 CDF | `ttest.go:371-384`（`calculateTPValue`）、`regression.go:851-863`（`studentTCDF`）、`correlation.go:290-295`、`correlation.go:393-398` | 4 處各自重新建立物件 |
| 兩尾 t p 值：`2*(1-CDF(|t|))` | `ttest.go:371`（`calculateTPValue`）、`regression.go:224`（`2.0 * studentTCDF(-|t|, df)`）、`correlation.go:295`（`2 * tDist.CDF(-|t|)`）| 3 種不同寫法，語義相同 |
| t 臨界值（Quantile）：`distuv.StudentsT{...}.Quantile(1-(1-cl)/2)` | `ttest.go:65`、`ttest.go:199`、`ttest.go:347`、`regression.go:258`、`regression.go:407`、`regression.go:543`、`regression.go:709` | **7 處**完全相同 |

**結論**：`calculateTPValue`（ttest.go）和 `studentTCDF`（regression.go）功能相同，只是介面略有不同；所有 t 臨界值計算都是 copy-paste。

---

### 2. 標準常態分佈

| 重複點 | 出現位置 |
|---|---|
| `norm`（`distuv.Normal{Mu:0, Sigma:1}`）用於 CDF/Quantile | `consts.go:7`（定義）、`ztest.go` 多處（使用）|
| 手寫 `normalCDF(z)` = `0.5*(1+math.Erf(z/√2))` | `correlation.go:412`（`kendallCorrelationWithStats` 呼叫）|
| `norm.CDF` 與 `normalCDF` 語義相同，卻是兩套實作 | `ztest.go` vs `correlation.go` |
| z 臨界值：`norm.Quantile(1-(1-cl)/2)` | `ztest.go:44`、`ztest.go:120` |

**結論**：`normalCDF` 可直接以 `norm.CDF` 取代；z 臨界值計算重複。

---

### 3. F 分佈 p 值

| 重複點 | 出現位置 |
|---|---|
| `1 - distuv.F{D1:df1, D2:df2}.CDF(f)` 單尾 | `anova.go:100`、`anova.go:224`（lambda fd）、`anova.go:309`、`ftest.go:152`、`ftest.go:176`、`ftest.go:233` |
| 兩尾 F：`2 * min(CDF(f), 1-CDF(f))` | `ftest.go:43` |

**結論**：6 處重複建立 `distuv.F` 物件；有兩種語義（單尾/兩尾）應分別封裝。

---

### 4. 卡方分佈 p 值

| 重複點 | 出現位置 |
|---|---|
| `1 - distuv.ChiSquared{K:df}.CDF(x)` | `chi_square.go:42`、`ftest.go:133`（BartlettTest）、`correlation.go:255`（BartlettSphericity）|

**結論**：3 處完全相同的模式。

---

### 5. 信賴水準解析

| 重複點 | 出現位置 |
|---|---|
| `if cl <= 0 \|\| cl >= 1 { cl = defaultConfidenceLevel }` | `ttest.go:61-63`、`ttest.go:193-195`、`ttest.go:341-343`（3 次）、`ztest.go:39-41`、`ztest.go:115-117`（2 次）|

**結論**：同一段驗證邏輯出現 **5 次**。

---

### 6. 對稱信賴區間建構

| 重複點 | 出現位置 |
|---|---|
| `ci := &[2]float64{center - margin, center + margin}` | `ttest.go:69`、`ttest.go:203`、`ttest.go:351`、`ztest.go`（多處）、`regression.go`（多處）|

---

### 7. `oneWayANOVA` 私有函數（ftest.go）與公開 `OneWayANOVA`（anova.go）

`ftest.go:193` 定義了一個私有 `oneWayANOVA(values []float64, labels []int, k int)`，供 `LeveneTest` 使用。  
此函數與 `anova.go` 中的 `OneWayANOVA` 計算邏輯完全相同（SSB、SSW、F、p），只是輸入格式不同（raw slice vs IDataList）。  
目前兩套並行存在，且底層的 F p 值計算又各自直接呼叫 `distuv.F`。

---

### 8. 簡單線性迴歸公式（Cramer's Rule OLS）

`ExponentialRegression`（第 327-346 行）和 `LogarithmicRegression`（第 467-500 行）都對轉換後的資料執行**完全相同的二參數 OLS 計算**：

```go
// 兩個函數都有這段（變數名稱略有不同）
sumX, sumY, sumXY, sumX2 := 0.0, 0.0, 0.0, 0.0
for i := range xs {
    sumX  += xs[i]      // 或 lx
    sumY  += transformedY[i]
    sumXY += xs[i] * transformedY[i]
    sumX2 += xs[i] * xs[i]
}
denom := n*sumX2 - sumX*sumX
// if denom == 0 { ... }
b := (n*sumXY - sumX*sumY) / denom
a := (sumY - b*sumX) / n
```

這是 `y = a + b·x` 的 Cramer's rule 解，屬於 `LinearRegression` 中通用矩陣 OLS 的特例，**完全重複**。

---

### 9. OLS 推斷迴圈（標準誤 / t 值 / p 值，矩陣版）

`LinearRegression`（第 206-230 行）和 `PolynomialRegression`（第 680-699 行）有**幾乎逐字相同**的推斷迴圈：

```go
// LinearRegression 版（第 218-230 行）
for i := 0; i <= p; i++ {
    if XTXInv != nil && XTXInv[i][i] >= 0 && !math.IsNaN(mse) {
        standardErrors[i] = math.Sqrt(mse * XTXInv[i][i])
        if standardErrors[i] > 0 {
            tValues[i] = coeffs[i] / standardErrors[i]
            if df > 0 {
                pValues[i] = 2.0 * studentTCDF(-math.Abs(tValues[i]), int(df))
            } else {
                pValues[i] = math.NaN()
            }
        }
    }
}

// PolynomialRegression 版（第 691-699 行）—— 完全相同，僅 p → degree
```

兩處唯一差異是迴圈上界（`p` vs `degree`），邏輯 100% 相同。

---

### 10. R² 與 Adjusted R²

`1 - sse/sst` 和 `1 - (1-r²)*(n-1)/df` 各出現 **4 次**：

| 函數 | R² 行 | Adjusted R² 行 |
|---|---|---|
| `LinearRegression` | 197 | 201 |
| `ExponentialRegression` | 372 | 373 |
| `LogarithmicRegression` | 525 | 527 |
| `PolynomialRegression` | 670 | 672-675 |

LinearRegression 和 Polynomial 還多了 `df > 0` 的邊界判斷（Exponential/Logarithmic 硬編碼 `n-2`）。

---

### 11. 兩係數模型的 CI 計算（Exponential + Logarithmic）

`ExponentialRegression`（第 402-418 行）和 `LogarithmicRegression`（第 538-554 行）有**完全相同**的 CI 計算區塊：

```go
confidenceLevel := 0.95
var ciIntercept, ciSlope [2]float64
if df > 0 {
    tDist := distuv.StudentsT{Mu: 0, Sigma: 1, Nu: float64(df)}
    criticalValue := tDist.Quantile(1 - (1-confidenceLevel)/2)
    ciIntercept = [2]float64{a - criticalValue*seA, a + criticalValue*seA}
    ciSlope     = [2]float64{b - criticalValue*seB, b + criticalValue*seB}
} else {
    ciIntercept = [2]float64{math.NaN(), math.NaN()}
    ciSlope     = [2]float64{math.NaN(), math.NaN()}
}
```

此區塊對應 `LinearRegression`（第 253-271 行）和 `PolynomialRegression`（第 705-720 行）的多係數版本，邏輯完全平行。

---

### 12. SSE / SST 累積 + meanY 計算

四個迴歸函數都有這段模式：

```go
meanY := sumY / float64(n)          // 或獨立迴圈計算
for i := range n {
    residuals[i] = ys[i] - yHat[i]
    sse += residuals[i] * residuals[i]
    sst += (ys[i] - meanY) * (ys[i] - meanY)
}
if sst == 0 { return nil }          // 零方差 guard
r2 := 1.0 - sse/sst
```

出現在 `LinearRegression:182-196`、`ExponentialRegression:354-372`、`LogarithmicRegression:502-525`、`PolynomialRegression:648-669`。

---

### 13. TwoSampleTTest 中 pooled variance 計算兩次

`ttest.go:TwoSampleTTest` 在等方差分支中，**同樣的 pooled variance 計算了兩次**：
- 第 171 行：計算 `poolVar` 用於 `standardError`
- 第 207 行：再次計算 `pooledVar` 用於 `effectSize`

兩者公式相同：`((n1-1)*var1 + (n2-1)*var2) / (n1+n2-2)`，可以共用一個變數。

---

### 14. Cohen's d EffectSizes 切片建構

```go
effectSizes := []EffectSizeEntry{{Type: "cohen_d", Value: effectSize}}
```

此模式出現 **5 次**（ttest.go 3 處 + ztest.go 2 處），每次都是單元素切片，型別名稱硬寫成字串 `"cohen_d"`。

---

### 15. 線性代數工具（gaussian elimination / matrix inversion）

`regression.go` 中私有的：
- `gaussianElimination(A, b)` — 帶部分主元的高斯消去法，求解 Ax=b
- `invertMatrix(A)` — Gauss-Jordan 矩陣求逆

這兩個函數目前**只在 regression.go 內部使用**，但屬於通用線性代數原語，與 `determinantGauss`（在 `correlation.go` 中）同屬矩陣運算，應集中放在一個 `linalg.go` 檔案，便於未來 PCA、Ridge Regression 等功能複用。

---

## 二、重構目標

1. 新增 `stats/distutil.go`：集中所有分佈相關的計算函數（不匯出）。
2. 新增 `stats/mathutil.go`：集中輔助工具函數（CI、信賴水準、誤差邊界、EffectSizes）。
3. 新增 `stats/linalg.go`：集中所有矩陣運算（高斯消去、矩陣求逆、行列式）。
4. 新增 `stats/olsutil.go`：集中 OLS 迴歸共用計算（二參數公式、推斷迴圈、R²、SSE/SST）。
5. 修改各現有檔案：改用新的共用函數，刪除重複實作。
6. **對外 API 零改動**：所有公開型別、函數簽章均不變。

---

## 三、新增檔案規格

### 3.1 `stats/distutil.go`（分佈計算原語）

此檔案為 package-private，僅供 stats 包內部使用。

```go
package stats

import (
    "math"
    "gonum.org/v1/gonum/stat/distuv"
)

// tTwoTailedPValue 回傳 Student's t 分佈的雙尾 p 值。
// 取代：calculateTPValue（ttest.go）、studentTCDF 的雙尾用法（regression.go）。
func tTwoTailedPValue(t, df float64) float64 {
    if df <= 0 {
        return 1.0
    }
    dist := distuv.StudentsT{Mu: 0, Sigma: 1, Nu: df}
    return 2 * (1 - dist.CDF(math.Abs(t)))
}

// tCDF 回傳 Student's t 分佈的 CDF（單尾）。
// 取代：studentTCDF（regression.go）。
func tCDF(t, df float64) float64 {
    if df <= 0 {
        return math.NaN()
    }
    return distuv.StudentsT{Mu: 0, Sigma: 1, Nu: df}.CDF(t)
}

// tQuantile 回傳 Student's t 分佈的 quantile（臨界值）。
// 取代：ttest.go 中 3 處、regression.go 中 4 處的 distuv.StudentsT{...}.Quantile(...) 呼叫。
func tQuantile(p, df float64) float64 {
    return distuv.StudentsT{Mu: 0, Sigma: 1, Nu: df}.Quantile(p)
}

// fOneTailedPValue 回傳 F 分佈的單尾（右尾）p 值。
// 取代：anova.go 中 3 處、ftest.go 中 3 處的 1-distuv.F{...}.CDF(f)。
func fOneTailedPValue(f, df1, df2 float64) float64 {
    return 1 - distuv.F{D1: df1, D2: df2}.CDF(f)
}

// fTwoTailedPValue 回傳 F 分佈的雙尾 p 值。
// 取代：ftest.go:43 的兩尾 F 檢定 p 值計算。
func fTwoTailedPValue(f, df1, df2 float64) float64 {
    dist := distuv.F{D1: df1, D2: df2}
    return 2 * math.Min(dist.CDF(f), 1-dist.CDF(f))
}

// chiSquaredPValue 回傳卡方分佈的右尾 p 值。
// 取代：chi_square.go:42、ftest.go:133、correlation.go:255。
func chiSquaredPValue(chi2, df float64) float64 {
    return 1 - distuv.ChiSquared{K: df}.CDF(chi2)
}

// zCDF 回傳標準常態分佈的 CDF。
// 取代：correlation.go 中的 normalCDF 手寫實作。
// 注意：直接使用 consts.go 中的 norm 物件。
func zCDF(z float64) float64 {
    return norm.CDF(z)
}
```

---

### 3.2 `stats/mathutil.go`（通用輔助工具）

```go
package stats

import "math"

// resolveConfidenceLevel 驗證並回傳信賴水準；無效值回傳預設值 0.95。
// 取代：ttest.go 中 3 處、ztest.go 中 2 處的相同驗證邏輯。
func resolveConfidenceLevel(cl float64) float64 {
    if cl > 0 && cl < 1 {
        return cl
    }
    return defaultConfidenceLevel
}

// symmetricCI 建立以 center 為中心、margin 為半徑的對稱信賴區間。
// 取代：各處 &[2]float64{center - margin, center + margin} 的重複寫法。
func symmetricCI(center, margin float64) *[2]float64 {
    ci := [2]float64{center - margin, center + margin}
    return &ci
}

// nanCI 回傳兩端皆為 NaN 的信賴區間，用於自由度不足時。
// 取代：regression.go 中多處 [2]float64{math.NaN(), math.NaN()}。
func nanCI() [2]float64 {
    return [2]float64{math.NaN(), math.NaN()}
}

// tMarginOfError 計算給定信賴水準和自由度下的 t 誤差邊界。
// 組合 tQuantile + standardError，供 ttest、regression 複用。
func tMarginOfError(confidenceLevel, df, standardError float64) float64 {
    return tQuantile(1-(1-confidenceLevel)/2, df) * standardError
}

// zMarginOfError 計算給定信賴水準下的 z 誤差邊界。
// 供 ztest 複用。
func zMarginOfError(confidenceLevel, standardError float64) float64 {
    return norm.Quantile(1-(1-confidenceLevel)/2) * standardError
}

// cohenDEffectSizes 建立單一 cohen_d EffectSizeEntry 切片。
// 取代：ttest.go 3 處、ztest.go 2 處的重複切片建構。
func cohenDEffectSizes(d float64) []EffectSizeEntry {
    return []EffectSizeEntry{{Type: "cohen_d", Value: d}}
}
```

---

### 3.3 `stats/linalg.go`（線性代數工具）

將 `regression.go` 和 `correlation.go` 中分散的矩陣運算集中到此檔案。

```go
package stats

import "math"

// gaussianElimination 使用帶部分主元的高斯消去法求解線性方程組 Ax = b。
// 回傳解向量；矩陣奇異時回傳 nil。
// 搬移自：regression.go:735-788（原 private，維持 private）。
func gaussianElimination(A [][]float64, b []float64) []float64 { /* 原邏輯不變 */ }

// invertMatrix 使用 Gauss-Jordan 消去法計算方陣的逆矩陣。
// 回傳逆矩陣；矩陣奇異時回傳 nil。
// 搬移自：regression.go:790-848（原 private，維持 private）。
func invertMatrix(A [][]float64) [][]float64 { /* 原邏輯不變 */ }

// determinantGauss 使用高斯消去法計算方陣的行列式。
// 搬移自：correlation.go:418-466（原 private，維持 private）。
func determinantGauss(matrix [][]float64) float64 { /* 原邏輯不變 */ }
```

> 注意：三個函數均為 package-private，搬移只是物理位置的整理，不改變任何呼叫語意。  
> `regression.go` 和 `correlation.go` 刪除對應函數定義後直接使用，因為同屬 `stats` 包。

---

### 3.4 `stats/olsutil.go`（OLS 迴歸共用計算）

```go
package stats

import "math"

// simpleOLSCoeffs 對資料對 (xs, ys) 執行二參數 OLS，回傳 (intercept, slope)。
// 公式：y = intercept + slope * x（Cramer's rule 解）。
// 取代：ExponentialRegression 和 LogarithmicRegression 中重複的手算過程。
// 呼叫端負責轉換輸入（例如 log(x)、log(y)）後再傳入。
// 回傳 (0, 0, false) 表示分母為零（資料共線）。
func simpleOLSCoeffs(xs, ys []float64) (intercept, slope float64, ok bool) {
    n := float64(len(xs))
    var sumX, sumY, sumXY, sumX2 float64
    for i := range xs {
        sumX  += xs[i]
        sumY  += ys[i]
        sumXY += xs[i] * ys[i]
        sumX2 += xs[i] * xs[i]
    }
    denom := n*sumX2 - sumX*sumX
    if denom == 0 {
        return 0, 0, false
    }
    slope     = (n*sumXY - sumX*sumY) / denom
    intercept = (sumY - slope*sumX) / n
    return intercept, slope, true
}

// simpleOLSSxx 計算 Σ(x - meanX)²，供後續 SE 計算使用。
func simpleOLSSxx(xs []float64) float64 {
    n := float64(len(xs))
    var sumX float64
    for _, x := range xs { sumX += x }
    meanX := sumX / n
    var sxx float64
    for _, x := range xs { d := x - meanX; sxx += d * d }
    return sxx
}

// computeGoodnessOfFit 計算殘差、SSE、SST、R²、Adjusted R²。
// yHatFn 是預測函數 yHat[i] = f(i)。
// 取代：四個迴歸函數中重複的殘差/R² 計算區塊。
func computeGoodnessOfFit(ys []float64, yHatFn func(i int) float64, df float64) (
    residuals []float64, r2, adjR2, sse float64, ok bool,
) {
    n := len(ys)
    var sumY float64
    for _, y := range ys { sumY += y }
    meanY := sumY / float64(n)

    residuals = make([]float64, n)
    var sst float64
    for i, y := range ys {
        yHat := yHatFn(i)
        residuals[i] = y - yHat
        sse += residuals[i] * residuals[i]
        sst += (y - meanY) * (y - meanY)
    }
    if sst == 0 {
        return nil, 0, 0, 0, false
    }
    r2 = 1.0 - sse/sst
    if df > 0 {
        adjR2 = 1.0 - (1.0-r2)*(float64(n-1)/df)
    } else {
        adjR2 = math.NaN()
    }
    return residuals, r2, adjR2, sse, true
}

// computeCoeffInference 從 XTXInv 對角線計算各係數的標準誤、t 值、p 值。
// 取代：LinearRegression（第 218-230 行）和 PolynomialRegression（第 691-699 行）的重複推斷迴圈。
func computeCoeffInference(coeffs []float64, XTXInv [][]float64, mse, df float64) (
    standardErrors, tValues, pValues []float64,
) {
    k := len(coeffs)
    standardErrors = make([]float64, k)
    tValues        = make([]float64, k)
    pValues        = make([]float64, k)

    for i := range k {
        if XTXInv != nil && XTXInv[i][i] >= 0 && !math.IsNaN(mse) {
            standardErrors[i] = math.Sqrt(mse * XTXInv[i][i])
            if standardErrors[i] > 0 {
                tValues[i] = coeffs[i] / standardErrors[i]
                if df > 0 {
                    pValues[i] = tTwoTailedPValue(tValues[i], df)
                } else {
                    pValues[i] = math.NaN()
                }
            }
        }
    }
    return
}

// buildTwoCoeffCIs 計算兩個係數（截距 a、斜率 b）的信賴區間。
// 取代：ExponentialRegression（第 402-418 行）和 LogarithmicRegression（第 538-554 行）的重複 CI 區塊。
func buildTwoCoeffCIs(a, b, seA, seB, df float64) (ciA, ciB [2]float64) {
    if df <= 0 {
        return nanCI(), nanCI()
    }
    criticalValue := tQuantile(1-(1-defaultConfidenceLevel)/2, df)
    ciA = [2]float64{a - criticalValue*seA, a + criticalValue*seA}
    ciB = [2]float64{b - criticalValue*seB, b + criticalValue*seB}
    return
}

// buildMultiCoeffCIs 計算多個係數的信賴區間切片。
// 取代：LinearRegression（第 253-271 行）和 PolynomialRegression（第 705-720 行）的重複 CI 迴圈。
func buildMultiCoeffCIs(coeffs, standardErrors []float64, df float64) [][2]float64 {
    k := len(coeffs)
    cis := make([][2]float64, k)
    if df <= 0 {
        for i := range cis { cis[i] = nanCI() }
        return cis
    }
    criticalValue := tQuantile(1-(1-defaultConfidenceLevel)/2, df)
    for i := range k {
        margin := criticalValue * standardErrors[i]
        cis[i] = [2]float64{coeffs[i] - margin, coeffs[i] + margin}
    }
    return cis
}
```

---

## 四、現有檔案修改規格

### 4.1 `ttest.go`

**刪除**：
- `calculateTPValue` 函數（第 371-385 行）→ 改用 `tTwoTailedPValue`

**修改**：
- `SingleSampleTTest`、`TwoSampleTTest`、`PairedTTest` 中的信賴水準解析：
  ```go
  // 舊
  var cl float64
  if len(confidenceLevel) > 0 {
      cl = confidenceLevel[0]
  } else {
      cl = defaultConfidenceLevel
  }
  if cl <= 0 || cl >= 1 {
      cl = defaultConfidenceLevel
  }
  // 新
  var rawCL float64
  if len(confidenceLevel) > 0 {
      rawCL = confidenceLevel[0]
  }
  cl := resolveConfidenceLevel(rawCL)
  ```
- t 臨界值計算：
  ```go
  // 舊
  tDist := distuv.StudentsT{Mu: 0, Sigma: 1, Nu: df}
  tCritical := tDist.Quantile(1 - (1-cl)/2)
  marginOfError := tCritical * standardError
  // 新
  marginOfError := tMarginOfError(cl, df, standardError)
  ```
- CI 建構：
  ```go
  // 舊
  ci := &[2]float64{mean - marginOfError, mean + marginOfError}
  // 新
  ci := symmetricCI(mean, marginOfError)
  ```
- p 值計算：
  ```go
  // 舊
  pValue := calculateTPValue(tValue, df)
  // 新
  pValue := tTwoTailedPValue(tValue, df)
  ```

**Import 調整**：移除 `"gonum.org/v1/gonum/stat/distuv"` 匯入（不再直接使用）。

---

### 4.2 `ztest.go`

**刪除**：
- `calculateZPValue` 函數 → 重命名為 `zPValue` 並移至 `distutil.go`（或保留原位但呼叫 `zCDF`）

  > 說明：`calculateZPValue` 在 `ztest.go` 中呼叫了 `norm.CDF`，邏輯正確。可以保留函數但以 `zCDF` 取代直接的 `norm.CDF` 呼叫，或整體移至 `distutil.go`，本計劃選擇移至 `distutil.go`（命名 `zPValue`）。

**修改**：
- 信賴水準解析：
  ```go
  // 舊（ztest.go 兩處）
  if !(confidenceLevel > 0 && confidenceLevel < 1) {
      confidenceLevel = defaultConfidenceLevel
  }
  // 新
  confidenceLevel = resolveConfidenceLevel(confidenceLevel)
  ```
- z 誤差邊界與 CI 建構：
  ```go
  // 舊
  zCritical := norm.Quantile(1 - (1-confidenceLevel)/2)
  marginOfError := zCritical * standardError
  // 新
  marginOfError := zMarginOfError(confidenceLevel, standardError)
  ```

**新增到 `distutil.go`**：
```go
// zPValue 回傳給定替代假設下的 z 檢定 p 值。
// 取代 ztest.go 中的 calculateZPValue。
func zPValue(z float64, alt AlternativeHypothesis) float64 {
    switch alt {
    case TwoSided:
        return 2 * (1 - zCDF(math.Abs(z)))
    case Greater:
        return 1 - zCDF(z)
    case Less:
        return zCDF(z)
    default:
        return math.NaN()
    }
}
```

---

### 4.3 `anova.go`

**修改**：
- 3 處 `1 - distuv.F{D1:..., D2:...}.CDF(F)` → `fOneTailedPValue(F, df1, df2)`
- 移除 `"gonum.org/v1/gonum/stat/distuv"` 匯入

具體位置：
- 第 100 行：`P := 1 - distuv.F{D1: float64(DFB), D2: float64(DFW)}.CDF(F)` → `P := fOneTailedPValue(F, float64(DFB), float64(DFW))`
- 第 219-225 行：`fd` lambda → `fd := func(d1, d2, f float64) float64 { return fOneTailedPValue(f, d1, d2) }`
- 第 309 行：同上模式

---

### 4.4 `ftest.go`

**修改**：
- `FTestForVarianceEquality` 中的兩尾 F p 值：
  ```go
  // 舊
  fDist := distuv.F{D1: df1, D2: float64(df2)}
  pValue := 2 * math.Min(fDist.CDF(fValue), 1-fDist.CDF(fValue))
  // 新
  pValue := fTwoTailedPValue(fValue, df1, float64(df2))
  ```
- `BartlettTest` 中的卡方 p 值：
  ```go
  // 舊
  pValue := 1 - distuv.ChiSquared{K: df}.CDF(chiSquared)
  // 新
  pValue := chiSquaredPValue(chiSquared, df)
  ```
- `FTestForRegression`、`FTestForNestedModels`、`oneWayANOVA` 中的單尾 F p 值 → `fOneTailedPValue`

**重構 `oneWayANOVA` 私有函數**：

目前此函數（第 193-243 行）與 `anova.go` 中的 `OneWayANOVA` 邏輯重複。`LeveneTest` 呼叫它是因為資料已轉成 raw slice + label 格式。

重構方案：將 raw slice 轉換成 `IDataList` 後直接呼叫公開的 `OneWayANOVA`：

```go
func oneWayANOVAOnSlices(values []float64, labels []int, k int) *FTestResult {
    // 按 label 分組建立 DataList
    groups := make([]*insyra.DataList, k)
    for i := range k {
        groups[i] = insyra.NewDataList()
    }
    for i, v := range values {
        groups[labels[i]].Append(v)
    }
    // 轉換為 IDataList 切片
    igroups := make([]insyra.IDataList, k)
    for i, g := range groups {
        igroups[i] = g
    }
    result := OneWayANOVA(igroups...)
    if result == nil {
        return nil
    }
    df1 := float64(result.Factor.DF)
    df2 := float64(result.Within.DF)
    return &FTestResult{
        testResultBase: testResultBase{
            Statistic: result.Factor.F,
            PValue:    result.Factor.P,
            DF:        &df1,
        },
        DF2: df2,
    }
}
```

> 注意：上方重構前提是 `OneWayANOVAResult.Factor.DF` 型別確認（目前為 `int`，轉 `float64` 時需注意）。

---

### 4.5 `chi_square.go`

**修改**：
- `calculateChiSquare` 中：
  ```go
  // 舊
  chiDist := distuv.ChiSquared{K: float64(df)}
  pValue := 1 - chiDist.CDF(chiSquare)
  // 新
  pValue := chiSquaredPValue(chiSquare, float64(df))
  ```
- 移除 `"gonum.org/v1/gonum/stat/distuv"` 匯入

---

### 4.6 `correlation.go`

**刪除**：
- `normalCDF(z float64) float64`（第 412-414 行）→ 改用 `zCDF`

**修改**：
- `kendallCorrelationWithStats` 中：
  ```go
  // 舊
  result.PValue = 2 * (1 - normalCDF(math.Abs(z)))
  // 新
  result.PValue = 2 * (1 - zCDF(math.Abs(z)))
  ```
- `pearsonCorrelationWithStats` 中（第 289-295 行）：
  ```go
  // 舊
  tDist := distuv.StudentsT{Mu: 0, Sigma: 1, Nu: n - 2}
  pValue := 2 * tDist.CDF(-math.Abs(tStat))
  // 新
  pValue := tTwoTailedPValue(tStat, n-2)
  ```
- `spearmanCorrelationWithStats` 中（第 393-398 行）：同上模式
- `BartlettSphericity` 中（第 255 行）：
  ```go
  // 舊
  pval := 1 - distuv.ChiSquared{K: float64(degreesOfFreedom)}.CDF(chisq)
  // 新
  pval := chiSquaredPValue(chisq, float64(degreesOfFreedom))
  ```
- 移除 `"gonum.org/v1/gonum/stat/distuv"` 匯入

---

### 4.7 `regression.go`

**搬移到 `linalg.go`（刪除定義，保留呼叫）**：
- `gaussianElimination`（第 735-788 行）
- `invertMatrix`（第 790-848 行）
- `studentTCDF`（第 850-863 行）→ 改用 `tCDF`（distutil.go）

**修改：t 分佈計算**：
- 所有 `2.0 * studentTCDF(-math.Abs(t), int(df))` → `tTwoTailedPValue(t, df)`
  - 第 224 行、第 400-401 行、第 535-536 行、第 697 行（共 6 處）
- 所有直接建立 `distuv.StudentsT{...}.Quantile(...)` → 改用 `tMarginOfError` 或 `buildTwoCoeffCIs` / `buildMultiCoeffCIs`

**修改：`LinearRegression`**：
- 係數推斷迴圈（第 218-230 行）→ `se, tv, pv := computeCoeffInference(coeffs, XTXInv, mse, df)`
- CI 計算（第 253-271 行）→ `confIntervals := buildMultiCoeffCIs(coeffs, standardErrors, df)`

**修改：`ExponentialRegression`**：
- 手算 `sumX/sumY/sumXY/sumX2` + Cramer's rule（第 327-346 行）→ `lnA, b, ok := simpleOLSCoeffs(xs, logYs)`
- 殘差/R² 區塊（第 349-372 行）→ `residuals, r2, adjR2, _, ok := computeGoodnessOfFit(ys, func(i int) float64 { return a * math.Exp(b*xs[i]) }, df)`
- CI 區塊（第 402-418 行）→ `ciIntercept, ciSlope := buildTwoCoeffCIs(a, b, seA, seB, df)`

**修改：`LogarithmicRegression`**：
- 手算 `sumLX/sumYLX/sumLX2` + Cramer's rule（第 467-500 行）→ `a, b, ok := simpleOLSCoeffs(logXs, ys)`
- `sxxLX` 計算（第 502-518 行）→ `sxx := simpleOLSSxx(logXs)`
- 殘差/R² 區塊 → `computeGoodnessOfFit`
- CI 區塊（第 538-554 行）→ `buildTwoCoeffCIs(a, b, seA, seB, df)`

**修改：`PolynomialRegression`**：
- 係數推斷迴圈（第 691-699 行）→ `computeCoeffInference(coeffs, XTXInv, mse, df)`
- CI 迴圈（第 705-720 行）→ `buildMultiCoeffCIs(coeffs, standardErrors, df)`
- 殘差/R² 區塊 → `computeGoodnessOfFit`

**修改：`TwoSampleTTest`（ttest.go）pooled variance 計算兩次**：
- 第 171 行計算 `poolVar` 後，第 207 行重算相同公式 → 提取為共用變數，等方差分支只計算一次：
  ```go
  // 舊：第 171 行
  poolVar := ((float64(n1-1)*var1 + float64(n2-1)*var2) / float64(n1+n2-2))
  standardError = math.Sqrt(poolVar * (1/n1Float + 1/n2Float))
  // ...後來第 207 行又算一次
  pooledVar := ((n1Float-1)*var1 + (n2Float-1)*var2) / (n1Float + n2Float - 2)
  pooledStd := math.Sqrt(pooledVar)
  effectSize = meanDiff / pooledStd

  // 新：只算一次
  pooledVar := ((n1Float-1)*var1 + (n2Float-1)*var2) / (n1Float + n2Float - 2)
  standardError = math.Sqrt(pooledVar * (1/n1Float + 1/n2Float))
  // ...
  effectSize = meanDiff / math.Sqrt(pooledVar)
  ```

**修改：EffectSizes 切片建構**：
- `ttest.go` 3 處 + `ztest.go` 2 處的 `[]EffectSizeEntry{{Type: "cohen_d", Value: v}}` → `cohenDEffectSizes(v)`

**移除**：`"gonum.org/v1/gonum/stat/distuv"` 匯入（regression.go 不再直接使用）

---

## 五、重構完成後架構圖

```
stats/
├── distutil.go    ← 新增：分佈原語（t/z/F/χ² 的 CDF、quantile、p 值）
├── mathutil.go    ← 新增：輔助工具（CI、信賴水準、誤差邊界、EffectSizes）
├── linalg.go      ← 新增：線性代數（gaussianElimination、invertMatrix、determinantGauss）
├── olsutil.go     ← 新增：OLS 共用計算（simpleOLSCoeffs、computeGoodnessOfFit、
│                            computeCoeffInference、buildTwoCoeffCIs、buildMultiCoeffCIs）
├── consts.go      ← 不變（norm 物件保留）
├── structs.go     ← 不變
├── types.go       ← 不變
├── init.go        ← 不變
├── ttest.go       ← 修改：刪除 calculateTPValue；改用共用函數；修復 pooledVar 重算；
│                           cohenDEffectSizes 取代切片字面量
├── ztest.go       ← 修改：刪除 calculateZPValue；改用 zPValue、zMarginOfError；
│                           cohenDEffectSizes 取代切片字面量
├── anova.go       ← 修改：F p 值改用 fOneTailedPValue
├── ftest.go       ← 修改：重構 oneWayANOVA；各 p 值改用共用函數
├── chi_square.go  ← 修改：chi2 p 值改用 chiSquaredPValue
├── correlation.go ← 修改：刪除 normalCDF 和 determinantGauss；改用 zCDF/tTwoTailedPValue
├── regression.go  ← 修改：刪除 studentTCDF/gaussianElimination/invertMatrix；
│                           改用 olsutil 和 linalg 函數
├── pca.go         ← 不需修改
├── moments.go     ← 不需修改
├── skewness.go    ← 不需修改
├── kurtosis.go    ← 不需修改
└── diag.go        ← 不需修改
```

**函數對應總覽**：

| 舊（分散各處） | 新（集中位置） |
|---|---|
| `calculateTPValue`（ttest.go）、`studentTCDF`（regression.go） | `tTwoTailedPValue`（distutil.go） |
| `distuv.StudentsT{...}.CDF(t)`（4 處）| `tCDF`（distutil.go）|
| `distuv.StudentsT{...}.Quantile(...)`（7 處）| `tQuantile`（distutil.go）|
| `calculateZPValue`（ztest.go）| `zPValue`（distutil.go）|
| `normalCDF`（correlation.go）| `zCDF`（distutil.go）|
| `1 - distuv.F{...}.CDF(f)`（6 處）| `fOneTailedPValue`（distutil.go）|
| `2*min(CDF,1-CDF)`（ftest.go）| `fTwoTailedPValue`（distutil.go）|
| `1 - distuv.ChiSquared{...}.CDF(x)`（3 處）| `chiSquaredPValue`（distutil.go）|
| 信賴水準驗證（5 處）| `resolveConfidenceLevel`（mathutil.go）|
| `&[2]float64{c-m, c+m}`（多處）| `symmetricCI`（mathutil.go）|
| `[2]float64{NaN, NaN}`（多處）| `nanCI`（mathutil.go）|
| `[]EffectSizeEntry{{...}}`（5 處）| `cohenDEffectSizes`（mathutil.go）|
| Cramer's rule（ExponentialRegression、LogarithmicRegression）| `simpleOLSCoeffs`（olsutil.go）|
| 殘差/SSE/SST/R²（4 個迴歸函數）| `computeGoodnessOfFit`（olsutil.go）|
| 係數推斷迴圈（LinearRegression、PolynomialRegression）| `computeCoeffInference`（olsutil.go）|
| 兩係數 CI（ExponentialRegression、LogarithmicRegression）| `buildTwoCoeffCIs`（olsutil.go）|
| 多係數 CI（LinearRegression、PolynomialRegression）| `buildMultiCoeffCIs`（olsutil.go）|
| `gaussianElimination`、`invertMatrix`（regression.go）| linalg.go |
| `determinantGauss`（correlation.go）| linalg.go |
| `oneWayANOVA` private（ftest.go）| 重構為呼叫 `OneWayANOVA`（anova.go）|
| pooledVar 計算兩次（TwoSampleTTest）| 合併為一次計算 |

---

## 六、對外 API 保證

以下公開函數/型別**完全不改動**：

| 函數/型別 | 簽章變更 |
|---|---|
| `SingleSampleTTest`, `TwoSampleTTest`, `PairedTTest` | 無 |
| `SingleSampleZTest`, `TwoSampleZTest` | 無 |
| `OneWayANOVA`, `TwoWayANOVA`, `RepeatedMeasuresANOVA` | 無 |
| `LeveneTest`, `BartlettTest`, `FTestForVarianceEquality` 等 | 無 |
| `LinearRegression`, `PolynomialRegression`, `ExponentialRegression`, `LogarithmicRegression` | 無 |
| `Correlation`, `CorrelationMatrix`, `CorrelationAnalysis` | 無 |
| `ChiSquareGoodnessOfFit`, `ChiSquareIndependenceTest` | 無 |
| `PCA` | 無 |
| 所有 Result 型別（`TTestResult`, `ZTestResult` 等） | 無 |
| `AlternativeHypothesis`, `TwoSided`, `Greater`, `Less` | 無 |
| `CorrelationMethod`, `PearsonCorrelation` 等常數 | 無 |

受影響的全是 package-private 函數（`calculateTPValue`、`studentTCDF`、`calculateZPValue`、`normalCDF`、`oneWayANOVA`、`gaussianElimination`、`invertMatrix`、`determinantGauss`），對外部使用者零影響。

---

## 七、實作順序

1. **新增 `linalg.go`**：搬移 `gaussianElimination`、`invertMatrix`（來自 regression.go）、`determinantGauss`（來自 correlation.go）
2. **新增 `distutil.go`**：`tTwoTailedPValue`、`tCDF`、`tQuantile`、`fOneTailedPValue`、`fTwoTailedPValue`、`chiSquaredPValue`、`zCDF`、`zPValue`
3. **新增 `mathutil.go`**：`resolveConfidenceLevel`、`symmetricCI`、`nanCI`、`tMarginOfError`、`zMarginOfError`、`cohenDEffectSizes`
4. **新增 `olsutil.go`**：`simpleOLSCoeffs`、`simpleOLSSxx`、`computeGoodnessOfFit`、`computeCoeffInference`、`buildTwoCoeffCIs`、`buildMultiCoeffCIs`
5. **修改 `regression.go`**：刪除已搬移的 3 個私有函數；改用 olsutil、linalg、distutil 函數
6. **修改 `correlation.go`**：刪除 `normalCDF`、`determinantGauss`；改用 distutil 函數
7. **修改 `ttest.go`**：刪除 `calculateTPValue`；修復 pooledVar；改用 mathutil/distutil
8. **修改 `ztest.go`**：刪除 `calculateZPValue`；改用 mathutil/distutil
9. **修改 `anova.go`**：F p 值改用 `fOneTailedPValue`
10. **修改 `ftest.go`**：重構 `oneWayANOVA`；各 p 值改用共用函數
11. **修改 `chi_square.go`**：卡方 p 值改用 `chiSquaredPValue`
12. **執行 `go test ./stats/...`**：確認所有測試通過，無回歸

> 建議先完成步驟 1-4（新增檔案），確認編譯通過後再逐步修改現有檔案。每完成一個現有檔案就跑一次測試。

---

## 八、潛在風險與備注

| 風險 | 說明 | 緩解方案 |
|---|---|---|
| `oneWayANOVA` 重構後效能差異 | 原版直接操作 slice，新版需建立 DataList 物件 | LeveneTest 資料量通常小；若有問題可保留 slice 路徑但共用 F p 值計算 |
| `tTwoTailedPValue` 邊界行為 | `calculateTPValue` 在 `df<=0` 回傳 `1.0`；`studentTCDF` 回傳 `NaN` | 統一為 `df<=0` 回傳 `1.0`（已納入規格） |
| `studentTCDF(t, int(df))` 介面不一致 | 原版接受 `int`，呼叫處有 `int()` 轉換 | 新 `tCDF` 接受 `float64`，同步移除 `int()` 轉換 |
| `computeGoodnessOfFit` 的 `yHatFn` closure 效能 | 每次呼叫一個 closure 而非 inline | 迴歸函數通常 n < 10^6，overhead 可忽略 |
| `simpleOLSCoeffs` 不支援平行化 | 原 LogarithmicRegression 有 `parallel.GroupUp` | 新函數為順序版；若需要平行化可保留原實作，但至少 Cramer's rule 部分統一 |
| `buildTwoCoeffCIs` 硬編碼 `defaultConfidenceLevel` | 原函數信賴水準也是硬編碼 0.95 | 行為一致，未來若需參數化兩處同步修改即可 |
| `TwoSampleTTest` pooledVar 重算修正 | 邏輯上等價，但需確認數值結果不變 | 測試覆蓋等方差路徑的所有回傳值（t、p、CI、effectSize）|
