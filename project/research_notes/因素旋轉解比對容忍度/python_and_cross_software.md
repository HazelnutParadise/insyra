# R 核心套件以外：Python 套件、商業軟體與跨軟體比較研究怎麼判定旋轉解「一致」

閱讀範圍與版本（2026-09-17 讀取）：statsmodels `main` commit `5032ffd`（最新 release v0.15.0，2026-08-27）、factor_analyzer `main` commit `de933d2`（套件版本 0.5.1）、scikit-learn `main` commit `bbd90ca`、semopy 2.3.11 sdist、JASP `jaspFactor` commit `83fe6db` 與 `jaspTools` commit `55ab6bc`、EFAtools 1.1.0（CRAN 鏡像）、fungible 2.4.8（CRAN 鏡像）、lavaan 0.7-2（CRAN 鏡像）、Stata 手冊 `[MV] rotate`/`rotatemat`（PDF 產生日期 2026-07-28）、Mplus User's Guide Chapter 16（PDF 產生日期 2017-03-25）、IBM SPSS Statistics 26 Command Syntax Reference 與 Algorithms、SAS/STAT 9.3 PROC FACTOR 文件頁。所有程式碼只讀不執行；唯一執行的是用本機 R 確認 `nchar(1e-5)` 的值（見 fungible 一節）。fungible、lavaan、EFAtools 在 `r_packages.md` 已有更完整的原始碼分析，這裡只記和跨軟體比對有關的部分。

---

## 1. statsmodels：GPA 旋轉測試比對什麼、容忍度多少

### Takeaway
statsmodels 的 GPA 旋轉測試只比 loadings、旋轉矩陣 T 和迭代表（準則值 f 與 log10 梯度範數），全部用 `np.allclose(..., atol=1e-05)`，**從來不比 Phi**，參考值來自 Bernaards & Jennrich 的 UCLA GPA 網站而不是 GPArotation 套件本身。唯一比對 Phi 的地方是 `Factor` 類別和 **Stata** 的比對，用 `rtol=1e-3`，而且同一個測試裡的因素分數也放寬到 `atol=1e-4, rtol=1e-3`，比同檔案裡 loadings 對 R psych 的 `decimal=8` 寬鬆很多。

