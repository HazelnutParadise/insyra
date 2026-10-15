# Tasks: specs-pass-strict-validation

## 1. Measure first

- [x] 1.1 `openspec validate --specs --strict` on 341bbf51：143 個 spec 有 40 個不過（26 個 Purpose、17 個 spec 的 27 條需求過長），這就是紅燈
- [x] 1.2 逐一比對 0.4 修正前（8f1f6396）的版本：27 個完全相同，13 個不同

## 2. Specs

- [x] 2.1 27 個相同的 spec 直接採用 0.4 修正後（aa2a9325）的版本
- [x] 2.2 13 個不同的 spec：Purpose 依本線的需求撰寫或沿用 0.4 已存在的 Purpose；與 0.4 修正前相同的長需求區塊換成 0.4 的拆法，其餘四條依同一規則手動拆分
- [x] 2.3 拆分後新增的唯一一個情境（`EWMCol` 以欄名讀欄）以一次性測試對照程式驗證

## 3. CI and the contract

- [x] 3.1 `.github/workflows/golangci-lint.yml` 新增 `OpenSpec strict validation` job（OpenSpec 1.14.1）
- [x] 3.2 `AGENTS.md`：封存步驟與嚴格驗證規則，刪除 follow-up
- [x] 3.3 `delivery-status.md` 的 Handoff Notes

## 4. Verification

- [x] 4.1 `openspec validate --specs --strict`（143 個全過）、`openspec validate specs-pass-strict-validation --strict`、gofmt、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`
