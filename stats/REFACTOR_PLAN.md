# stats 包重構計劃：抽取共用統計原語

> 日期：2026-04-07  
> 分支：enhance/stats  
> 目標：消除 stats 包內重複的統計分佈計算、p 值計算、信賴區間建構等邏輯，統一為單一實作，**不改變對外 API**。

---

## 一、現況分析：重複造輪子清單

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

## 二、重構目標

1. 新增 `stats/distutil.go`：集中所有分佈相關的計算函數（不匯出）。
2. 新增 `stats/mathutil.go`：集中輔助工具函數。
3. 修改各現有檔案：改用新的共用函數，刪除重複實作。
4. **對外 API 零改動**：所有公開型別、函數簽章均不變。

---

## 三、新增檔案規格

### 3.1 `stats/distutil.go`

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

### 3.2 `stats/mathutil.go`

```go
package stats

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

// tMarginOfError 計算給定信賴水準和自由度下的 t 誤差邊界。
// 組合 tQuantile + marginOfError 計算，供 ttest、regression 複用。
func tMarginOfError(confidenceLevel, df, standardError float64) float64 {
    alpha := (1 - confidenceLevel) / 2
    return tQuantile(1-alpha, df) * standardError
}

// zMarginOfError 計算給定信賴水準下的 z 誤差邊界。
// 供 ztest 複用。
func zMarginOfError(confidenceLevel, standardError float64) float64 {
    return norm.Quantile(1-(1-confidenceLevel)/2) * standardError
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

**刪除**：
- `studentTCDF(t float64, df int) float64`（第 850-863 行）→ 改用 `tCDF`

**修改**：
- 所有 `2.0 * studentTCDF(-math.Abs(t), int(df))` → `tTwoTailedPValue(t, df)`
  - 第 224 行、第 400-401 行、第 535-536 行、第 697 行（共 6 處）
- 所有 `distuv.StudentsT{Mu: 0, Sigma: 1, Nu: df}.Quantile(...)` → `tQuantile(p, df)` + `tMarginOfError`
  - 第 258 行、第 407 行、第 543 行、第 709 行（共 4 處）
- CI 建構 → `symmetricCI`
- 移除 `"gonum.org/v1/gonum/stat/distuv"` 匯入

---

## 五、重構完成後架構圖

```
stats/
├── distutil.go      ← 新增：所有分佈計算原語（t, z, F, χ², p值, quantile）
├── mathutil.go      ← 新增：通用工具（CI, 信賴水準解析, 誤差邊界）
├── consts.go        ← 不變（norm 物件保留，供 distutil.go 引用）
├── structs.go       ← 不變
├── types.go         ← 不變
├── init.go          ← 不變
├── ttest.go         ← 修改：移除重複邏輯，呼叫共用函數
├── ztest.go         ← 修改：移除 calculateZPValue 定義，呼叫 distutil
├── anova.go         ← 修改：F p值改用 fOneTailedPValue
├── ftest.go         ← 修改：移除/重構 oneWayANOVA，使用共用 p 值函數
├── chi_square.go    ← 修改：chi2 p值改用 chiSquaredPValue
├── correlation.go   ← 修改：移除 normalCDF，使用 zCDF/tTwoTailedPValue
├── regression.go    ← 修改：移除 studentTCDF，使用 tCDF/tTwoTailedPValue
├── pca.go           ← 不需修改（無重複分佈計算）
├── moments.go       ← 不需修改
├── skewness.go      ← 不需修改
├── kurtosis.go      ← 不需修改
└── diag.go          ← 不需修改
```

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

唯一可能的影響：`calculateTPValue` 和 `studentTCDF` 是私有函數，可以直接刪除/重命名，不影響外部使用者。

---

## 七、實作順序

1. **新增 `distutil.go`**（含 `tTwoTailedPValue`, `tCDF`, `tQuantile`, `fOneTailedPValue`, `fTwoTailedPValue`, `chiSquaredPValue`, `zCDF`, `zPValue`）
2. **新增 `mathutil.go`**（含 `resolveConfidenceLevel`, `symmetricCI`, `tMarginOfError`, `zMarginOfError`）
3. **修改 `ttest.go`**（刪除 `calculateTPValue`，更新三個公開函數）
4. **修改 `ztest.go`**（刪除 `calculateZPValue`，更新兩個公開函數）
5. **修改 `correlation.go`**（刪除 `normalCDF`，更新各處呼叫）
6. **修改 `anova.go`**（更新 F p 值）
7. **修改 `ftest.go`**（重構 `oneWayANOVA`，更新各 p 值計算）
8. **修改 `chi_square.go`**（更新卡方 p 值）
9. **修改 `regression.go`**（刪除 `studentTCDF`，更新所有 t 計算）
10. **執行 `go test ./stats/...`** 確認所有測試通過

---

## 八、潛在風險與備注

| 風險 | 說明 | 緩解 |
|---|---|---|
| `oneWayANOVA` 重構後效能差異 | 原版直接操作 slice，新版需建立 DataList 物件 | LeveneTest 的資料量通常不大；若有效能問題可保留原 slice 版本但共用 F p值計算 |
| `tTwoTailedPValue` 與 `calculateTPValue` 的邊界行為差異 | `calculateTPValue` 在 `df<=0` 時回傳 `1.0`；新函數需保持一致 | 已納入規格：`if df <= 0 { return 1.0 }` |
| `regression.go` 中 `studentTCDF(t, int(df))` 接受 `int`，新版接受 `float64` | 呼叫端已做 `int(df)` 轉換，改為直接傳 `df float64` 即可 | 修改時同步移除 `int()` 轉換 |
| `PairedTTest` 中 `confidenceLevel` 為 variadic，但 `ztest` 中是必要參數 | `resolveConfidenceLevel` 需對應不同呼叫模式 | ttest 用 variadic 展開後傳入；ztest 直接傳入 |
