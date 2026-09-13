# test-suite-integrity Specification

## Purpose
測試套件的完整性契約：存在的測試必須有真實斷言，參考實作對照必須有 workflow 實際執行，repo 不追蹤編譯出的測試二進位檔。

## Requirements
### Requirement: Tests assert, and reference comparisons run

`datalist_test.go` 中的 DataList 轉換測試 SHALL 有真實斷言；因子分析邊界測試 SHALL 斷言預期結果；`reference-verification.yml` 的 scikit-learn 步驟 SHALL 同時匹配 `AgainstScikitLearn` 與 `MatchesScikitLearnPredictions`；repo SHALL NOT 追蹤 `*.test` 二進位檔。

#### Scenario: Empty test body
- **WHEN** 檢視 `TestDataListMovingAverage`
- **THEN** 它對 `MovingAverage(2)` 的結果與無效視窗的 nil 回傳做斷言

