# R 因素分析套件如何比對旋轉解（loadings、Phi、structure、scores）與使用的容忍度

閱讀範圍與版本（2026-09-17 讀取）：GPArotation 2026.8-2（CRAN 鏡像 commit `3d15692`）、psych 2.6.5（`2b17d1e`）、fungible 2.4.8（`d19e1e5`）、lavaan 0.7-2（CRAN 鏡像 `7b78a73`，GitHub `yrosseel/lavaan` HEAD `620fa97`）、EFAtools 1.1.0（`ce93c76`）、waldo 0.6.2、semTools 0.5-9。本機 R 只裝了 GPArotation 2026.8.2 與 psych 2.6.5，其餘套件只讀原始碼、沒有執行。

## GPArotation：tests/ 比對什麼、容忍度多少、隨機起點怎麼判定「同一個最小值」、有沒有比對 Phi

### Takeaway
GPArotation 的測試會比對 Phi，而且每一處都跟 loadings 用**同一個** fuzz。取值依比對對象而定：跟已存參考值比是 1e-5，演算法不同但收斂到同一最小值時是 1e-4 到 1e-3，跟 SPSS 三位小數輸出比是 1e-3 到 4e-3。`.GPA_RS_engine` 判斷「同一最小值」的做法，是把準則值四捨五入到 eps 的整數倍（預設 1e-5，絕對量）再比相等。它只比準則值，不比 loadings 或 Phi。

