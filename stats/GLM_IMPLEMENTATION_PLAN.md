# GLM 實作計畫(Issue #164)

> 廣義線性模型(Generalized Linear Models)— Logistic / Poisson / 通用 GLM
> 分支:`feat/glm-stats`(自 `dev`)
> 對拍 baseline:**R `stats::glm()`(主)** + Python `statsmodels.GLM`(輔)
>
> 本文件是 issue #164 的完整、可執行實作規格,涵蓋**全部階段**(非僅 Phase 1)。
> 每一節都對齊 [stats/CLAUDE.md](CLAUDE.md) 的分層架構與「複用既有建構塊」原則。

---

## 目錄

1. [設計定案(對 issue 原文的修正)](#1-設計定案對-issue-原文的修正)
2. [檔案佈局與分層](#2-檔案佈局與分層)
3. [數學規格(IRLS / Family / Link / 推論)](#3-數學規格)
4. [R / Python baseline 對拍規格](#4-r--python-baseline-對拍規格)
5. [Phase 0 — 地基與重構](#phase-0--地基與重構)
6. [Phase 1 — IRLS 核心 + Family/Link](#phase-1--irls-核心--familylink)
7. [Phase 2 — LogisticRegression / PoissonRegression 公開 API](#phase-2--logisticregression--poissonregression-公開-api)
8. [Phase 3 — 通用 GLM 入口 + Predict](#phase-3--通用-glm-入口--predict)
9. [Phase 4 — isr 暴露](#phase-4--isr-暴露)
10. [Phase 5 — Separation ridge / over-dispersion 強化](#phase-5--separation-ridge--over-dispersion-強化)
11. [Phase 6 — 文件、決策樹、CI 全綠](#phase-6--文件決策樹ci-全綠)
12. [明確延後(後續 issue)](#12-明確延後後續-issue)
13. [驗收條件對照表](#13-驗收條件對照表)

---

## 1. 設計定案(對 issue 原文的修正)

issue 原文的方向正確,但有幾個會在實作時卡住的點,**本計畫採用以下定案**:

| # | issue 原文 | 定案 | 理由 |
|---|---|---|---|
| D1 | 把 GLM 放在 `stats/glm/` **子套件** | **不開子套件**,檔案攤平放 `package stats`,以 `glm_*.go` 前綴分組 | 子套件看不到 `package stats` 的未匯出 helper(`computeCoeffInference`、`zCDF`、`buildMultiCoeffCIs`、`resolveConfidenceLevel`),原文「重用 olsutil.go」與「子套件」無法並存。攤平才符合既有「一層一檔、同 package 共用」結構。 |
| D2 | IRLS 收斂後直接套 `computeCoeffInference` / `buildMultiCoeffCIs` | **新增 Layer-3 `glmutil.go`** 走 z(非 t)、共變異數用 `φ·(XᵀWX)⁻¹`(非 `mse·(XᵀX)⁻¹`)、CI 吃 `ConfidenceLevel` | `computeCoeffInference` 綁 t 分布且做 `mse` 縮放;`buildMultiCoeffCIs` 把信賴水準 **hardcode 成 `defaultConfidenceLevel`**([olsutil.go:154](olsutil.go))。GLM 不能直接重用。 |
| D3 | `SeparationPolicy` 含 `ridge`,Phase 1 就實作 | `warn` / `error` 進 Phase 2;**`ridge` 延到 Phase 5**,且若啟用必須在 result 標記 SE 為 penalized | ridge 會改變估計量並讓回報的 SE/p 變成 penalized-likelihood 值,與使用者預期的 MLE 推論不同;混進 p 值很危險。 |
| D4 | doc comment 用 `WithPositiveClass()` / `WithOffset()`,型別又用 Options struct | **統一用 Options struct**(`PositiveClass any`、`Offset IDataList`),移除 `With*` 字樣 | 與 `LogisticRegressionWithOptions(opts, ...)` 命名一致;避免兩套並存。 |
| D5 | 二元正類預設「第二個出現的值」 | **預設「兩相異值排序後較大者為正類」**(對齊 R factor 最後一個 level),決定性、可文件化;仍允許 `PositiveClass` 覆寫 | 「第二個出現」order-dependent、不決定性。 |
| D6 | (未提) | **抽 `gatherRegressionInputs` / `buildDesignMatrix`** 共用輸入層,LinearRegression 與 GLM 與 Predict 共用 | [regression.go:140-194](regression.go) 的「並行 AtomicDo 拉資料→驗長度→建 design matrix」會被整段複製。 |
| D7 | (未提) | **Gaussian + identity GLM 對拍 `LinearRegression`** 當免費內部 smoke test(不需 R) | 等價性是最便宜的正確性保證。 |

---

## 2. 檔案佈局與分層

全部在 `package stats`(D1)。括號為 CLAUDE.md 分層。

```
stats/
  distutil.go            (Layer 0, 擴充)  + zQuantile(p)
  consts.go              (擴充)            + GLM 相關列舉常數
  glm_links.go           (Layer 1, 新)     Link 介面 + Logit/Log/Identity(+ Probit/Cloglog hook)
  glm_families.go        (Layer 1, 新)     Family 介面 + Binomial/Poisson/Gaussian
  regression_shared.go   (Layer 3, 新)     gatherRegressionInputs / buildDesignMatrix(D6)
  glm_irls.go            (Layer 3, 新)     fitIRLS 核心 + 收斂 + null 模型 + separation 偵測
  glmutil.go             (Layer 3, 新)     GLM 專用 z 推論 / z 版 CI / pseudo-R² / dispersion
  regression_glm.go      (Layer 4, 新)     GLM / GLMResult / GLMOptions / (*GLMResult).Predict
  regression_logistic.go (Layer 4, 新)     LogisticRegression(+Options) / Result / Predict
  regression_poisson.go  (Layer 4, 新)     PoissonRegression(+Options) / Result / Predict

  glm_links_test.go  glm_families_test.go  glm_irls_test.go  glmutil_test.go
  regression_logistic_test.go  regression_poisson_test.go  regression_glm_test.go
  crosslang_public_methods_test.go  (擴充: logistic / poisson / glm 三組對拍)
  testdata/crosslang_baseline.R   (擴充: logistic_reg / poisson_reg / glm 方法)
  testdata/crosslang_baseline.py  (擴充: 同上)

isr/
  glm.go        (新)  LogisticRegression / PoissonRegression / GLM 薄包裝 + 型別 alias
  glm_test.go   (新)
```

**分層規則(CLAUDE.md §分層架構):Layer N 只能呼叫 Layer 0~N-1。**
`glm_irls.go`(L3)呼叫 `glm_families.go`/`glm_links.go`(L1)、`gonum/mat`(L0)、`parutil`(內部);
`glmutil.go`(L3)呼叫 `distutil.go`(L0,`zPValue`/`zQuantile`)、`mathutil.go`(L2);
Layer 4 的 `regression_*.go` 不得放任何私有計算函式(CLAUDE.md §步驟 5)。

---

## 3. 數學規格

所有公式以 **R `glm()` 慣例** 為準(deviance / logLik / AIC 對齊 `stats::glm`)。

### 3.1 Link 介面(`glm_links.go`, Layer 1)

```go
type glmLink interface {
    eta(mu float64) float64        // g(μ)         link
    mu(eta float64) float64        // g⁻¹(η)       inverse link (mean function)
    muEta(eta float64) float64     // dμ/dη        for working weights/response
    name() string
}
```

| Link | g(μ)=η | μ=g⁻¹(η) | dμ/dη | 數值穩定 |
|---|---|---|---|---|
| `logitLink` | `log(μ/(1-μ))` | `sigmoid(η)` | `μ(1-μ)` | sigmoid 用 η≥0 / η<0 兩條 path 避免 `exp` 溢位;μ clamp 到 `[ε, 1-ε]`,ε=1e-10 |
| `logLink` | `log(μ)` | `exp(η)` | `exp(η)` | η 上限 clamp(避免 `exp` Inf,如 η≤700) |
| `identityLink` | `μ` | `η` | `1` | — |
| `probitLink`(hook,延後) | `Φ⁻¹(μ)` | `Φ(η)` | `φ(η)` | — |
| `cloglogLink`(hook,延後) | `log(-log(1-μ))` | `1-exp(-exp(η))` | — | — |

> `sigmoid` 放 [mathutil.go](mathutil.go)(Layer 2 工具),供 logit link 與 logistic Predict 共用。
> 介面以 `muEta(eta)` 而非 `gPrime(mu)` 表達,因 IRLS 直接需要 `dμ/dη`(= 1/g'(μ)),少一次倒數。

### 3.2 Family 介面(`glm_families.go`, Layer 1)

```go
type glmFamily interface {
    canonicalLink() glmLink
    variance(mu float64) float64                  // V(μ)
    devianceResidualSq(y, mu, w float64) float64  // 單點 deviance 貢獻(已乘 prior weight)
    logLikContrib(y, mu, w float64) float64       // 單點 log-likelihood(含常數項,供 AIC)
    initMu(y, w float64) float64                  // IRLS 初始 μ
    dispersionFixed() (phi float64, fixed bool)   // binomial/poisson → (1, true)
    name() string
}
```

| Family | V(μ) | canonical link | initMu | dispersion |
|---|---|---|---|---|
| `binomialFamily` | `μ(1-μ)` | logit | `(y+0.5)/2` | (1, true) |
| `poissonFamily` | `μ` | log | `y+0.1` | (1, true) |
| `gaussianFamily` | `1` | identity | `y` | (φ=Pearson χ²/df, false) |

**Deviance 貢獻(R 慣例,0log0≡0):**

- Binomial(0/1):`dᵢ = 2w[ y·log(y/μ) + (1-y)·log((1-y)/(1-μ)) ]`
- Poisson:`dᵢ = 2w[ y·log(y/μ) − (y−μ) ]`,`y=0` 時 `y·log(y/μ)=0`
- Gaussian:`dᵢ = w·(y−μ)²`

**log-likelihood 貢獻(含常數項,AIC 必須對齊 R):**

- Binomial:`w[ y·log μ + (1−y)·log(1−μ) ]`
- Poisson:`w[ y·log μ − μ − lgamma(y+1) ]` ← **`lgamma(y+1)` 不可省**,否則 AIC 對不上 R
- Gaussian:`−0.5·w[ log(2πφ) + (y−μ)²/φ ]`(φ 為估計 dispersion)

`Deviance = Σdᵢ`;`AIC = −2·logLik + 2·k`(k = 參數數,Gaussian 額外 +1 計 dispersion,對齊 R);`BIC = −2·logLik + k·log(n)`。

### 3.3 IRLS 核心(`glm_irls.go`, Layer 3)

```go
type irlsOptions struct {
    maxIter   int        // 預設 25(對齊 R glm.control maxit)
    tolerance float64    // 預設 1e-8(deviance 相對變化)
    offset    []float64  // 長度 n,可為 nil(視為 0)
    weights   []float64  // prior weights,可為 nil(視為全 1)
    ridge     float64    // L2 penalty(Phase 5;Phase 1~4 恒為 0)
}

type irlsFit struct {
    beta       []float64    // β₀..βₖ
    covUnscaled [][]float64  // (XᵀWX)⁻¹,最終一輪;乘 dispersion 後即 Wald 共變異數
    eta        []float64    // 線性預測子(含 offset)
    mu         []float64    // 擬合均值
    weightsW   []float64    // 最終 IRLS working weights(供 Pearson dispersion)
    deviance   float64
    iterations int
    converged  bool
    sepFlag    bool         // separation 疑慮(|β| 或 SE 爆炸)
}

func fitIRLS(X *mat.Dense, y []float64, fam glmFamily, link glmLink, opts irlsOptions) (*irlsFit, error)
```

**演算法(每輪):**

```
初始化 μᵢ = fam.initMu(yᵢ, wᵢ);ηᵢ = link.eta(μᵢ)
loop:
  dμdηᵢ = link.muEta(ηᵢ)
  Wᵢ    = wᵢ · dμdηᵢ² / fam.variance(μᵢ)          // working weight
  zᵢ    = (ηᵢ − offsetᵢ) + (yᵢ − μᵢ) / dμdηᵢ        // working response
  解 (XᵀWX + ridge·I') β = XᵀW z                    // mat.SolveVec;I' 不含截距列
  ηᵢ    = (Xβ)ᵢ + offsetᵢ
  μᵢ    = link.mu(ηᵢ)            // 含 clamp
  dev   = Σ fam.devianceResidualSq(yᵢ, μᵢ, wᵢ)
  if |dev − dev_old| / (|dev| + 0.1) < tol: converged; break
收斂後:covUnscaled = (XᵀWX)⁻¹(最終 W);存 weightsW
```

- `XᵀWX` 與 `XᵀWz` 的建構:大型 design matrix 走 [parutil](internal/parutil) 平行 reduce(對齊 [regression.go:188](regression.go) 的 `n·(p+1) >= 50_000` 門檻判斷)。
- 解線性系統用 `mat.SolveVec`;反矩陣用 `mat.Dense.Inverse`(與 [regression.go:34](regression.go) `solveOLS` 同模式,可考慮把 `solveWeightedNormalEquations` 抽成共用)。
- **Separation 偵測**:收斂後若 `max|βⱼ| > sepBetaThreshold`(預設 30)或任一對角 `covUnscaled[j][j]` 異常大(SE > sepSEThreshold,預設 1e3)→ `sepFlag = true`。Binomial 專屬。
- **數值穩定**:μ clamp(binomial `[1e-10, 1-1e-10]`);η clamp(log link 上限 700);若 `dev` 變 `NaN`/`Inf` 立即停並回 `converged=false`。

**Null 模型**:`fitNullIRLS(y, fam, link, offset, weights)` 只用截距欄(X = 全 1 的 n×1),回傳 `nullDeviance`、`nullLogLik`,供 pseudo-R²。

### 3.4 GLM 推論(`glmutil.go`, Layer 3)— **D2 核心**

```go
// 走 Wald z(非 t);共變異數 = φ · (XᵀWX)⁻¹(非 mse 縮放)。
func computeGLMInference(beta []float64, covUnscaled [][]float64, dispersion float64) (se, z, p []float64)

// z 版多係數 CI,吃 confidence level(修正 OLS 版 hardcode defaultConfidenceLevel)
func buildGLMCoeffCIs(beta, se []float64, cl float64) [][2]float64

// pseudo-R²(logistic 用)
func mcFaddenR2(logLik, nullLogLik float64) float64
func coxSnellR2(logLik, nullLogLik float64, n int) float64
func nagelkerkeR2(logLik, nullLogLik float64, n int) float64  // = coxSnell / (1 - exp(2·nullLogLik/n))

// dispersion(gaussian / over-dispersion 檢查用)
func pearsonChiSq(y, mu, weightsPriorW []float64, fam glmFamily) float64
func pearsonDispersion(pearsonChi2 float64, dfResidual int) float64  // χ²/df
```

- `se_j = sqrt(dispersion · covUnscaled[j][j])`(binomial/poisson φ=1)
- `z_j = β_j / se_j`;`p_j = zPValue(z_j, TwoSided)`(複用 [distutil.go:48](distutil.go))
- `CI_j = β_j ± zcrit·se_j`,`zcrit = zQuantile(1-(1-cl)/2)`(Phase 0 新增,見下)
- Odds ratio / IRR:`exp(β_j)`;其 CI = `exp(CI_j)`(對 CI 端點取 exp,單調保序)

### 3.5 Predict

```go
type PredictType string
const (
    PredictLinear   PredictType = "linear"   // η = Xβ
    PredictResponse PredictType = "response" // g⁻¹(η)
    PredictClass    PredictType = "class"    // logistic only:μ ≥ threshold(預設 0.5)→ 正類
)
```

Predict 重建 design matrix(複用 `buildDesignMatrix`,D6)→ `η = Xβ`(若提供 offset 則加)→ 依 type 回 `η` / `g⁻¹(η)` / 類別標籤(對映回 `ClassLabels`)。回傳 `(*insyra.DataList, error)`。

---

## 4. R / Python baseline 對拍規格

對拍機制沿用既有 [crosslang_helpers_test.go](crosslang_helpers_test.go):
`runRBaseline(t, method, payload)` → 跑 [testdata/crosslang_baseline.R](testdata/crosslang_baseline.R)(method dispatch + `toJSON(out, digits=16)`),結果由 SHA256(script+method+payload)快取於 `testdata/baseline_cache/`。**改 .R 腳本會自動失效快取。**

### 4.1 R baseline(主)— 用 base R `stats::glm()`

新增三個 method 分支到 `crosslang_baseline.R`。**R 套件:base `stats`(內建 glm)+ 既有 `jsonlite`,零新增相依。**

```r
} else if (method == "logistic_reg") {
  y  <- as.double(unlist(payload$y))
  xs <- lapply(payload$xs, function(v) as.double(unlist(v)))
  df <- data.frame(y = y, do.call(cbind, xs))
  names(df) <- c("y", paste0("x", seq_along(xs)))
  fit <- glm(y ~ ., data = df, family = binomial(link = "logit"),
             control = glm.control(epsilon = 1e-10, maxit = 100))
  s   <- summary(fit)$coefficients          # Estimate / Std.Error / z value / Pr(>|z|)
  ci  <- confint.default(fit)               # Wald CI(z 版),不是 profile confint()！
  out <- list(
    coefficients   = unname(s[, "Estimate"]),
    standard_errors= unname(s[, "Std. Error"]),
    z_values       = unname(s[, "z value"]),
    p_values       = unname(s[, "Pr(>|z|)"]),
    ci             = unname(cbind(ci[, 1], ci[, 2])),
    odds_ratios    = unname(exp(s[, "Estimate"])),
    log_likelihood = as.numeric(logLik(fit)),
    null_deviance  = fit$null.deviance,
    deviance       = fit$deviance,
    aic            = fit$aic,
    bic            = as.numeric(BIC(fit)),
    iterations     = fit$iter,
    fitted         = unname(fitted(fit))
  )
} else if (method == "poisson_reg") {
  y  <- as.double(unlist(payload$y))
  xs <- lapply(payload$xs, function(v) as.double(unlist(v)))
  off <- if (!is.null(payload$offset)) as.double(unlist(payload$offset)) else rep(0, length(y))
  df <- data.frame(y = y, do.call(cbind, xs))
  names(df) <- c("y", paste0("x", seq_along(xs)))
  fit <- glm(y ~ ., data = df, family = poisson(link = "log"),
             offset = off, control = glm.control(epsilon = 1e-10, maxit = 100))
  s   <- summary(fit)$coefficients
  ci  <- confint.default(fit)
  pear <- sum(residuals(fit, type = "pearson")^2)
  out <- list(
    coefficients    = unname(s[, "Estimate"]),
    standard_errors = unname(s[, "Std. Error"]),
    z_values        = unname(s[, "z value"]),
    p_values        = unname(s[, "Pr(>|z|)"]),
    ci              = unname(cbind(ci[, 1], ci[, 2])),
    irr             = unname(exp(s[, "Estimate"])),
    log_likelihood  = as.numeric(logLik(fit)),
    deviance        = fit$deviance,
    null_deviance   = fit$null.deviance,
    pearson_chi2    = pear,
    dispersion      = pear / fit$df.residual,
    aic             = fit$aic,
    iterations      = fit$iter
  )
} else if (method == "glm_generic") {
  fam_name  <- as.character(payload$family)   # binomial / poisson / gaussian
  link_name <- as.character(payload$link)     # logit / log / identity
  fam <- switch(fam_name,
    binomial = binomial(link = link_name),
    poisson  = poisson(link = link_name),
    gaussian = gaussian(link = link_name))
  y  <- as.double(unlist(payload$y))
  xs <- lapply(payload$xs, function(v) as.double(unlist(v)))
  w  <- if (!is.null(payload$weights)) as.double(unlist(payload$weights)) else rep(1, length(y))
  off<- if (!is.null(payload$offset))  as.double(unlist(payload$offset))  else rep(0, length(y))
  df <- data.frame(y = y, do.call(cbind, xs)); names(df) <- c("y", paste0("x", seq_along(xs)))
  fit <- glm(y ~ ., data = df, family = fam, weights = w, offset = off,
             control = glm.control(epsilon = 1e-10, maxit = 100))
  s  <- summary(fit)$coefficients
  ci <- confint.default(fit)
  out <- list(
    coefficients = unname(s[, "Estimate"]),
    standard_errors = unname(s[, "Std. Error"]),
    z_values = unname(s[, ncol(s) - 1]),     # gaussian 是 t value,容差另計
    p_values = unname(s[, ncol(s)]),
    ci = unname(cbind(ci[, 1], ci[, 2])),
    deviance = fit$deviance, null_deviance = fit$null.deviance,
    aic = fit$aic, log_likelihood = as.numeric(logLik(fit)),
    dispersion = summary(fit)$dispersion
  )
}
```

**關鍵陷阱(務必照做):**
- **CI 一定用 `confint.default()`(Wald / z)**,不要用 `confint()`——後者是 profile-likelihood CI,跟我們算的 Wald CI 不同,會對不上。
- R `glm` 預設 `epsilon=1e-8, maxit=25`;對拍時把 R 收到 **`epsilon=1e-10`** 讓 baseline 更收斂,Go 端 IRLS 也跑到同等收斂,避免「兩邊都半收斂」造成的假性誤差。
- Poisson AIC 對齊需要 logLik 含 `lgamma(y+1)`;R 的 `fit$aic` 已含,Go 端 `logLikContrib` 必須同樣含(見 §3.2)。
- Gaussian GLM 的 `summary` 末兩欄是 t value / Pr(>|t|)(因 dispersion 估計),與 binomial/poisson 的 z 不同——gaussian 對拍 z/p 容差另放寬,或直接以 `LinearRegression` 內部等價(D7)為主。

### 4.2 Python baseline(輔)— `statsmodels`

`crosslang_baseline.py` 新增對應 method,用 `sm.GLM(y, sm.add_constant(X), family=sm.families.Binomial())` 等;`fit().bse / tvalues / pvalues / conf_int() / deviance / null_deviance / aic / llf`。statsmodels `conf_int()` 即 Wald,可直接用。Python 已是既有相依(`requireCrossLangTools` 已檢查 statsmodels)。

### 4.3 容差(`crosslang_public_methods_test.go`)

| 量 | 容差 | 備註 |
|---|---|---|
| coefficients β | abs/rel 1e-6 | IRLS 收到 1e-10 應達此精度 |
| standard errors | rel 1e-5 | 依賴最終 information matrix,略放寬 |
| z values | rel 1e-5 | |
| p values | abs 1e-6,或 p<1e-8 時只比對數量級 | p 接近 0 時絕對容差會偽陰性 |
| deviance / null_deviance | rel 1e-6 | |
| AIC / BIC | rel 1e-6 | 需 logLik 常數項對齊 |
| log_likelihood | rel 1e-6 | |
| dispersion(poisson) | rel 1e-6 | |

對拍 helper 既有 `baselineFloat` / `baselineFloatSlice` 可直接用;CI 為 `[][2]` 需自行展平比對。

---

## Phase 0 — 地基與重構

**目標:** 不改任何對外行為,先把共用建構塊到位。

- [ ] `distutil.go`:新增 `zQuantile(p float64) float64 { return norm.Quantile(p) }`(顯式暴露,避免走 `tQuantile(p, +Inf)` 退化)。
- [ ] `mathutil.go`:新增 `sigmoid(x float64) float64`(η≥0 / η<0 兩條 path,避免 `exp` 溢位)。
- [ ] `consts.go`:新增列舉
  ```go
  type GLMFamily string
  type GLMLink string
  type SeparationPolicy string
  const ( Binomial GLMFamily = "binomial"; Poisson = "poisson"; Gaussian = "gaussian" )
  const ( Logit GLMLink = "logit"; Log = "log"; Identity = "identity"; Probit = "probit"; Cloglog = "cloglog" )
  const ( SepWarn SeparationPolicy = "warn"; SepError = "error"; SepRidge = "ridge" )
  ```
- [ ] `regression_shared.go`(Layer 3):抽出
  ```go
  // 並行 AtomicDo 拉 y + xs(+offset/weights),驗長度;回切片
  func gatherRegressionInputs(dlY insyra.IDataList, dlXs []insyra.IDataList,
      extra ...insyra.IDataList) (y []float64, xs [][]float64, extras [][]float64, n int, err error)
  // 建 n×(p+1) design matrix(leading intercept),直接寫 RawMatrix,大尺寸走 parutil
  func buildDesignMatrix(xs [][]float64, n int) *mat.Dense
  ```
- [ ] **重構 `LinearRegression` 改用上述兩個 helper**(D6),確保 `go test ./stats/...` 全綠、行為不變(回歸測試把關)。
- [ ] 單元測試:`zQuantile` 對拍 `qnorm`;`sigmoid` 大正負值不溢位;`buildDesignMatrix` 形狀 / 截距欄正確。

**Phase 0 完成準則:** `go test -race ./stats/...` 與 `go build ./...` 全綠,且 LinearRegression 既有測試一字不改通過。

---

## Phase 1 — IRLS 核心 + Family/Link

**目標:** 純內部、不對外暴露,先把計算引擎做對。

- [ ] `glm_links.go`:`glmLink` 介面 + `logitLink` / `logLink` / `identityLink`(`probitLink` / `cloglogLink` 留 struct stub + `// TODO Phase: 後續 issue`,但不註冊)。
- [ ] `glm_families.go`:`glmFamily` 介面 + `binomialFamily` / `poissonFamily` / `gaussianFamily`,含 §3.2 全部方法。
- [ ] `glm_irls.go`:`fitIRLS` + `fitNullIRLS` + separation 偵測 + clamp;`solveWeightedNormalEquations(X, W, z)` 用 `mat.SolveVec`,大尺寸 `XᵀWX` 走 parutil。
- [ ] `glmutil.go`:`computeGLMInference` / `buildGLMCoeffCIs` / `mcFaddenR2` / `coxSnellR2` / `nagelkerkeR2` / `pearsonChiSq` / `pearsonDispersion`。
- [ ] 單元測試:
  - **D7 等價測試**:Gaussian+identity 的 `fitIRLS` β / SE 對拍同資料的 `solveOLS` / `LinearRegression`,容差 1e-9(同一套 normal equations,應幾乎位元一致)。
  - Link 反函數往返:`link.mu(link.eta(x)) ≈ x`。
  - Family deviance / logLik 對單點手算值。
  - IRLS 收斂性:小型 logistic / poisson 資料 `converged==true` 且 `iterations < 25`。

**Phase 1 完成準則:** 引擎層測試全綠;Gaussian GLM == OLS 通過。

---

## Phase 2 — LogisticRegression / PoissonRegression 公開 API

**目標:** 第一個對外可用版本 + R/Python 對拍。

- [ ] `regression_logistic.go`:
  - `LogisticRegressionOptions`(`ConfidenceLevel` / `MaxIter` / `Tolerance` / `PositiveClass any` / `SeparationPolicy`,Phase 2 只接受 `warn`/`error`;`ridge` 回 `errors.New("ridge policy not yet available")` 直到 Phase 5)。
  - `LogisticRegressionResult`(完整欄位見 issue,含 OddsRatios / OddsRatioCIs / McFadden / CoxSnell / Nagelkerke / Deviance 系列 / AIC / BIC / Pearson&Deviance residuals / PositiveClass / ClassLabels)。
  - `LogisticRegression(dlY, dlXs...)` 與 `LogisticRegressionWithOptions(opts, dlY, dlXs...)`。
  - **正類解析(D5)**:y 容許 0/1、true/false、或兩相異任意值;預設正類 = 兩相異值排序後較大者;`PositiveClass` 可鎖定;非二元 → error。
  - separation:`sepFlag && policy==error` → return error;`==warn` → 照回結果但 `Converged` 反映實況(不另設旗標欄位,以免破壞 issue 定義的 struct;warn 走 `insyra` logger 之外的注意:依 CLAUDE.md「stats 不用 LogWarning 當失敗訊號」,warn 僅記於 result 的 `Converged`/SE 反映,文件說明)。
- [ ] `regression_poisson.go`:
  - `PoissonRegressionOptions`(+`Offset IDataList` / `DispersionCheck bool`)。
  - `PoissonRegressionResult`(含 IncidenceRateRatios / IRRConfidenceIntervals / PearsonChi2 / DispersionStatistic 等)。
  - `PoissonRegression` / `PoissonRegressionWithOptions`;offset 走 IRLS `opts.offset`。
  - over-dispersion:`DispersionCheck && χ²/df > 1.5` → 不報錯,於文件與 `DispersionStatistic` 欄位呈現(CLAUDE.md 禁止用 log 當失敗訊號;此為「提醒」非「失敗」,僅以欄位表達)。
- [ ] baseline:`crosslang_baseline.R` + `.py` 加 `logistic_reg` / `poisson_reg`(§4)。
- [ ] `crosslang_public_methods_test.go`:logistic / poisson 各 ≥2 組資料(含多自變量、含 offset 的 poisson rate model)對拍 R + Python,容差 §4.3。
- [ ] 單元測試:正類解析(0/1、bool、字串對、鎖定、非二元報錯);separation `warn`/`error` 兩路徑;dispersion 計算正確。

**Phase 2 完成準則:** logistic / poisson 對拍 R `glm()` 全綠;separation warn/error、over-dispersion 警示有測試。

---

## Phase 3 — 通用 GLM 入口 + Predict

- [ ] `regression_glm.go`:
  - `GLMOptions`(`Family` / `Link`(空則用 canonical) / `ConfidenceLevel` / `MaxIter` / `Tolerance` / `Offset` / `Weights`)。
  - `GLMResult`(係數推論 + deviance/aic + dispersion;為三家共用的「素」結果型別)。
  - `GLM(opts, dlY, dlXs...)`:解析 family+link → `fitIRLS` → `computeGLMInference`。
  - link/family 相容性檢查(如 `binomial+identity` 允許但警示;不支援組合報 error)。
- [ ] `Predict`(三型別各一,§3.5):`(*LogisticRegressionResult).Predict`、`(*PoissonRegressionResult).Predict`、`(*GLMResult).Predict`,支援 `linear`/`response`/`class`(class 僅 logistic;poisson/generic 傳 class 報 error)。
- [ ] baseline:`crosslang_baseline.R`/`.py` 加 `glm_generic`;**prior weights 與 offset 各一條 baseline 對拍**。
- [ ] 測試:
  - `GLM{Gaussian,Identity}` == `LinearRegression`(對拍既有 Go 結果,免 R)。
  - `GLM{Binomial,Logit}` == `LogisticRegression`(同資料,Go 內部一致性)。
  - `Predict` 三型別:對訓練資料 `response` == `FittedProbabilities`/`FittedRates`;`linear` == `LinearPredictors`;`class` 門檻邏輯。
  - weights / offset 對拍 R。

**Phase 3 完成準則:** 通用 GLM + Predict 全綠;weights/offset baseline 通過;三套 API 內部一致。

---

## Phase 4 — isr 暴露

- [ ] `isr/glm.go`:
  ```go
  type LogisticRegressionResult = stats.LogisticRegressionResult
  type PoissonRegressionResult  = stats.PoissonRegressionResult
  type GLMResult                = stats.GLMResult
  type GLMOptions               = stats.GLMOptions   // + Family/Link 常數 alias
  func LogisticRegression(y insyra.IDataList, xs ...insyra.IDataList) (*stats.LogisticRegressionResult, error)
  func PoissonRegression(y insyra.IDataList, xs ...insyra.IDataList) (*stats.PoissonRegressionResult, error)
  func GLM(opts stats.GLMOptions, y insyra.IDataList, xs ...insyra.IDataList) (*stats.GLMResult, error)
  ```
  風格對齊 [isr/groupby.go](../isr/groupby.go) / [isr/pivot.go](../isr/pivot.go)(薄包裝 + doc 範例)。
- [ ] `isr/glm_test.go`:呼叫三入口跑通 + 與 `stats` 直呼結果一致。

**Phase 4 完成準則:** `isr.LogisticRegression/PoissonRegression/GLM` 可用,測試綠。

---

## Phase 5 — Separation ridge / over-dispersion 強化

> D3:`ridge` 在此階段才實作,且**明確標記推論為 penalized**。

- [ ] `irlsOptions.ridge`:在 `XᵀWX` 對角(**排除截距列**)加 `λ`;`LogisticRegressionOptions.SeparationPolicy==ridge` 時以 `Ridge` 值(預設小,如 1e-4)啟用。
- [ ] result 文件 / 註解明示:ridge 啟用時 SE/p/CI 為 penalized-likelihood 值,非 MLE Wald;`Converged` 反映 ridge 後收斂。
- [ ] 測試:完全可分資料(complete separation)三策略——`warn`(回結果、SE 巨大)、`error`(拒絕)、`ridge`(收斂、β 有限)各覆蓋。quasi-complete separation 另一組。
- [ ] over-dispersion:`DispersionCheck` 路徑與 `DispersionStatistic` 正確性對拍 R(`sum(pearson resid²)/df`)。

**Phase 5 完成準則:** separation 三路徑 + over-dispersion 測試全綠。

---

## Phase 6 — 文件、決策樹、CI 全綠

- [ ] `Docs/stats.md`:新增 GLM 章節(`LogisticRegression` / `PoissonRegression` / `GLM` / `Predict` 簽名、Options、Result 欄位、error-first 範例)。
- [ ] `Docs/tutorials/`:新增 GLM 教學,含 **「OLS → logistic → Poisson」決策樹**:
  - 連續 y、常態殘差 → `LinearRegression`
  - 二元 y(0/1、成功/失敗)→ `LogisticRegression`
  - 計數 y(非負整數、事件數)→ `PoissonRegression`;`χ²/df ≫ 1.5` → 考慮(未來)Negative Binomial
  - 比率 y(事件/暴露)→ `PoissonRegression` + `Offset = log(exposure)`
  - 需要自訂 family/link → `GLM`
- [ ] `golangci-lint run` 全綠;`govulncheck ./...` 無新增;`go test -race ./...` 全綠。
- [ ] 確認**純加法**:既有 import path 與既有方法簽名零變更(Phase 0 的 LinearRegression 重構為內部實作替換,對外簽名不動)。
- [ ] 更新本檔狀態為「completed」或移除。

**Phase 6 完成準則:** 文件齊備、三大 CI(lint/vuln/race)全綠、純加法確認。

---

## 12. 明確延後(後續 issue)

本提案**不**實作以下項目,但 `glmFamily` / `glmLink` 介面必須能無破壞性地容納:

| 延後項 | 介面預留 |
|---|---|
| Negative Binomial(over-dispersion 計數) | 新增 `negativeBinomialFamily`,`variance(μ)=μ+μ²/θ`;θ 估計另需迭代 hook |
| Gamma / Inverse Gaussian | 新增對應 `glmFamily`,canonical link 各為 inverse / 1/μ² |
| Probit / Cloglog link | `glm_links.go` 已留 stub,實作 `eta/mu/muEta` 即可註冊 |
| Firth bias-reduced logistic | IRLS 改 penalized score(Jeffreys prior);取代 Phase 5 的簡化 ridge |
| Multinomial logistic | 需獨立 multi-response 路徑,不在現有單一 η 框架 |

> 驗收條件:加入上述任一項**不得**改動 `LogisticRegression` / `PoissonRegression` / `GLM` 的對外簽名或既有 `glmFamily` / `glmLink` 方法集(只新增 struct)。

---

## 13. 驗收條件對照表(issue #164)

| issue 驗收條件 | 對應 Phase | 狀態 |
|---|---|---|
| `LogisticRegression(+Options)` / `PoissonRegression(+Options)` / `GLM`,皆 `(*Result, error)` | P2 / P3 | ☐ |
| 三 result 型別皆有 `Predict(typ, newXs...)`(linear/response/class) | P3 | ☐ |
| 共用 IRLS 核心 + Family/Link 介面,與 OLS 計算分離,守 CLAUDE.md 分層 | P1 | ☐ |
| 對拍 R `glm(binomial/poisson)` + statsmodels,β/SE/z/p/deviance/AIC 容差通過 | P2 / P3 | ☐ |
| separation `warn`/`error`/`ridge` 三路徑單元測試 | P2(warn/error)/ P5(ridge) | ☐ |
| over-dispersion 警示 + `DispersionStatistic` 測試 | P2 / P5 | ☐ |
| offset 與 prior weights baseline 對拍 | P3 | ☐ |
| `isr/glm.go` 暴露 + alias + test | P4 | ☐ |
| `Docs/stats.md` + tutorials + 決策樹 | P6 | ☐ |
| `golangci-lint` / `govulncheck` / `go test -race` 全綠 | P6 | ☐ |
| 純加法,不破壞既有簽名 | P0 / P6 | ☐ |
| 不實作 NB/Gamma/IG/Probit/Cloglog/Firth/multinomial,但介面可擴充 | §12 | ☐ |

---

_實作順序建議:P0 → P1 → P2 →(可發第一個 PR)→ P3 → P4 → P5 → P6。每個 Phase 結束都應 `go test -race ./stats/...` 綠燈再進下一階段。_
