# 變更紀錄

影響 Insyra 使用者的變更，依套件分類，分法與 release note 相同。`## Unreleased` 收錄下一個版本會包含的內容。

v0.3.0 及更早的版本不重複收錄於此，請見 [GitHub Releases](https://github.com/HazelnutParadise/insyra/releases)。

English: [CHANGELOG.md](CHANGELOG.md)

## Unreleased

### Core

- 新增 `CSVReadOptions`，以及 `ReadCSV_FileWithOptions`、`ReadCSV_StringWithOptions`。將 `RawStrings` 設為 true 時，每個 cell 都保留原始字串、跳過欄位級型別推斷，股票代號這類值不會再掉開頭的 0，空白 cell 也維持 `""` 而不是變成 NaN。`ReadCSV_File` 與 `ReadCSV_String` 的簽名和行為維持不變。
- 新增 fitted `SimpleImputer`，支援平均數、中位數、眾數與常數替代值。它會記住訓練表格的替代值並套用到後續表格，可在避免資料洩漏的前處理 pipeline 中使用；既有的 in-place `FillWith*` 方法維持不變。 它刻意沒有 `InverseTransform`，也不是 `insyra.Scaler`：補值無法還原，而一個永遠失敗的方法會讓型別斷言誤以為該能力存在。
- 新增 `Config.SetAcceleration` 與 `Config.GetAccelerationEnabled`。加速預設開啟，程式控制項會控管裝置呼叫點；`INSYRA_ACCEL_DISABLE_WGPU=1` 仍是部署環境的覆寫開關，兩者同時設定時由環境變數優先。

### `isr`

- `CSV_inOpts` 新增 `RawStrings` 欄位，由 `DT.From` 傳遞給讀取端。

### `accel`

- 大型 `nn` 二維 float32 MatMul 現在透過 `accel.DeviceMatMul` 預設使用裝置。它保留每個輸出沿 `k` 的序列累加順序，透過既有 `accel` report 記錄 fallback 理由，沒有裝置或後端失敗時仍由精確的 CPU 路徑作答。設定 `INSYRA_ACCEL_DISABLE_WGPU=1` 可關閉後端。
- wgpu 後端升級到 v0.30.35，修掉上游 Metal 的 checkptr 崩潰——`go test -race` 現在能完整覆蓋裝置路徑而不再跳過，race guard 已全部移除。升級前後裝置數值 parity 不變。
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
- **BREAKING**：無法讀成有限數字的值改為拒絕，不再當成零。影響 `LinearRegression`、`PolynomialRegression`、`ExponentialRegression`、`LogarithmicRegression`、`PoissonRegression`、`GLM`、`Correlation`、`Covariance`、`CorrelationMatrix` 與 `CorrelationAnalysis`，預測變數與目標變數皆同。這些路徑原本把每個值送進一個沒有失敗管道的轉換，缺值、空白或文字會靜默變成 `0`——六筆觀測中的一個空白，就把 Pearson 係數從 0.9992 移到 0.9879，沒有錯誤，下游也分辨不出來。分群、PCA 與 KNN 原本就拒絕，因素分析原本就刪除該筆觀測。錯誤訊息會指出序列與列號，`Docs/stats.md` 也列出每個家族的處理方式。
- 新增 `RidgeRegression` 與 `LassoRegression`，完全採用 scikit-learn 的目標函數——L2 懲罰 `||y − Xβ||² + α·||β||²` 以封閉解求解、L1 懲罰 `(1/2n)·||y − Xβ||² + α·||β||₁` 以座標下降求解——截距不受懲罰、不做標準化，兩者都逐係數對 scikit-learn 驗證通過。Ridge 能處理讓 `LinearRegression` 失敗的共線性預測變數；lasso 把被懲罰淘汰的係數壓到精確的零，且未收斂時如實回報而非隱藏。兩種結果都不帶標準誤、t 值與 p 值，因為古典推論不適用於受懲罰的估計。參照實作是 scikit-learn 而非 R 的 glmnet：glmnet 預設標準化且懲罰縮放不同，同一份資料會算出不同的係數。
- 新增 `WeightedLinearRegression`（WLS）：加權常態方程式搭配精確古典推論——係數、標準誤、t 值、p 值、加權 R² 與預測全部逐欄位對 statsmodels 的 `WLS` 驗證通過。權重必須嚴格為正；零權重直接拒絕而不是猜測排除語意，因為各參照實作對自由度的處理不一致，猜出來的標準誤誰都對不上。
- 自動演算法的 KNN 可使用 GPU：blank import `accel/knnbridge` 後，有利可圖的形狀會走 exact-nearest 裝置運算，答案由 CPU 以 `float64` 重算裁定——裝置結果與暴力搜尋逐 index 相同，已在硬體上驗證，且加速以接線自身的方向實測（100k×32 對全核心：2k 測試列 1.4 倍到 10k 測試列 3.7 倍）。明確指名的演算法、`k > 7`、小形狀與沒有裝置的機器都照舊走 CPU 路徑；不 import 則 `stats` 完全不帶加速器依賴。

### `nn`

- 新增 resize 算子波，範圍由兩個新發佈 checkpoint 的盤點決定：`Resize`（nearest 與 linear、scales 或 sizes、asymmetric 與 pytorch_half_pixel 座標模式）、opset-9 `Upsample`、`Floor`、`InstanceNormalization`，以及 reflect 模式的 `Pad`。FCN-ResNet50（語意分割）與 mosaic-9（快速風格轉換）現在原封不動跑通並與 `onnxruntime` 一致；ConvTranspose 與 TopK 因為沒有目標模型需要而不建。
- 大型二維 float32 MatMul 現在不需 blank import 就會預設使用裝置。只有達到實測 16Mi MAC 地板的乘法會詢問裝置，批次或較小形狀維持位元一致的 CPU 路徑。設定 `INSYRA_ACCEL_DISABLE_WGPU=1` 或呼叫 `nn.RegisterDeviceMatMul(nil)` 可恢復 CPU-only。在 8 核 M3／Metal 上實測，裝置結果在階梯每一階都與 CPU 位元一致，勝幅從地板處的 1.35 倍到 4096 方陣的 52 倍，實測 encoder layer 從全核 CPU 約 0.9 秒降到 234 毫秒。
- autodiff tape 新增 2-D Conv、MaxPool、AveragePool、GlobalAveragePool 與推論模式 BatchNormalization 的 CNN VJP。Grouped Conv、非對稱 padding、stride、pooling denominator 語意，以及 BatchNormalization 三種梯度都通過 finite difference；固定權重 CNN 走一步 Adam 後與 PyTorch 對齊。
- 在 autodiff tape 上新增 `Layer` 與 `Sequential` surface，支援 eager 維度檢查、以 seed 控制的 He 初始化 `Dense`、activation、`Dropout`、`Flatten` 與 `Func` layer。`Predict` 會以結構方式略過 training-only layer，參數命名遵循 torch `nn.Sequential`，並在 LoadWeights 邊界把 torch Linear 的 `[out,in]` SafeTensors 權重轉成 `[in,out]`。Sequential MNIST proof 重現 `0.350281` 與 `0.163855` 的平均 loss，兩個 epoch 後達到 `95.84%` 準確率。
- 完成可訓練的 layer catalog，新增 `Conv2D`、`MaxPool2D`、`AvgPool2D`、`GlobalAvgPool`、訓練／推論語意不同的 `BatchNorm2D`、`LayerNorm` 與 `Embedding`。訓練模式 BatchNorm 對齊 torch 的 biased normalization、unbiased running variance 更新與三項梯度；Embedding 對重複 index 做 scatter-add。Torch state-dict 可載入卷積與 normalization buffers，gated CNN catalog proof 使用文件化的 30,000 筆子集，兩個 epoch 達到 `97.27%` MNIST 測試準確率。
- 新增 deterministic 的 `SaveSafeTensors`、`Sequential.SaveWeights` 與 `Sequential.ExportONNX`。儲存的 state dict 使用 torch 名稱、包含 BatchNorm running statistics，並反轉 Dense transpose；ONNX 匯出支援的推論 layer、略過 Dropout，並依 layer 位置拒絕 Func 或 Embedding。訓練後的 MLP 與 CNN 匯出結果可在 `nn` 精確 round-trip，也能在 `onnxruntime` 以 float32 容差通過。
- 新增融合且取 mean 的 `MSELoss` 與 `BCEWithLogitsLoss` tape 運算、每個參數各自保存狀態的 `SGDMomentum`、`CosineAnnealingLR` 與 global-norm `ClipGradNorm`。六步 BCE 訓練流程在每一步的 learning rate、loss、clip 前 norm 與所有參數都與 PyTorch 對齊。
- 新增與 torch 相容的 `MultiHeadAttention(embed, heads)` 與 `Residual(layers...)` layer。無 mask 的 batch-first self-attention 組合既有 tape 運算，巢狀 layer state 名稱可透過 SafeTensors round-trip 並處理 torch projection transpose，layer 組成的 encoder 走一步 AdamW 後與 PyTorch 對齊。ONNX 匯出會依 layer 位置與 kind 拒絕這兩種 composite layer。

### `ml`

- ONNX 匯出現在能產生獨立 runtime 接受的模型。兩個缺陷之所以存活，是因為 round-trip 驗證需要裝了 `onnxruntime` 的 `python3`，而它在跑過的每一台機器上都被靜默跳過。每個非字串屬性都帶著一個多餘的空字串資料欄，onnxruntime 在執行前就以 invalid graph 拒絕；樹節點又是最深葉子先寫入，而 runtime 把每棵樹在陣列中的第一個節點當作根——於是它走了一個節點就拒絕整個模型。節點現在以根為先寫入，round-trip（線性、logistic、兩種樹、pipeline）已對 onnxruntime 通過。

- `mltest.RunConformance` 現在會使用傳入的訓練標籤。它原本收下之後就丟掉，所以呼叫端傳的值沒有任何東西拿去檢查。對 `Classifier` 而言，現在會驗證 `Classes()` 涵蓋模型 fit 時看過的每一個標籤——類別集合少了一個，就代表模型永遠預測不出那個標籤。

- 新增 `ml.Clusterer`，分群模型實作這個 optional 介面來宣告自己的預測是分組指派而不是量測值，並回報 fit 收斂出幾個群。回歸指標現在會拒絕這種模型。之前 `KMeansModel` 既不是 `Classifier` 也不是連續值預測器，所以沒有東西阻止 `RMSE` 去計算它的群編號，回傳一個算術上正確但沒有意義的數字。

- 在套件外撰寫的指標現在可以宣告自己需要什麼輸入。`ClassLabelMetric` 和 `ProbabilityMetric` 已匯出，呼叫端自己寫的指標可以像內建指標一樣要求類別標籤或機率。之前的路由是靠未匯出的標記介面，所以外部指標永遠無法要求機率——它會靜默收到模型的預測值，`Prediction.Probabilities` 為 nil 且沒有任何錯誤。兩個介面現在會讀取回傳值而不只是偵測方法存在：實作了但回傳 `false`，與完全沒實作等價。`Prediction` 的欄位也記載了各自在什麼情況下會被填入。

- `mltest.RunConformance` 現在會以數值而不只是欄位名稱檢查機率順序。它原本比對欄名與類別名，而一個用自己的 `Classes()` 產生欄名的模型從構造上就一定通過——所以機率值放在錯誤標籤底下的模型，能通過一個正是為了抓這件事而寫的檢查。現在每一列裡 `Predict` 回傳的類別，必須是機率最大的那一欄所對應的類別。改名欄位的檢查也一併加嚴：改名後的名稱是原名的超字串，所以只提到傳入欄位的錯誤訊息也能矇混過關。

- `ROCAUC` 現在會拒絕不屬於任何一個機率類別的真實標籤，而不是把它當成負類。之前它會對自己根本沒理解的資料回報完美的鑑別力（AUC 為 1 且 error 為 nil），而 `LogLoss` 對同樣的輸入是拒絕的。兩個指標不再互相矛盾。

- 新增統一的估計器與轉換器協定，包裝 `stats` 現有 fitted 模型，支援依欄名繫結特徵、可選的機率能力、PCA 轉換、KNN wrapper，以及 `ml/mltest` conformance 檢查。
- 新增 `ml.NewPipeline` 與 `ml.NewColumnTransformer`，可將前處理與模型一起 fitted、限制轉換器只處理指定欄位，並重用 fitted 結果避免前處理資料洩漏。
- 新增可由 seed 重現的 k-fold 與分層切分、每折重新 fit 的估計器交叉驗證，以及分類與迴歸指標，包含 accuracy、log loss、ROC AUC、混淆矩陣、RMSE、MAE 與 R²。
- 新增可重現的 histogram 決策樹分類與迴歸，支援數值欄位分位數分箱、類別子集合切分、學習缺失值路徑、成長界限、類別機率與特徵重要度。
- 強化 `ml` 協定與 pipeline：拒絕未命名特徵欄位和目前無法支援的迴歸 offset，讓 logistic 模型區分類別標籤與機率，讓 fitted pipeline 保留支援的能力與輸入欄位順序，並保留小量級目標值的決策樹迴歸精度。
- 新增不依賴 C 的 ONNX 匯出，支援線性與 logistic 模型、決策樹，以及包含支援的 scaler 和 encoder 的 fitted pipeline。無法支援的模型會在寫入前拒絕，匯出測試在環境具備 `onnxruntime` 時會做獨立 runtime round-trip。
- `Score` 用指定的指標評估已配適的模型，不重新配適。它走的是 `CrossValidate` 同一套相容性檢查與預測組裝，因此需要機率的指標、或需要從回報機率的模型取得類別標籤的指標，兩條路徑得到的服務完全一致。scikit-learn 把預設指標掛在 estimator 類別上，Go 沒有地方掛，所以指標是參數。
- 對套件外自訂的指標為 **BREAKING**：`Metric` 新增 `Direction`，宣告分數越大越好還是越小越好。`CrossValidationResult` 會帶著它，`Better` 依它比較兩個結果。沒有這個宣告，比較兩個平均值的呼叫者有一半機率挑到較差的模型，因為內建指標一半是越大越好、一半是越小越好。回傳可排序分數卻宣告 `NoDirection` 的指標會被拒絕，而不是給它一個預設方向。
- 配適完成的 pipeline 會回報 `TransformedFeatureNames`，也就是所有步驟跑完後最終估計器實際配適的欄位。兩欄輸入、其中一欄編碼成三欄的 pipeline，原本回報兩個特徵名稱與四個重要度，兩者無從對齊。一致性檢查工具現在要求模型的重要度數量與特徵名稱數量相符。
- 新增 `PrecisionMetric`、`RecallMetric` 與 `F1Metric`，支援 macro、micro、weighted 與 binary 平均，並提供 `Precision`、`Recall`、`F1` 直接函式，全部對 scikit-learn 的 `precision_recall_fscore_support` 驗證通過。預設平均是 macro，而不是 scikit-learn 的 binary 配 `pos_label=1`——在任意標籤上那是猜測；binary 平均必須指名正類，因為與 ROC AUC 不同，這些分數會隨正類選擇而改變。從未被預測的類別貢獻 precision 0，與 `zero_division=0` 一致。
- 新增 `FitRidgeRegression` 與 `FitLassoRegression`，把新的 `stats` 估計器包進 estimator 協定，依名稱綁定特徵並通過一致性檢查。
- `GridSearch` 在完全相同的折上交叉驗證各個具名候選估計器——未提供種子時抽取一個並回報在結果上，讓比較公平且可重現——依指標宣告的方向排名、平手保留較早的候選，並回傳以全部資料重新配適的贏家。網格以具名估計器清單的形式直接提供，因為參數網格展開正是本協定刻意不做的 `clone()` 反射。
- 新增隨機森林（`FitRandomForestClassifier`、`FitRandomForestRegressor`）：bootstrap 重抽以列索引 multiset 表達、共用一份特徵編碼，每個分裂限制在隨機特徵子集（分類 √p、迴歸全部 p），以機率平均做預測，重要度為各樹重要度的再正規化平均。樹平行配適，但所有隨機抽取都源自單一種子，未指定時抽一個並回報在模型上——同一種子永遠重現同一座森林。類別在重抽前就從完整目標收集，因此某棵樹的 bootstrap 樣本缺類別也不可能造成機率欄位錯位。
- 新增梯度提升（`FitGradientBoostingRegressor`、`FitGradientBoostingClassifier`）：迴歸以平方損失擬合殘差，二元分類以 logistic 損失搭配 Newton 葉值更新，預設值採 scikit-learn，殘差歸零時提前停止並回報實際輪數。多類別目標明確拒絕並說明限制，不做近似。
- 新增 `FitWeightedLinearRegression(x, y, weights)`。權重只作用於該次配適：`CrossValidate` 沒有權重通道，此限制明文記載而非讓權重與折列靜默錯位。
- `CrossValidateWeighted` 把樣本權重送進配適：每折的估計器收到的權重，是用建構該折的同一份索引子集出來的，對齊由建構保證。`Estimator` 新增選用的 `FitWeighted`——沒提到權重的一切照舊。留出集評分維持不加權，與 scikit-learn 預設一致；沒有 `FitWeighted` 的估計器直接拒絕，不會靜默改用不加權配適。
- ONNX 匯出涵蓋新家族：ridge、lasso 與 WLS 走線性迴歸器路徑，兩種森林與兩種提升以多樹 ensemble 匯出——森林葉值乘 1/T 讓 runtime 的加總等於平均，boosting 把學習率烘進葉權重、先驗作為 base value。二元分類器採用 runtime 的單分數慣例：寫兩類權重時機率完全正確但 label 全部回傳 1，因此雙類 ensemble 每葉只帶一個分數，補數與 0.5 門檻由 runtime 計算。七個家族全部通過獨立 onnxruntime round-trip。
- 決策樹新增 `ExactSplits`：每對相鄰相異數值的中點都是分裂候選——scikit-learn 的 CART 搜尋——與預設的直方圖搜尋並存。分裂準則本來就相同（Gini、變異數），因此 exact 樹對 scikit-learn 做逐預測驗證：分類在探測網格上逐 label 精確、迴歸在單精度容差內。直方圖因 O(MaxBins) 成本維持預設；兩個選項同時設定會被拒絕，ensemble 透過 Tree 選項繼承此選擇。
- 匯出的 logistic 模型改帶兩列係數（skl2onnx 的二元慣例），修正 onnxruntime 下的機率輸出：單列形式仰賴 runtime 在二元路徑套用 LOGISTIC 轉換，而 onnxruntime 不套用——它把原始決策分數當機率回傳，之前 round-trip 只比 label 所以沒人發現。由 `nn` 的雙參照 round-trip 抓到；label 一直是對的。

### `nn`

- 新增帶 seed 的 tape `Dropout` wrapper，使用 inverted scaling，並在 VJP
  透過相同 mask 傳遞梯度。eval 路徑由呼叫端不呼叫 wrapper 來保持 identity。
- 新增與 PyTorch 相容的 decoupled `AdamW` weight decay 與 `StepLR` helper，
  以固定 PyTorch MLP 驗證多步 loss 和參數 parity，並明確驗證它與 coupled L2
  的差異。
- 新增純 Go 的 float32 ONNX 推論，支援聚焦的 MLP operator 家族。模型以 `protowire` 解碼，在載入時驗證，再用具名輸入與輸出執行。格式錯誤會回傳錯誤而不 panic，未支援的 operator 會一次列出。獨立張量 kernel 與固定權重 MLP 都已對 `onnxruntime` 驗證。
- `nn` 現在能以純 Go 讀回 `ml` 匯出的迴歸器、樹 ensemble 與帶前處理的 pipeline，使用 `ai.onnx.ml` 運算子域執行。`BindRegressor` 與 `BindClassifier` 以結構型介面把載入的網路接進 `ml` 協定，依欄名綁定輸入並通過一致性檢查。strict closure 測試以配適模型與 `onnxruntime` 雙重比對二元 logistic 分類器的 label 與機率。
- `nn` 現在支援帶有 NumPy 風格前導 batch 廣播的 N-D batched `MatMul`、LayerNormalization、GELU、reduction 與 shape-control kernel、比較、切片、分割，以及任意 rank 的 Transpose。二維 MatMul 路徑仍保留為快速路徑。
- 新增各 kernel 的單算子 `onnxruntime` parity，以及固定權重 transformer encoder proof，涵蓋雙頭 self-attention、feed-forward GELU、residual connection 與 LayerNormalization。另提供 `INSYRA_NN_REAL_MODEL` 手動 smoke path，執行支援的本機模型並列印輸出 shape。
- 新增純 Go 的 CNN 推論 kernel，支援 2-D Conv、MaxPool、AveragePool、GlobalAveragePool、推論模式 BatchNormalization 與 constant Pad。Conv 支援顯式與自動 padding、strides、dilations、groups、depthwise groups 與選用 bias；pooling 和 Pad 會清楚拒絕不支援的 ONNX 模式。單算子 parity 會列舉屬性組合，固定權重的 MNIST 類 CNN 也已與 `onnxruntime` 完成端到端比對。
- `MatMul`（二維與批次路徑）和 `Conv` 現在會在大型工作負載使用所有 CPU 核心，同時保留每個輸出的序列累加順序，因此結果與序列版本位元完全一致。8 核心 M3 實測中，encoder layer 從 3.35 秒降至約 0.9 秒、MNIST 級 CNN forward 從 526 毫秒降至約 120 毫秒（可重現約 3.8 倍與 4.4 倍，單次最佳達 4.6 倍與 5.4 倍）；小型輸入仍走序列路徑。
- 新增 `LoadSafeTensors(io.Reader)`，會以原生 dtype 精確載入具名的 `F32`、`I64` 與 `BOOL` 張量。格式錯誤、offset、shape、重複名稱與未支援的 dtype 都會回傳包含名稱的錯誤；`__metadata__` 會被接受並忽略。loader 與混合 dtype fixture 已對 Python `safetensors` 做 round-trip 驗證。
- 新增 MLP kernel 的 float32 反向模式 autodiff tape，包含 `MatMul`、支援廣播的 `Add`、`Relu`、`Sigmoid`、`Tanh`、`Gemm` 屬性、融合的 `SoftmaxCrossEntropy`，以及單步 `SGD` 更新。tape 重用推論 kernel，圖執行器維持不變。
- 將 float32 autodiff tape 擴充到 attention 訓練家族：支援廣播還原的批次 `MatMul`、軸向 `Softmax`、`LayerNormalization`、精確式 `Gelu`、`Erf`、`Sqrt`、`Pow`、`ReduceMean`、shape 運算的 VJP，以及每個參數各自保存狀態的 bias-corrected `Adam` 更新。未實作的反向運算會指出運算名稱並拒絕，不會偽造零梯度。
- 新增 `Clip`、`ConstantOfShape` 與執行期計算的 int64 shape/control tensor，補齊 published MobileNetV2 與 MiniLM-L6-v2 checkpoint 所需的執行支援。新增 gated `INSYRA_NN_REAL_MODELS_DIR` 測試，以固定輸入逐元素和 `onnxruntime` 比對兩個模型。
- 新增由資料集 gate 控制的 MNIST 收斂驗證：固定 seed 的 He 初始化 `784 -> 128 -> 10` MLP 以 Adam 訓練每批 128 筆、每個 epoch 重新 shuffle 的 minibatch，在本機 IDX 資料集兩個 epoch 內達到 95% 以上測試準確率，並加入不依賴資料集的二元 micro-convergence 測試。IDX reader 與初始化 helper 維持在測試端，不新增公開 API。
- `LoadSafeTensors` 現在接受 `F16` 與 `BF16` checkpoint，並以位元完全相同的方式拓寬成 f32。ONNX `FLOAT16` 與 `BFLOAT16` initializer 走同一條路徑，Cast 到半精度時會先依儲存格式四捨五入再拓寬回來；圖內運算維持 f32，quantized dtype 仍拒絕載入。
- 新增 detector 所需的 ONNX 推論支援：`LeakyRelu`、`Exp`、`Ceil`、`Round`、`Tile`、`ReduceMin`、批次 `NonMaxSuppression`，以及帶有 validated GraphProto body、子 scope、loop-carried value 與 scan output 的 `Loop`。單算子與 synthetic Loop parity 已對 `onnxruntime` 通過，gated tiny-YOLOv3 的 selection indices 精確相同，boxes 與 scores 通過 f32 容差。

### CLI

- `load <file.csv>` 新增 `infer true|false` 選項，預設 `true`。指定 `infer false` 時所有 cell 都讀為原始字串。JSON 與 Excel 檔案不接受這個選項。
