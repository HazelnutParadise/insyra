# Tasks: gmaps-review-extra-fields

## 1. 查證
- [x] 1.1 在存下的 20 則評論上確認語言代碼、內文、評論數與相片數的位置，並與評論視窗顯示的數字比對。

## 2. 測試（先寫、先看它失敗）
- [x] 2.1 評論頁測試期望四個新欄位的值。
- [x] 2.2 `ToDataTable` 測試確認四個新欄與值。
- [x] 2.3 live test 確認評論 ID、語言與評論數有值。

## 3. 實作
- [x] 3.1 `GoogleMapsStoreReview` 加上 `ReviewID`、`Language`、`ReviewerReviewCount`、`ReviewerPhotoCount`，`GetReviews` 填入。
- [x] 3.2 `ToDataTable` 加上四欄。

## 4. 收尾
- [x] 4.1 `gofmt -l .`、`go build ./...`、`go test ./...`、`golangci-lint run`，以及 live test。
- [x] 4.2 兩份 CHANGELOG 的 `datafetch` 段落。
- [x] 4.3 `Docs/datafetch.md` 的欄位與 DataTable 欄位說明，`delivery-status.md`。
