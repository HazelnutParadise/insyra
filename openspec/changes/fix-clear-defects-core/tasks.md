# Tasks: fix-clear-defects-core

## 1. 先重現
- [x] 1.1 為每一項寫測試，確認修正前是紅的。

## 2. internal/utils (#337)
- [x] 2.1 微秒時間戳區間，並補上毫秒與秒之間的空隙。
- [x] 2.2 `ParseColIndex` 直接累積 0-based 值，讓 `CalcColIndex` 的整個範圍都能往返。
- [x] 2.3 `FormatValue` 不把非整數顯示成整數。
- [x] 2.4 `ConvertDateFormat` 改寫成掃描器，加上 `MMMM`／`MMM`／`A`／`[...]`。
- [x] 2.5 `ToFloat64`／`ToFloat64Safe` 對具名數值型別加上反射後備。

## 3. internal/core (#338) 與 internal/algorithms (#340)
- [x] 3.1 `BiIndex.Set` 把名字的舊 id 放回 free list。
- [x] 3.2 三個插值函式拒絕 NaN。

## 4. DataTable (#228 前半)
- [x] 4.1 垂直合併以位置對齊無名欄。
- [x] 4.2 釘住「重複欄名會被自動改名」，因為那讓原本的檢查只剩誤判。

## 5. 收尾
- [x] 5.1 `go test ./...`、`golangci-lint run`。
- [x] 5.2 兩份 CHANGELOG。
- [x] 5.3 `api-review.md` 與 `delivery-status.md`。
- [ ] 5.4 關閉 #337、#338、#340；#228 留言說明只做了前半。