### Cited Findings
- **GPA 測試的參考來源**：`TestGPARotation` 各測試的 docstring 標示範例出自 `http://www.stat.ucla.edu/research/gpa`，參考 loadings 以 5 到 6 位小數字串寫死在測試裡 — [statsmodels/multivariate/factor_rotation/tests/test_rotation.py L115-L180, L270-L306](https://github.com/statsmodels/statsmodels/blob/5032ffdf1e706b8b8a90248fd661995d224c2797/statsmodels/multivariate/factor_rotation/tests/test_rotation.py#L115-L306)
- **容忍度**：quartimax、quartimin、biquartimin、CF、target、partial target 的比對一律是 `np.allclose(table, table_required, atol=1e-05)` 與 `np.allclose(L, L_required, atol=1e-05)`；比對的是 loadings `L` 與迭代表 `table`，不是 Phi — [同檔 L305-L306、L499-L500、L507-L508、L516-L517](https://github.com/statsmodels/statsmodels/blob/5032ffdf1e706b8b8a90248fd661995d224c2797/statsmodels/multivariate/factor_rotation/tests/test_rotation.py#L491-L527)
- **不同演算法之間**（有導數的 GPA 對無導數 GPA、或 analytic target）：`_test_template` 比對 `L1, L2` 與 `T1, T2`，都是 `atol=1e-5` — [同檔 L610-L621](https://github.com/statsmodels/statsmodels/blob/5032ffdf1e706b8b8a90248fd661995d224c2797/statsmodels/multivariate/factor_rotation/tests/test_rotation.py#L610-L621)
- **Phi 從未被斷言**：`GPA()` 回傳的 `phi` 在測試裡只被接收（如 `L, phi, T, table = GPA(...)`），全檔沒有任何對 `phi` 的 assert — [同檔 L111、L280、L322、L382、L411](https://github.com/statsmodels/statsmodels/blob/5032ffdf1e706b8b8a90248fd661995d224c2797/statsmodels/multivariate/factor_rotation/tests/test_rotation.py)（以 `grep phi` 查證）
- **`np.allclose` 的實際門檻**：numpy 文件說明 allclose 是拿 `atol + rtol * abs(b)` 跟絕對差比較；statsmodels 只傳 `atol`，所以還會加上 numpy 的預設 `rtol`（numpy 簽名預設 `rtol=1e-05`，這個預設值是依 numpy 公開簽名，本次沒有抓到頁面原文，信心高） — [numpy.allclose](https://numpy.org/doc/stable/reference/generated/numpy.allclose.html)
- **GPA 停止條件**：`GPA(A, ..., max_tries=501, rotation_method="orthogonal", tol=1e-5)`，停止條件是投影梯度的 Frobenius 範數 `s < tol`，與 GPArotation 的 `eps` 同型 — [statsmodels/multivariate/factor_rotation/_gpa_rotation.py L29-L30、L118-L126](https://github.com/statsmodels/statsmodels/blob/5032ffdf1e706b8b8a90248fd661995d224c2797/statsmodels/multivariate/factor_rotation/_gpa_rotation.py#L29-L126)
- **`Factor` 類別對 R psych/GPArotation**：註解附上 `fa(Y, nfactors=2, fm="pa", rotate=..., min.err=1e-10)` 的 R 程式，varimax、quartimax、oblimin 的 loadings 用 `assert_array_almost_equal(..., decimal=8)` 比對，且手動把某些欄乘以 -1 對齊符號 — [statsmodels/multivariate/tests/test_factor.py L119-L208](https://github.com/statsmodels/statsmodels/blob/5032ffdf1e706b8b8a90248fd661995d224c2797/statsmodels/multivariate/tests/test_factor.py#L119-L208)
- **`decimal=8` 的實際門檻**：numpy 文件寫明判準是 `abs(desired-actual) < 1.5 * 10**(-decimal)`，即 1.5e-8 — [numpy.testing.assert_array_almost_equal](https://numpy.org/doc/stable/reference/generated/numpy.testing.assert_array_almost_equal.html)
- **不一致就不測，而不是放寬**：同一測試對 equamax、promax、biquartimin 只寫註解「Not the same as R fa」並呼叫 `rotate`，沒有任何斷言 — [test_factor.py L195-L199](https://github.com/statsmodels/statsmodels/blob/5032ffdf1e706b8b8a90248fd661995d224c2797/statsmodels/multivariate/tests/test_factor.py#L195-L199)
- **唯一比對 Phi 與分數的測試（對 Stata）**：`test_factor_scoring` 標記為 `@pytest.mark.smoke`，註解說「mostly smoke tests for now」；varimax 與 oblimin 的迴歸分數、Bartlett 分數對 Stata 用 `assert_allclose(..., atol=1e-4, rtol=1e-3)`，oblimin 的因素相關用 `assert_allclose(res._corr_factors()[0, 1], (-1) * 0.25651037, rtol=1e-3)`，並註明 Stata 第二因素符號相反 — [test_factor.py L320-L364](https://github.com/statsmodels/statsmodels/blob/5032ffdf1e706b8b8a90248fd661995d224c2797/statsmodels/multivariate/tests/test_factor.py#L320-L364)
- **`assert_allclose` 的門檻公式**：比較 `atol + rtol * abs(desired)` — [numpy.testing.assert_allclose](https://numpy.org/doc/stable/reference/generated/numpy.testing.assert_allclose.html)

### Inferences
- 換算成絕對量：Phi 對 Stata 的容忍度約是 `1e-3 × 0.2565 ≈ 2.6e-4`；分數約 `1e-4 + 1e-3×|分數|`，分數量級 1 時約 1.1e-3。對照同一套件 loadings 對 R 的 1.5e-8，Phi 與分數確實被放寬了 4 個數量級以上。但這個放寬是「換了參考軟體（Stata）加上 smoke test」造成的，loadings 在那個測試裡根本沒對 Stata 比，所以**不能解讀成 statsmodels 有「Phi 比 loadings 寬」的刻意規則**。
- statsmodels 旋轉測試比對迭代表（準則值與梯度範數每一步），等於要求演算法路徑一致，比只比最終 loadings 嚴格；但它的範例都是 8×2 的小矩陣，沒有涵蓋多個局部解或近秩一的情況。

### Gaps
- 問題描述說 statsmodels 的 GPA 測試「對照 GPArotation」：原始碼只引用 UCLA GPA 網站的範例輸出，沒有提到 GPArotation 套件版本；`test_factor.py` 的 varimax 註解寫的是「Same as R GRArotation」（原文拼字），指的是透過 psych 呼叫 GPArotation。
- `factors_stata.csv` 用的 Stata 版本與旋轉設定沒有在檔案內說明，未查證。

---

## 2. factor_analyzer（ETS）：對 R psych 比對哪些輸出、容忍度多少、怎麼處理符號

### Takeaway
factor_analyzer 的比對非常寬鬆：每個元素用 `math.isclose(..., abs_tol=0.1)`，只要 **90%**（整體 FA）或 **95%**（單純旋轉）的元素落在 0.1 以內就算通過；loadings、structure、scores 一律同一個 0.1，**完全沒有比對 Phi**。唯一要求全部元素都過的是 geomin 旋轉，但門檻仍是 0.1。沒有明確的符號或欄位順序對齊。

### Cited Findings
- **比對函式**：`check_close(data1, data2, rel_tol=0.0, abs_tol=0.1, with_normalize=True, absolute=False)` 逐元素呼叫 `math.isclose`，回傳「符合比例」而不是布林值 — [factor_analyzer/test_utils.py L206-L256](https://github.com/EducationalTestingService/factor_analyzer/blob/de933d26808039d965c117874c67260ec5149690/factor_analyzer/test_utils.py#L206-L256)
- **比對的輸出種類**：`OUTPUT_TYPES = ["value", "evalues", "loading", "uniquenesses", "communalities", "structure", "scores"]`；Python 端 structure 取 `fa.structure_`、scores 取 `fa.transform(data)`，沒有 Phi — [test_utils.py L30-L38、L105-L113](https://github.com/EducationalTestingService/factor_analyzer/blob/de933d26808039d965c117874c67260ec5149690/factor_analyzer/test_utils.py#L30-L113)
- **structure 與 scores 是選配**：`check_scenario` 預設只比 loading、evalues，`check_scores=True` 才加 scores，`check_structure=True` 才加 structure（docstring 註明只用於斜交旋轉），所有種類共用同一組 `rel_tol=0, abs_tol=0.1` — [test_utils.py L259-L348](https://github.com/EducationalTestingService/factor_analyzer/blob/de933d26808039d965c117874c67260ec5149690/factor_analyzer/test_utils.py#L259-L348)
- **通過門檻**：`tests/test_expected_factor_analyzer.py` 設 `THRESHOLD = 0.9` 並 `assertGreater(check, THRESHOLD)`；例如 `test_01_promax_minres_2_factors` 同時開 `check_structure=True, check_scores=True`，沒有另傳容忍度 — [tests/test_expected_factor_analyzer.py L17、L63-L79](https://github.com/EducationalTestingService/factor_analyzer/blob/de933d26808039d965c117874c67260ec5149690/tests/test_expected_factor_analyzer.py#L17-L79)
- **單純旋轉的比對**：`check_rotation` 拿 R 的未旋轉 loadings 餵給 `Rotator`，再跟 R 的旋轉後 loadings 比，只比 loadings — [test_utils.py L351-L396](https://github.com/EducationalTestingService/factor_analyzer/blob/de933d26808039d965c117874c67260ec5149690/factor_analyzer/test_utils.py#L351-L396)；門檻 `THRESHOLD = 0.95`，oblimin、quartimin、oblimax 等都用它，geomin（`geomin_obl`、`geomin_ort`）則要求 `assertEqual(check, 1)` — [tests/test_expected_rotator.py L14、L45-L52、L163-L188](https://github.com/EducationalTestingService/factor_analyzer/blob/de933d26808039d965c117874c67260ec5149690/tests/test_expected_rotator.py#L14-L188)
- **符號**：`check_close` 預設 `absolute=False`，`check_scenario` 與 `check_rotation` 都沒改它，所以沒有取絕對值，也沒有欄位重排或反射 — [test_utils.py L168-L203、L206-L208](https://github.com/EducationalTestingService/factor_analyzer/blob/de933d26808039d965c117874c67260ec5149690/factor_analyzer/test_utils.py#L168-L208)
- **R 參考值產生方式**：`tests/generate_r_output.r` 對 `cor(df)` 呼叫 `psych::fa(..., rotate = rot)`，只寫出 loadings、values、e.values、uniquenesses、communalities — [tests/generate_r_output.r L55-L96](https://github.com/EducationalTestingService/factor_analyzer/blob/de933d26808039d965c117874c67260ec5149690/tests/generate_r_output.r)
- **CFA 的比對**：`THRESHOLD = 1.0`，但用 `rel_tol=0.1` 或 `abs_tol=0.05`，比對 loadings、error variances、factor covariances、標準誤與 transform 分數 — [tests/test_expected_confirmatory_factor_analyzer.py L22-L61](https://github.com/EducationalTestingService/factor_analyzer/blob/de933d26808039d965c117874c67260ec5149690/tests/test_expected_confirmatory_factor_analyzer.py)
- **版本**：最新 release 為 v0.5.1（2024-02-08），`main` 最後 commit 為 2025-01-21 — [GitHub releases](https://github.com/EducationalTestingService/factor_analyzer/releases)

### Inferences
- 0.1 的門檻加上「只要 90% 元素符合」，等於只驗證「解大致相同」，它能容忍一整欄符號相反甚至少數元素差很多。這是驗證「大方向正確」的做法，不是數值一致性驗證，不適合當 insyra 2e-5 等級的參考。
- 在這個套件裡，loadings、structure、scores 是同一個門檻，沒有任何輸出被特別放寬或收緊。

### Gaps
- `structure_*.csv`、`scores_*.csv` 參考檔不是 `generate_r_output.r` 產生的，可能來自 `tests/notebooks/get_results_in_r.ipynb`，因此 R 端分數用哪種方法、psych 版本多少，沒有查證。

---

## 3. semopy、scikit-learn 與其他 Python／應用程式層的 EFA 工具

### Takeaway
scikit-learn 只用絕對值、3 位小數（|差| < 1.5e-3）比對 varimax loadings 對 R `psych::principal`，不比 Phi（它只有正交旋轉）。semopy 沒有任何因素旋轉功能可比。JASP 的 EFA 迴歸測試把因素相關、pattern loadings、structure loadings 都四捨五入到同樣 4 位精度再比，**Phi 與 loadings 待遇相同**。

### Cited Findings
- **scikit-learn**：`test_factor_analysis` 註解說 R 的因素分析結果差很多，所以「只測旋轉本身」；拿 `_ortho_rotation(..., method="varimax")` 對寫死的 R `psych::principal(rotate="varimax")` 結果比，`assert_array_almost_equal(np.abs(rotated), np.abs(r_solution), decimal=3)`，參考值只有 3 位小數 — [sklearn/decomposition/tests/test_factor_analysis.py L93-L109](https://github.com/scikit-learn/scikit-learn/blob/bbd90caf8b2f74e722f6a71364e886075b80c001/sklearn/decomposition/tests/test_factor_analysis.py#L93-L109)
- **scikit-learn 旋轉只有正交**：`_ortho_rotation(components, method="varimax", tol=1e-6, max_iter=100)`，測試裡只跑 `None`、`"varimax"`、`"quartimax"` — [sklearn/decomposition/_factor_analysis.py L438](https://github.com/scikit-learn/scikit-learn/blob/bbd90caf8b2f74e722f6a71364e886075b80c001/sklearn/decomposition/_factor_analysis.py#L438)；[test_factor_analysis.py L81-L91](https://github.com/scikit-learn/scikit-learn/blob/bbd90caf8b2f74e722f6a71364e886075b80c001/sklearn/decomposition/tests/test_factor_analysis.py#L81-L91)
- **semopy 2.3.11**：`efa.py` 的模組說明寫明只用自家「unorthodox」的分群法（OPTICS、SparsePCA）找潛在因素；整個套件的 `.py` 檔搜尋 `rotat|oblimin|varimax|promax|geomin` 無任何結果，`tests/` 也只有 means、model、polycorr、reproducibility、standard errors 等測試 — [semopy 2.3.11 sdist（PyPI）](https://pypi.org/project/semopy/2.3.11/)（下載 sdist 後以 grep 查證）
- **JASP 的表格比對精度**：`jaspTools::expect_equal_tables` 文件寫預設以 `signif(round(x, digits = 4), digits = 4)` 比較數字，可透過 `options("jaspRoundToPrecision")` 改 — [jaspTools R/testthat-helper-tables.R L9-L10、L76-L82](https://github.com/jasp-stats/jaspTools/blob/55ab6bcd2e3828fc923a9bdf513c50a06da3d6ad/R/testthat-helper-tables.R#L9-L82)
- **JASP EFA 測試同時比 Phi、loadings、structure**：`test-exploratoryfactoranalysis.R` 分別有「Factor Correlations table results match」、「Factor Loadings table results match」、「Factor Loadings (Structure Matrix) table results match」，全部呼叫同一個 `expect_equal_tables`，沒有個別容忍度；斜交設定包含 `promax` 與 `geominQ`，並 `set.seed(1)` — [jaspFactor tests/testthat/test-exploratoryfactoranalysis.R L14-L16、L57-L70、L93-L101、L138-L144](https://github.com/jasp-stats/jaspFactor/blob/83fe6db186b69be295027ef18c6992b68172e168/tests/testthat/test-exploratoryfactoranalysis.R)
- **JASP 的跨軟體「verified」測試**：`test-verified-exploratoryfactoranalysis.R` 的 loadings 測試標題寫「match R, SPSS, SAS, MiniTab」並連到 JASP Verification Project；檔頭註明不測正交旋轉以外的一些項目 — [jaspFactor tests/testthat/test-verified-exploratoryfactoranalysis.R L3-L7、L107-L125](https://github.com/jasp-stats/jaspFactor/blob/83fe6db186b69be295027ef18c6992b68172e168/tests/testthat/test-verified-exploratoryfactoranalysis.R)
- **JASP Verification Project 的 EFA 比對**：用 Field (2018) 的 SAQ 資料跑 PAF + varimax，把各軟體輸出的 SS loadings 並列，數值分別印成 3.0336、3.033、3.034、3.03 這種各軟體自己的顯示精度，並附上 SPSS `/CRITERIA ITERATE(25)`、SAS `Method=prinit Rotate=varimax`、R `factor.pa` 的程式 — [JASP Verification Project: Factor, 8.2](https://jasp-stats.github.io/jasp-verification-project/factor.html#exploratory-factor-analysis)

### Inferences
- 應用程式層（JASP）的做法是「所有表格同一個四捨五入精度」，Phi 與 loadings 同等對待；它的跨軟體驗證停在各軟體的顯示精度（2 到 4 位小數），而且只驗正交旋轉，沒有處理斜交旋轉的局部解問題。
- Python 生態裡沒有找到任何一個對斜交 GPA 旋轉的 Phi 做數值比對、而且容忍度比 loadings 嚴或寬的測試。

### Gaps
- 沒有查 pingouin、`horns`、`psynlig` 等較小的 Python 工具；以已查到的主要套件判斷，它們不太可能有更嚴格的斜交旋轉驗證，但沒有查證。
- JASP Verification Project 的總覽表欄位標題沒有被擷取到，所以 3.033 與 3.034 分別屬於 SPSS 或 SAS 無法確定。

---

## 4. 商業軟體文件：SPSS、SAS、Stata、Mplus、CEFA／FACTOR 的收斂準則與「同一解」

### Takeaway
商業軟體的旋轉收斂準則大多是單一數字，沒有任何一家文件把 Phi 和 loadings 分開設定精度目標。SPSS（`RCONVERGE` 0.0001、最多 25 次）與 SAS（簡單度函數的尺度化變化 1E-9）都沒有隨機起點；有隨機起點的 Stata（`protect()`）與 Mplus（`RSTARTS`，預設 30）判斷「同一解」都是看**準則值**，而且文件都沒寫比到幾位。Mplus 的 GPA 收斂準則預設 0.00001，和 GPArotation 的 eps 相同。

### Cited Findings
- **SPSS FACTOR `/CRITERIA`**：`RCONVERGE(n)` 是旋轉收斂準則，預設 0.0001；`ECONVERGE` 萃取預設 0.001；`ITERATE(n)` 同時管萃取與旋轉，預設 25 — [IBM SPSS Statistics 26 Command Syntax Reference, FACTOR CRITERIA（PDF）](https://public.dhe.ibm.com/software/analytics/spss/documentation/statistics/26.0/en/client/Manuals/IBM_SPSS_Statistics_Command_Syntax_Reference.pdf)
- **SPSS 旋轉演算法**：正交旋轉是對因素配對循環旋轉（Harman 1976），達到最大迭代數或收斂準則即停；斜交旋轉是 Jennrich & Sampson (1966) 的 direct oblimin，逐對旋轉並更新因素相關矩陣 C，最後 structure 由最終迭代的因素相關矩陣算出；文件沒有描述隨機起點 — [IBM SPSS Statistics 26 Algorithms, FACTOR Algorithms: Factor Rotations（PDF）](https://public.dhe.ibm.com/software/analytics/spss/documentation/statistics/26.0/en/client/Manuals/IBM_SPSS_Statistics_Algorithms.pdf)
- **SAS PROC FACTOR**：`RCONVERGE=` 指定旋轉循環的收斂準則，當簡單度函數值的尺度化變化小於它就停，預設 ε 為 1E-9；`RITER=` 預設為變數數的 10 倍與 100 取大者（promax、Procrustes 不適用）；`CONVERGE=` 是萃取用，看共同性最大變化，預設 0.001；選項表中唯一與亂數有關的 `RANDOM=` 只用於 `PRIORS=RANDOM` — [SAS/STAT 9.3 User's Guide, PROC FACTOR Statement](https://support.sas.com/documentation/cdl/en/statug/63962/HTML/default/statug_factor_sect006.htm)
- **Stata `rotatemat` 收斂準則**：`tolerance(#)`（旋轉矩陣 T 的相對變化，預設 1e-6）、`gtolerance(#)`（投影梯度範數，預設 1e-6）、`ltolerance(#)`（準則值相對變化，預設 1e-6）、`iterate(#)` 預設 1000 — [Stata Manual: [MV] rotatemat（PDF）](https://www.stata.com/manuals/mvrotatemat.pdf)
- **Stata `protect(#)`**：跑 # 次隨機起點並回報最好的解，輸出會指出「是否所有起點收斂到同一解」；存回 `r(fmin)`（找到的各最小值）與 `r(nnconv)`（未收斂次數） — [[MV] rotatemat](https://www.stata.com/manuals/mvrotatemat.pdf)
- **Stata 手冊範例**：`rotate, oblimin oblique normalize protect(10)` 逐次印出「min criterion」，9 次是 `.0181657`、1 次是 `458260.7`，說明文字把「收斂到同一準則值」當成找到全域最佳的依據；loadings 印 4 位小數 — [Stata Manual: [MV] rotate（PDF）, p. 16](https://www.stata.com/manuals/mvrotate.pdf)
- **Mplus `RCONVERGENCE`**：GPA 旋轉演算法的收斂準則，預設 .00001 — [Mplus User's Guide, Chapter 16（PDF）, 約 p. 699](https://www.statmodel.com/download/usersguide/Chapter16.pdf)
- **Mplus `RSTARTS`**：指定 GPA 旋轉的隨機起點數，以及要印出「best unique rotation function values」的解有幾個；預設 30 個起點、只印最好的一個；例 `RSTARTS = 10 2;` — [Chapter 16, pp. 692-693](https://www.statmodel.com/download/usersguide/Chapter16.pdf)
- **Mplus geomin**：預設 ε 依因素數而定（2 因素 .0001、3 因素 .001、4 以上 .01），並說明 geomin 常有多個局部解，預設 30 個隨機起點 — [Chapter 16, pp. 678-679](https://www.statmodel.com/download/usersguide/Chapter16.pdf)
- **Mplus 開發者對多個局部解的處理原則**：Asparouhov 在論壇回覆說 Mplus 會為每個局部最小值報告簡單度函數值；若最好的解明顯較佳就用它，若前幾個值差不多，就由研究者選最好解讀的那個 — [Mplus Discussion: EFA Analysis "RSTARTS =" command (2015-01-21)](http://www.statmodel.com/discussion/messages/8/20878.html)；提問者引述的輸出標題為「LOCAL MINIMUM # FOR ROTATION ALGORITHM WITH FUNCTION VALUE」（二手引述）
- **Asparouhov & Muthén 的 ESEM 技術報告**：Mplus 的旋轉用 Jennrich (2001, 2002) 的 GPA；geomin 常產生「similar rotation function values」的多個局部解；舉例兩個解的函數值 0.28 與 0.30 各有約一半起點收斂到；在模擬研究中若每次都選全域最小值，樣本小時兩個解會交替出現，平均起來會得到無用結果 — [Asparouhov & Muthén, Exploratory Structural Equation Modeling（技術報告版）](https://www.statmodel.com/download/EFACFA810.pdf)

### Inferences
- 「收斂準則」與「輸出一致的精度」是兩回事：SPSS 0.0001、Mplus 1e-5、Stata 1e-6、SAS 1e-9 都是停止條件，文件沒有承諾 loadings 或 Phi 會準到哪一位。沒有任何一家文件針對 Phi 另設精度。
- Stata 與 Mplus 判斷「同一解」的依據都是準則值，這跟 psych 用 hyperplane count 挑選、lavaan 用最小準則值挑選（見第 5 節）不同；在準則值幾乎相同但 loadings 可沿平坦方向移動的情況下，這些軟體之間本來就可能回報不同的 Phi。
- Stata 手冊文字說「三次試驗收斂到不同旋轉」，但印出的輸出只有一次不同（458260.7）；文件範例是示意用途，不宜當精確規格引用。

### Gaps
- **Mplus 怎麼判定「unique」旋轉函數值**（比到幾位、是否也比 loadings）文件沒寫，論壇回覆也沒說；信心：低。
- **Stata 怎麼判定「同一解」**（`r(fmin)` 是否經四捨五入）文件沒寫；從範例輸出推測是比準則值，信心：中。
- SPSS Algorithms PDF 的收斂公式是圖片，文字擷取遺失，所以 `RCONVERGE` 比的是準則值變化還是角度變化無法從原文確認；信心：中低。
- SAS 只讀到 9.3 版文件頁；目前的 SAS/STAT（Viya）文件頁是 JavaScript 渲染，無法抓取，預設值是否變動未查證，信心：中高（此類預設極少改）。
- **CEFA**（Browne, Cudeck, Tateneni & Mels）與 **FACTOR**（Lorenzo-Seva & Ferrando）的手冊都沒有讀到，無法說明它們對 loadings 與因素相關的精度目標；只查到 FACTOR 程式與 Robust Promin 的論文存在 — [FACTOR: A computer program to fit the EFA model (BRM 2006)](https://link.springer.com/article/10.3758/BF03192753)、[Robust Promin (Liberabit 2019)](http://www.scielo.org.pe/pdf/liber/v25n1/a08v25n1.pdf)
- 各商業軟體預設輸出的小數位數（例如 SPSS 3 位、Mplus 3 位）本次沒有從官方文件查證。

---

## 5. 發表過的跨軟體與局部解比較研究：用什麼指標、有沒有單獨看 Phi、多大差異算無關緊要

### Takeaway
Grieder & Steiner（2022, BRM）比較 R psych 與 SPSS 時，只看 loadings/pattern 係數的整體、平均、最大絕對差，加上 Tucker 一致性係數與「指標到因素對應」（顯著門檻 ≥ .20），**沒有單獨比較 Phi**；他們用自己的 EFAtools 重現 psych 到小數第 14 位、重現 SPSS 旋轉結果到小數第 4 位，並把兩軟體平均最大差 0.12 稱為不可忽略。局部解研究（Hattori 等 2017；Nguyen & Waller 2023）用準則值區分解、用一致性係數 .98/.92 或 RMSD 衡量相似；Nguyen & Waller 明確示範**準則值在 5 位小數相同、loadings 與 Phi 卻明顯不同**的解。沒有找到 lavaan 對 Mplus 旋轉結果的系統性比較研究。

### Cited Findings
**Grieder & Steiner (2022)** "Algorithmic jingle jungle: A comparison of implementations of principal axis factoring and promax rotation in R and SPSS", *Behavior Research Methods* 54, 54–74 — [PMC8863761](https://www.ncbi.nlm.nih.gov/pmc/articles/PMC8863761/)
- **重現精度**：EFAtools 重現 psych `fa` 的未旋轉 PAF、varimax 與 promax pattern 係數「至少到小數第 14 位」；重現 SPSS 的未旋轉 PAF 到第 9 位、varimax 與 promax 到「至少第 4 位」（Table S1） — [PMC8863761, Reproduction with the EFAtools package](https://www.ncbi.nlm.nih.gov/pmc/articles/PMC8863761/)
- **SPSS 重現的限制**：沒有 SPSS 原始碼，只能照演算法手冊實作；varimax 準則經過調整後才更接近 SPSS 輸出 — [同上, Limitations](https://www.ncbi.nlm.nih.gov/pmc/articles/PMC8863761/)
- **比較指標**：每個資料集計算 loadings/pattern 係數的整體、平均、最大絕對差，以及整體、平均、最小因素一致性係數；另計「指標到因素對應」是否不同 — [同上, Methods](https://www.ncbi.nlm.nih.gov/pmc/articles/PMC8863761/)
- **顯著負荷門檻**：可接受解要求無 Heywood case（共同性或未旋轉 loadings ≥ .998）且每個因素至少兩個 ≥ .20 的 pattern 係數；對應不同的定義是同一指標在兩解顯著負荷於不同因素，或只在一解顯著 — [同上, Methods](https://www.ncbi.nlm.nih.gov/pmc/articles/PMC8863761/)
- **結果數字**（真實資料）：未旋轉 PAF 平均絕對差 0.002、平均最大差 0.01；varimax 後 0.003 與 0.02；promax 後 0.03 與 0.12；一致性係數平均 .997、.996、.99，最小值平均 .98、.98、.95；promax 解有 41.7% 資料集的對應不同 — [同上, Results](https://www.ncbi.nlm.nih.gov/pmc/articles/PMC8863761/)
- **研究者的判讀**：多數差異小，但兩軟體 promax 解的平均最大差 0.12 被形容為「a non-negligible 0.12」 — [同上, Discussion](https://www.ncbi.nlm.nih.gov/pmc/articles/PMC8863761/)
- **Phi**：文中 Φ 只出現在模擬研究產生母體相關矩陣（R = ΛΦΛᵀ）與設計因子（因素間相關高低）；真實資料比較與模擬評估（RMSE、Heywood、對應錯誤）都只針對 pattern 矩陣，沒有比較兩軟體估出的因素相關 — [同上, Methods/Statistical analyses](https://www.ncbi.nlm.nih.gov/pmc/articles/PMC8863761/)
- **收斂準則**：PAF 迭代的預設收斂準則在 psych 與 SPSS 都是 10⁻³ — [同上](https://www.ncbi.nlm.nih.gov/pmc/articles/PMC8863761/)

**EFAtools 1.1.0 的比較工具 `efa_compare`**（上述研究的工具，現行版本）
- 比較 loadings 或共同性，回傳平均、中位數、最小、最大絕對差、RMSE、「兩者在幾位小數內完全一致」（`are_equal`，依絕對值比，所以正負號相反也算一致）、以及顯著負荷（`thresh = .3`）對應不同的數量；預設用 Tucker 一致性係數做一對一重排 — [EFAtools R/efa_compare.R L13-L85](https://github.com/cran/EFAtools/blob/master/R/efa_compare.R)
- 顯示門檻預設：平均與中位數大於 `.001` 標紅、最大值大於 `.001` 標紅、一致小數位數少於 `3` 位標紅，畫圖門檻 `plot_red = .01` — [efa_compare.R L30-L43、L109-L122](https://github.com/cran/EFAtools/blob/master/R/efa_compare.R)

**Hattori, Zhang & Preacher (2017)** "Multiple local solutions and geomin rotation", *Multivariate Behavioral Research* 52(6), 720–731 — [作者網站 PDF](https://www.quantpsy.org/pubs/hattori_zhang_preacher_2017.pdf)
- 用 30、100、1,000 個隨機起點與 ε = .02、.01、.001、.0001；以 geomin 準則值區分全域解與局部解（例如文中一個範例 100 個起點得到準則值 3.280 的全域解，以及 3.817、3.820 等三個局部解） — [同上](https://www.quantpsy.org/pubs/hattori_zhang_preacher_2017.pdf)
- 解的相似度用欄位對應的 Tucker 一致性係數，嚴格標準是所有因素 > .98，寬鬆標準是 > .92（引 MacCallum 等 1999 為「good」對應）；表格雖並列 loadings 與因素相關，但相似度指標只算在 loadings 上 — [同上](https://www.quantpsy.org/pubs/hattori_zhang_preacher_2017.pdf)；[線上附錄](http://quantpsy.org/pubs/hattori_zhang_preacher_2017_appx.pdf)
- 結論包括 ε = .01 多數情況表現令人滿意、100 個起點似乎足以檢視多重解 — [同上](https://www.quantpsy.org/pubs/hattori_zhang_preacher_2017.pdf)

**Nguyen & Waller (2023)** "Local minima and factor rotations in exploratory factor analysis", *Psychological Methods* 28(5), 1122–1141, [doi:10.1037/met0000467](https://doi.org/10.1037/met0000467)；補充資料 — [APA supplemental page](https://supp.apa.org/psycarticles/supplemental/met0000467/met0000467_supp.html)
- **解集合的分法**：用 fungible `faMain` 從多個隨機起點旋轉，依複雜度（準則）值分成「solution sets」；fungible 文件說把準則值四捨五入到 `epsilon`（預設 1e-5）對應的位數再分組 — [fungible R/faMain.R L149-L153](https://github.com/cran/fungible/blob/master/R/faMain.R)
- **實際四捨五入位數**：程式是 `round(DisVal, digits = nchar(cnRotate$epsilon))`；本機 R 查得 `nchar(1e-5)` 與 `nchar(1e-6)` 都是 5（字串 `"1e-05"`），所以實際是**小數點後 5 位（絕對量）**，不是文件說的 5 位有效數字，而且調小 epsilon 不會增加位數 — [faMain.R L1031-L1036](https://github.com/cran/fungible/blob/master/R/faMain.R)
- **同準則值、不同解的實例 1**（補充資料 Table S3、S4）：模擬資料 Model 21 第 26 樣本，oblique geomin 200 個起點，兩個解準則值都是 .82126；對母體對齊後 RMSE 分別 .18630 與 .19089，兩解互相對齊後 RMSD = .10816；兩解的 Phi 也不同，例如 f2–f5 為 0.20 對 0.06、f3–f5 為 0.23 對 0.09、f2–f4 為 0.15 對 −0.10 — [met0000467_sm2.pdf, pp. 29-31](https://supp.apa.org/psycarticles/supplemental/met0000467/met0000467_sm2.pdf)
- **同準則值、不同解的實例 2**（Table S17）：54 題人格資料 9 因素 geominQ、1000 個起點得到 108 個解集合；Set 49 的 17 個解準則值（到 5 位小數）都是 1.47211，但分成兩組質性不同的 pattern；註解列出兩解準則值 1.472106 與 1.472109，對應因素一致性係數低到 .55 — [met0000467_sm2.pdf, p. 84 起](https://supp.apa.org/psycarticles/supplemental/met0000467/met0000467_sm2.pdf)；[met0000467_sm3.Rmd](https://supp.apa.org/psycarticles/supplemental/met0000467/met0000467_sm3.Rmd)
- **作者判讀**：只靠準則值分組可能低估局部解數；他們計算同一解集合內、對齊母體後的兩兩 RMSD，只要任一 RMSD > 0 就記為含不同 pattern，結論是這種情況機率很小，所以準則值分組提供「very tight lower bound」 — [met0000467_sm2.pdf, pp. 32, 64](https://supp.apa.org/psycarticles/supplemental/met0000467/met0000467_sm2.pdf)
- **fungible 的配套工具**：`faLocalMin` 專門計算「共同複雜度值」解集合內 pattern 的兩兩 RMSD（先用 Hungarian 法對齊），預設印 5 位有效數字 — [fungible R/faLocalMin.R L5-L22](https://github.com/cran/fungible/blob/master/R/faLocalMin.R)

**lavaan 與 Mplus**
- lavaan 0.7-2 的 `rotation.args` 預設：`rstarts = 30`、`algorithm = "gpa"`、`gpa_tol = 1e-05`、`tol = 1e-08`、`geomin_epsilon = 0.001` — [lavaan R/lav_options_default.R L273-L296](https://github.com/cran/lavaan/blob/master/R/lav_options_default.R)
- lavaan 在多個起點中用 `which.min` 取準則值最小者 — [lavaan R/lav_matrix_rotate.R L165-L218](https://github.com/cran/lavaan/blob/master/R/lav_matrix_rotate.R)
- lavaan 官方 ESEM/EFA 教學示範以 `rotation = "geomin"` 搭配 `geomin.epsilon = 0.0001`、`rstarts = 30` 等設定來模仿 Mplus — [lavaan.org: ESEM and EFA](https://lavaan.ugent.be/tutorial/efa.html)（本次只讀到搜尋摘要，未抓全文）

### Inferences
- 跨軟體研究判斷「實質差異」的尺度是 0.01 到 0.1 級（平均最大差 0.02、0.12；顯著負荷 .20/.30；一致性 .98/.92），自動化「完全重現」的尺度是 1e-3 到 1e-14（EFAtools 標紅 .001、重現 SPSS 到 4 位、psych 到 14 位）。insyra 提議的 5e-3 落在兩者之間：比任何「完全重現」測試寬，但比任何「實質差異」門檻嚴一個數量級以上。
- Nguyen & Waller 的實例直接說明「準則值相同」不足以證明「同一個解」：1.472106 對 1.472109 的相對差約 2.0e-6（本機 R 計算），遠小於 insyra 用的相對 1e-4 門檻，這兩個解卻有一致性 .55 的因素。所以 insyra 的「準則值同一最小值」判定本身擋不住這類不同解，真正擋住它們的是後面的 loadings/Phi 容忍度：這兩個實例的 loadings RMSD 約 .11、Phi 元素差最大到 .25，都遠大於 5e-3，仍會失敗。
- insyra 實測的情況（40 個起點準則值差最多 2.7e-9、Phi 差 3.1e-3）跟 Nguyen & Waller 的實例性質不同：前者是同一個平坦最小值附近的數值漂移，後者是不同的局部解。已查到的研究都沒有處理「同一最小值但 Phi 漂移比 loadings 大」這種數值現象，只處理「不同局部解」。
- 研究者層面沒有任何來源把 Phi 當成比 loadings 更寬鬆的比較對象；多數研究根本不比 Phi，只比 loadings/pattern 係數。

### Gaps
- 沒有找到同時比較 lavaan、Mplus、psych 斜交旋轉輸出（含 Phi）的已發表研究；只有 lavaan 教學宣稱可模仿 Mplus，信心：可能不存在或未被索引。
- Grieder & Steiner 的 Table S1（重現精度細節，是否含因素相關）在補充資料中，本次沒有讀；正文沒有提到 Phi 的重現精度。
- Nguyen & Waller 正文（非補充資料）付費牆後，未讀；「RMSD > 0」是否經過四捨五入後才判斷，補充資料沒寫清楚。
- EFAtools 2020–2022 年（論文當時）`COMPARE` 函式的預設值是否與 1.1.0 版 `efa_compare` 相同，未查證。

---

## 6. 總結：實作與驗證者實際用多少容忍度比 loadings 與 Phi？有沒有人把 Phi 放寬或收緊？

### Takeaway
沒有任何已查來源以原則性理由把 Phi（或 structure、scores）設得比 loadings 寬鬆或嚴格：factor_analyzer、JASP 都對所有輸出用同一個門檻，多數跨軟體研究與 statsmodels 的 GPA 測試根本不比 Phi。唯一出現「Phi 與分數比 loadings 寬」的是 statsmodels 對 Stata 的 smoke test（Phi 約 2.6e-4、分數約 1e-3），但那是換參考軟體造成的附帶結果。insyra 的 5e-3 比所有數值重現測試寬，但仍比所有「實質差異」門檻嚴 4 倍以上，並足以抓出文獻中的不同局部解。

### Cited Findings
| 來源 | loadings | Phi | structure / scores | 「同一解」判定 |
| --- | --- | --- | --- | --- |
| statsmodels GPA 測試（對 UCLA GPA 範例） | `atol=1e-5`（加 numpy 預設 rtol） | 不比 | 不比 | 不處理；另比迭代表 f 與 log10 梯度 — [test_rotation.py](https://github.com/statsmodels/statsmodels/blob/5032ffdf1e706b8b8a90248fd661995d224c2797/statsmodels/multivariate/factor_rotation/tests/test_rotation.py) |
| statsmodels `Factor`（對 R psych） | `decimal=8`（< 1.5e-8） | 不比 | 不比 | 手動翻符號 — [test_factor.py L119-L208](https://github.com/statsmodels/statsmodels/blob/5032ffdf1e706b8b8a90248fd661995d224c2797/statsmodels/multivariate/tests/test_factor.py#L119-L208) |
| statsmodels `Factor`（對 Stata，smoke） | 不比 | `rtol=1e-3`（≈2.6e-4） | scores `atol=1e-4, rtol=1e-3` | 手動翻符號 — [test_factor.py L320-L364](https://github.com/statsmodels/statsmodels/blob/5032ffdf1e706b8b8a90248fd661995d224c2797/statsmodels/multivariate/tests/test_factor.py#L320-L364) |
| factor_analyzer（對 R psych） | `abs_tol=0.1`，≥90%/95% 元素 | 不比 | 同 0.1 | 無對齊 — [test_utils.py](https://github.com/EducationalTestingService/factor_analyzer/blob/de933d26808039d965c117874c67260ec5149690/factor_analyzer/test_utils.py#L206-L396) |
| scikit-learn（對 psych::principal） | 絕對值、`decimal=3` | 無斜交 | 不比 | 取絕對值 — [test_factor_analysis.py L93-L109](https://github.com/scikit-learn/scikit-learn/blob/bbd90caf8b2f74e722f6a71364e886075b80c001/sklearn/decomposition/tests/test_factor_analysis.py#L93-L109) |
| JASP（迴歸測試） | 4 位 round + 4 位 signif | 同 loadings | structure 同 | 不處理（測試固定 `set.seed(1)`） — [jaspTools](https://github.com/jasp-stats/jaspTools/blob/55ab6bcd2e3828fc923a9bdf513c50a06da3d6ad/R/testthat-helper-tables.R#L76-L82) |
| EFAtools `efa_compare` | 標紅 > .001、< 3 位 | 函式不處理 | — | 一致性係數重排 — [efa_compare.R](https://github.com/cran/EFAtools/blob/master/R/efa_compare.R) |
| Grieder & Steiner 2022 | 重現 psych 14 位、SPSS 4 位；實質差：最大差 .12「不可忽略」、顯著 ≥ .20 | 不比 | 不比 | — [PMC8863761](https://www.ncbi.nlm.nih.gov/pmc/articles/PMC8863761/) |
| Hattori 等 2017 | 一致性 > .98（嚴）/ > .92（寬） | 列出但不算指標 | — | 準則值 — [PDF](https://www.quantpsy.org/pubs/hattori_zhang_preacher_2017.pdf) |
| Nguyen & Waller 2023 / fungible | 對齊後 RMSD | 列出但不算指標 | — | 準則值四捨五入到小數 5 位 — [faMain.R](https://github.com/cran/fungible/blob/master/R/faMain.R) |
| Stata `protect()` | 手冊印 4 位 | — | — | 看準則值（位數未載明） — [[MV] rotate](https://www.stata.com/manuals/mvrotate.pdf) |
| Mplus `RSTARTS` | — | — | — | 「unique rotation function values」（位數未載明） — [Chapter 16](https://www.statmodel.com/download/usersguide/Chapter16.pdf) |
| lavaan 0.7-2 | — | — | — | 準則值最小者 — [lav_matrix_rotate.R](https://github.com/cran/lavaan/blob/master/R/lav_matrix_rotate.R) |
| SPSS / SAS | 收斂準則 0.0001 / 1E-9，無隨機起點 | 無另設 | 無另設 | 不適用 — [SPSS CSR](https://public.dhe.ibm.com/software/analytics/spss/documentation/statistics/26.0/en/client/Manuals/IBM_SPSS_Statistics_Command_Syntax_Reference.pdf)、[SAS 9.3](https://support.sas.com/documentation/cdl/en/statug/63962/HTML/default/statug_factor_sect006.htm) |

（表中每格的原始出處與細節見第 1 到第 5 節。`r_packages.md` 記錄 GPArotation 自己的測試也是 Phi 與 loadings 同一個 fuzz。）

### Inferences
- **對「Phi/structure/scores 比 loadings 寬」的先例**：只有 statsmodels 對 Stata 那一處，且是附帶而非設計。業界主流是「一次比對一個門檻、所有輸出同門檻」，或乾脆不比 Phi。所以 insyra 的提案**在形式上沒有直接先例**，需要靠自己量到的數據（同一最小值下 Phi 漂移 3.1e-3 而 loadings 2.1e-4）來支撐，而不是引用慣例。
- **對 5e-3 這個量級**：比 statsmodels 對 Stata 的 Phi（≈2.6e-4）寬約 20 倍、比 EFAtools 標紅門檻 .001 寬 5 倍，但比 Grieder & Steiner 報告的最小實質差（varimax 平均最大差 .02）嚴 4 倍、比顯著負荷門檻 .20 嚴 40 倍。以「不會把不同局部解誤判為相同」來看是安全的（文獻實例的差異是 .1 級）。
- **真正的風險在準則值閘門，不在 5e-3**：insyra 用相對 1e-4 判定「同一最小值」，比 fungible 的 5 位小數絕對量（在準則值約 1.47 時相當於相對約 7e-6）寬，也寬於 Nguyen & Waller 那組不同解的相對差 2e-6。既然實測同一最小值的準則差只有 2.7e-9，可以考慮把閘門收緊到例如相對 1e-6 到 1e-7，讓「放寬 Phi」只在真正的同一最小值時發生。這是推論，需要用 insyra 自己的案例驗證不會誤殺。
- **可參考的替代做法**：若不想只靠元素容忍度，可仿照 Hattori 等與 EFAtools，另加一個不受數值漂移影響的結構性檢查，例如對齊後 loadings 的 Tucker 一致性 > .98，或顯著負荷（≥ .20 或 .30）對應完全相同，作為「同一解」的第二道確認。
- **為何 Phi 在近秩一解中漂移較大**：已查來源都沒有討論這個數值現象。從 Phi = TᵀT、L = A(Tᵀ)⁻¹ 的關係推測，兩因素高度相關時旋轉矩陣沿平坦方向的小擾動，對 Phi 的一階影響可能大於對 loadings 的影響；這是推論，未經任何來源證實。

### Gaps
- 沒有找到任何來源專門研究「同一旋轉最小值下，Phi 的數值穩定性相對 loadings 如何」，因此無法引用外部證據說明 Phi 應該放寬多少。
- 商業軟體（Mplus、Stata）判定「同一解」的確切位數沒有公開文件；CEFA、FACTOR 的精度目標沒有讀到。
