# Tasks: nn-data-gates-in-ci

## 1. 查證
- [x] 1.1 本機以資料齊備的目錄跑五個閘測試，記下時間與結果。
- [x] 1.2 確認 manifest 上的每個 URL 下載回來的位元組與本機已驗證過的檔案一致（sha256 逐一比對）。

## 2. 實作
- [x] 2.1 `.github/nn-data-manifest.txt`：九個檔案的 sha256、資料目錄下的路徑與不可變來源 URL。
- [x] 2.2 `.github/workflows/nn-data-gates.yml`：`nn/` 與自身檔案觸發、手動觸發、不排程；快取、下載、校驗、跑測試、確認五個測試都真的執行。
- [x] 2.3 `ENG.md` 寫明 GPU 閘維持手動及跑法；`Docs/nn.md` 指向 manifest。

## 3. 驗證與紀錄
- [x] 3.1 以 workflow 的下載與校驗步驟在本機模擬一次：既有檔案略過、`.gz` 解壓、九個檔案校驗全數 OK。
- [ ] 3.2 推上 `0.4` 後 workflow 通過，log 顯示五個測試實際執行並 PASS。
- [ ] 3.3 `api-review.md` TS-5、`delivery-status.md`。
- [ ] 3.4 歸檔；在 #303 留言附證據並關閉。
