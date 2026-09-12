# Tasks: cli-message-and-help-fixes

- [x] 1.1 `parseFloatArg`／`parseIntArg`，13 個站點改用；另外 8 個已具名的去掉 strconv 尾巴。
- [x] 1.2 `kmeans`／`knn` 選項不分大小寫、列出支援清單、驗證列舉值。
- [x] 1.3 `varTypeError`，五個指令改用。
- [x] 2.1 `LookupCommand`／`SnapshotRegistry`，`help` 與補全改用。
- [x] 2.2 help 欄寬依最長名稱；`read`、`env` 補 Forms 與 Examples。
- [x] 2.3 九個 regexp 提到套件層。
- [x] 3.1 測試；`go test ./...`、`go vet`、`golangci-lint run`。
- [x] 3.2 兩份 CHANGELOG、`api-review.md`、`delivery-status.md`。
- [ ] 3.3 關閉 #324、#326、#327、#296。
