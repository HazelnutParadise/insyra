# Tasks: chisq-observed-expected

## 1. 以 R 的 chisq.test 產生參考值

- [x] 1.1 新增 `stats/testdata/chisq_test_reference.R`，直接呼叫 `chisq.test`：適合度三個以上案例（均等、指定 `p`、`rescale.p = TRUE`），獨立性三個案例（3×2、2×2、3×3，一律 `correct = FALSE`），輸出統計量、p 值、自由度、觀察與期望次數及類別標籤；執行產生 `stats/testdata/chisq_test_reference.txt` 並提交

## 2. 結果改為兩張表

- [x] 2.1 先寫會失敗的測試：`Observed`、`Expected` 的形狀、列名、欄名與數值（對 1.1 的參考值）、`Expected` 欄 `Sum()` 不為 `NaN`；在舊程式上編譯失敗於 `res.Observed undefined`
- [x] 2.2 `stats/chi_square.go`：以 `Observed`、`Expected` 取代 `ContingencyTable`，`Show()` 印出兩張表；既有的卡方測試與跨語言測試改讀新欄位，斷言的值不改；2.1 通過

## 3. 適合度檢定的機率以類別為鍵

- [x] 3.1 先寫會失敗的測試：map 依類別對位（與輸入順序無關）、未出現的鍵與缺少的類別的錯誤訊息、nil 與空 map 為均等機率、`p` 不被修改；在舊程式上編譯失敗於型別不符
- [x] 3.2 `ChiSquareGoodnessOfFit` 改收 `map[string]float64`，移除 doc comment 的 IMPORTANT 警告，改寫為以類別為鍵的說明與「未出現的類別無法納入」的限制；3.1 通過

## 4. CLI

- [x] 4.1 先寫會失敗的測試：`chisq gof colors red=0.5 green=0.3 blue=0.2` 的輸出等於程式庫以同樣 map 的結果；`chisq gof colors 0.5 0.3 0.2` 回傳 `chisq gof: expected label=proportion, got "0.5"`；重複類別回傳錯誤
- [x] 4.2 `cli/commands/hypothesis.go` 的 `chisq gof` 解析 `label=p`，更新 `Forms` 與 `Examples`；4.1 通過

## 5. 驗證

- [x] 5.1 `gofmt`、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`（0 issues）通過；stats 跨語言套件以 `INSYRA_REQUIRE_REFERENCE_TOOLCHAINS=1` 執行通過

- [x] 5.2 對抗式審查（Claude Code Agent，Opus）沒有 P0、P1。四項 P2 已處理：`Docs/cli-dsl.md` 補上負數開頭的標籤在單次指令模式要加 `--`；`Docs/stats.md` 的交叉驗證章節改列 `chisq.test` 並更新腳本數與 R 版本；刪除 `AGENTS.md` 已解決的 `help chisq` 範例項目；補上兩張表欄名、含 `=` 的標籤與空標籤的測試。標籤用 `fmt` 而非程式庫的數字文字規則、獨立性檢定分兩次讀清單兩件事，記為 `AGENTS.md` follow-up

## 6. 文件

- [x] 6.1 `Docs/stats.md` 的卡方章節：新簽名、`Observed`／`Expected` 的形狀與讀法、以類別為鍵的機率與錯誤、範例改寫並實際編譯執行；總覽表的簽名同步
- [x] 6.2 `Docs/cli-dsl.md` 的卡方段落改為 `label=p` 形式
- [x] 6.3 `CHANGELOG.md` 與 `CHANGELOG_TW.md` 的 `## Unreleased`：`` ### `stats` `` 末尾兩條 BREAKING（兩張表、map），`### CLI` 末尾一條 BREAKING（`chisq gof`）
- [x] 6.4 `api-review.md` 的 ST-6 列標為 `~~Med~~ 已修正（chisq-observed-expected）`，問題清單同步
- [x] 6.5 `delivery-status.md` 的 Latest Milestones 最上方新增條目
- [x] 6.6 檢查 `skills/insyra/` 與 `skills/use-insyra-cli/`：兩者都不教卡方的參數或欄位，本變更不需修改
- [x] 6.7 `Docs/tutorials/ab-test-decision-with-statistics.md` 的 CLI 附錄：刪去 `chisq gof variant 0.1,…`（把轉換率當類別，且原本就因整串只是一個參數而無法執行），`ztest two` 補上缺少的 `sigma2`（原本同樣無法執行）；兩段都以建好的 CLI 實際跑過
- [x] 6.8 `openspec validate chisq-observed-expected --strict` 通過
