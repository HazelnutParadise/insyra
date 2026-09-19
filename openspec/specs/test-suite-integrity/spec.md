# test-suite-integrity Specification

## Purpose
測試必須有斷言，參考對照必須有 workflow 執行，文件承諾的行為與函式庫賴以運作的基礎型別必須有測試釘住，需要外部工具的套件其不需要該工具的部分也必須測得到。

## Requirements

### Requirement: Tests assert, and reference comparisons run

`datalist_test.go` 中的 DataList 轉換測試 SHALL 有真實斷言；因子分析邊界測試 SHALL 斷言預期結果；`reference-verification.yml` 的 scikit-learn 步驟 SHALL 同時匹配 `AgainstScikitLearn` 與 `MatchesScikitLearnPredictions`；repo SHALL NOT 追蹤 `*.test` 二進位檔。

#### Scenario: Empty test body
- **WHEN** 檢視 `TestDataListMovingAverage`
- **THEN** 它對 `MovingAverage(2)` 的結果與無效視窗的 nil 回傳做斷言

### Requirement: Documented behaviour is pinned by a test

每一個在 `Docs/` 中以範例或文字承諾的行為，SHALL 有一個測試在該行為停止成立時失敗。合併模式 `MergeModeLeft` 與 `MergeModeRight` SHALL 被斷言到逐格的值與列順序；`SortBy` 承諾的穩定性 SHALL 被一個含重複鍵、且不靠第二個排序條件打破平手的測試釘住。

#### Scenario: An unstable sort
- **WHEN** `SortBy` 改成不穩定排序
- **THEN** 相同鍵的列失去原始相對順序，穩定性測試失敗

#### Scenario: A left join that drops unmatched rows
- **WHEN** `MergeModeLeft` 漏掉左表沒有配對的列
- **THEN** 合併測試比對到缺少的列與 `nil` 欄位，測試失敗

### Requirement: The primitives the library is built on are tested directly

`internal/core` 的 `Ring`、`BiIndex` 與 `AtomicActor` SHALL 有直接的單元測試涵蓋其邊界：空容器的取值與彈出、成長與環繞、越界索引、空名稱與負 id、複製後的獨立性，以及 actor 的序列化、同 actor 與跨 actor 重入、`AtomicDoN` 的鎖定順序與去重，和 `Close` 之後各條路徑的行為。這些測試 SHALL NOT 依賴 `insyra` 套件間接觸發。

#### Scenario: An empty ring
- **WHEN** 對空的 `Ring` 呼叫 `Get`、`PopFront` 或 `PopBack`
- **THEN** 回傳零值與 `false`，不 panic

#### Scenario: Two goroutines on one actor
- **WHEN** 兩個 goroutine 同時對同一個 actor 呼叫 `AtomicDo`
- **THEN** 兩段 callback 不重疊執行

### Requirement: Solver-free and bridge code is tested without its external dependency

不需要外部工具就能執行的程式碼 SHALL 有不依賴該工具的測試。`lp` 解析 GLPK 輸出的函式 SHALL 以固定的輸出樣本測試，不呼叫 `glpsol`；`parquet` 的 CCL 介接層 SHALL 以測試自行寫出的 Parquet 檔案測試 `FilterWithCCL`、`ApplyCCL` 與 `parquetContext` 的存取方法。

#### Scenario: GLPK is not installed
- **WHEN** 在沒有 `glpsol` 的機器上執行 `go test ./lp/...`
- **THEN** 解析函式的測試照常執行並通過

#### Scenario: A CCL filter over a Parquet file
- **WHEN** 對測試寫出的 Parquet 檔執行 `FilterWithCCL`
- **THEN** 回傳的 DataTable 只含符合條件的列

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

### Requirement: A ported numerical routine is tested against its own mathematics

從其他實作移植過來的數值程式，若原始實作在這個環境中無法執行、拿不到對照數值，SHALL 以該數學本身保證的性質測試，而不是只確認它會回傳東西。解析梯度 SHALL 與自己目標函式的數值微分相符；正交旋轉 SHALL 回傳正交矩陣且不改變每個變數的共同性；斜交旋轉的旋轉矩陣各欄 SHALL 維持單位長度。與原始實作的數值對照屬於參考驗證（#302／#303）的範圍。

#### Scenario: A criterion's gradient is wrong
- **WHEN** 某個 `vgQ*` 的解析梯度與目標函式不一致
- **THEN** 梯度測試失敗，而不是等到旋轉收斂到錯誤的位置卻仍回報成功

#### Scenario: An orthogonal rotation that is not orthogonal
- **WHEN** `GPForth` 回傳的旋轉矩陣不滿足 T'T = I
- **THEN** 測試失敗，因為旋轉後的載荷已經不代表同一個模型
