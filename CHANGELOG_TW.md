# 變更紀錄

影響 Insyra 使用者的變更，依套件分類，分法與 release note 相同。`## Unreleased` 收錄下一個版本會包含的內容。

v0.3.0 及更早的版本不重複收錄於此，請見 [GitHub Releases](https://github.com/HazelnutParadise/insyra/releases)。

English: [CHANGELOG.md](CHANGELOG.md)

## Unreleased

### Core

- 新增 `CSVReadOptions`，以及 `ReadCSV_FileWithOptions`、`ReadCSV_StringWithOptions`。將 `RawStrings` 設為 true 時，每個 cell 都保留原始字串、跳過欄位級型別推斷，股票代號這類值不會再掉開頭的 0，空白 cell 也維持 `""` 而不是變成 NaN。`ReadCSV_File` 與 `ReadCSV_String` 的簽名和行為維持不變。
- 新增 fitted `SimpleImputer`，支援平均數、中位數、眾數與常數替代值。它會記住訓練表格的替代值並套用到後續表格，可在避免資料洩漏的前處理 pipeline 中使用；既有的 in-place `FillWith*` 方法維持不變。 它刻意沒有 `InverseTransform`，也不是 `insyra.Scaler`：補值無法還原，而一個永遠失敗的方法會讓型別斷言誤以為該能力存在。

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
- `ExecuteNearestExact` 在有 GPU 的機器上要求九個以上的鄰居時不再 panic。候選清單寬度會被夾到裝置的八個槽位，但判斷仍然索引第 `m-1` 個位置；現在裝置服務不了的請求會直接略過裝置，改由主機作答。單精度距離溢位成無限大時也不再信任那份候選清單——那種情況下排序沒有任何資訊，而邊界檢查會因為錯誤的理由通過。判斷候選清單可否信任的誤差界也放寬了，涵蓋差值本身的捨入，以及平方項小於最小正規 `float32` 的情況。
- **破壞性變更：**移除 `OpSum`、`OpSquaredDistance`、`OpNearestQuery`，連同 `ExecuteDataList`、`ExecuteDataTable`、`ExecuteProjectedDataset`、`ExecuteDistances`、`ExecuteNearestQuery`、`SquaredDistancesCPU`、`NearestQueryCPU`、對應的 WGSL kernel，以及 CLI 的 `accel run <var>`。每一個都拿吃滿所有核心的主機量過而且輸了：欄位加總是 0.7 倍，因為每個元素搬一次值只做一次加法；距離矩陣要讀回的結果隨列數乘查詢點數成長；單精度最近鄰回傳 f32，而它原本要服務的 float64 呼叫端用不了。`ExecuteNearestExact` 取代了最後這個，回傳精確的 float64 答案。被移除的表面從未出現在任何 release。`accel devices`、`accel cache`、`accel plan` 不受影響。

### `stats`

- `PCA` 對沒有欄位的表格回傳錯誤而不是 panic。形狀守衛本來就存在，但位置在它要保護的 `mat.NewDense` 呼叫下方 34 行，所以零欄位的表格會用 `mat: zero length in matrix dimension` 讓呼叫端崩潰。

- 線性、多項式、指數與對數回歸結果新增 `Predict`。方法沿用 GLM 的 prediction 簽名，針對新資料回傳 response scale 的點估計，並檢查 predictor 數量與資料列長度。R 的標準誤與 prediction interval 目前仍不在 API 範圍內。
- 新增 `KMeansResult.Assign`，將已 fitted 的中心套用到新觀測值，回傳從 1 開始的中心索引與該中心的平方歐氏距離。
- `PCAResult` 現在會回傳每個欄位 fitted 時使用的中心化與縮放參數，以及訓練資料的 scores，呼叫端可以用同一組 decomposition 投影新觀測值。
- Logistic 與 Poisson 回歸結果現在公開 fitted `Link`，命名與 `GLMResult.Link` 一致，讓呼叫端能在 `stats` 外用線性預測值重現 response prediction。
- 對分群或降維的進入點傳入 nil 表格現在會回傳錯誤而不是 panic。`KMeans`、`DBSCAN`、`Silhouette`、`HierarchicalAgglomerative`、`PCA` 和 `KMeansResult.Assign` 都在驗證之前就解參考表格，所以 nil interface 和 typed nil 兩種都會讓呼叫端崩潰。

### `ml`

- 在套件外撰寫的指標現在可以宣告自己需要什麼輸入。`ClassLabelMetric` 和 `ProbabilityMetric` 已匯出，呼叫端自己寫的指標可以像內建指標一樣要求類別標籤或機率。之前的路由是靠未匯出的標記介面，所以外部指標永遠無法要求機率——它會靜默收到模型的預測值，`Prediction.Probabilities` 為 nil 且沒有任何錯誤。兩個介面現在會讀取回傳值而不只是偵測方法存在：實作了但回傳 `false`，與完全沒實作等價。`Prediction` 的欄位也記載了各自在什麼情況下會被填入。

- `mltest.RunConformance` 現在會以數值而不只是欄位名稱檢查機率順序。它原本比對欄名與類別名，而一個用自己的 `Classes()` 產生欄名的模型從構造上就一定通過——所以機率值放在錯誤標籤底下的模型，能通過一個正是為了抓這件事而寫的檢查。現在每一列裡 `Predict` 回傳的類別，必須是機率最大的那一欄所對應的類別。改名欄位的檢查也一併加嚴：改名後的名稱是原名的超字串，所以只提到傳入欄位的錯誤訊息也能矇混過關。

- `ROCAUC` 現在會拒絕不屬於任何一個機率類別的真實標籤，而不是把它當成負類。之前它會對自己根本沒理解的資料回報完美的鑑別力（AUC 為 1 且 error 為 nil），而 `LogLoss` 對同樣的輸入是拒絕的。兩個指標不再互相矛盾。

- 新增統一的估計器與轉換器協定，包裝 `stats` 現有 fitted 模型，支援依欄名繫結特徵、可選的機率能力、PCA 轉換、KNN wrapper，以及 `ml/mltest` conformance 檢查。
- 新增 `ml.NewPipeline` 與 `ml.NewColumnTransformer`，可將前處理與模型一起 fitted、限制轉換器只處理指定欄位，並重用 fitted 結果避免前處理資料洩漏。
- 新增可由 seed 重現的 k-fold 與分層切分、每折重新 fit 的估計器交叉驗證，以及分類與迴歸指標，包含 accuracy、log loss、ROC AUC、混淆矩陣、RMSE、MAE 與 R²。
- 新增可重現的 histogram 決策樹分類與迴歸，支援數值欄位分位數分箱、類別子集合切分、學習缺失值路徑、成長界限、類別機率與特徵重要度。
- 強化 `ml` 協定與 pipeline：拒絕未命名特徵欄位和目前無法支援的迴歸 offset，讓 logistic 模型區分類別標籤與機率，讓 fitted pipeline 保留支援的能力與輸入欄位順序，並保留小量級目標值的決策樹迴歸精度。
- 新增不依賴 C 的 ONNX 匯出，支援線性與 logistic 模型、決策樹，以及包含支援的 scaler 和 encoder 的 fitted pipeline。無法支援的模型會在寫入前拒絕，匯出測試在環境具備 `onnxruntime` 時會做獨立 runtime round-trip。

### CLI

- `load <file.csv>` 新增 `infer true|false` 選項，預設 `true`。指定 `infer false` 時所有 cell 都讀為原始字串。JSON 與 Excel 檔案不接受這個選項。
