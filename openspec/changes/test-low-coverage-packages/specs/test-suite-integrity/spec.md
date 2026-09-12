## ADDED Requirements

### Requirement: A ported numerical routine is tested against its own mathematics

從其他實作移植過來的數值程式，若原始實作在這個環境中無法執行、拿不到對照數值，SHALL 以該數學本身保證的性質測試，而不是只確認它會回傳東西。解析梯度 SHALL 與自己目標函式的數值微分相符；正交旋轉 SHALL 回傳正交矩陣且不改變每個變數的共同性；斜交旋轉的旋轉矩陣各欄 SHALL 維持單位長度。與原始實作的數值對照屬於參考驗證（#302／#303）的範圍。

#### Scenario: A criterion's gradient is wrong
- **WHEN** 某個 `vgQ*` 的解析梯度與目標函式不一致
- **THEN** 梯度測試失敗，而不是等到旋轉收斂到錯誤的位置卻仍回報成功

#### Scenario: An orthogonal rotation that is not orthogonal
- **WHEN** `GPForth` 回傳的旋轉矩陣不滿足 T'T = I
- **THEN** 測試失敗，因為旋轉後的載荷已經不代表同一個模型
