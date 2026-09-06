## ADDED Requirements

### Requirement: CCL methods return the receiver after a recovered panic

`AddColUsingCCL`、`EditColByIndexUsingCCL`、`EditColByNameUsingCCL`、`ExecuteCCL` 在內部 panic 被 recover 後 SHALL 回傳接收者本身並設定 `Err()`，SHALL NOT 回傳 nil。

#### Scenario: Chained call after a CCL failure
- **WHEN** CCL 評估 panic（以會觸發 panic 的運算式測試）後接著呼叫 `.NumCols()`
- **THEN** 不 nil deref，`Err()` 非 nil
