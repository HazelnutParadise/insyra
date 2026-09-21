# Tasks: ccl-identifier-is-only-an-index

## 1. 測試先紅
- [x] 1.1 `qty_1 * 2` 這種帶底線的識別字回錯誤，訊息教 `['qty_1']`。
- [x] 1.2 `price * 2` 的訊息說明它被讀成欄索引，並指出這張表有名為 price 的欄。
- [x] 1.3 表上沒有同名欄位時，訊息維持原樣，不亂建議括號寫法。
- [x] 1.4 賦值左側同一套規則：`price = …` 回錯誤，`['price'] = …` 與 `A = A * 10` 照常。

## 2. 實作
- [x] 2.1 `Bind` 的識別字分支只解析 Excel 索引，欄名表只拿來寫訊息。
- [x] 2.2 `checkCCLColRange` 取得出界識別字的原文與欄名表，組出教學式訊息。
- [x] 2.3 `executeAssignment` 的目標解析拿掉退回欄名那段。
- [x] 2.4 `parquet` 的 `resolveAssignTarget` 同步，錯誤訊息比照。
- [x] 2.5 `[qty_1]` 這種括號索引改在編譯階段失敗。

## 3. 文件與紀錄
- [x] 3.1 `Docs/CCL.md` 改寫解析規則那兩處，範例檢查一遍。
- [x] 3.2 兩份 CHANGELOG 標 BREAKING；`skills/insyra/` 同步。
- [x] 3.3 `api-review.md` CCL-1 與 issue 對照列、`delivery-status.md`。
- [ ] 3.4 全套驗證（gofmt、build、vet、test、golangci-lint）後歸檔，在 #341 留言附證據並關閉。
