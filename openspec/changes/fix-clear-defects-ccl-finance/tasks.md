# Tasks: fix-clear-defects-ccl-finance

- [x] 1.1 為四項各寫測試，確認修正前是紅的（#368 確認已經是綠的）。
- [x] 2.1 parser 檢查 `ParseFloat` 的錯誤。
- [x] 2.2 tokenizer 支援指數字尾，且不吃掉識別字。
- [x] 2.3 `TOSTR`／`TEXT` 偵測格式錯誤標記並回報。
- [x] 2.4 `finance` 加上 `Options.finish`，所有結尾捨入改走它。
- [x] 3.1 `go test ./...`、`golangci-lint run`。
- [ ] 3.2 `Docs/CCL.md`、兩份 CHANGELOG。
- [ ] 3.3 `api-review.md`、`delivery-status.md`。
- [ ] 3.4 關閉 #365、#366、#368；#248 留言說明只做了 panic 那部分。
