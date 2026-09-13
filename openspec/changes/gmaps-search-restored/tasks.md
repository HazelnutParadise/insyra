# Tasks: gmaps-search-restored

## 1. 查證
- [x] 1.1 實測 `Search` 對任何查詢都回傳 0 家店且沒有警告，`listugcposts` 回 403。
- [x] 1.2 找出 Maps 網頁用來列搜尋結果的請求，以純 HTTP 重送並確認拿得到店家 ID 與名稱。
- [x] 1.3 逐一拿掉 `pb` 欄位，找出與完整字串結果相同的最短版本（鼎泰豐、星巴克兩個查詢比對）。

## 2. 實作
- [x] 2.1 `Search` 改打 `/search?tbm=map`，從結果清單讀 ID 與名稱，去重、略過無效項目，沒結果時警告。
- [x] 2.2 移除執行期下載的遠端設定，端點與 user agent 改為常數，`GoogleMapsStores()` 不回傳 nil。
- [x] 2.3 所有請求共用一個有 30 秒逾時的 client，回應讀取上限 64 MiB。
- [x] 2.4 `GetReviews` 進度改記 debug log，修正等待上限 1000 時 panic，零值選項視為預設且不警告。

## 3. 測試
- [x] 3.1 本地伺服器測試：搜尋結果解析、無結果警告、請求失敗警告、建立爬蟲不需網路。
- [x] 3.2 等待上限 1000 抓兩頁不 panic；把等待公式改回舊版時測試會失敗。
- [x] 3.3 只設一個選項時沒有警告，排序參數正確送出。
- [x] 3.4 `gmaps_live` build tag 的實際搜尋測試，跑一次並通過。

## 4. 收尾
- [x] 4.1 `gofmt -l .`、`go build ./...`、`go test ./...`、`golangci-lint run`。
- [x] 4.2 兩份 CHANGELOG 的 `datafetch` 段落。
- [x] 4.3 `Docs/datafetch.md`：建立爬蟲不回傳 nil、搜尋結果上限、評論目前被 Google 拒絕、選項零值。
- [x] 4.4 `api-review.md` 的 DF-1、SEC-14，`delivery-status.md`，在 #249 留言說明進度（issue 保持開啟，評論仍在研究）。
