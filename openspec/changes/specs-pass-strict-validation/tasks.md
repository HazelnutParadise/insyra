# Tasks: specs-pass-strict-validation

## 1. Measure first

- [x] 1.1 `openspec validate --specs --strict` on 8f1f6396：173 個 spec 有 78 個不過（35 個 Purpose、49 個 spec 的 91 條需求過長），這就是紅燈
- [x] 1.2 寫一個與 openspec 算法相同的本地長度檢查，數字與 openspec 一致（91 條）

## 2. Purposes

- [x] 2.1 35 個 Purpose 依各自的需求重寫；英文需求的 spec 用英文，原本是中文的短 Purpose 用中文擴寫

## 3. Long requirements

- [x] 3.1 91 條過長需求拆成各自只講一件事的需求，第一段沿用原標題，每條規則與每個情境都保留（拆分工具會在情境遺失時失敗）
- [x] 3.2 沒有情境的拆分段落補上只重述既有規則的情境，共 9 個，逐一對照程式、`go doc`、文件或 workflow 驗證：`Rolling(...).Apply(nil)`、`Headers`／`SetHeaders`、`Docs/cli-dsl.md` 的另存變數與存活時間、`cli/env` 的 Deprecated 名稱、`nn` 的十二個 `New…` 建構函式、`Docs/parquet.md` 的批次說明、`py-musl.yml`、`res.PValue`、`TwoSampleTTest(x, y, true)` 無法編譯

## 4. CI and the contract

- [x] 4.1 `.github/workflows/golangci-lint.yml` 新增 `OpenSpec strict validation` job（OpenSpec 1.14.1）；確認目前的樹通過、放入一個 TBD Purpose 時失敗
- [x] 4.2 `AGENTS.md`：封存步驟要求同一個 commit 補上 Purpose，新增嚴格驗證與拆分方法的段落，刪除兩條 follow-up
- [x] 4.3 `delivery-status.md`

## 5. Verification

- [x] 5.1 `openspec validate --specs --strict`（173 個全過）、`openspec validate specs-pass-strict-validation --strict`、gofmt、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`
