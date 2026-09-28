## MODIFIED Requirements

### Requirement: A chart is tested by rendering it

繪圖套件的每一個 `CreateXxx` SHALL 由測試產生實際輸出檔（`gplot` 存成圖檔、`plot` 存成 HTML）並斷言檔案非空，而不是只斷言回傳的錯誤是 nil。文件寫明會回傳錯誤的每一種輸入 SHALL 也各有一個測試，斷言錯誤不為 nil 且圖表為 nil。

#### Scenario: A chart that builds but cannot render
- **WHEN** 某個 `CreateXxx` 回傳的圖表無法寫出檔案
- **THEN** 該圖表的測試失敗，即使建構本身沒有回傳錯誤
