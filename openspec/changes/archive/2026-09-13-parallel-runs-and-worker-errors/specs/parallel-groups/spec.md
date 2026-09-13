## ADDED Requirements

### Requirement: A group can run many times, each run on its own

`parallel.GroupUp` SHALL 複製傳入的函式清單，之後修改呼叫端的 slice SHALL NOT 影響這個 group。每次呼叫 `(*ParallelGroup).Run` SHALL 啟動一次獨立的執行，回傳一個新的 `*RunningGroup`，各次執行的結果 SHALL 分開存放。多個 goroutine 同時對同一個 group 呼叫 `Run` SHALL NOT 產生資料競爭。

#### Scenario: Running a group twice
- **WHEN** `g := GroupUp(f)` 之後呼叫 `g.Run()` 兩次並分別等待
- **THEN** `f` 被呼叫兩次，兩次執行各自拿到自己的結果

#### Scenario: Concurrent runs under the race detector
- **WHEN** 多個 goroutine 同時對同一個 group 呼叫 `Run().AwaitResult()`，並以 `go test -race` 執行
- **THEN** 每個 goroutine 拿到完整的結果，race detector 沒有回報

### Requirement: Only a run can be awaited

`*ParallelGroup` SHALL 只有 `Run` 方法，SHALL NOT 有 `AwaitResult` 或 `AwaitNoResult`。`*RunningGroup` SHALL 只有 `AwaitResult` 與 `AwaitNoResult`，SHALL NOT 有 `Run`。等待同一次執行多次 SHALL 回傳相同的結果與錯誤。

#### Scenario: Awaiting a group that was never run
- **WHEN** 程式碼寫成 `GroupUp(f).AwaitResult()`
- **THEN** 編譯失敗

### Requirement: A slot holds what its function returned, and a failure is a WorkerError

`(*RunningGroup).AwaitResult` SHALL 回傳 `([][]any, error)`，結果長度 SHALL 等於函式數量，每一格 SHALL 只放該函式的回傳值（沒有回傳值時為 nil），函式自己回傳的 `error` SHALL 留在結果格裡。函式 panic、或傳入的值無法呼叫（不是函式、nil 函式、需要參數的函式）時，該格 SHALL 為 nil，並 SHALL 以 `*parallel.WorkerError` 回報，含 `Index`、`Panic`（panic 的值）、`Stack`（panic 當下的堆疊）與 `Err`（無法呼叫的原因）。多個失敗 SHALL 依 index 順序以 `errors.Join` 回傳。panic 的值本身是 `error` 時，`errors.Is` SHALL 能比對到它。`AwaitNoResult` SHALL 回傳同一個錯誤。套件 SHALL NOT 讓 worker 的 panic 傳出去。

#### Scenario: A function returns an error of its own
- **WHEN** 函式回傳 `(0, errors.New("mine"))`
- **THEN** 該格是 `[]any{0, err}`，`AwaitResult` 回傳的 error 為 nil

#### Scenario: A function panics
- **WHEN** 三個函式中第二個 panic
- **THEN** 其餘兩格有值，第二格為 nil，`errors.As(err, &we)` 得到 `we.Index == 1` 與 panic 的值

#### Scenario: A value that is not a function
- **WHEN** `GroupUp(42)`
- **THEN** 該格為 nil，錯誤是 `Err` 不為 nil、`Panic` 為 nil 的 `*WorkerError`
