# Tasks: hypothesis-tests-refuse-non-finite

## 1. 共用檢查

- [x] 1.1 `stats/numericinput.go` 新增 `appendNumericValues(dst, raw, label, position)`，標籤只在拒絕某格時才組出來，`numericValues` 改為以 `"row"` 呼叫它；`stats/numericinput_test.go` 斷言 `position` 會出現在兩種錯誤訊息中、值會接在 `dst` 後面、沒有拒絕時不會組標籤，且 `numericValues` 的訊息逐字不變，`go test -run 'TestAppendNumericValues|TestNumericValues' ./stats/` 通過
- [x] 1.2 `stats/asdatalist.go` 對不是 `*insyra.DataList` 的實作改用它 `AtomicDo` 交出來的 `*insyra.DataList`，不再以 `NewDataList` 重建而攤平 slice 格；`stats/type_guard_test.go` 的 `TestNonConcreteDataListKeepsSliceCells` 先失敗後通過，涵蓋八個函式；以 `isr` list 交替量測 `OneWayANOVA` 與 `KruskalWallis`，速度與配置和 base commit 相當

## 2. 單樣本與雙樣本檢定

- [x] 2.1 先寫會失敗的測試：`PairedTTest`、`SingleSampleWilcoxon`、`PairedWilcoxon`、`MannWhitneyU` 各自遇到 `NaN`、`+Inf`、`-Inf`、nil 格與文字格時回傳 spec 規定的完整錯誤字串且結果為 nil，nil 與帶型別的 nil list 回傳錯誤而不 panic；記錄實作前的失敗輸出
- [x] 2.2 四個函式改用 `asDataList` 與 `numericValues`，標籤為 `data`、`data1`、`data2`；長度與空樣本檢查維持在轉換之前，轉換在鎖外進行；2.1 的測試與 `go test ./stats/` 全數通過，既有測試不修改

## 3. 多組檢定

- [x] 3.1 實作前先跑 `go test -run '^$' -bench BenchmarkTwoWayANOVA -count 6 ./stats/`，記錄結果：32.1、34.3、34.8、41.4、51.5、56.2 µs/op
- [x] 3.2 先寫會失敗的測試：`OneWayANOVA`、`TwoWayANOVA`、`RepeatedMeasuresANOVA`、`KruskalWallis`、`FriedmanTest` 各自遇到 `NaN`、`+Inf`、`-Inf`、nil 格與文字格時回傳 spec 規定的完整錯誤字串；nil 與帶型別的 nil list 回傳錯誤，`OneWayANOVA`、`KruskalWallis`、`FriedmanTest` 不再讓測試程式崩潰；`LeveneTest`、`BartlettTest` 與上述函式其他提到組、格、受試者的錯誤都從 1 起算；記錄實作前的失敗輸出
- [x] 3.3 五個函式在啟動 goroutine 或上鎖前先以 `asDataList` 轉換每個參數，再以 `appendNumericValues` 檢查並直接接到原本的值 slice 後面，標籤與位置詞依 design.md 的表；`LeveneTest`、`BartlettTest` 的組號與其他位置錯誤改為從 1 起算；`OneWayANOVA`、`KruskalWallis`、`FriedmanTest` 保留平行讀取；3.2 的測試與 `go test ./stats/` 全數通過，既有測試不修改
- [x] 3.4 實作後以同一指令重跑 `BenchmarkTwoWayANOVA`，把前後數字寫進本任務；若差距超過前後各自的波動範圍，改為就地檢查再量一次。第一版每組先以 `fmt.Sprintf` 組標籤、另配一個 slice，與 base commit 交替各跑 10 次、每次 2 秒，中位數從 33.4 µs 變成 40.2 µs，慢了約 20%，兩次量測一致，因此改成 `appendNumericValues`。改完後同樣交替量測：base 中位數 15.7 µs、59 allocs/op，本變更 16.5 µs、39 allocs/op，兩邊最快的一次是 14.9 µs 與 14.7 µs，差距在波動範圍內

## 4. 文件

- [x] 4.1 `Docs/stats.md` 的「Values that are not numbers」改成所有假設檢定同一種處理，訊息表改為每個函式的新標籤與位置詞，刪掉 `NaN`／±Inf 會混進結果的段落；`TwoWayANOVA` 說明錯誤訊息中 A、B 水準從 1 起算與 `cells` 索引的對應；`LeveneTest`、`BartlettTest`、`KruskalWallis`、`FriedmanTest` 的說明與新行為一致；文件中每一則錯誤訊息都以 2.1、3.2 的測試或實際執行結果核對過
- [x] 4.2 `skills/insyra/SKILL.md` 第 67 行改寫：統計函式拒收非數值與非有限值並指出從 1 起算的位置，`NaN` 與 ±Inf 只剩 `PCA` 與 `KNNRegress` 的目標值會帶進結果
- [x] 4.3 `CHANGELOG.md` 與 `CHANGELOG_TW.md` 的 `## Unreleased` 新增 `` ### `stats` `` 小節與條目，寫明受影響的函式、實測的舊結果、新訊息格式、nil list 行為，以及有限輸入的結果不變；兩份內容一致
- [x] 4.4 `AGENTS.md` 的 Follow-ups：`stats` nil list 那一條刪去本變更已處理的 `OneWayANOVA`、`KruskalWallis`、`FriedmanTest`、`TwoWayANOVA`，保留尚未處理的部分；新增一條記錄假設檢定以外從 0 起算的位置與 `PCA`、`KNNRegress` 接受 `NaN` 的實測

## 5. 整合驗證

- [x] 5.1 `go build ./...`、`go test ./...`、`golangci-lint run`、`npx -y @fission-ai/openspec@latest validate hypothesis-tests-refuse-non-finite --strict` 全數通過
- [x] 5.2 以變更前量測用的探測程式重跑九個函式的 `NaN`、`+Inf`、`-Inf`、nil 格、文字格與 nil list，確認每個都回傳錯誤且不 panic
