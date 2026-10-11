# Tasks: chisq-labels-follow-number-text

## 1. 獨立性檢定同一時點讀兩個 list

- [x] 1.1 測試：沿用 `resizingPair`，另一個 goroutine 以 `AtomicDoAll` 同時改變兩個 list 的長度，反覆呼叫 `ChiSquareIndependenceTest`，不得出現長度不一致的錯誤（`TestChiSquareIndependenceReadsBothListsAtOneMoment`）
- [x] 1.2 `stats/chi_square.go`：以 `insyra.AtomicDoAll` 讀取兩個 list
