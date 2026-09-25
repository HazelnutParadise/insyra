# Tasks: selector-follow-through

## 1. 錯誤訊息
- [x] 1.1 測試先紅：scaler、imputer、OneHot、Label、Ordinal 收到裸字串欄名時，錯誤訊息要教 `Name(...)`。
- [x] 1.2 `resolveEncodingColumn` 回傳原因，各呼叫端帶進錯誤訊息。

## 2. 文件
- [x] 2.1 文件與 skill 範例把欄名改成 `Name(...)`，修正無法編譯的範例、過時的 struct／簽名、描述「先比欄名」的文字與 Go doc 註解，刪掉 `mkt` 過時範例。
- [x] 2.2 `Docs/isr.md` 的 `isr.Row`／`isr.Rows` 鍵改成 `isr.Name(...)`。
- [x] 2.3 兩份 CHANGELOG。

## 3. 驗證與紀錄
- [ ] 3.1 全套驗證；`delivery-status.md`；歸檔。
