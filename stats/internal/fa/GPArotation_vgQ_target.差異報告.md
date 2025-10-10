檔案: GPArotation_vgQ_target.go
對應 R 檔案: GPArotation_vgQ.target.R

摘要（高階差異）

- target 相關的 vgQ 會處理目標矩陣的損失函數與梯度計算，Go 需確保 target 矩陣的形狀與 lower.tri 行為一致。

建議

- 檢查 target 參數的傳遞與 target 矩陣索引是否與 R 一致。
