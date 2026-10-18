# 斜交因素旋轉：Φ、結構矩陣、因素分數相對於負荷量的決定精度，以及兩組旋轉解的等價判定

符號約定（沿用 GPArotation / Jennrich 2002 的斜交參數化）：A 為未旋轉負荷量，T 為各行單位長度的 k×k 旋轉矩陣，型態負荷量 L = A (T′)⁻¹，因素相關 Φ = T′T，結構矩陣 S = LΦ。以下「來源說」與「我的推論」分開列，推論一律放在 Inferences。

## Q1 旋轉不定性與識別：L、Φ、S 怎麼連在一起？準則平坦或病態時，誰比較不準？

### Takeaway
L、Φ、S 都由同一個 T 決定，但準則只透過 L 感受 T。於是 T 在「L 幾乎看不見」的方向上移動時，準則值和 L 幾乎不變，Φ 和 S 卻照常移動。這是代數結構本身造成的，不是 bug。不過我找不到任何文獻明講「Φ 的數值精度天生比負荷量差」，這一點是推導加上已量測數據支持，不是文獻結論。

### Cited Findings
- GPArotation 的輸出定義 `Phi` 為 `t(Th) %*% Th`，且各 GPA 函式在單位長度行的限制下對 T 做投影梯度下降。使用指南以程式驗證 L·T′ = A、L = A(T′)⁻¹、Φ = T′T 在機器精度內成立，並註明這些定義出自 Bernaards & Jennrich (2005) 第 678、695 頁。— GPArotation 2026.8.2 本機安裝的 `GPA.Rd`、`GPFoblq` 原始碼與 `GPA1guide.Rnw` 約第 120–149 行（[CRAN GPArotation](https://cran.r-project.org/package=GPArotation)）；斜交方法原始論文為 Jennrich (2002), *Psychometrika* 67:7–19, [doi:10.1007/BF02294706](https://doi.org/10.1007/BF02294706)（全文未取得，只核對書目）
- GPArotation 使用指南說明標準化變項下結構矩陣就是 Corr(X, f) = ΛΦ，而且因素斜交時，結構矩陣會讓變項與因素的表面關係看起來更強。— GPArotation 2026.8.2 vignette `GPA1guide.Rnw`（本機檔案，約第 196–212 行；[CRAN GPArotation](https://cran.r-project.org/package=GPArotation)）
- Browne (2001) 回顧斜交旋轉史時指出：以電腦疊代旋轉參照結構時，常出現「factor collapse」，也就是因素間相關在疊代中一路逼近 1。這是 Φ 在某些斜交準則下本身就不穩定的早期文獻證據（是準則設計問題，不是數值容忍度問題）。— Browne (2001), *Multivariate Behavioral Research* 36(1):111–150, [doi:10.1207/S15327906MBR3601_05](https://doi.org/10.1207/S15327906MBR3601_05)，[開放 PDF](http://www.statpower.net/Content/312/Handout/Browne2001.pdf)
- Schmitt & Sass (2011) 的結論是：因素間相關和型態負荷量都會隨旋轉準則與型態複雜度「大幅」變動。這是實質層面的敏感度（換準則），不是同一準則下的數值敏感度。— Schmitt & Sass (2011), *Educational and Psychological Measurement* 71(1):95–113, [doi:10.1177/0013164410387348](https://doi.org/10.1177/0013164410387348)（只讀到摘要層級的搜尋結果，全文未取得）
- Nguyen & Waller 的補充材料給了一個反例：兩個 oblique geomin 解的準則值都是 .82126，但負荷量對齊後 RMSD = .10816，兩個 Φ 矩陣也明顯不同（例如 f4–f5 為 .48 對 .53，f1–f2 為 .10 對 .04，f2–f5 為 .20 對 .06）。— Nguyen & Waller (2023), *Psychological Methods* 28(5):1122–1141, [doi:10.1037/met0000467](https://doi.org/10.1037/met0000467)；[補充材料 PDF](https://supp.apa.org/psycarticles/supplemental/met0000467/met0000467_sm2.pdf) Table S3、S4（主文付費牆，未取得）
- Hattori 等人指出，不同 ε 或不同局部解得到的旋轉後負荷量和因素相關在數學上等價、對資料的適配完全一樣；而且母體的全域解可能對應到樣本的局部解。— Hattori, Zhang & Preacher (2017), *Multivariate Behavioral Research* 52(6):720–731, [doi:10.1080/00273171.2017.1361312](https://doi.org/10.1080/00273171.2017.1361312)，[開放 PDF](https://www.quantpsy.org/pubs/hattori_zhang_preacher_2017.pdf)

### Inferences
- 一階擾動推導（我自己推的，建議請統計背景的人再核一次）：令 E = T⁻¹ dT，則
  - dL = −L E′
  - dΦ = ΦE + E′Φ
  - dS = S E（因為 S = LΦ = A(T′)⁻¹T′T = A T，S 對 T 是線性的）
  準則 Q 只看 L，所以 E 裡被 L 近乎消掉的成分（L 的小奇異值方向）只會讓 L 和 Q 產生很小的變化，Φ 和 S 卻不受 L 的奇異值縮小，會照 O(‖E‖) 移動。這解釋了為什麼 Φ 和 S 的差距幾乎一樣大，而且都比 L 大。
- 已量測數據和這個推導一致：近秩一的兩因素 ML 解（oblimin），Φ／L 最大差距比約 3.1e-3 / 2.1e-4 ≈ 15，S 和 Φ 幾乎相同。
- 條件良好的 PCA 解（geomin）比例也約 10（9.1e-5 / 9.0e-6），所以放大現象不只出在近秩虧損的情況。放大倍數取決於 L、Φ 的條件數，以及殘留誤差落在哪個方向，不是固定常數。
- 單位行長限制 diag(T′dT) = 0 會排除部分方向（例如 k = 2 時，完全落在 L 零空間的方向通常不可行），所以放大倍數有上限，但上限因資料而異。要拿推導算出精確界限，得針對每組測試資料算雅可比矩陣（Jacobian），不能套一個通用倍數。
- 最重要的推論：量測顯示，就算負荷量差距在 2e-5 以內（PCA／geomin 為 9.0e-6），同一最小值上的 Φ 仍可差到 9.1e-5 > 2e-5。所以目前規則裡「負荷量已在 2e-5 內一致，Φ／S／分數就用 2e-5 比」這一支，在數學上本來就站不住。

### Gaps
- 我找不到任何同儕審查文獻明確說「同一準則最小值附近，Φ 的數值決定程度比負荷量差」。上面的機制是推導，不是引用。
- Jennrich (2001, 2002) 與 Bernaards & Jennrich (2005) 全文都沒取得，無法確認他們是否討論過 Φ 的數值敏感度。

## Q2 抽樣標準誤：旋轉後負荷量和因素相關，誰的精度比較差？這和數值決定程度有關嗎？

### Takeaway
文獻沒有支持「因素相關的抽樣標準誤普遍大於負荷量」。Zhang & Preacher (2015) 的實例裡兩者同一量級。真正的共同點是：旋轉準則在最佳點附近越平坦（旋轉識別越弱），抽樣標準誤就會膨脹。這和數值上「T 在谷底方向決定不準」是同一件事的兩面，但這個連結是推論。

### Cited Findings
- 旋轉後負荷量與因素相關的漸近共變異，可由加上旋轉限制導數的擴增資訊矩陣求逆得到（Jennrich, 1974）。斜交旋轉的限制條件來自 Jennrich (1973b) 第 28 式。另一條路是 delta method（Cudeck & O'Dell, 1994）。— Zhang & Preacher (2015), *Journal of Educational and Behavioral Statistics* 40(6):579–603, [doi:10.3102/1076998615606098](https://doi.org/10.3102/1076998615606098)，[開放 PDF](https://quantpsy.org/pubs/zhang_preacher_2015.pdf)，第 587–588 頁
- 同一份資料用 oblique CF-varimax 和 CF-quartimax，點估計很接近，標準誤卻差很多。在我讀到的 Table 2 各列中，負荷量 SE 約 .03–.28（CF-quartimax 的部分負荷量可到 .25–.28），因素相關 SE 約 .03–.15。看不出因素相關系統性地比較不精確。— 同上，Table 1–2
- CF 族準則的 κ 很小時，漸近標準誤會嚴重膨脹：Table 4 括號內的 ASE（= √n · s_n）最高到 70.1、23.47；κ 很大時 CF 旋轉則表現不穩。— 同上，Concluding Remarks 與 Table 4
- 因素結構是獨立叢集型態時，各 CF 準則的漸近標準誤相近；結構越複雜，差異越大。— 同上，摘要（[ERIC 紀錄](https://eric.ed.gov/?id=EJ1084509)）
- Cudeck & O'Dell (1994) 的主題就是非限制性因素分析中負荷量與「因素相關」的顯著性檢定，表示因素相關也有標準誤理論與實務。— Cudeck & O'Dell (1994), *Psychological Bulletin* 115(3):475–487, [doi:10.1037/0033-2909.115.3.475](https://doi.org/10.1037/0033-2909.115.3.475)（只經 Crossref 核對書目，全文未取得）
- Jennrich (1973) 推導了斜交旋轉負荷量的標準誤。— *Psychometrika* 38:593–604, [doi:10.1007/BF02291497](https://doi.org/10.1007/BF02291497)（只核對書目）
- Browne (2001) 提到 CEFA 已提供旋轉後負荷量的標準誤等統計資訊。— Browne (2001)，[PDF](http://www.statpower.net/Content/312/Handout/Browne2001.pdf) p. 113

### Inferences
- 擴增資訊矩陣裡的旋轉限制導數，描述的正是準則在 T 附近的彎曲程度。準則在某方向平坦時，該方向的抽樣變異大，數值上也最難收斂到同一點。所以「抽樣上決定得差」和「數值上決定得差」出自同一個幾何結構。這是推論，Zhang & Preacher 沒有這樣表述。
- 但抽樣標準誤（實例約 .03–.15）和軟體比對容忍度（2e-5 到 5e-3）差了 1–3 個數量級。5e-3 的 Φ 差異在實質解釋上幾乎不可能造成影響，但這只能說「放寬不會誤導使用者」，不能拿來證明「放寬後仍抓得到移植錯誤」。軟體等價要靠數值分析來論證，不能靠抽樣標準誤。
- 文獻不支持用「Φ 抽樣上比較不準」當放寬理由，因為在找到的例子裡這個前提不成立。能支持放寬的是 Q1 的代數推導加上實測的起點間差距。

### Gaps
- 沒找到同時比較大量模型下 Φ 與 L 抽樣標準誤相對大小的系統性研究。
- Jennrich (1973, 1974)、Cudeck & O'Dell (1994) 全文未取得，無法確認是否有 Φ 與 L 精度的一般性陳述。

## Q3 投影梯度收斂準則 eps = 1e-5，對負荷量與 Φ 的精度代表什麼？

### Takeaway
GPArotation 的 eps 管的是投影梯度的 Frobenius 範數，不是參數誤差。參數誤差大約是 eps 除以準則在谷底方向的曲率，準則值誤差則是 eps² 除以曲率。所以準則值一致到 1e-9，完全不代表 T、Φ、L 一致到同等級。我沒找到心理計量文獻明寫這個關係，這是標準最佳化理論加上已量測數據的推論。

### Cited Findings
- `GPFoblq` 的收斂判斷：Gp = G − T·diag(colSums(T∗G))，s = sqrt(sum(Gp²))，s < eps 就停；預設 eps = 1e-5、maxit = 2000、algorithm = "bb"。說明文件寫法是 "convergence is assumed when the norm of the gradient is smaller than eps"。— GPArotation 2026.8.2 本機原始碼與 `GPA.Rd`（[CRAN refman](https://cran.r-project.org/web/packages/GPArotation/refman/GPArotation.html)）
- 同一份文件說明，"bb" 預設 `fwindow = 10` 的非單調 Armijo 條件允許準則值暫時上升，用來脫離平坦區與淺的局部極小。— 同上，`GPA.Rd`
- Nguyen & Waller (2024) 在 760 萬次以上的旋轉中，對 `GPFoblq()` 一律使用 eps = .00001、maxit = 15000；GPFoblq 在 7,680,000 次旋轉中只有兩次不收斂。— Nguyen & Waller (2024), *Educational and Psychological Measurement* 84:1045–1075, [doi:10.1177/00131644231223722](https://doi.org/10.1177/00131644231223722)，[PMC11523187](https://pmc.ncbi.nlm.nih.gov/articles/PMC11523187/)
- Hattori 等人提醒，有些「局部解」其實只是最小化演算法太早停下、沒有真正收斂。— Hattori et al. (2017)，[PDF](https://www.quantpsy.org/pubs/hattori_zhang_preacher_2017.pdf) Discussion
- Browne (2001) 也說過早終止的 oblique CF-varimax 解不能算是局部極小。— Browne (2001)，[PDF](http://www.statpower.net/Content/312/Handout/Browne2001.pdf) p. 136–137
- Nguyen & Waller 的補充材料有一個 varimax 母體例子（eps 用 fungible 預設 1e-5、500 個起點），印出的其中一組解中，var1 那列是 (0.1264927, 0.3794728)，var5 那列是 (0.3794739, 0.1264895)。— [補充材料 PDF](https://supp.apa.org/psycarticles/supplemental/met0000467/met0000467_sm2.pdf) p. 21

### Inferences
- 標準二階泰勒展開（推論，未另外調閱最佳化教科書）：在非退化極小點附近，梯度 g ≈ Hδ，準則差 ΔQ ≈ ½ δ′Hδ。所以 ‖δ‖ ≲ ‖g‖/λ_min(H) ≤ eps/λ_min，ΔQ ≲ eps²/(2λ_min)。參數誤差和梯度同階，準則誤差則和梯度平方同階。
- 拿已量測數據做量級檢查（假設性估算，只求量級）：近秩一 oblimin 案例 Φ 差距 3.1e-3，換算 ‖E‖ 約 1.5e-3。配合準則差距 ≤ 2.7e-9，曲率 λ ≈ 2·2.7e-9/(1.5e-3)² ≈ 2.4e-3，推得 eps/λ ≈ 4e-3。這和實測 Φ 差距 3.1e-3 同一量級，所以「起點間 Φ 差 3e-3」完全可以用「eps = 1e-5 遇上極平坦的谷」解釋，不需要假設有演算法錯誤。
- 由此看，Φ／S 的合理容忍度應該和 eps/λ_min 成比例，而 λ_min 因資料而異：近秩一或弱因素時很小。固定 5e-3 在這個案例只留 1.6 倍餘裕（5e-3 / 3.1e-3），換一組條件更差的資料就可能不夠；反過來，對條件良好的 PCA／geomin 案例（實測約 9e-5）又寬了約 50 倍。
- varimax 例子中，母體結構是鏡像對稱，精確解理論上應該有 var1 = var5 反轉。印出值差 1.1e-6 與 3.2e-6，看起來是 eps = 1e-5 下條件良好問題的殘餘收斂誤差量級。這是弱證據：對稱性假設是我從表格推的，原文沒有這樣說。

### Gaps
- Bernaards & Jennrich (2005, *EPM* 65:676–696, [doi:10.1177/0013164404272507](https://doi.org/10.1177/0013164404272507)) 與 Jennrich (2001, 2002) 全文都沒取得（SAGE／Springer 付費牆）。無法確認他們是否討論 eps 和參數精度的關係，或選 1e-5 的理由。
- 沒有取得最佳化教科書（例如 Nocedal & Wright）的原文來引用「梯度容忍度 → 參數誤差」的界限，這裡以推導呈現。

## Q4 多起點與局部極小：文獻怎麼判定不同起點得到的是「同一個解」？

### Takeaway
主流做法幾乎都先看準則值（四捨五入到 eps = 1e-5 分組），部分研究再補負荷量對齊後的 RMSD 或一致性係數。我找到的來源中，沒有任何一個拿 Φ 當判定依據。已量測的 40 個起點準則差距最多 2.7e-9，遠小於 1e-5 的分組寬度，所以依 GPArotation 自己的定義，它們全都是「同一個最小值」。

### Cited Findings
- GPArotation 的隨機起點引擎把準則值四捨五入到 eps（`Q_round <- round(Qvalues/eps)*eps`），用它計算 `atMinimum`（落在最小分組的起點數）和 `localMins`（不同分組數），完全不比較負荷量或 Φ。— GPArotation 2026.8.2 本機原始碼 `GPArotation:::.GPA_RS_engine`（[CRAN](https://cran.r-project.org/package=GPArotation)）
- GPArotation 的局部極小 vignette（作者 Bernaards）示範的 `GPFallMinima` 同樣用四捨五入到 eps 的準則值分組，預設 `minimumInclusion = 2`，理由是只出現一次的極小值很可能是數值假象；建議做完整分析時用 500 個以上起點。— GPArotation 2026.8.2 vignette `GPA2local.Rnw`（本機檔案）
- 同一份 vignette 的 CCAI 例子：oblimin 在 200 個隨機起點下全部收斂到同一解，simplimax 則找到 16 個不同局部極小。— 同上
- fungible 的 `faMain` 把旋轉複雜度值四捨五入到 `rotateControl` 裡 `epsilon`（預設 1e-5）的位數，再分成「局部解集合」。— [fungible faMain 說明](https://rdrr.io/cran/fungible/man/faMain.html)（經擷取工具轉述，原句未逐字核對）
- Nguyen & Waller (2023) 補充材料的輸出直接寫明同一個 1e-05 同時用於旋轉收斂和解集合分群。他們也提醒：準則值相同的集合裡可能混有實質不同的型態，所以對齊到母體後計算集合內兩兩 RMSD，只要大於 0 就算不同解。他們認為這種情況機率很小，因此以準則值分群算出的局部解數是很緊的下界。— [補充材料 PDF](https://supp.apa.org/psycarticles/supplemental/met0000467/met0000467_sm2.pdf) p. 32、Table S5
- Nguyen & Waller (2024) 沿用這套做法：每組資料 200 個隨機起點、以相等複雜度值分組、再用 `faAlign()` 對齊並算 RMSD。38,400 個組合中只找到 20 例同集合內 RMSD 偏大，也就是 "indicating qualitatively distinct rotations"。例子中的準則值報到小數第五位（.66145 對 .66876）。— [PMC11523187](https://pmc.ncbi.nlm.nih.gov/articles/PMC11523187/)
- Hattori 等人：100 個隨機起點（Mplus 預設 30，Browne 2001 用 100）；比較解時先處理行交換與行反號，再用一致性係數。他們在模擬中以「所有因素一致性係數 > .98」為嚴格標準，另外也用 .92 當門檻，引用 MacCallum et al. (1999) 的分級：.98–1.00 極佳、.92–.98 良好。— Hattori et al. (2017)，[PDF](https://www.quantpsy.org/pubs/hattori_zhang_preacher_2017.pdf) p. 722–724、726
- Hattori 等人建議實務上要檢查推定的全域解、準則值接近的幾個解，以及出現最頻繁的幾個解。— 同上，Discussion
- Browne (2001)：24 項心理測驗資料用 20 個隨機起點，Box 資料用 100 個；描述人工構造的解和旋轉結果 "agrees with it to three decimal places"，並稱該解在隨機重啟下穩定。— Browne (2001)，[PDF](http://www.statpower.net/Content/312/Handout/Browne2001.pdf) 約 p. 131–140（頁碼依文字轉檔推估）
- Weide & Beauducel（arXiv 預印本，2018）判定「結果相等」的條件是：平均一致性係數差 ≤ .001 且 Varimax 準則差 ≤ .0001。— [arXiv:1809.04885](https://arxiv.org/abs/1809.04885)（預印本，正式發表狀態未核實）
- psych 2.6.5 的 `faRotations` 從多個起點中挑 hyperplane 比例（|負荷量| < `hyper` = .15 的比例）最大者；同分時比 complexity，再同分取第一個，並不是挑準則值最小者。— psych 2.6.5 本機原始碼（deparse 第 195–215 行；[CRAN psych](https://cran.r-project.org/package=psych)）

### Inferences
- 文獻的「同一解」尺度大約在 1e-3（三位小數、一致性係數差 .001）到 1e-5（準則值分組）之間，而且判定對象是準則值和負荷量。Φ 從來不是判定依據，最多像 Nguyen & Waller 的 Table S4 那樣並列描述。
- 依 GPArotation 本身的定義（準則值四捨五入到 1e-5），實測準則差 ≤ 2.7e-9 的 40 個起點全部屬於同一個最小值。所以把「準則值是同一最小值」當作判定前提，和上游套件的定義一致。
- 但比較規則上有兩個細節不同。第一，parity 測試用「相對 1e-4」，GPArotation 用「絕對值四捨五入到 eps」：準則值約 0.5 時，相對 1e-4 等於絕對 5e-5，比 1e-5 寬。第二，四捨五入分組有邊界效應，例如 0.123455 和 0.123445 只差 1e-5，卻可能被分進不同組。兩者都不影響目前 2.7e-9 的情況，只是若要宣稱「沿用 GPArotation 定義」，要注意寫法不完全相同。
- psych 用 hyperplane 比例挑解，而 .15 門檻對 1e-4 級差異不敏感，同一最小值上的起點幾乎必然同分，最後落到 complexity 或「取第一個」。再加上 20 個起點沒有固定種子，psych 的參考值本身就是谷底上一個隨機位置。因此「Go 對 psych 逐元素 2e-5」在 Φ／S／分數上測的其實有一部分是 psych 的亂數，不是移植正確性。這是支持放寬的最強理由。
- 旁註（信心低，未執行驗證）：deparse 出來的 tie-break 程式碼中，`best <- which(stats[best, "complexity"] == min(...))` 回傳的是子集合內的位置，不是原索引；緊接的 fit tie-break 看起來也沒有指派回 `best`。如果屬實，psych 的挑選更接近任意。這需要實際讀完整原始碼確認，不應直接引用。

### Gaps
- Nguyen & Waller (2023) 主文付費牆未取得，無法確認主文是否另有數值容忍度（例如 RMSD 門檻）的規定。
- Hattori et al. 線上附錄的 R 函式如何判定「不同解」（準則值取幾位）沒有細看。
- 沒找到任何來源為 Φ 或結構矩陣另設等價容忍度。

## Q5 比較因素解的既有指標：一致性係數門檻、Procrustes／RMSD，有沒有人建議另外比 Φ？

### Takeaway
既有指標（Tucker 一致性係數、對齊後 RMSD）都是以負荷量行為單位的實質相似度指標，尺度是 1e-2 到 1e-3，對 1e-5 到 1e-3 的軟體比對幾乎沒有鑑別力。沒有找到任何來源建議分開比較 Φ，或給 Φ 不同的容忍度。

### Cited Findings
- Lorenzo-Seva & ten Berge (2006)：一致性係數 .85–.94 代表尚可的相似，高於 .95 可視兩個因素或成分相同。— *Methodology* 2(2):57–64, [doi:10.1027/1614-2241.2.2.57](https://doi.org/10.1027/1614-2241.2.2.57)，[Groningen 研究入口](https://research.rug.nl/en/publications/tuckers-congruence-coefficient-as-a-meaningful-index-of-factor-si/)（門檻數字來自摘要層級的搜尋結果；全文 PDF 下載失敗，未核對內文）
- MacCallum, Widaman, Zhang & Hong (1999) 的分級：.98–1.00 excellent、.92–.98 good、.82–.92 borderline、.68–.82 poor、< .68 terrible。— 轉引自 Hattori et al. (2017)，[PDF](https://www.quantpsy.org/pubs/hattori_zhang_preacher_2017.pdf) p. 724（二手引用，未查原文）
- Nguyen & Waller 用對齊後的負荷量 RMSD（公式為所有 p×k 元素平方差平均的平方根）比較局部解，對齊用 fungible 的 `faAlign()`。— [PMC11523187](https://pmc.ncbi.nlm.nih.gov/articles/PMC11523187/) 式 (18)
- Hattori 等人建議計算所有局部解之間的一致性係數，找出規律，也要和其他旋轉方法的結果比較。— Hattori et al. (2017)，Discussion
- 比較前必須先處理行交換與行反號，因為兩者不改變準則值。— Hattori et al. (2017) p. 722；GPArotation `GPA1guide.Rnw` 也說同一解在不同隨機起點下可能出現不同因素順序或正負號（本機 vignette，約第 157–165 行）

### Inferences
- 一致性係數對小差距極不敏感。假設性例子：10 個變項、一行負荷量平方和約 1，每個元素都差 1e-2，就算差異完全垂直於原向量，係數仍約 0.9995。所以 .95 或 .98 這類門檻完全不能拿來定 2e-5 或 5e-3 的 parity 容忍度，只能說明 5e-3 等級的差異在實質上等於「同一個因素」。
- 行反號或行交換會讓 Φ 的非對角元素跟著變號或換位，對齊錯誤造成的 Φ 差距是 2|φ_ij| 等級，遠大於 5e-3。因此即使放寬到 5e-3，對齊錯誤仍會被抓到。
- 文獻沒有 Φ 專屬門檻，代表「Φ 用不同容忍度」既沒有被支持，也沒有被反對。能不能這樣做，要靠 Q1 和 Q3 的數值論證。

### Gaps
- Lorenzo-Seva & ten Berge (2006) 全文未能下載，無法確認他們是否談到斜交解或 Φ。
- 沒找到專門討論「兩個斜交解的 Φ 該怎麼比」的方法論文獻。

## Q6 用「準則值＋負荷量」判定等價，並把平坦區內的 Φ／S 差異視為非錯誤，學理上成立嗎？有什麼反論？

### Takeaway
方向上成立：判定依據（準則值加負荷量）和 GPArotation、fungible、Nguyen & Waller、Hattori 的做法一致；Φ／S 在谷底上比負荷量飄得更遠，有代數推導和兩組實測支持。實測也說明現行「負荷量一致就用 2e-5 比 Φ」的規則本身不一致。但「一律 5e-3」這個數字沒有任何文獻或推導撐腰：對近秩一案例餘裕只有 1.6 倍，對條件良好案例又寬了約 50 倍。比較好的做法，是用精確的內部一致性檢查和駐點證明，取代寬鬆的逐元素比較。

### Cited Findings
- 反論一：準則值相同不代表同一解。Nguyen & Waller 的例子中準則值相同（.82126），負荷量 RMSD 卻有 .108，Φ 也不同。— [補充材料 PDF](https://supp.apa.org/psycarticles/supplemental/met0000467/met0000467_sm2.pdf) Table S3–S4
- 反論二：Φ 會被拿來做實質解釋，而且結構矩陣的解讀依賴 Φ（ΛΦ），不能當成附屬輸出。— GPArotation `GPA1guide.Rnw`（本機）；Schmitt & Sass (2011), [doi:10.1177/0013164410387348](https://doi.org/10.1177/0013164410387348)
- 局部解會讓相同作答反應得到不同的潛在特質估計和測量標準誤，也就是因素分數層面的後果。— Nguyen & Waller (2024)，[PMC11523187](https://pmc.ncbi.nlm.nih.gov/articles/PMC11523187/) 摘要
- psych 的 Thurstone（迴歸法）分數權重是 `solve(r, S)`，其中 `S <- f %*% Phi`。所以分數權重直接繼承結構矩陣的差異，再乘上 R⁻¹。— psych 2.6.5 本機原始碼 `factor.scores`（[CRAN psych](https://cran.r-project.org/package=psych)）
- 因素分數另有「資料層級的不定性」：模型在結構層級有定義，在資料層級卻不唯一（Grice, 2001）。這是統計上的不定性，和數值收斂無關。— psych 2.6.5 `factor.scores.Rd`；Grice (2001), *Psychological Methods* 6(4):430–450, [doi:10.1037/1082-989X.6.4.430](https://doi.org/10.1037/1082-989X.6.4.430)（只經 Crossref 核對書目）
- GPArotation 指南以程式驗證 L·T′ = A 與 Φ = T′T 在機器精度內成立；Hattori 等人也說不同旋轉解對資料的適配完全相同。由前兩式可推得 LΦL′ = L T′T L′ = AA′，也就是共同變異重建不受旋轉影響（這一步是我的代數推導，指南沒有直接寫）。— GPArotation `GPA1guide.Rnw` 約第 120–149 行（本機）；Hattori et al. (2017) p. 722–723

### Inferences
- 支持放寬的論證鏈：(1) 準則值一致到 2.7e-9，依 GPArotation 定義屬同一最小值；(2) 準則只透過 L 看 T，所以谷底方向上 Φ／S 的移動會被放大；(3) eps = 1e-5 時殘留參數誤差約 eps/λ_min，實測量級吻合；(4) psych 用無種子的 20 個起點、依 hyperplane 挑解，參考值本身就在谷底隨機分布。所以 Φ／S／分數在谷底範圍內的差異，不應算成移植錯誤。
- 反對「一律 5e-3」的理由：
  - 這個數字沒有推導基礎，餘裕因資料差很多（近秩一約 1.6 倍，PCA／geomin 約 50 倍）。
  - 5e-3 有機會蓋掉只影響 Φ／S／分數下游的小錯誤，例如分數方法選錯、R⁻¹ 計算誤差、Kaiser 常態化後 Φ 的處理細節。
  - 反向風險：分數多乘了 R⁻¹，條件數大時，5e-3 對分數可能還不夠寬。
- 建議的替代設計（依嚴謹度排列，都是推論，需實作驗證）：
  1. **精確內部一致性（2e-5 或更嚴）**：檢查 Go 自己的 Φ = T′T、L = A(T′)⁻¹、S = LΦ = AT、分數權重 = R⁻¹S。確認每個輸出都由同一個 T 推出，這樣下游公式錯誤不會躲進放寬的容忍度裡。
  2. **旋轉不變量對 psych 嚴格比對（2e-5）**：LΦL′ = AA′（共同性與重建矩陣）不受谷底位置影響，可以照原本的嚴格標準比。
  3. **駐點證明**：用 psych／GPArotation 的準則與梯度定義，重算 Go 解的投影梯度範數，要求 < eps，準則值不高於 psych 解加容忍度。這直接證明「Go 落在同一個谷底」，不必靠逐元素猜測。
  4. **若仍要逐元素比 Φ／S／分數**：容忍度改成依資料計算，不要全域固定。例如 max(2e-5, c × 參考套件自己跨起點的實測差距)，或用一階雅可比從負荷量差距預測 Φ 差距上限（dL = −LE′ 反解 E，再算 dΦ = ΦE + E′Φ），允許的偏差就是「負荷量差距能解釋的 Φ 差距」，不是任意 5e-3。
- 對提案的具體判斷：提案裡「負荷量已在 2e-5 內一致時，Φ／S／分數也放寬」這部分，有 PCA／geomin 實測直接支持（負荷量 9.0e-6 但 Φ 9.1e-5），學理上正當。「固定用 5e-3」這部分只算經驗值，要寫進測試說明時應標成暫定，並附上 1.6 倍餘裕這個風險。

### Gaps
- 沒有找到任何文獻直接討論「移植或軟體等價測試」應該怎麼比斜交解，所有建議都是從方法論慣例和數值分析類推。
- 上述一階雅可比界限和駐點證明，都沒有在 insyra 的測試資料上實際算過（本次任務為唯讀研究），可行性和能否穩定通過還沒驗證。
- psych `faRotations` tie-break 的索引疑點只看了 deparse 片段，未完整確認。
