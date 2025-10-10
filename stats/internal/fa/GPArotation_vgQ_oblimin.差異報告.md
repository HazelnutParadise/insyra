檔案: GPArotation_vgQ_oblimin.go
對應 R 檔案: GPArotation_vgQ.oblimin.R

摘要（高階差異）

- `vgQOblimin` 在 Go 與 R 中基本等價：先計算 L^2，再乘上非對角矩陣 (notDiag)，如果 gam != 0 則套用 (I - gam/p *J) 變換，計算 Gq = L* X 與 f = sum(L^2 * X)/4。

逐行重要差異摘錄

- Go 將 boolean matrix 轉為 1/0 並以數值矩陣運算實作，等價於 R 的邏輯。
- 函式回傳額外 error，用以處理可能發生的算術錯誤。

相容性風險與建議

- 與其他 vgQ 函式一樣，建議驗證在極端 gamma 值或奇異輸入下的數值穩定性。

## 優先次序（建議）

1. 數值穩健性驗證（高），2. 測試與 fixtures 建立（中），3. 文件化（低）。

## 下一步

- 在 `tests/fa/fixtures/oblimin` 建立由 R 產生的幾組測資（含極端 gamma、接近奇異矩陣），並新增 Go 單元測試確認 f/Gq 與 residual 行為。
