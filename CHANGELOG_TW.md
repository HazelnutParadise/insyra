# 變更紀錄

影響 Insyra 使用者的變更，依套件分類，分法與 release note 相同。`## Unreleased` 收錄下一個版本會包含的內容。

v0.3.0 及更早的版本不重複收錄於此，請見 [GitHub Releases](https://github.com/HazelnutParadise/insyra/releases)。

English: [CHANGELOG.md](CHANGELOG.md)

## Unreleased

### Core

- 新增 `CSVReadOptions`，以及 `ReadCSV_FileWithOptions`、`ReadCSV_StringWithOptions`。將 `RawStrings` 設為 true 時，每個 cell 都保留原始字串、跳過欄位級型別推斷，股票代號這類值不會再掉開頭的 0，空白 cell 也維持 `""` 而不是變成 NaN。`ReadCSV_File` 與 `ReadCSV_String` 的簽名和行為維持不變。

### `isr`

- `CSV_inOpts` 新增 `RawStrings` 欄位，由 `DT.From` 傳遞給讀取端。

### `accel`

- `accel` 現在會在真實硬體上執行。`ExecuteDataList`、`ExecuteDataTable`、`ExecuteProjectedDataset` 會在 `ExecutionResult.Reductions` 回傳每個欄位算出來的值，並附上實測的 `Transfer`、`Dispatch`、`Readback` 時間與 `BytesUploaded`。
- `accel.Session` 現在可以並發使用。所有公開方法都在 session 鎖後序列化，多個 goroutine 可以共用同一個 session；先前並發呼叫 `ExecuteDataList` 會在快取與 report 狀態上產生資料競爭。裝置提交也在行程層級序列化，因為所有 session 共用同一個 GPU handle。
- 新增 `accel.Default()`，這是整個行程共用、第一次取用時才建立的 session。探測只會執行一次，常駐快取跨運算共用，對它呼叫 `Close` 不會有作用，因為沒有任何呼叫端擁有它的生命週期。單純 import 這個套件仍然不會開啟任何裝置。
- 新增 `accel.OpSquaredDistance` 與 `Session.ExecuteDistances`，在 GPU 上計算每一列到各查詢點的平方歐氏距離，並提供 `accel.SquaredDistancesCPU` 作為參考實作。裝置結果會在執行平台上驗證與參考實作位元一致。
- 新增 `accel.OpNearestQuery` 與 `Session.ExecuteNearestQuery`、`accel.NearestQueryCPU`，回報每一列最接近的查詢點及其平方距離。取最小值的動作在裝置上完成，所以結果大小隨列數成長而非隨「列×查詢點」成長——在 Apple M3 上、64 個查詢點時比 CPU 快 13.6 倍。距離相同時取索引較小者。
- `accel` 現在包含在 `allpkgs` 裡，照標準方式 `go get .../allpkgs` 安裝就會自動註冊 GPU 後端。註冊是惰性的——在開啟 accel session 之前不會探測任何裝置。
- GPU 執行改為內建。後端是純 Go 的 WebGPU 實作（[gogpu/wgpu](https://github.com/gogpu/wgpu)），在 `accel` 初始化時自行註冊，不需要額外安裝、也不需要為了 side effect 而 import。可在 `CGO_ENABLED=0` 下建置，macOS 走 Metal、Linux 與 Windows 走 Vulkan、Windows 也可走 DirectX 12。沒有 import `accel` 的程式不會編譯到它，但 gogpu 的 module 會出現在 `go list -m all`。設定 `INSYRA_ACCEL_DISABLE_WGPU=1` 可以不改程式就關掉。
- **破壞性變更：** `BackendAllocator`、`RegisterBackendAllocator`、`AllocationRecord`、`AllocatorKind` 由 `BackendExecutor`、`RegisterBackendExecutor`、`ExecuteRequest`/`ExecuteResponse`、`ExecutorKind` 取代。舊的介面既不能帶入運算，也無法回傳值，真實後端無法實作。
- **破壞性變更：** `ExecutionResult.Allocator` 與 `ExecutionResult.AllocatorKind` 改名為 `Executor` 與 `ExecutorKind`；`ExecutionResult.BytesMoved` 移除。該欄位是用固定的後端常數推算而非實測，改由 `BytesUploaded` 與三個實測時間取代。
- GPU 執行需要明確指定精度。WGSL 沒有 `f64`，Apple GPU 也沒有雙精度硬體，因此除非把 `WorkloadEstimate.Precision` 設為 `accel.PrecisionFloat32`，否則不會把 `float64` 欄位降精度。未指定時會回退到 CPU，理由為 `precision-not-accepted`。
- 新增 fallback 理由 `no-backend-executor`、`precision-not-accepted`、`dtype-not-eligible`、`shader-compile-failed`、`buffer-too-large`、`readback-timeout`、`execution-failed`。
- 後端回報的 CPU／軟體 adapter 一律不視為加速裝置，因此沒有 GPU 驅動的機器會回退到 CPU，而不是跑軟體直譯器還宣稱是加速。
- 投影欄位便宜很多。資料集指紋不再把每個值轉成文字再 hash，`projectValues` 也不再對每個元素走 `reflect`（舊路徑每個值都會在 heap 上配一次記憶體）。4 Mi 的 `float64` 欄位：投影從 4,194,308 次配置降到 4 次、`ProjectDataList` 從 357 ms 降到 43 ms、端到端的 GPU 欄位加總從 354 ms 降到 48 ms。指紋只存在於單一 session，所以 hash 值改變不會被外部觀察到。
- `ExecuteDistances` 和 `ExecuteNearestQuery` 現在不論裝置有沒有執行都會回傳結果。之前沒有 GPU 的機器只會拿到 `Accelerated: false`、理由 `no-accelerator` 和一個空 slice，等於每個呼叫端都得自己發現並改叫 CPU 版本。裝置存在但執行失敗、逾時或超過緩衝區上限時也一樣。`Accelerated` 和 `FallbackReason` 仍然照實回報工作跑在哪裡，可觀察性沒有降低。因為請求本身被拒絕的情況——`precision-not-accepted`、`dtype-not-eligible`、`workload-unsupported`——仍然不回傳結果，因為在 CPU 上算出來的正好是呼叫端拒絕的東西。strict GPU 模式仍然回傳錯誤而不是 CPU 結果。
- 新增 `accel.OpNearestShortlist` 和 `Session.ExecuteNearestExact`，回傳每列最近的 M 個查詢點，值是精確的 `float64`，但大部分計算仍然跑在 GPU 上。裝置用單精度排序，每列回傳一份候選清單，外加它捨棄掉的最好候選的距離；主機把這份清單用 `float64` 重算後決定，遇到單精度分不出勝負的列就改用全部查詢點重算。結果與 `accel.NearestExactCPU` 完全相同，所以不需要降精度的 opt-in。在 Apple M3 上以 200,000 列對照吃滿八核的主機實測：16 維 2.5 倍、64 維 3.4 倍（都是 1024 個查詢點），而每列的距離計算量低於約 2048 次時會比主機慢，此時執行期會拒絕使用裝置並回報 `workload-not-profitable`。`ExactNearestResult.Rechecked` 會回報有多少列走了完整路徑。
- `ExecuteNearestExact` 的主機端現在會用滿所有核心。沒有裝置時的路徑，以及驗證裝置回傳候選清單的那一段，工作量超過門檻時都會切給 `GOMAXPROCS` 個 goroutine，低於門檻則維持單執行緒。200,000 列乘 16 維、1024 個查詢點時，沒有 GPU 的機器會走的那條路徑從 1.575 秒降到 306 毫秒。候選清單也改成以列為主而不是以候選為主的排列，驗證單一列不再需要跨越整個陣列。

### CLI

- `load <file.csv>` 新增 `infer true|false` 選項，預設 `true`。指定 `infer false` 時所有 cell 都讀為原始字串。JSON 與 Excel 檔案不接受這個選項。
