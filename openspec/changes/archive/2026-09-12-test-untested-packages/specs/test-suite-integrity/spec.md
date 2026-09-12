## ADDED Requirements

### Requirement: A package's exported surface is tested in that package

每一個對外可用的套件 SHALL 有自己的測試檔，直接呼叫它的匯出函式，而不是只靠上游套件的測試間接觸發。純粹轉出內部型別的套件（`engine/*`）SHALL 至少驗證轉出的函式確實委派，且別名在編譯期就對應到同一個型別。

#### Scenario: A re-export wired to the wrong thing
- **WHEN** `engine/atomic.AtomicDoWithInit` 把參數順序傳錯
- **THEN** `engine/atomic` 自己的測試失敗，不必等到某個上游套件剛好用到

### Requirement: A chart is tested by rendering it

繪圖套件的每一個 `CreateXxx` SHALL 由測試產生實際輸出檔（`gplot` 存成圖檔、`plot` 存成 HTML）並斷言檔案非空，而不是只斷言回傳值不是 nil。文件寫明「輸入不足時回傳 nil」的情況 SHALL 也各有一個測試。

#### Scenario: A chart that builds but cannot render
- **WHEN** 某個 `CreateXxx` 回傳的圖表無法寫出檔案
- **THEN** 該圖表的測試失敗，即使建構本身沒有回傳 nil
