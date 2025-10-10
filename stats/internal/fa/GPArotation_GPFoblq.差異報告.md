檔案: GPArotation_GPFoblq.go
對應 R 檔案: GPArotation_GPFoblq.R

摘要（高階差異）

- Go `GPFoblq` 為 R `GPArotation::GPFoblq` 的對應實作，負責 oblique rotation（包含 NormalizingWeight 與 T 矩陣迭代更新）。
- 實作了 computeL、computeGMatrix、computeGp 等輔助函式來對應 R 的矩陣操作。

逐行重要差異摘錄

- 內部 step acceptance 條件 (improvement > 0.5 *s^2* alpha) 與 R 相同，且在無改善情況下使用最後嘗試的 Tnew。
- 對於 normalize 的處理與 R 相符：在迭代前除以 row norms，結束後再乘回。
- 在某些錯誤或數值問題時 Go 會 panic 而非 R 的 try/warning 機制。

相容性風險與建議

- 與 `GPForth` 相同，建議降低 panic 的使用並提供返回 error 的選項。
- 建立測試案例以驗證在各種 method（oblimin, quartimin, simplimax, geomin 等）下的數值一致性。
