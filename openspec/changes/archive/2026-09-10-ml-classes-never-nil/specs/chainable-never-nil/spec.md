## ADDED Requirements

### Requirement: Getters returning a DataList without an error never return nil

A method in `ml` or `nn` that returns an `*insyra.DataList` or `*insyra.DataTable` and no `error` SHALL NOT return nil. When it has nothing to give, it SHALL return an empty, usable value whose `Err()` states the reason. A guard against nil SHALL be kept wherever it can still fire — a value that may come from a third-party implementation of a public interface, or from an in-repo caller that passes nil deliberately — and SHALL be removed only where it has been shown to be unreachable.

#### Scenario: An unfitted classifier is asked for its classes
- **WHEN** 對尚未 fit 的分類器呼叫 `Classes()`
- **THEN** 回傳長度為 0 的 `*insyra.DataList`，其 `Err()` 說明模型尚未 fit，且呼叫任何方法都不會 panic

#### Scenario: A third-party classifier still returns nil
- **WHEN** 函式庫外部的 `ml.Classifier` 實作從自己的 `Classes()` 回傳 nil
- **THEN** insyra 消費該值的位置仍有 nil 防護，且 `ml/mltest` 的一致性檢查會指出這違反協定，而不是自己 panic