### Cited Findings
- **隨機起點挑選**：每個起點取迭代紀錄 `Table` 最後一列的準則值。只有嚴格更小（`<`）才取代目前最佳，同分時保留先找到的那個 — [R/GPFRS.R L76-89](https://github.com/cran/GPArotation/blob/3d15692ba675723fd91dfb9fb01273a86826a7b9/R/GPFRS.R#L76-L89)
- **`randStartChar` 的「同一最小值」判定（已查證）**：`Q_round <- round(Qvalues / eps) * eps`，`atMinimum = sum(Q_round == Qmin_round)`，`localMins = length(unique(Q_round))`，`Converged = sum(Qconverged)`。只有 `randomStarts > 1` 才計算，而且沒收斂的起點也算進 `atMinimum`／`localMins` — [R/GPFRS.R L92-101](https://github.com/cran/GPArotation/blob/3d15692ba675723fd91dfb9fb01273a86826a7b9/R/GPFRS.R#L92-L101)
- 引擎在隨機起點流程裡對 Phi 只做一件事：設定 dimnames。不拿 Phi 判斷解是否相同 — [R/GPFRS.R L104-106](https://github.com/cran/GPArotation/blob/3d15692ba675723fd91dfb9fb01273a86826a7b9/R/GPFRS.R#L104-L106)
- **Harman.R**：`fuzz <- 1e-5`，註解說明用 eps=1e-5 時測試無法做得比這更精確。oblimin 的 loadings（L123）、Phi（L144）、Th（L165）都用同一個 fuzz 比對寫死的參考值（取最大絕對差）— [tests/Harman.R L9](https://github.com/cran/GPArotation/blob/3d15692ba675723fd91dfb9fb01273a86826a7b9/tests/Harman.R#L9)、[L139-150](https://github.com/cran/GPArotation/blob/3d15692ba675723fd91dfb9fb01273a86826a7b9/tests/Harman.R#L139-L150)
- **Thurstone.R**：fuzz 1e-5。Phi 比對參考值（L163），並用同一個 fuzz 檢查 `Phi == t(Th) %*% Th` 的內部一致性 — [tests/Thurstone.R L26](https://github.com/cran/GPArotation/blob/3d15692ba675723fd91dfb9fb01273a86826a7b9/tests/Thurstone.R#L26)、[L157-184](https://github.com/cran/GPArotation/blob/3d15692ba675723fd91dfb9fb01273a86826a7b9/tests/Thurstone.R#L157-L184)
- **legacyVsBB.R（路徑不同、最小值相同）**：`fuzz <- 1e-3`，註解寫「different paths, same minimum」。`.check()` 先用 `.sortGPALoadings` 標準化因素順序，再以 `max(abs(abs(L1)-abs(L2)))` 比 loadings、`max(abs(abs(Phi1)-abs(Phi2)))` 比 Phi，**兩者用同一個 tol** — [tests/legacyVsBB.R L3-8](https://github.com/cran/GPArotation/blob/3d15692ba675723fd91dfb9fb01273a86826a7b9/tests/legacyVsBB.R#L3-L8)、[L18-43](https://github.com/cran/GPArotation/blob/3d15692ba675723fd91dfb9fb01273a86826a7b9/tests/legacyVsBB.R#L18-L43)
- **algorithmTests.R**：`fuzz <- 1e-4`，註解說明比舊版測試寬鬆，因為路徑不同但最小值相同。只比 loadings（abs 後取最大差），整個檔案沒有比 Phi。隨機起點測試只檢查 `randStartChar` 有產生、起點數正確 — [tests/algorithmTests.R L10](https://github.com/cran/GPArotation/blob/3d15692ba675723fd91dfb9fb01273a86826a7b9/tests/algorithmTests.R#L10)、[L23-25](https://github.com/cran/GPArotation/blob/3d15692ba675723fd91dfb9fb01273a86826a7b9/tests/algorithmTests.R#L23-L25)、[L120-128](https://github.com/cran/GPArotation/blob/3d15692ba675723fd91dfb9fb01273a86826a7b9/tests/algorithmTests.R#L120-L128)
- **legacyVsBB_extra.R**：比的是準則值，不是矩陣。`f.bb > f.leg + fuzz`（fuzz 1e-4）就判失敗，也就是要求 BB 找到的解同樣好或更好 — [tests/legacyVsBB_extra.R L8](https://github.com/cran/GPArotation/blob/3d15692ba675723fd91dfb9fb01273a86826a7b9/tests/legacyVsBB_extra.R#L8)、[L18-23](https://github.com/cran/GPArotation/blob/3d15692ba675723fd91dfb9fb01273a86826a7b9/tests/legacyVsBB_extra.R#L18-L23)
- **legacyVsNew.R（同一演算法的重寫版 vs 舊版）**：`tol <- 1e-5`，loadings、Th、Phi 全部用同一個 tol，其中包含一組隨機起點測試（Test 8 的 Phi，L174）— [tests/legacyVsNew.R L4-10](https://github.com/cran/GPArotation/blob/3d15692ba675723fd91dfb9fb01273a86826a7b9/tests/legacyVsNew.R#L4-L10)、[L126](https://github.com/cran/GPArotation/blob/3d15692ba675723fd91dfb9fb01273a86826a7b9/tests/legacyVsNew.R#L126)、[L174](https://github.com/cran/GPArotation/blob/3d15692ba675723fd91dfb9fb01273a86826a7b9/tests/legacyVsNew.R#L174)
- **print-GPArotation.R（兩個隨機起點收斂到同一個局部最小值）**：挑 seed 238 與 46，因為兩者收斂到同一最小值。排序前要求 loadings、Th、Phi 都**不同**（差異需大於 1e-5）。排序後要求三者都在 `fuzz = 1e-5` 內一致。另外把 factanal 內建旋轉跟 GPArotation 兩步驟做法對照，loadings（取 abs）與 Phi 都用 1e-4 — [tests/print-GPArotation.R L12](https://github.com/cran/GPArotation/blob/3d15692ba675723fd91dfb9fb01273a86826a7b9/tests/print-GPArotation.R#L12)、[L22-74](https://github.com/cran/GPArotation/blob/3d15692ba675723fd91dfb9fb01273a86826a7b9/tests/print-GPArotation.R#L22-L74)、[L99-111](https://github.com/cran/GPArotation/blob/3d15692ba675723fd91dfb9fb01273a86826a7b9/tests/print-GPArotation.R#L99-L111)
- **rotationsRS.R（隨機起點）**：標準 `fuzz 1e-5`，受 BLAS 影響的測試用 `fuzz_lo 1e-4`。100 個起點的 `Varimax` 對照 `stats::varimax`，normalize=FALSE 用 0.001、normalize=TRUE 用 0.01。其餘測試只比排序後的 loadings 與已存參考值，檔案中沒有比 Phi — [tests/rotationsRS.R L6-13](https://github.com/cran/GPArotation/blob/3d15692ba675723fd91dfb9fb01273a86826a7b9/tests/rotationsRS.R#L6-L13)、[L37-54](https://github.com/cran/GPArotation/blob/3d15692ba675723fd91dfb9fb01273a86826a7b9/tests/rotationsRS.R#L37-L54)、[L246](https://github.com/cran/GPArotation/blob/3d15692ba675723fd91dfb9fb01273a86826a7b9/tests/rotationsRS.R#L246)
- **KaiserNormalization.R（對照 SPSS 三位小數輸出）**：quartimax loadings 用 1e-3。oblimin pattern 用 3e-3。structure matrix（`loadings %*% Phi`）用 4e-3。註解把差異歸因於和 SPSS 在第四位小數不同。這是檔案中唯一「structure 比 loadings 寬」的地方，而參考值本身只有三位小數 — [tests/KaiserNormalization.R L11](https://github.com/cran/GPArotation/blob/3d15692ba675723fd91dfb9fb01273a86826a7b9/tests/KaiserNormalization.R#L11)、[L36-74](https://github.com/cran/GPArotation/blob/3d15692ba675723fd91dfb9fb01273a86826a7b9/tests/KaiserNormalization.R#L36-L74)
- **target_vs_bb_clash.R**：會算 loadings 是否在 1e-4 內一致（`Matrices_Match`），但通過條件只看收斂。比對矩陣的條件被註解掉，旁邊註明是舊的有問題檢查 — [tests/target_vs_bb_clash.R L49](https://github.com/cran/GPArotation/blob/3d15692ba675723fd91dfb9fb01273a86826a7b9/tests/target_vs_bb_clash.R#L49)、[L99-109](https://github.com/cran/GPArotation/blob/3d15692ba675723fd91dfb9fb01273a86826a7b9/tests/target_vs_bb_clash.R#L99-L109)
- 其他容忍度：vgQ 準則值 f 與梯度 Gq 新舊版比對用 1e-10（[tests/vgQtestOldVsNew.R L12](https://github.com/cran/GPArotation/blob/3d15692ba675723fd91dfb9fb01273a86826a7b9/tests/vgQtestOldVsNew.R#L12)）。殘差診斷用 1e-6（[tests/diagnosticsTests.R L8](https://github.com/cran/GPArotation/blob/3d15692ba675723fd91dfb9fb01273a86826a7b9/tests/diagnosticsTests.R#L8)）。Cureton-Mulaik 對照 Browne (2001) 表格用 1e-3（[tests/cm_browne.R L8](https://github.com/cran/GPArotation/blob/3d15692ba675723fd91dfb9fb01273a86826a7b9/tests/cm_browne.R#L8)）
- 測試寫法：全部是 base R 腳本，自己設 `all.ok` 旗標、失敗時 `stop()`，沒有用 `all.equal`／testthat，一律取最大絕對差（見上列各檔）
- NEWS：2026.8-2 只改測試。2026.6-1 起預設改為 `algorithm = "bb"`，要重現舊結果需設 `"legacy"`。2026.4-1 新增 GPA2local vignette（局部最小值診斷）— [NEWS L1-3、L22-26、L90-92](https://github.com/cran/GPArotation/blob/3d15692ba675723fd91dfb9fb01273a86826a7b9/NEWS)
- **GPA2local vignette**：`GPFallMinima` 用同一套 `round(Q/eps)*eps` 分群（L93-94），`minimumInclusion` 用來排除數值雜訊造成的假最小值（L112-113、L350-352）。建議把 deltaF < 0.001 視為通常可忽略、> 0.01 則可能看得出 loadings 不同（L354-357）。比較不同最小值用排序後的絕對 loadings 圖（L282-285）。全文沒有提出 Phi 的比較方法 — [vignettes/GPA2local.Rnw](https://github.com/cran/GPArotation/blob/3d15692ba675723fd91dfb9fb01273a86826a7b9/vignettes/GPA2local.Rnw#L93-L94)（本機 `/opt/homebrew/lib/R/4.6/site-library/GPArotation/doc/GPA2local.Rnw` 與鏡像檔 diff 相同）
- **本機重現 GPArotation 自己的測試資料（GPArotation 2026.8.2，Rscript，腳本在 scratchpad `rsrc/probe.R`）**：print-GPArotation 的 seed 238/46：ΔQ 1.9e-11，loadings 1.9e-6、Phi 1.5e-6、Th 2.3e-6。legacyVsBB 的斜交測試（BB vs legacy）：T9 quartimin/Harman8 loadings 1.1e-6、Phi 1.7e-6。T10 oblimin/box26 1.3e-5、6.5e-7。T11/T16 geomin/CCAI 9.0e-7、1.6e-7。T13 bentlerQ/CCAI 1.0e-4、4.5e-5。T14 oblimin/CCAI 5.6e-6、2.6e-6。這幾組的 Phi 差異都不超過 loadings 差異的約 1.5 倍，多數比 loadings 小

### Inferences
- GPArotation 對「同一最小值」的判準是準則值絕對量化到 eps（預設 1e-5）。insyra 用相對 1e-4：Q≈1 時比 GPArotation 寬（1e-4 對 1e-5），Q≈0.01 時反而比較嚴（1e-6 對 1e-5）。另外四捨五入到格點還會出現交界問題：差 1e-9 的兩個值可能剛好落在不同格而被判成不同最小值，差將近 eps 的兩個值則可能被判成相同。
- GPArotation 測試中，Phi 從來沒有比 loadings 更寬的容忍度（structure 對 SPSS 三位小數的 4e-3 是唯一例外，原因在參考值精度）。「路徑不同、最小值相同」的放寬是 loadings 與 Phi **一起**放寬（1e-4 或 1e-3）。
- GPArotation 自己的測試資料條件數都不差，Phi 差異與 loadings 同量級甚至更小。insyra 量到的「近似一階的雙因素 ML/oblimin 解，Phi 差異約 15 倍於 loadings」，這種情況並沒有被 GPArotation 的測試涵蓋。推論：Phi 相對 loadings 被放大，來自解本身的條件數很差（因素高度相關、旋轉矩陣 T 接近奇異，以下簡稱病態），所以不能引用 GPArotation 的測試當反證，也不能當支持。

### Gaps
- 找不到 GPArotation 作者說明為何 legacyVsBB 用 1e-3、algorithmTests 用 1e-4 的文件，檔案註解之外沒有依據。
- GPArotation 沒有任何測試比對 factor scores（本來就不算分數），也沒有測試驗證 `randStartChar$atMinimum` 的數值。

## psych：如何跨起點或跨軟體比較旋轉解，`test.psych` 與 testthat 測試檢查什麼、精度多少

### Takeaway
psych 在隨機起點之間挑解，看的是 hyperplane count，不是準則值，而且完全不比 Phi。它的 testthat 測試只有 loadings 與 communalities（容忍度 1e-3 到 1e-5），沒有斜交、沒有 Phi、沒有 scores，而且看起來 R CMD check 不會執行這些測試。`test.psych()` 只把函式跑一遍，沒有任何數值斷言。

### Cited Findings
- 預設：`fa()` 的 `n.rotations = 20`、`hyper = .15`。內部的 `fac()` 預設是 1。`n.rotations > 1` 才呼叫 `faRotations` — [R/fa.R L40](https://github.com/cran/psych/blob/2b17d1e78b536c0e4cc807d8a83f546e0a0da85d/R/fa.R#L40)、[L159](https://github.com/cran/psych/blob/2b17d1e78b536c0e4cc807d8a83f546e0a0da85d/R/fa.R#L159)、[L638](https://github.com/cran/psych/blob/2b17d1e78b536c0e4cc807d8a83f546e0a0da85d/R/fa.R#L638)
- 2.6.5 才把預設從 1 改成 20，起因是 Keith Widaman 的範例資料。`faRotations` 與 hyperplane count 在 2.2.3 加入，並註明參考 Niels Waller — [inst/NEWS.Rd L7](https://github.com/cran/psych/blob/2b17d1e78b536c0e4cc807d8a83f546e0a0da85d/inst/NEWS.Rd#L7)、[L578-582](https://github.com/cran/psych/blob/2b17d1e78b536c0e4cc807d8a83f546e0a0da85d/inst/NEWS.Rd#L578-L582)
- `faRotations` 註明改寫自 fungible `faMain`。隨機起點用 `qr.Q(qr(matrix(rnorm(...))))`，沒有設 seed，而**第一個起點固定是單位矩陣**（所以是 19 個隨機起點加 1 個單位矩陣，不是 20 個全隨機）— [R/faRotate.r L3-19](https://github.com/cran/psych/blob/2b17d1e78b536c0e4cc807d8a83f546e0a0da85d/R/faRotate.r#L3-L19)
- 斜交 GPArotation 準則在每個起點的呼叫只傳 `list(loadings, Tmat=initial)`，沒有傳 `...`，所以 eps、maxit 等用 GPArotation 預設值 — [R/faRotate.r L120-130](https://github.com/cran/psych/blob/2b17d1e78b536c0e4cc807d8a83f546e0a0da85d/R/faRotate.r#L120-L130)
- 每個起點算完後依 `colSums` 的正負號翻轉 loadings 與 Phi，再記錄 `hyperplane = mean(abs(loadings) < hyper)`、`complexity`、`fit = tr(var(loadings^2))`、`indetermin` — [R/faRotate.r L136-151](https://github.com/cran/psych/blob/2b17d1e78b536c0e4cc807d8a83f546e0a0da85d/R/faRotate.r#L136-L151)
- **挑選規則**：hyperplane 最大者（精確相等比較）。同分時比 complexity 最小。第三層 fit 的比較結果沒有被指派（L167 等於沒作用）。還是同分就取第一個。最後用最佳起點的 `Tmat` 從原始 loadings 重新旋轉一次（L173-185）。整個過程完全不用準則值 — [R/faRotate.r L164-185](https://github.com/cran/psych/blob/2b17d1e78b536c0e4cc807d8a83f546e0a0da85d/R/faRotate.r#L164-L185)
- **同分處理的索引錯誤（程式碼判讀＋本機驗證 R 語意）**：L166 `best <- which(stats[best,"complexity"]==min(...))` 回傳的是「同分子集合裡的位置」，不是原本的起點編號。本機用合成資料重現：同分起點是 2、4、5，complexity 最小的是起點 4，但回傳 2，接著 `starts[[2]]` 就被拿去重新旋轉 — [R/faRotate.r L166](https://github.com/cran/psych/blob/2b17d1e78b536c0e4cc807d8a83f546e0a0da85d/R/faRotate.r#L166)
- `fa.congruence`／`factor.congruence`：預設 `digits = 2`，結果以 `round(..., digits)` 輸出。`structure=TRUE` 時比 Structure，否則比 loadings。沒有比 Phi 的選項 — [R/factor.congruence.R L9-28](https://github.com/cran/psych/blob/2b17d1e78b536c0e4cc807d8a83f546e0a0da85d/R/factor.congruence.R#L9-L28)、[L127](https://github.com/cran/psych/blob/2b17d1e78b536c0e4cc807d8a83f546e0a0da85d/R/factor.congruence.R#L127)
- `faCor` 用 `factor.congruence(f1,f2)` 比較同一資料的兩個因素解 — [R/faCor.R L46](https://github.com/cran/psych/blob/2b17d1e78b536c0e4cc807d8a83f546e0a0da85d/R/faCor.R#L46)
- **testthat 測試 `test_fa.r`**：Harman.Burt 的 minres loadings 對照書上表格（三位小數）用 `tolerance=.001`。communalities 用 .002 與 .0025。`fa(Thurstone,3,fm="mle",rotate="varimax")` 對照 `factanal` 用 `expect_equivalent(..., tolerance=10^-5)`。沒有斜交、Phi 或 scores 的測試 — [tests/testthat/test_fa.r L9-34](https://github.com/cran/psych/blob/2b17d1e78b536c0e4cc807d8a83f546e0a0da85d/tests/testthat/test_fa.r#L9-L34)
- `test_omega.r`：Schmid-Leiman loadings 對照 Jensen & Weng，以絕對差總和用 tolerance .001 — [tests/testthat/test_omega.r L19-23](https://github.com/cran/psych/blob/2b17d1e78b536c0e4cc807d8a83f546e0a0da85d/tests/testthat/test_omega.r#L19-L23)
- testthat 測試是 2023 年 10 月應 Coen Bernaards 要求才開始寫的 — [tests/testthat/ReadMe.text L3](https://github.com/cran/psych/blob/2b17d1e78b536c0e4cc807d8a83f546e0a0da85d/tests/testthat/ReadMe.text#L3)。NEWS 也記載「10/15/23 開始、進度很慢」— [inst/NEWS.Rd L195](https://github.com/cran/psych/blob/2b17d1e78b536c0e4cc807d8a83f546e0a0da85d/inst/NEWS.Rd#L195)
- DESCRIPTION 的 Suggests 沒有 testthat — [DESCRIPTION L11](https://github.com/cran/psych/blob/2b17d1e78b536c0e4cc807d8a83f546e0a0da85d/DESCRIPTION#L11)。鏡像樹中也沒有 `tests/testthat.R` 執行入口（GitHub tree API 列出的 tests/ 只有 testthat 子目錄下的檔案）
- `test.psych()`：對 5 組資料跑 `principal`、`fa`、`fa.parallel`、`VSS`、`ICLUST`、`omega`、`factor.congruence` 等，把結果收進清單，沒有任何比對或斷言 — [R/test.psych.r L19-55](https://github.com/cran/psych/blob/2b17d1e78b536c0e4cc807d8a83f546e0a0da85d/R/test.psych.r#L19-L55)、[L234-235](https://github.com/cran/psych/blob/2b17d1e78b536c0e4cc807d8a83f546e0a0da85d/R/test.psych.r#L234-L235)

### Inferences
- psych 不需要判斷起點之間是不是同一最小值，因為它根本不用準則值。hyperplane 是離散比例，同一最小值的起點幾乎一定同分，實際上會落到 complexity 比較。再加上索引錯誤，被選去重新旋轉的起點可能是任意一個。psych 報出來的解等於「某個起點重新旋轉後的結果」，在病態解上，各起點之間 Phi 的抖動會直接反映到 psych 的輸出。這支持 insyra 採取「跟 psych 逐元素比對時，Phi 無法期待比起點間的抖動更精確」的觀點。
- 沒有 testthat 在 Suggests、也沒有 `tests/testthat.R`，一般 R CMD check 應該不會執行這些測試（推論：依 R CMD check 只執行 `tests/*.R` 的慣例，沒實際跑 check 驗證）。
- `expect_equivalent` 在未宣告 edition 時走 `all.equal`，tolerance 是平均相對差，不是最大絕對差（推論自 testthat 第 2 版語意，未在本機執行）。

### Gaps
- psych 的測試、文件、NEWS 都沒有提到 Phi、structure 或 scores 在隨機起點之間的比較或容忍度。
- 沒有找到 psych 對照其他軟體（SPSS、Mplus）的自動化測試。跨軟體比較散見於文件，不是測試。

## fungible（faMain）：如何從隨機起點辨識不同解（四捨五入位數、numberStarts、局部解回報），有沒有比 Phi

### Takeaway
`faMain` 依準則值分群局部解：程式把準則值 `round` 到 `nchar(epsilon)` 位小數。這個寫法對 1e-3 到 1e-8 的 epsilon 都得到 5，所以實際上固定是 5 位小數，跟文件說的「依 epsilon 的有效位數」不符。同一群內的差異診斷（`faLocalMin`）只算 loadings 的 RMSD，沒有門檻，也不比 Phi。fungible 的 testthat 測試沒有涵蓋 faMain 或旋轉。

### Cited Findings
- `rotateControl` 預設：`numberStarts = 10`，第一個起點一律從未旋轉的方向開始。`epsilon = 1e-5`。`maxItr = 15000`。`Seed = 1` — [R/faMain.R L92-95](https://github.com/cran/fungible/blob/d19e1e59705ea956d23a6aa67e263866abd40977/R/faMain.R#L92-L95)、[L506-514](https://github.com/cran/fungible/blob/d19e1e59705ea956d23a6aa67e263866abd40977/R/faMain.R#L506-L514)、[L405](https://github.com/cran/fungible/blob/d19e1e59705ea956d23a6aa67e263866abd40977/R/faMain.R#L405)
- 旋轉交給 GPArotation（例如 `GPArotation::oblimin(lambda, Tmat = spinMatrix, eps = cnRotate$epsilon, ...)`）。起點在 `set.seed(Seed)` 之後產生 — [R/faMain.R L591-625](https://github.com/cran/fungible/blob/d19e1e59705ea956d23a6aa67e263866abd40977/R/faMain.R#L591-L625)、[L893-896](https://github.com/cran/fungible/blob/d19e1e59705ea956d23a6aa67e263866abd40977/R/faMain.R#L893-L896)
- 每個起點的準則值取 `min(attempt$Table[, 2])`，也就是迭代過程中的最小值，不是最後一步的值 — [R/faMain.R L936](https://github.com/cran/fungible/blob/d19e1e59705ea956d23a6aa67e263866abd40977/R/faMain.R#L936)
- 依準則值升冪排序。非 target 旋轉會先用 `orderFactors(..., salient = .01, reflect = TRUE)` 同時排序、翻轉 loadings 與 Phi — [R/faMain.R L953-993](https://github.com/cran/fungible/blob/d19e1e59705ea956d23a6aa67e263866abd40977/R/faMain.R#L953-L993)
- **分群規則**：`DisVal <- round(x = DisVal, digits = nchar(cnRotate$epsilon))`，`localMins <- unique(DisVal)`，依相等分到各組。回報的解就是準則值最小的那個（排序後第一個）的 loadings 與 Phi — [R/faMain.R L1029-1057](https://github.com/cran/fungible/blob/d19e1e59705ea956d23a6aa67e263866abd40977/R/faMain.R#L1029-L1057)
- 文件說預設 1e-5 時只比到「五位有效數字」— [R/faMain.R L149-153](https://github.com/cran/fungible/blob/d19e1e59705ea956d23a6aa67e263866abd40977/R/faMain.R#L149-L153)
- **本機驗證 `nchar`（R 4.6）**：`nchar(1e-3)`、`nchar(1e-4)`、`nchar(1e-5)`、`nchar(1e-6)`、`nchar(1e-8)` 都是 5（`as.character` 分別是 "0.001"、"1e-04"、"1e-05"、"1e-06"、"1e-08"），`nchar(0.01)` 是 4。`round(x, digits)` 取的是小數位數，不是有效位數
- 輸出包含 `localSolutions`（每個解都有 loadings、Phi、RotationComplexityValue、facIndeterminacy、RotationConverged）、`numLocalSets` 與 `localSolutionSets` — [R/faMain.R L193-208](https://github.com/cran/fungible/blob/d19e1e59705ea956d23a6aa67e263866abd40977/R/faMain.R#L193-L208)、[L1404-1406](https://github.com/cran/fungible/blob/d19e1e59705ea956d23a6aa67e263866abd40977/R/faMain.R#L1404-L1406)
- **`faLocalMin`**：對同一解集合（共用同一個 complexity 值）裡的解兩兩計算 RMSD，先用 Hungarian 對齊。`RMSD = sqrt(mean((F1 - F2)^2))`，只用 `$loadings`。有任一方沒收斂就記 999。排序後四捨五入到 `digits`（預設 5）。另外回報 hyperplane count。沒有任何門檻，也不比 Phi — [R/faLocalMin.R L5-12](https://github.com/cran/fungible/blob/d19e1e59705ea956d23a6aa67e263866abd40977/R/faLocalMin.R#L5-L12)、[L229](https://github.com/cran/fungible/blob/d19e1e59705ea956d23a6aa67e263866abd40977/R/faLocalMin.R#L229)、[L247-300](https://github.com/cran/fungible/blob/d19e1e59705ea956d23a6aa67e263866abd40977/R/faLocalMin.R#L247-L300)
- Bootstrap 時會用 `faAlign` 把每個樣本的 loadings 與 Phi 對齊到主解，只是為了算 SE／CI，不是比對 — [R/faMain.R L1192-1214](https://github.com/cran/fungible/blob/d19e1e59705ea956d23a6aa67e263866abd40977/R/faMain.R#L1192-L1214)
- fungible 的 testthat 測試只有 `test_cb`、`test_cfi`、`test_get_wb_mod`、`test_noisemaker`、`test_rmsea`、`test_tkl`、`test_wb`，沒有 faMain 或旋轉 — [tests/testthat/（GitHub 鏡像）](https://github.com/cran/fungible/tree/d19e1e59705ea956d23a6aa67e263866abd40977/tests/testthat)

### Inferences
- `faLocalMin` 的存在說明 fungible 作者（Waller）接受「準則值相同但 loadings 不完全相同」這件事，處理方式是提供 loadings RMSD 給使用者判斷，不設通過門檻，也不看 Phi。
- `min(Table[,2])` 在 GPArotation 2026.6-1 改成非單調 BB 線搜尋後，可能不等於最後回報那一步的準則值（推論：非單調搜尋允許中途上升，未實際驗證 fungible 2.4.8 搭配 GPArotation 2026.8-2 的差異大小）。

### Gaps
- 沒有找到 fungible 比較 Phi 或 factor scores 的任何測試或文件。
- 沒查 fungible 的 NEWS（CRAN 鏡像的 NEWS.md 只有 14 bytes）。

## lavaan `efa()`：旋轉結果怎麼測試、容忍度多少、多個 rstarts 回報哪個解、怎麼判斷相等

### Takeaway
lavaan 的 rstarts 只取準則值最小者（`which.min`），不判斷起點之間是否相等、不回報局部最小值，也不比 loadings 或 Phi。CRAN 鏡像與 GitHub repo 都沒有 tests/ 目錄。唯一的數值自我檢查是開發者用的梯度檢查函式（numDeriv 對解析梯度，`all.equal` tolerance 1e-7），而且找不到任何呼叫它的地方。

### Cited Findings
- rstarts 流程：每個起點產生隨機正交矩陣，跑 GPA 或 pairwise，存下準則值與旋轉矩陣。`best_idx <- which.min(rep_1[1, ])`。`keep_rep` 為 TRUE 時保留全部結果 — [R/lav_matrix_rotate.R L165-219](https://github.com/cran/lavaan/blob/7b78a73ef9d6436405c25c0f4e800e4f0e7566f1/R/lav_matrix_rotate.R#L165-L219)
- 函式層級預設 `rstarts = 100L`、`gpa_tol = 0.00001`、`tol = 1e-07` — [R/lav_matrix_rotate.R L25-33](https://github.com/cran/lavaan/blob/7b78a73ef9d6436405c25c0f4e800e4f0e7566f1/R/lav_matrix_rotate.R#L25-L33)。`efa()` 的 options 預設則是 `rstarts 30L`、`gpa_tol 1e-05`、`tol 1e-08`、`std_ov TRUE`、`geomin_epsilon 0.001` — [R/lav_options_default.R L278-293](https://github.com/cran/lavaan/blob/7b78a73ef9d6436405c25c0f4e800e4f0e7566f1/R/lav_options_default.R#L278-L293)
- 旋轉後依 `order_lv_by` 重排 loadings、Phi 與旋轉矩陣 — [R/lav_matrix_rotate.R L316-329](https://github.com/cran/lavaan/blob/7b78a73ef9d6436405c25c0f4e800e4f0e7566f1/R/lav_matrix_rotate.R#L316-L329)
- 文件：geomin、target 等方法「類似 GPArotation，但為了更好控制而重寫」。`target.strict` 等同 GPArotation 的 target。範例用 `rstarts = 1` — [man/efa.Rd L37-51](https://github.com/cran/lavaan/blob/7b78a73ef9d6436405c25c0f4e800e4f0e7566f1/man/efa.Rd#L37-L51)、[L120-126](https://github.com/cran/lavaan/blob/7b78a73ef9d6436405c25c0f4e800e4f0e7566f1/man/efa.Rd#L120-L126)
- 自己實作而不用 GPArotation 的理由（理解演算法、直接取得梯度、少一個依賴、方便實驗）— [R/lav_matrix_rotate_methods.R L23-30](https://github.com/cran/lavaan/blob/7b78a73ef9d6436405c25c0f4e800e4f0e7566f1/R/lav_matrix_rotate_methods.R#L23-L30)
- 梯度檢查：`ilav_mat_rotate_grad_test` 在隨機 20×5 矩陣上比對 `numDeriv::grad` 與解析梯度，`all.equal(gq1, gq2, tolerance = 1e-07)`。`ilav_mat_rotate_grad_test_all` 印出 OK/FAILED — [R/lav_matrix_rotate_methods.R L617-660](https://github.com/cran/lavaan/blob/7b78a73ef9d6436405c25c0f4e800e4f0e7566f1/R/lav_matrix_rotate_methods.R#L617-L660)。在鏡像下載的全部 231 個 R 檔中 grep，這兩個函式在定義檔之外都沒有被呼叫
- 沒有測試目錄：CRAN 鏡像 tree 沒有 `tests/`。GitHub `yrosseel/lavaan` HEAD 的頂層只有 `.github`、`R`、`data`、`inst`、`man` 等。CI 只跑 `r-lib/actions/check-r-package`（R-CMD-check）— [.github/workflows/CI.yml L9、L47](https://github.com/yrosseel/lavaan/blob/620fa97892392cef37e834c2059b40ea451df185/.github/workflows/CI.yml#L47)

### Inferences
- lavaan 回報的就是準則值最小的單一起點。在病態解上，不同 seed 之間的 Phi 會有跟 insyra 量到相同性質的抖動，lavaan 並沒有去控制或測試這件事。
- lavaan 在這個題目上沒有可以引用的容忍度慣例，只有「梯度對到 1e-7」這個跟解的比對無關的數字。

### Gaps
- 搜尋結果摘要提到 lavaan 教學頁（https://lavaan.ugent.be/tutorial/efa.html）有模仿 Mplus 設定的範例，但我沒有打開驗證，也沒找到 lavaan 對照 Mplus 輸出的自動化測試或公開容忍度。
- 不確定 lavaan 維護者是否在 repo 以外有私下的回歸測試集。

## EFAtools：COMPARE／`efa_compare` 與對照 psych、SPSS、GPArotation 的一致性指標和門檻

### Takeaway
在讀過的五個套件中，EFAtools 1.1.0 的旋轉對照測試最多、也最細。做法是：先要求準則值「不差於」參考（`native <= ref + 1e-6`，當作主要判準），再用**同一個容忍度**比 loadings 與 Phi（一般 1e-4，最適點平坦的 bentlerQ／bifactorQ 兩者一起放寬到 1e-3，對 SPSS 兩者都是 1e-3 或 2e-2）。多峰的 simplimax 則完全放棄逐元素比對，改成只驗準則值的量級。Phi 的比對用排序後的上三角絕對值「指紋」，而且部分區塊透過 waldo 用平均相對差，跟 loadings 的最大絕對差不是同一種度量。factor scores 只驗證「給相同 loadings 與 Phi 時跟 psych 逐位元相同」，不跨起點比。

### Cited Findings
- `efa_compare()`（舊名 `COMPARE()`）用途寫明是比較 loadings 或 communalities。預設 `reorder = "congruence"`（以 Tucker 同餘係數做一對一配對並翻正負號）。輸出平均、中位數、最小、最大絕對差、`are_equal`（所有元素的絕對值一致到第幾位小數）、`diff_corres`、`g`（RMSE）— [R/efa_compare.R L1-5](https://github.com/cran/EFAtools/blob/ce93c76517ad6d5fe2a15a0b1a19bbd6915ef6b3/R/efa_compare.R#L1-L5)、[L17-19](https://github.com/cran/EFAtools/blob/ce93c76517ad6d5fe2a15a0b1a19bbd6915ef6b3/R/efa_compare.R#L17-L19)、[L60-86](https://github.com/cran/EFAtools/blob/ce93c76517ad6d5fe2a15a0b1a19bbd6915ef6b3/R/efa_compare.R#L60-L86)
- 顯示門檻預設：`m_red = .001`、`range_red = .001`（平均／範圍超過 .001 顯示紅色）、`round_red = 3`（一致位數少於 3 位顯示紅色）、`plot_red = .01`、`thresh = .3`、`digits = 4`。這些只控制顯示，不是通過／失敗判準 — [R/efa_compare.R L7-11](https://github.com/cran/EFAtools/blob/ce93c76517ad6d5fe2a15a0b1a19bbd6915ef6b3/R/efa_compare.R#L7-L11)、[L30-43](https://github.com/cran/EFAtools/blob/ce93c76517ad6d5fe2a15a0b1a19bbd6915ef6b3/R/efa_compare.R#L30-L43)、[L109-122](https://github.com/cran/EFAtools/blob/ce93c76517ad6d5fe2a15a0b1a19bbd6915ef6b3/R/efa_compare.R#L109-L122)
- **旋轉回歸測試的總體設計**：loadings 用 `efa_compare` 對齊後取 `max_abs_diff`。斜交的 Phi 用 `phi_fingerprint(Phi) = sort(abs(Phi[upper.tri(Phi)]))`（跟順序、正負號無關）。GPArotation 參考值與本套件引擎共用 seed（平滑準則 100 個起點，simplimax 300 個）。註解說容忍度相對實際觀察到的一致程度刻意放寬，以利跨平台 — [tests/testthat/test-regression-rotations.R L1-20](https://github.com/cran/EFAtools/blob/ce93c76517ad6d5fe2a15a0b1a19bbd6915ef6b3/tests/testthat/test-regression-rotations.R#L1-L20)
- **oblimin／quartimin**：準則非凸所以不要求逐位元相同，主要判準是 `expect_lte(native$value, ref_Q + 1e-6)`。loadings `expect_lt(aligned_max_diff, 1e-4)`。Phi `expect_equal(phi_fingerprint(native$Phi), phi_fingerprint(ref$Phi), tolerance = 1e-4)`。公開 API 對 GPArotation 參考值也是 loadings 1e-4、Phi 1e-4 — [L347-386](https://github.com/cran/EFAtools/blob/ce93c76517ad6d5fe2a15a0b1a19bbd6915ef6b3/tests/testthat/test-regression-rotations.R#L347-L386)。gamma=0.5 同一套 — [L388-404](https://github.com/cran/EFAtools/blob/ce93c76517ad6d5fe2a15a0b1a19bbd6915ef6b3/tests/testthat/test-regression-rotations.R#L388-L404)
- **geominQ**：註解說主要判準是準則值不差於參考。這些資料在 100 個起點下準則識別良好，所以 loadings 與 Phi 也一致，「若在相同 Q 下出現分歧，Q 的比較仍會成立」。loadings 1e-4、Phi 指紋 1e-4。公開 API 對原生引擎是 1e-6 與 1e-6 — [L406-452](https://github.com/cran/EFAtools/blob/ce93c76517ad6d5fe2a15a0b1a19bbd6915ef6b3/tests/testthat/test-regression-rotations.R#L406-L452)
- **bentlerQ（最適點平坦）**：註解說兩個引擎在相同 Q 下只一致到約 1e-4（凸引擎約 1e-6），所以 loadings **與 Phi** 一起放寬到約 1e-3，並說真正的旋轉回歸會讓 loadings 偏移至少 1e-2。Phi 在這裡用最大絕對差 `max(abs(fp1 - fp2)) < 1e-3` — [L454-491](https://github.com/cran/EFAtools/blob/ce93c76517ad6d5fe2a15a0b1a19bbd6915ef6b3/tests/testthat/test-regression-rotations.R#L454-L491)
- **bifactorQ**：同樣理由，loadings 與 Phi 都是 1e-3（k ≥ 3 時）。公開 API 對原生引擎兩者都是 1e-6 — [L493-534](https://github.com/cran/EFAtools/blob/ce93c76517ad6d5fe2a15a0b1a19bbd6915ef6b3/tests/testthat/test-regression-rotations.R#L493-L534)
- **simplimax（多峰）**：因為兩邊起點序列獨立、最小值密集，連 GPArotation 自己在變數重新編號後達到的準則值都會變動約 45%，所以**不做 loadings 比對**，只驗 (a) 回報值確實是回報旋轉的準則值（1e-10）、(b) 比未旋轉的好、(c) `native$value < 1.5 * ref_Q`（1.5 是實測值：300 個起點最差比值 1.27，退化成單一起點時 1.95 到 5.14）、Phi 對角線為 1 — [L214-282](https://github.com/cran/EFAtools/blob/ce93c76517ad6d5fe2a15a0b1a19bbd6915ef6b3/tests/testthat/test-regression-rotations.R#L214-L282)
- 其他：cfT、geominT、bentlerT 的 loadings 1e-4（[L85-92](https://github.com/cran/EFAtools/blob/ce93c76517ad6d5fe2a15a0b1a19bbd6915ef6b3/tests/testthat/test-regression-rotations.R#L85-L92)、[L128-136](https://github.com/cran/EFAtools/blob/ce93c76517ad6d5fe2a15a0b1a19bbd6915ef6b3/tests/testthat/test-regression-rotations.R#L128-L136)、[L162-170](https://github.com/cran/EFAtools/blob/ce93c76517ad6d5fe2a15a0b1a19bbd6915ef6b3/tests/testthat/test-regression-rotations.R#L162-L170)）。Kaiser varimax 對 `stats::varimax` 用 2e-3（[L557](https://github.com/cran/EFAtools/blob/ce93c76517ad6d5fe2a15a0b1a19bbd6915ef6b3/tests/testthat/test-regression-rotations.R#L557)）。promax（normalize=FALSE）對 psych `Promax` 的 loadings 與 Phi 指紋都是 1e-4（[L585-586](https://github.com/cran/EFAtools/blob/ce93c76517ad6d5fe2a15a0b1a19bbd6915ef6b3/tests/testthat/test-regression-rotations.R#L585-L586)）
- **容忍度語意**：EFAtools 使用 testthat 第 3 版 — [DESCRIPTION L72](https://github.com/cran/EFAtools/blob/ce93c76517ad6d5fe2a15a0b1a19bbd6915ef6b3/DESCRIPTION#L72)。`expect_equal(..., tolerance)` 經 waldo 的 `num_equal`：只對不相等的元素取平均絕對差，若平均 |y| 大於 tolerance 再除以它，所以是**平均相對差** — [waldo 0.6.2 R/num_equal.R L28-46](https://github.com/cran/waldo/blob/de67bbeeb81f623eea4851ec6a85635b0aca570a/R/num_equal.R#L28-L46)。因此 oblimin、geominQ、promax 區塊的 Phi 比對是「平均相對差 < 1e-4」，loadings 是「最大絕對差 < 1e-4」
- **對照 SPSS 的測試（`test-regression-spss.R`）**：刻意用「單一係數的最大絕對偏差」，理由是對整個矩陣取平均的相對差會掩蓋單一位置錯誤的 loading（L16-19）。只旋轉 SPSS 自己的 `paf_load` 時，varimax loadings、promax loadings、**promax Phi 都用 2e-2**。17 個參考解中最大偏差 1.2e-2（29 變數、7 因素的 WJIV_3_5 promax），主要來自收斂鬆弛，`precision` 改成 1e-8 會降到 1.1e-3。註解說真正的回歸會動到第二位小數（L38-61）。走完整 SPSS 預設路徑時，loadings 與 Phi 都用 1e-3，最差 3.4e-4（L106-117）— [tests/testthat/test-regression-spss.R](https://github.com/cran/EFAtools/blob/ce93c76517ad6d5fe2a15a0b1a19bbd6915ef6b3/tests/testthat/test-regression-spss.R#L16-L61)、[L106-117](https://github.com/cran/EFAtools/blob/ce93c76517ad6d5fe2a15a0b1a19bbd6915ef6b3/tests/testthat/test-regression-spss.R#L106-L117)
- **Factor scores**：`FACTOR_SCORES` 對 `psych::factor.scores` 的 weights、r.scores、scores 用 tolerance 1e-12（部分 1e-10），但兩邊輸入的是**同一組** L 與 Phi，驗的是計分邏輯，不是旋轉的穩定性 — [tests/testthat/test-efa_scores.R L485-538](https://github.com/cran/EFAtools/blob/ce93c76517ad6d5fe2a15a0b1a19bbd6915ef6b3/tests/testthat/test-efa_scores.R#L485-L538)
- 萃取對照 psych：PAF／ULS 的 communalities 與重製相關用 1e-4。其他初始 communality 路徑（mac、unity）放寬到 5e-3 — [tests/testthat/test-regression-estimators.R L74-76](https://github.com/cran/EFAtools/blob/ce93c76517ad6d5fe2a15a0b1a19bbd6915ef6b3/tests/testthat/test-regression-estimators.R#L74-L76)、[L251-254](https://github.com/cran/EFAtools/blob/ce93c76517ad6d5fe2a15a0b1a19bbd6915ef6b3/tests/testthat/test-regression-estimators.R#L251-L254)
- NEWS 0.8.0：所有準則式旋轉改用內建 C++ 梯度投影引擎取代 GPArotation，文件稱結果「numerically equivalent」。`randomStarts` 預設從 10 提高到 100。另外修了 Phi、structure、解釋變異量沒有跟著 loadings 一起翻轉正負號與重排的錯誤：以前某個因素被翻成正向時 Phi 沒調整，`order_type = "ss_factors"` 時 Phi 完全沒重排 — [NEWS.md L247-253](https://github.com/cran/EFAtools/blob/ce93c76517ad6d5fe2a15a0b1a19bbd6915ef6b3/NEWS.md#L247-L253)、[L337-348](https://github.com/cran/EFAtools/blob/ce93c76517ad6d5fe2a15a0b1a19bbd6915ef6b3/NEWS.md#L337-L348)。同版也把 `COMPARE` 的同餘配對從逐一貪婪改成一對一最佳指派 — [L365-370](https://github.com/cran/EFAtools/blob/ce93c76517ad6d5fe2a15a0b1a19bbd6915ef6b3/NEWS.md#L365-L370)。0.7.1 應 Coen Bernaards 建議加入 `randomStarts` — [L377](https://github.com/cran/EFAtools/blob/ce93c76517ad6d5fe2a15a0b1a19bbd6915ef6b3/NEWS.md#L377)
- **論文（Grieder & Steiner, BRM，PMC8863761）**：EFAtools 重現 psych 的未旋轉 PAF loadings、varimax loadings、promax pattern「至少到第 14 位小數」。重現 SPSS 的未旋轉 PAF 到至少第 9 位，varimax 與 promax 到至少第 4 位。psych 與 SPSS 在真實資料上的差異用絕對差與因素同餘係數描述：未旋轉的每解平均 M_mean 0.002、M_max 0.01。varimax 0.003、0.02。promax 0.03、0.12。同餘係數 .85 到 .94 算尚可、> .95 算良好（引 Lorenzo-Seva & ten Berge 2006）。全文檢索沒有找到 Phi 的比較指標（「intercorrelation」只出現在模擬設計與結果討論中）— [Grieder & Steiner, Behavior Research Methods (PMC 全文)](https://pmc.ncbi.nlm.nih.gov/articles/PMC8863761/)
- JOSS 論文（Steiner & Grieder 2020）只說能重現 psych 與 SPSS 的實作，沒有給數值門檻 — [paper/paper.md L40、L44](https://github.com/mdsteiner/EFAtools/blob/HEAD/paper/paper.md)。[JOSS 10.21105/joss.02521](https://joss.theoj.org/papers/10.21105/joss.02521)

### Inferences
- EFAtools 的慣例很清楚：以準則值當主要判準，再依「同一準則值下實測的抖動」決定逐元素容忍度，而 loadings 與 Phi 一起調整。它沒有「loadings 緊、Phi 寬」的不對稱設定。最接近不對稱的是度量方式不同（Phi 用平均相對差、loadings 用最大絕對差），而且沒有說明理由。
- EFAtools 在 0.8.0 真的出過「Phi／structure 沒跟 loadings 一起翻轉或重排」的錯誤。這類錯誤在 Phi 上造成的差異大約是 2|φ_ij|，所以只要容忍度比最小的 |φ_ij| 的兩倍小就抓得到。insyra 若把 Phi 放寬到 5e-3，只會漏掉 |φ_ij| < 2.5e-3 的翻號錯誤（推論，未實測）。
- 「真正的回歸會動到第二位小數（≥ 1e-2）」是 EFAtools 對 loadings 的經驗判斷，被用來說明 1e-3 或 2e-2 的容忍度仍然安全。拿同樣邏輯套用到 Phi，5e-3 仍低於這個量級。
- simplimax 的處理顯示：當準則下的解本身無法唯一識別，EFAtools 的選擇是不再逐元素比對，而不是把容忍度一路放大。

### Gaps
- EFAtools 沒有說明為什麼 Phi 有時用 waldo 的平均相對差、有時用最大絕對差。看起來是各區塊寫法不一致，找不到刻意這樣設計的理由（推論）。
- 沒有找到 EFAtools 跨隨機起點比較 structure matrix 或 factor scores 的測試。
- BRM 論文的補充表 S1 沒有打開，無法確認裡面是否列了 Phi 的差異。

## semTools、EFA.dimensions 等其他套件

### Takeaway
semTools 已經移除自己的旋轉函式，改叫使用者用 `lavaan::efa()`，所以沒有可引用的旋轉比對慣例。EFA.dimensions 沒有查。

### Cited Findings
- semTools 0.5-9 的 NEWS：`efaUnrotate()`、`orthRotate()`、`oblqRotate()`、`funRotate()` 自 2022 年起棄用、現已移除，建議改用 `lavaan::efa()` 的 `rotation=` — [semTools NEWS.md L37-43、L68](https://github.com/cran/semTools/blob/a74ff02d34061cbf6e108495b31088542a6ea5dc/NEWS.md)
- semTools 0.5-9 鏡像樹中跟 EFA 相關的只剩 `man/efa.ekc.Rd`，沒有 tests/ 目錄 — [cran/semTools tree](https://github.com/cran/semTools/tree/a74ff02d34061cbf6e108495b31088542a6ea5dc)

### Inferences
- 在 R 生態系裡，旋轉結果的比對慣例實際上集中在 GPArotation 與 EFAtools。psych、lavaan、fungible、semTools 都沒有可當標準的 Phi 容忍度。

### Gaps
- 沒查 EFA.dimensions、nFactors、MBESS 等套件。Mplus、SPSS、SAS 等商業軟體的內部驗證做法也不在範圍內。

## 總結：有沒有套件把 Phi 設得比 loadings 緊或寬、甚至根本不測？可接受的差異是多少？

### Takeaway
所有會測 Phi 的 R 套件（GPArotation、EFAtools），在同一個測試區塊裡都讓 Phi 跟 loadings 用**相同**數值的容忍度。放寬時兩者一起放寬。沒有找到「loadings 維持緊、Phi 單獨放寬」的先例。psych、fungible、lavaan 根本不測 Phi，也沒有任何套件跨隨機起點測 structure 或 factor scores。insyra 的提案（準則值相同就把 Phi／structure／scores 放到 5e-3，loadings 已一致時仍守 2e-5）在數學上說得通，但不算業界慣例，屬於需要自己寫清楚理由的專案決定。

### Cited Findings
- 同一區塊內 Phi 與 loadings 容忍度相同的例子：GPArotation 已存參考值 1e-5（[Harman.R L9、L144](https://github.com/cran/GPArotation/blob/3d15692ba675723fd91dfb9fb01273a86826a7b9/tests/Harman.R#L144)）。同一最小值的兩個隨機起點 1e-5（[print-GPArotation.R L55-74](https://github.com/cran/GPArotation/blob/3d15692ba675723fd91dfb9fb01273a86826a7b9/tests/print-GPArotation.R#L55-L74)）。factanal 對 GPArotation 1e-4（[L99-111](https://github.com/cran/GPArotation/blob/3d15692ba675723fd91dfb9fb01273a86826a7b9/tests/print-GPArotation.R#L99-L111)）。路徑不同、最小值相同 1e-3（[legacyVsBB.R L8、L18-43](https://github.com/cran/GPArotation/blob/3d15692ba675723fd91dfb9fb01273a86826a7b9/tests/legacyVsBB.R#L18-L43)）。EFAtools 跨引擎 1e-4、平坦最適點 1e-3、同引擎 1e-6（[test-regression-rotations.R L374-376、L479-482、L488-489](https://github.com/cran/EFAtools/blob/ce93c76517ad6d5fe2a15a0b1a19bbd6915ef6b3/tests/testthat/test-regression-rotations.R#L374-L376)）。對 SPSS 2e-2 或 1e-3（[test-regression-spss.R L58-61、L114-117](https://github.com/cran/EFAtools/blob/ce93c76517ad6d5fe2a15a0b1a19bbd6915ef6b3/tests/testthat/test-regression-spss.R#L58-L61)）
- 唯一「structure 比 pattern 寬」的例子是 GPArotation 對照 SPSS 三位小數的輸出：pattern 3e-3、structure 4e-3 — [KaiserNormalization.R L47、L66](https://github.com/cran/GPArotation/blob/3d15692ba675723fd91dfb9fb01273a86826a7b9/tests/KaiserNormalization.R#L47-L66)
- 判斷「同一最小值」的做法：GPArotation `round(Q/eps)*eps`（[GPFRS.R L93](https://github.com/cran/GPArotation/blob/3d15692ba675723fd91dfb9fb01273a86826a7b9/R/GPFRS.R#L93)）。fungible `round(Q, nchar(epsilon))`，實際是 5 位小數（[faMain.R L1032-1033](https://github.com/cran/fungible/blob/d19e1e59705ea956d23a6aa67e263866abd40977/R/faMain.R#L1032-L1033)）。EFAtools 用 `Q_native <= Q_ref + 1e-6`（[L374](https://github.com/cran/EFAtools/blob/ce93c76517ad6d5fe2a15a0b1a19bbd6915ef6b3/tests/testthat/test-regression-rotations.R#L374)）。GPArotation `f.bb <= f.leg + 1e-4`（[legacyVsBB_extra.R L23](https://github.com/cran/GPArotation/blob/3d15692ba675723fd91dfb9fb01273a86826a7b9/tests/legacyVsBB_extra.R#L23)）。lavaan 只取 `which.min`（[lav_matrix_rotate.R L213](https://github.com/cran/lavaan/blob/7b78a73ef9d6436405c25c0f4e800e4f0e7566f1/R/lav_matrix_rotate.R#L213)）。psych 看 hyperplane count，不看準則值（[faRotate.r L164-168](https://github.com/cran/psych/blob/2b17d1e78b536c0e4cc807d8a83f546e0a0da85d/R/faRotate.r#L164-L168)）
- 找得到的 Phi 容忍度中，最寬的是 EFAtools 旋轉 SPSS 自己 loadings 時的 2e-2，理由是 SPSS 收斂鬆弛。跨實作、同一最小值的 Phi 容忍度最寬是 1e-3（GPArotation legacyVsBB、EFAtools bentlerQ／bifactorQ）（出處同上）
- GPArotation 自己的資料上，同一最小值的兩個解之間 Phi 差異 ≤ loadings 差異的約 1.5 倍（本機重現，見第一節）。insyra 在近似一階的 ML/oblimin 解上量到約 15 倍（題目給定的事實）

### Inferences
- **跟慣例的差距**：(1) 沒有套件讓 Phi 比 loadings 寬。(2) 5e-3 比任何「跨實作、同一最小值」的 Phi 容忍度（最寬 1e-3）還寬 5 倍，只比對照四捨五入過的 SPSS 輸出時的 2e-2 窄。(3) 沒有套件跨起點比 structure 或 scores。
- **支持提案的論點**：Phi = TᵀT、structure = L·Phi。解接近一階時 T 病態，loadings 的小差異會在 Phi 上放大。GPArotation 測試資料都是良態，所以沒看到這個現象。EFAtools「依實測抖動、留安全餘裕來訂容忍度」的原則（例如 1.2e-2 實測對 2e-2 門檻、1.27 對 1.5）可以直接用來為 insyra 的 5e-3 辯護：insyra 實測最大 3.1e-3，5e-3 大約是 1.6 倍餘裕，而且仍小於 EFAtools 認定的回歸量級 1e-2。
- **比較符合慣例的替代方案**（供維護者評估）：
  1. 準則值相同時，loadings 與 Phi 用同一個由實測決定的容忍度（例如都用 5e-3）。這最接近 GPArotation legacyVsBB 與 EFAtools bentlerQ 的做法，缺點是良態解的 loadings 會失去 2e-5 的保護。
  2. 保留不對稱，但把 Phi 的容忍度跟條件數或 loadings 實際差異掛勾，不寫死 5e-3。良態解（如 PCA/geomin：loadings 9e-6、Phi 9.1e-5，約 10 倍）自然得到比較緊的門檻。
  3. 仿 EFAtools 把「準則值不差於參考」當主要判準，逐元素比對只當次要檢查。另外加一個不依賴隨機起點的一致性檢查，例如 `Phi == TᵀT`、`structure == L·Phi`，在 1e-10 等級驗算（GPArotation Thurstone.R L176-177 就有這種檢查）。這樣就算 Phi 跨實作的容忍度放寬，翻號、重排、公式錯誤仍會被緊的內部一致性測試抓到。
- **放寬 Phi 的風險**：EFAtools 0.8.0 修過的「Phi 沒跟 loadings 一起翻號或重排」這類錯誤，差異約 2|φ|，在 5e-3 下仍抓得到，除非 |φ| < 2.5e-3。不過，如果 structure 或 scores 的公式只在少數位置差一點點（小於 5e-3），這個容忍度就看不出來。第 3 點的內部一致性檢查可以抓到這種錯誤。
- insyra 目前用「相對 1e-4」判斷同一最小值，比 GPArotation 的 `atMinimum`（絕對 1e-5 格點）與 EFAtools 的 `+1e-6` 更寬或更窄要看 Q 的大小。可以考慮在文件中寫明它跟這兩個套件的對應關係。

### Gaps
- 沒有找到任何 R 套件公開討論 Phi 對起點抖動的敏感度，也沒找到對病態（近似一階）解的比對慣例。這部分只能靠 insyra 自己的實測與推導。
- 沒有檢視 R 以外的參考實作（Python 的 factor_analyzer、Mplus、SPSS 內部測試）。
