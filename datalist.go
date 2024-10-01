package insyra

import (
	"fmt"
	"math"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/HazelnutParadise/Go-Utils/asyncutil"
	"github.com/HazelnutParadise/Go-Utils/conv"
	"github.com/HazelnutParadise/Go-Utils/sliceutil"
)

// DataList is a generic dynamic data list.
type DataList struct {
	data                  []interface{}
	name                  string
	creationTimestamp     int64
	lastModifiedTimestamp atomic.Int64
	mu                    sync.Mutex
}

// IDataList defines the behavior expected from a DataList.
type IDataList interface {
	isFragmented() bool
	GetCreationTimestamp() int64
	GetLastModifiedTimestamp() int64
	updateTimestamp()
	GetName() string
	SetName(string) *DataList
	Data() []interface{}
	Append(values ...interface{})
	Get(index int) interface{}
	Clone() *DataList
	Count(value interface{}) int
	Counter() map[interface{}]int
	Update(index int, value interface{})
	InsertAt(index int, value interface{})
	FindFirst(interface{}) interface{}
	FindLast(interface{}) interface{}
	FindAll(interface{}) []int
	Filter(func(interface{}) bool) *DataList
	ReplaceFirst(interface{}, interface{})
	ReplaceLast(interface{}, interface{})
	ReplaceAll(interface{}, interface{})
	ReplaceOutliers(float64, float64) *DataList
	Pop() interface{}
	Drop(index int) *DataList
	DropAll(...interface{}) *DataList
	DropIfContains(interface{}) *DataList
	Clear() *DataList
	ClearStrings() *DataList
	ClearNumbers() *DataList
	ClearNaNs() *DataList
	ClearOutliers(float64) *DataList
	Normalize() *DataList
	Standardize() *DataList
	FillNaNWithMean() *DataList
	MovingAverage(int) *DataList
	WeightedMovingAverage(int, interface{}) *DataList
	ExponentialSmoothing(float64) *DataList
	DoubleExponentialSmoothing(float64, float64) *DataList
	MovingStdev(int) *DataList
	Len() int
	Sort(acending ...bool) *DataList
	Rank() *DataList
	Reverse() *DataList
	Upper() *DataList
	Lower() *DataList
	Capitalize() *DataList

	// Statistics
	Sum() float64
	Max() float64
	Min() float64
	Mean() float64
	WeightedMean(weights interface{}) float64
	GMean() float64
	Median() float64
	Mode() float64
	MAD() float64
	Stdev() float64
	StdevP() float64
	Var() float64
	VarP() float64
	Range() float64
	Quartile(int) float64
	IQR() float64
	Percentile(float64) float64
	Difference() *DataList

	// comparison
	IsEqualTo(*DataList) bool
	IsTheSameAs(*DataList) bool

	// conversion
	ParseNumbers() *DataList
	ParseStrings() *DataList
	ToF64Slice() []float64
	ToStringSlice() []string

	// Interpolation
	LinearInterpolation(float64) float64
	QuadraticInterpolation(float64) float64
	LagrangeInterpolation(float64) float64
	NearestNeighborInterpolation(float64) float64
	NewtonInterpolation(float64) float64
	HermiteInterpolation(float64, []float64) float64
}

// Data returns the data stored in the DataList.
func (dl *DataList) Data() []interface{} {
	defer func() {
		go reorganizeMemory(dl)
	}()
	return dl.data
}

// NewDataList creates a new DataList, supporting both slice and variadic inputs,
// and flattens the input before storing it.
func NewDataList(values ...interface{}) *DataList {
	var flatData []interface{}

	flatData, _ = sliceutil.Flatten[interface{}](values)
	LogDebug("NewDataList(): Flattened data:", flatData)

	continuousMemData := make([]interface{}, len(flatData))
	copy(continuousMemData, flatData)

	timestamp := time.Now().Unix()
	dl := &DataList{
		data:              continuousMemData,
		creationTimestamp: timestamp,
	}
	dl.lastModifiedTimestamp.Store(timestamp)

	return dl
}

// Append adds a new values to the DataList.
// The value can be of any type.
// The value is appended to the end of the DataList.
func (dl *DataList) Append(values ...interface{}) {
	defer func() {
		dl.mu.Unlock()
		go reorganizeMemory(dl)
	}()
	dl.mu.Lock()

	// Append data and update timestamp
	dl.data = append(dl.data, values...)
	go dl.updateTimestamp()
}

// Get retrieves the value at the specified index in the DataList.
// Supports negative indexing.
// Returns nil if the index is out of bounds.
// Returns the value at the specified index.
func (dl *DataList) Get(index int) interface{} {
	defer func() {
		dl.mu.Unlock()
		go reorganizeMemory(dl)
	}()
	dl.mu.Lock()
	// 支持負索引
	if index < 0 {
		index += len(dl.data)
	}
	if index < 0 || index >= len(dl.data) {
		LogWarning("DataList.Get(): Index out of bounds, returning nil.")
		return nil
	}
	return dl.data[index]
}

// Clone creates a deep copy of the DataList.
func (dl *DataList) Clone() *DataList {
	defer func() {
		dl.mu.Unlock()
	}()
	dl.mu.Lock()
	newDL := NewDataList(dl.data)
	newDL.SetName(dl.name)
	return newDL
}

// Count returns the number of occurrences of the specified value in the DataList.
func (dl *DataList) Count(value interface{}) int {
	found := dl.FindAll(value)
	if found == nil {
		return 0
	}
	return len(found)
}

// Counter returns a map of the number of occurrences of each value in the DataList.
func (dl *DataList) Counter() map[interface{}]int {
	counter := make(map[interface{}]int)
	for _, value := range dl.Data() {
		counter[value]++
	}
	return counter
}

// Update replaces the value at the specified index with the new value.
func (dl *DataList) Update(index int, newValue interface{}) {
	defer func() {
		dl.mu.Unlock()
		go reorganizeMemory(dl)
	}()
	dl.mu.Lock()
	if index < 0 {
		index += len(dl.data)
	}
	if index < 0 || index >= len(dl.data) {
		LogWarning("ReplaceAtIndex(): Index %d out of bounds", index)
	}
	dl.data[index] = newValue
	go dl.updateTimestamp()
}

// InsertAt inserts a value at the specified index in the DataList.
// If the index is out of bounds, the value is appended to the end of the list.
func (dl *DataList) InsertAt(index int, value interface{}) {
	defer func() {
		dl.mu.Unlock()
		go reorganizeMemory(dl)
	}()
	dl.mu.Lock()
	// Handle negative index
	if index < 0 {
		index += len(dl.data) + 1
	}

	// If index is out of bounds, append the value to the end
	if index < 0 || index > len(dl.data) {
		LogWarning("InsertAt(): Index out of bounds, appending value to the end.")
		dl.data = append(dl.data, value)
	} else {
		var err error
		dl.data, err = sliceutil.InsertAt(dl.data, index, value)
		if err != nil {
			LogWarning("InsertAt(): Failed to insert value at index:", err)
			return
		}
	}

	go dl.updateTimestamp()
}

// FindFirst returns the index of the first occurrence of the specified value in the DataList.
// If the value is not found, it returns nil.
func (dl *DataList) FindFirst(value interface{}) interface{} {
	defer func() {
		dl.mu.Unlock()
		go reorganizeMemory(dl)
	}()
	dl.mu.Lock()
	for i, v := range dl.data {
		if v == value {
			return i
		}
	}
	LogWarning("FindFirst(): Value not found, returning nil.")
	return nil
}

// FindLast returns the index of the last occurrence of the specified value in the DataList.
// If the value is not found, it returns nil.
func (dl *DataList) FindLast(value interface{}) interface{} {
	defer func() {
		dl.mu.Unlock()
		go reorganizeMemory(dl)
	}()
	dl.mu.Lock()
	for i := len(dl.data) - 1; i >= 0; i-- {
		if dl.data[i] == value {
			return i
		}
	}
	LogWarning("FindLast(): Value not found, returning nil.")
	return nil
}

// FindAll returns a slice of all the indices where the specified value is found in the DataList using parallel processing.
// If the value is not found, it returns an empty slice.
func (dl *DataList) FindAll(value interface{}) []int {
	defer func() {
		dl.mu.Unlock()
		go reorganizeMemory(dl)
	}()
	dl.mu.Lock()
	length := len(dl.data)
	if length == 0 {
		LogWarning("FindAll(): DataList is empty, returning an empty slice.")
		return []int{}
	}

	var indices []int
	for i, v := range dl.data {
		if v == value {
			indices = append(indices, i)
		}
	}
	return indices
}

// Filter filters the DataList based on a custom filter function provided by the user.
// The filter function should return true for elements that should be included in the result.
func (dl *DataList) Filter(filterFunc func(interface{}) bool) *DataList {
	defer func() {
		dl.mu.Unlock()
		go reorganizeMemory(dl)
	}()
	dl.mu.Lock()
	filteredData := []interface{}{}

	for _, v := range dl.data {
		if filterFunc(v) {
			filteredData = append(filteredData, v)
		}
	}

	return NewDataList(filteredData...)
}

// ReplaceFirst replaces the first occurrence of oldValue with newValue.
func (dl *DataList) ReplaceFirst(oldValue, newValue interface{}) {
	defer func() {
		dl.mu.Unlock()
		go reorganizeMemory(dl)
	}()
	dl.mu.Lock()
	for i, v := range dl.data {
		if v == oldValue {
			dl.data[i] = newValue
			go dl.updateTimestamp()
			return
		}
	}
	LogWarning("ReplaceFirst(): value not found.")
}

// ReplaceLast replaces the last occurrence of oldValue with newValue.
func (dl *DataList) ReplaceLast(oldValue, newValue interface{}) {
	defer func() {
		dl.mu.Unlock()
		go reorganizeMemory(dl)
	}()
	dl.mu.Lock()
	for i := len(dl.data) - 1; i >= 0; i-- {
		if dl.data[i] == oldValue {
			dl.data[i] = newValue
			go dl.updateTimestamp()
			return
		}
	}
	LogWarning("ReplaceLast(): value not found.")
}

// ReplaceAll replaces all occurrences of oldValue with newValue in the DataList.
// If oldValue is not found, no changes are made.
func (dl *DataList) ReplaceAll(oldValue, newValue interface{}) {
	defer func() {
		dl.mu.Unlock()
		go reorganizeMemory(dl)
	}()
	dl.mu.Lock()
	length := len(dl.data)
	if length == 0 {
		LogWarning("ReplaceAll(): DataList is empty, no replacements made.")
		return
	}

	// 單線程處理資料替換
	for i, v := range dl.data {
		if v == oldValue {
			dl.data[i] = newValue
		}
	}

	go dl.updateTimestamp()
}

// ReplaceOutliers replaces outliers in the DataList with the specified replacement value (e.g., mean, median).
func (dl *DataList) ReplaceOutliers(stdDevs float64, replacement float64) *DataList {
	mean := dl.Mean()
	stddev := dl.Stdev()
	threshold := stdDevs * stddev

	for i, v := range dl.Data() {
		val := conv.ParseF64(v)
		if math.Abs(val-mean) > threshold {
			dl.data[i] = replacement
		}
	}
	return dl
}

// Pop removes and returns the last element from the DataList.
// Returns the last element.
// Returns nil if the DataList is empty.
func (dl *DataList) Pop() interface{} {
	defer func() {
		dl.mu.Unlock()
		go reorganizeMemory(dl)
	}()
	dl.mu.Lock()
	n, err := sliceutil.Drt_PopFrom(&dl.data)
	if err != nil {
		LogWarning("DataList.Pop(): DataList is empty, returning nil.")
		return nil
	}
	go dl.updateTimestamp()
	return n
}

// Drop removes the element at the specified index from the DataList and updates the timestamp.
// Returns an error if the index is out of bounds.
func (dl *DataList) Drop(index int) *DataList {
	defer func() {
		dl.mu.Unlock()
		go reorganizeMemory(dl)
	}()
	dl.mu.Lock()
	if index < 0 {
		index += len(dl.data)
	}
	if index >= len(dl.data) {
		LogWarning("DataList.Drop(): Index out of bounds, returning.")
		return dl
	}
	dl.data = append(dl.data[:index], dl.data[index+1:]...)
	go dl.updateTimestamp()
	return dl
}

// DropAll removes all occurrences of the specified values from the DataList.
// Supports multiple values to drop.
func (dl *DataList) DropAll(toDrop ...interface{}) *DataList {
	defer func() {
		dl.mu.Unlock()
		go reorganizeMemory(dl)
	}()
	dl.mu.Lock()
	length := len(dl.data)
	if length == 0 {
		return dl
	}

	// 決定要開多少個線程
	numGoroutines := runtime.NumCPU()
	if numGoroutines == 0 {
		numGoroutines = 1
	}

	chunkSize := length / numGoroutines
	if length%numGoroutines != 0 {
		chunkSize++
	}

	// 儲存所有的 Awaitable
	var awaitables []*asyncutil.Awaitable

	// 啟動 Awaitables 處理每個部分
	for i := 0; i < numGoroutines; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if end > length {
			end = length
		}

		awaitable := asyncutil.Async(func(dataChunk []interface{}) []interface{} {
			var result []interface{}
			for _, v := range dataChunk {
				shouldDrop := false
				for _, drop := range toDrop {
					if v == drop {
						shouldDrop = true
						break
					}
				}
				if !shouldDrop {
					result = append(result, v)
				}
			}
			return result
		}, dl.data[start:end])

		awaitables = append(awaitables, awaitable)
	}

	// 收集所有結果並合併
	var finalResult []interface{}
	for _, awaitable := range awaitables {
		results, err := awaitable.Await()
		if err != nil {
			LogWarning("DropAll(): Error in async task:", err)
			continue
		}

		if len(results) > 0 {
			finalResult = append(finalResult, results[0].([]interface{})...)
		}
	}

	// 更新 DataList
	dl.data = finalResult
	go dl.updateTimestamp()
	return dl
}

// DropIfContains removes all elements from the DataList that contain the specified value.
func (dl *DataList) DropIfContains(value interface{}) *DataList {
	dl.mu.Lock()
	defer func() {
		dl.mu.Unlock()
		go reorganizeMemory(dl)
	}()

	// 創建一個臨時切片存放保留的元素
	var newData []interface{}

	for _, v := range dl.data {
		if str, ok := v.(string); ok {
			// 如果當前元素不包含指定的值，將其添加到 newData 中
			if !strings.Contains(str, value.(string)) {
				newData = append(newData, v)
			}
		} else {
			// 如果元素不是字符串類型，也將其保留
			newData = append(newData, v)
		}
	}

	// 將新的數據賦值回 dl.data
	dl.data = newData
	go dl.updateTimestamp()
	return dl
}

// Clear removes all elements from the DataList and updates the timestamp.
func (dl *DataList) Clear() *DataList {
	defer func() {
		dl.mu.Unlock()
		go reorganizeMemory(dl)
	}()
	dl.mu.Lock()
	dl.data = []interface{}{}
	go dl.updateTimestamp()
	return dl
}

func (dl *DataList) Len() int {
	return len(dl.data)
}

// ClearStrings removes all string elements from the DataList and updates the timestamp.
func (dl *DataList) ClearStrings() *DataList {
	defer func() {
		dl.mu.Unlock()
		go reorganizeMemory(dl)
	}()
	dl.mu.Lock()
	length := len(dl.data)
	if length == 0 {
		return dl
	}

	// 獲取可用的 CPU 核心數量
	numGoroutines := min(runtime.NumCPU(), length)

	// 決定每個線程處理的數據量
	chunkSize := length / numGoroutines
	if length%numGoroutines != 0 {
		chunkSize++
	}

	// 構建任務切片
	var tasks []asyncutil.Task

	for i := 0; i < numGoroutines; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if end > length {
			end = length
		}

		task := asyncutil.Task{
			ID: fmt.Sprintf("Task-%d", i),
			Fn: func(dataChunk []interface{}) []interface{} {
				var result []interface{}
				for _, v := range dataChunk {
					if _, ok := v.(string); !ok {
						result = append(result, v)
					}
				}
				return result
			},
			Args: []interface{}{dl.data[start:end]},
		}

		tasks = append(tasks, task)
	}

	// 使用 ParallelProcess 進行平行處理
	taskResults := asyncutil.ParallelProcess(tasks)

	// 合併所有的結果
	var finalResult []interface{}
	for _, taskResult := range taskResults {
		finalResult = append(finalResult, taskResult.Results[0].([]interface{})...)
	}

	// 更新 DataList
	dl.data = finalResult
	go dl.updateTimestamp()
	return dl
}

// ++++ 此處之後尚未提升性能 ++++

// ClearNumbers removes all numeric elements (int, float, etc.) from the DataList and updates the timestamp.
func (dl *DataList) ClearNumbers() *DataList {
	defer func() {
		dl.mu.Unlock()
		go reorganizeMemory(dl)
	}()
	dl.mu.Lock()
	filteredData := dl.data[:0] // Create a new slice with the same length as the original

	for _, v := range dl.data {
		// If the element is not a number, keep it
		switch v.(type) {
		case int, int8, int16, int32, int64:
		case uint, uint8, uint16, uint32, uint64:
		case float32, float64:
		default:
			filteredData = append(filteredData, v)
		}
	}

	dl.data = filteredData
	go dl.updateTimestamp()
	return dl
}

// ClearNaNs removes all NaN values from the DataList and updates the timestamp.
func (dl *DataList) ClearNaNs() *DataList {
	defer func() {
		go reorganizeMemory(dl)
		go dl.updateTimestamp()
	}()
	for i, v := range dl.Data() {
		if math.IsNaN(v.(float64)) {
			dl.Drop(i)
		}
	}
	return dl
}

// ClearOutliers removes values from the DataList that are outside the specified number of standard deviations.
// This method modifies the original DataList and returns it.
func (dl *DataList) ClearOutliers(stdDevs float64) *DataList {
	defer func() {
		r := recover()
		if r != nil {
			LogWarning("DataList.ClearOutliers(): Data types cannot be compared.")
		}
		go dl.updateTimestamp()
	}()

	mean := dl.Mean()
	stddev := dl.Stdev()
	threshold := stdDevs * stddev

	// 打印調試信息，確保計算值與 R 相同
	LogDebug("Mean: %f\n", mean)
	LogDebug("Standard Deviation: %f\n", stddev)
	LogDebug("Threshold: %f\n", threshold)

	for i := dl.Len() - 1; i >= 0; i-- {
		val := conv.ParseF64(dl.Data()[i])
		LogDebug("Checking value: %f\n", val) // 打印每個檢查的值
		if math.Abs(val-mean) > threshold {
			LogDebug("Removing outlier: %f\n", val) // 打印要移除的異常值
			dl.Drop(i)
		}
	}
	return dl
}

// Normalize normalizes the data in the DataList, skipping NaN values.
// Directly modifies the DataList.
func (dl *DataList) Normalize() *DataList {
	defer func() {
		r := recover()
		if r != nil {
			LogWarning("Normalize: Data types cannot be compared, returning nil.")
		}

		go reorganizeMemory(dl)
		go dl.updateTimestamp()
	}()
	min, max := dl.Min(), dl.Max()
	if math.IsNaN(min) || math.IsNaN(max) {
		LogWarning("Normalize: Cannot normalize due to invalid Min/Max values.")
		return nil
	}

	for i, v := range dl.Data() {
		vfloat := conv.ParseF64(v)
		dl.data[i] = (vfloat - min) / (max - min)
	}
	return dl
}

// Standardize standardizes the data in the DataList.
// Directly modifies the DataList.
func (dl *DataList) Standardize() *DataList {
	defer func() {
		r := recover()
		if r != nil {
			LogWarning("Standardize(): Data types cannot be compared, returning nil.")
		}
		dl.mu.Unlock()
		go reorganizeMemory(dl)
		go dl.updateTimestamp()
	}()
	mean := dl.Mean()
	stddev := dl.Stdev()
	dl.mu.Lock()
	for i, v := range dl.Data() {
		vfloat := conv.ParseF64(v)
		dl.data[i] = (vfloat - mean) / stddev
	}
	return dl
}

// FillNaNWithMean replaces all NaN values in the DataList with the mean value.
// Directly modifies the DataList.
func (dl *DataList) FillNaNWithMean() *DataList {
	defer func() {
		r := recover()
		if r != nil {
			LogWarning("FillNaNWithMean(): Data types cannot be compared, returning nil.")
		}
		dl.mu.Unlock()
		go reorganizeMemory(dl)
		go dl.updateTimestamp()
	}()
	dlclone := dl.Clone()
	dlNoNaN := dlclone.ClearNaNs()
	mean := dlNoNaN.Mean()
	dl.mu.Lock()
	for i, v := range dl.Data() {
		vfloat := conv.ParseF64(v)
		if math.IsNaN(vfloat) {
			dl.data[i] = mean
		} else {
			dl.data[i] = vfloat
		}
	}
	return dl
}

// MovingAverage calculates the moving average of the DataList using a specified window size.
// Returns a new DataList containing the moving average values.
func (dl *DataList) MovingAverage(windowSize int) *DataList {
	if windowSize <= 0 || windowSize > dl.Len() {
		return nil
	}
	movingAverageData := make([]float64, dl.Len()-windowSize+1)
	for i := 0; i < len(movingAverageData); i++ {
		windowSum := 0.0
		for j := 0; j < windowSize; j++ {
			windowSum += dl.Data()[i+j].(float64)
		}
		movingAverageData[i] = windowSum / float64(windowSize)
	}
	return NewDataList(movingAverageData)
}

// WeightedMovingAverage applies a weighted moving average to the DataList with a given window size.
// The weights parameter should be a slice or a DataList of the same length as the window size.
// Returns a new DataList containing the weighted moving average values.
func (dl *DataList) WeightedMovingAverage(windowSize int, weights interface{}) *DataList {
	weightsSlice, sliceLen := ProcessData(weights)
	if windowSize <= 0 || windowSize > dl.Len() || sliceLen != windowSize {
		LogWarning("DataList.WeightedMovingAverage(): Invalid window size or weights length.")
		return nil
	}

	// 計算權重總和，避免直接除以 windowSize
	weightsSum := 0.0
	for _, w := range weightsSlice {
		weightsSum += w.(float64)
	}

	movingAvgData := make([]float64, dl.Len()-windowSize+1)
	for i := 0; i < len(movingAvgData); i++ {
		window := dl.Data()[i : i+windowSize]
		sum := 0.0
		for j := 0; j < windowSize; j++ {
			sum += window[j].(float64) * weightsSlice[j].(float64)
		}
		movingAvgData[i] = sum / weightsSum // 使用權重總和
	}
	return NewDataList(movingAvgData)
}

// ExponentialSmoothing applies exponential smoothing to the DataList.
// The alpha parameter controls the smoothing factor.
// Returns a new DataList containing the smoothed values.
func (dl *DataList) ExponentialSmoothing(alpha float64) *DataList {
	if alpha < 0 || alpha > 1 {
		LogWarning("ExponentialSmoothing: Invalid alpha value.")
		return nil
	}

	smoothedData := make([]float64, dl.Len())
	smoothedData[0] = dl.Data()[0].(float64) // 使用初始值作為第一個平滑值
	for i := 1; i < dl.Len(); i++ {
		smoothedData[i] = alpha*dl.Data()[i].(float64) + (1-alpha)*smoothedData[i-1]
	}
	return NewDataList(smoothedData)
}

// DoubleExponentialSmoothing applies double exponential smoothing to the DataList.
// The alpha parameter controls the level smoothing, and the beta parameter controls the trend smoothing.
// Returns a new DataList containing the smoothed values.
func (dl *DataList) DoubleExponentialSmoothing(alpha, beta float64) *DataList {
	if alpha < 0 || alpha > 1 || beta < 0 || beta > 1 {
		LogWarning("DoubleExponentialSmoothing: Invalid alpha or beta value.")
		return nil
	}

	smoothedData := make([]float64, dl.Len())
	trend := 0.0
	level := dl.Data()[0].(float64)

	smoothedData[0] = level
	for i := 1; i < dl.Len(); i++ {
		prevLevel := level
		level = alpha*dl.Data()[i].(float64) + (1-alpha)*(level+trend)
		trend = beta*(level-prevLevel) + (1-beta)*trend
		smoothedData[i] = level + trend
	}
	return NewDataList(smoothedData)
}

// MovingStdDev calculates the moving standard deviation for the DataList using a specified window size.
func (dl *DataList) MovingStdev(windowSize int) *DataList {
	if windowSize <= 0 || windowSize > dl.Len() {
		LogWarning("MovingStdDev: Invalid window size.")
		return nil
	}
	movingStdDevData := make([]float64, dl.Len()-windowSize+1)
	for i := 0; i < len(movingStdDevData); i++ {
		window := NewDataList(dl.Data()[i : i+windowSize])
		movingStdDevData[i] = window.Stdev()
	}
	return NewDataList(movingStdDevData)
}

// Sort sorts the DataList using a mixed sorting logic.
// It handles string, numeric (including all integer and float types), and time data types.
// If sorting fails, it restores the original order.
func (dl *DataList) Sort(ascending ...bool) *DataList {
	defer func() {
		dl.mu.Unlock()
		go reorganizeMemory(dl)
	}()
	dl.mu.Lock()
	if len(dl.data) == 0 {
		LogWarning("DataList.Sort(): DataList is empty, returning.")
		return dl
	}

	// Save the original order
	originalData := make([]interface{}, len(dl.data))
	copy(originalData, dl.data)

	defer func() {
		if r := recover(); r != nil {
			LogWarning("DataList.Sort(): Sorting failed, restoring original order:", r)
			dl.data = originalData
		}
	}()

	ascendingOrder := true
	if len(ascending) > 0 {
		ascendingOrder = ascending[0]
	}
	if len(ascending) > 1 {
		LogWarning("DataList.Sort(): Too many arguments, returning.")
		return dl
	}

	// Mixed sorting
	sort.SliceStable(dl.data, func(i, j int) bool {
		v1 := dl.data[i]
		v2 := dl.data[j]

		switch v1 := v1.(type) {
		case string:
			if v2, ok := v2.(string); ok {
				return (v1 < v2) == ascendingOrder
			}
		case int, int8, int16, int32, int64:
			v1Float := ToFloat64(v1)
			if v2Float, ok := ToFloat64Safe(v2); ok {
				return (v1Float < v2Float) == ascendingOrder
			}
		case uint, uint8, uint16, uint32, uint64:
			v1Float := ToFloat64(v1)
			if v2Float, ok := ToFloat64Safe(v2); ok {
				return (v1Float < v2Float) == ascendingOrder
			}
		case float32, float64:
			v1Float := ToFloat64(v1)
			if v2Float, ok := ToFloat64Safe(v2); ok {
				return (v1Float < v2Float) == ascendingOrder
			}
		case time.Time:
			if v2, ok := v2.(time.Time); ok {
				return v1.Before(v2) == ascendingOrder
			}
		}

		// Fallback: compare as strings
		return fmt.Sprint(v1) < fmt.Sprint(v2)
	})

	go dl.updateTimestamp()
	return dl
}

// Rank assigns ranks to the elements in the DataList.
func (dl *DataList) Rank() *DataList {
	data := dl.ToF64Slice()
	ranked := make([]float64, len(data))

	// 建立一個索引來追蹤原始位置
	indexes := make([]int, len(data))
	for i := range data {
		indexes[i] = i
	}

	// 根據數據排序，並追蹤索引
	sort.Slice(indexes, func(i, j int) bool {
		return data[indexes[i]] < data[indexes[j]]
	})

	// 分配秩次，處理重複值的情況
	for i := 0; i < len(indexes); {
		sumRank := 0.0
		count := 0
		val := data[indexes[i]]
		for j := i; j < len(indexes) && data[indexes[j]] == val; j++ {
			sumRank += float64(j + 1)
			count++
		}
		avgRank := sumRank / float64(count) // 計算平均秩次
		for j := 0; j < count; j++ {
			ranked[indexes[i+j]] = avgRank
		}
		i += count
	}

	return NewDataList(ranked)
}

// Reverse reverses the order of the elements in the DataList.
func (dl *DataList) Reverse() *DataList {
	defer func() {
		dl.mu.Unlock()
		go reorganizeMemory(dl)
		go dl.updateTimestamp()
	}()
	dl.mu.Lock()
	sliceutil.Reverse(dl.data)
	return dl
}

// Upper converts all string elements in the DataList to uppercase.
func (dl *DataList) Upper() *DataList {
	defer func() {
		dl.mu.Unlock()
		go reorganizeMemory(dl)
	}()
	dl.mu.Lock()
	for i, v := range dl.data {
		if str, ok := v.(string); ok {
			dl.data[i] = strings.ToUpper(str)
		}
	}
	go dl.updateTimestamp()
	return dl
}

// Lower converts all string elements in the DataList to lowercase.
func (dl *DataList) Lower() *DataList {
	defer func() {
		dl.mu.Unlock()
		go reorganizeMemory(dl)
	}()
	dl.mu.Lock()
	for i, v := range dl.data {
		if str, ok := v.(string); ok {
			dl.data[i] = strings.ToLower(str)
		}
	}
	go dl.updateTimestamp()
	return dl
}

// Capitalize capitalizes the first letter of each string element in the DataList.
func (dl *DataList) Capitalize() *DataList {
	defer func() {
		dl.mu.Unlock()
		go reorganizeMemory(dl)
	}()
	dl.mu.Lock()
	for i, v := range dl.data {
		if str, ok := v.(string); ok {
			dl.data[i] = strings.Title(strings.ToLower(str))
		}
	}
	go dl.updateTimestamp()
	return dl
}

// ======================== Statistics ========================

// Sum calculates the sum of all elements in the DataList.
// Returns math.NaN() if the DataList is empty or if no elements can be converted to float64.
func (dl *DataList) Sum() float64 {
	if len(dl.data) == 0 {
		LogWarning("DataList.Sum(): DataList is empty.")
		return math.NaN()
	}

	sum := 0.0
	count := 0
	for _, v := range dl.data {
		vfloat, ok := ToFloat64Safe(v)
		if !ok {
			LogWarning("DataList.Sum(): Element %v cannot be converted to float64, skipping.", v)
			continue
		}
		sum += vfloat
		count++
	}

	if count == 0 {
		LogWarning("DataList.Sum(): No valid elements to compute sum.")
		return math.NaN()
	}

	return sum
}

// Max returns the maximum value in the DataList.
// Returns math.NaN() if the DataList is empty or if no elements can be converted to float64.
func (dl *DataList) Max() float64 {
	if len(dl.data) == 0 {
		LogWarning("DataList.Max(): DataList is empty.")
		return math.NaN()
	}

	var max float64
	var foundValid bool

	for _, v := range dl.data {
		vfloat, ok := ToFloat64Safe(v)
		if !ok {
			LogWarning("DataList.Max(): Element %v is not a numeric type, skipping.", v)
			continue
		}
		if !foundValid {
			max = vfloat
			foundValid = true
			continue
		}
		if vfloat > max {
			max = vfloat
		}
	}

	if !foundValid {
		LogWarning("DataList.Max(): No valid elements to compute maximum.")
		return math.NaN()
	}

	return max
}

// Min returns the minimum value in the DataList.
// Returns math.NaN() if the DataList is empty or if no elements can be converted to float64.
func (dl *DataList) Min() float64 {
	if len(dl.data) == 0 {
		LogWarning("DataList.Min(): DataList is empty.")
		return math.NaN()
	}

	var min float64
	var foundValid bool

	for _, v := range dl.data {
		vfloat, ok := ToFloat64Safe(v)
		if !ok {
			LogWarning("DataList.Min(): Element %v is not a numeric type, skipping.", v)
			continue
		}
		if !foundValid {
			min = vfloat
			foundValid = true
			continue
		}
		if vfloat < min {
			min = vfloat
		}
	}

	if !foundValid {
		LogWarning("DataList.Min(): No valid elements to compute minimum.")
		return math.NaN()
	}

	return min
}

// Mean calculates the arithmetic mean of the DataList.
// Returns math.NaN() if the DataList is empty or if no elements can be converted to float64.
func (dl *DataList) Mean() float64 {
	mean := math.NaN()
	if len(dl.data) == 0 {
		LogWarning("DataList.Mean(): DataList is empty.")
		return mean
	}

	var sum float64
	var count int
	for _, v := range dl.data {
		if val, ok := ToFloat64Safe(v); ok {
			sum += val
			count++
		} else {
			LogWarning("DataList.Mean(): Element %v is not a numeric type, skipping.", v)
			// 跳過非數字類型的元素
			continue
		}
	}

	if count == 0 {
		LogWarning("DataList.Mean(): No elements could be converted to float64.")
		return mean
	}

	mean = sum / float64(count)
	return mean
}

// WeightedMean calculates the weighted mean of the DataList using the provided weights.
// The weights parameter should be a slice or a DataList of the same length as the DataList.
// Returns math.NaN() if the DataList is empty, weights are invalid, or if no valid elements can be used.
func (dl *DataList) WeightedMean(weights interface{}) float64 {
	if dl.Len() == 0 {
		LogWarning("DataList.WeightedMean(): DataList is empty.")
		return math.NaN()
	}

	weightsSlice, sliceLen := ProcessData(weights)
	if sliceLen != dl.Len() {
		LogWarning("DataList.WeightedMean(): Weights length does not match data length.")
		return math.NaN()
	}

	totalWeight := 0.0
	weightedSum := 0.0
	validElements := 0

	for i, v := range dl.Data() {
		vfloat, ok1 := ToFloat64Safe(v)
		wfloat, ok2 := ToFloat64Safe(weightsSlice[i])
		if !ok1 {
			LogWarning(fmt.Sprintf("DataList.WeightedMean(): Data element at index %d cannot be converted to float64, skipping.", i))
			continue
		}
		if !ok2 {
			LogWarning(fmt.Sprintf("DataList.WeightedMean(): Weight at index %d cannot be converted to float64, skipping.", i))
			continue
		}
		weightedSum += vfloat * wfloat
		totalWeight += wfloat
		validElements++
	}

	if validElements == 0 {
		LogWarning("DataList.WeightedMean(): No valid elements to compute weighted mean.")
		return math.NaN()
	}

	if totalWeight == 0 {
		LogWarning("DataList.WeightedMean(): Total weight is zero, returning NaN.")
		return math.NaN()
	}

	return weightedSum / totalWeight
}

// GMean calculates the geometric mean of the DataList.
// Returns the geometric mean.
// Returns math.NaN() if the DataList is empty or if no elements can be converted to float64.
func (dl *DataList) GMean() float64 {
	gmean := math.NaN()
	if len(dl.data) == 0 {
		LogWarning("DataList.GMean(): DataList is empty.")
		return gmean
	}

	product := 1.0
	count := 0
	for _, v := range dl.data {
		if val, ok := ToFloat64Safe(v); ok {
			if val <= 0 {
				LogWarning("DataList.GMean(): Non-positive value encountered, skipping.")
				continue
			}
			product *= val
			count++
		} else {
			LogWarning("DataList.GMean(): Element %v is not a numeric type, skipping.", v)
			// 跳過無法轉換為 float64 的元素
			continue
		}
	}

	if count == 0 {
		LogWarning("DataList.GMean(): No valid elements to compute geometric mean.")
		return gmean
	}

	gmean = math.Pow(product, 1.0/float64(count))
	return gmean
}

// Median calculates the median of the DataList.
// Returns math.NaN() if the DataList is empty or if no valid elements can be used.
func (dl *DataList) Median() float64 {
	if len(dl.data) == 0 {
		LogWarning("DataList.Median(): DataList is empty.")
		return math.NaN()
	}

	// Convert data to float64 and skip invalid elements
	var validData []float64
	for _, v := range dl.data {
		vfloat, ok := ToFloat64Safe(v)
		if !ok {
			LogWarning("DataList.Median(): Element %v is not a numeric type, skipping.", v)
			continue
		}
		validData = append(validData, vfloat)
	}

	if len(validData) == 0 {
		LogWarning("DataList.Median(): No valid elements to compute median.")
		return math.NaN()
	}

	// Sort the valid data
	sort.Float64s(validData)

	mid := len(validData) / 2

	if len(validData)%2 == 0 {
		// Even number of elements, return the average of the middle two
		mid1 := validData[mid-1]
		mid2 := validData[mid]
		return (mid1 + mid2) / 2
	}

	// Odd number of elements, return the middle one
	return validData[mid]
}

// Mode calculates the mode of the DataList.
// Only works with numeric data types.
// Returns math.NaN() if the DataList is empty or if no valid elements can be used.
func (dl *DataList) Mode() float64 {
	if len(dl.data) == 0 {
		LogWarning("DataList.Mode(): DataList is empty.")
		return math.NaN()
	}

	freqMap := make(map[float64]int)
	for _, v := range dl.data {
		vfloat, ok := ToFloat64Safe(v)
		if !ok {
			LogWarning("DataList.Mode(): Element %v is not a numeric type, skipping.", v)
			continue
		}
		freqMap[vfloat]++
	}

	if len(freqMap) == 0 {
		LogWarning("DataList.Mode(): No valid elements to compute mode.")
		return math.NaN()
	}

	var mode float64
	maxFreq := 0
	for k, v := range freqMap {
		if v > maxFreq {
			mode = k
			maxFreq = v
		}
	}

	return mode
}

// MAD calculates the mean absolute deviation of the DataList.
// Returns math.NaN() if the DataList is empty or if no valid elements can be used.
func (dl *DataList) MAD() float64 {
	if len(dl.data) == 0 {
		LogWarning("DataList.MAD(): DataList is empty.")
		return math.NaN()
	}

	// Calculate the median using the modified Median function
	median := dl.Median()
	if math.IsNaN(median) {
		LogWarning("DataList.MAD(): Median calculation failed.")
		return math.NaN()
	}

	// Calculate the mean absolute deviation
	var sum float64
	var count int
	for _, v := range dl.data {
		val, ok := ToFloat64Safe(v)
		if !ok {
			LogWarning("DataList.MAD(): Element %v is not a numeric type, skipping.", v)
			continue
		}
		sum += math.Abs(val - median)
		count++
	}

	if count == 0 {
		LogWarning("DataList.MAD(): No valid elements to compute MAD.")
		return math.NaN()
	}

	return sum / float64(count)
}

// Stdev calculates the standard deviation (sample) of the DataList.
// Returns math.NaN() if the DataList is empty or if no valid elements can be used.
func (dl *DataList) Stdev() float64 {
	if len(dl.data) == 0 {
		LogWarning("DataList.Stdev(): DataList is empty.")
		return math.NaN()
	}

	variance := dl.Var()
	if math.IsNaN(variance) {
		LogWarning("DataList.Stdev(): Variance calculation failed.")
		return math.NaN()
	}

	return math.Sqrt(variance)
}

// StdevP calculates the standard deviation (population) of the DataList.
// Returns math.NaN() if the DataList is empty or if no valid elements can be used.
func (dl *DataList) StdevP() float64 {
	if len(dl.data) == 0 {
		LogWarning("DataList.StdevP(): DataList is empty.")
		return math.NaN()
	}

	varianceP := dl.VarP()
	if math.IsNaN(varianceP) {
		LogWarning("DataList.StdevP(): Variance calculation failed.")
		return math.NaN()
	}

	return math.Sqrt(varianceP)
}

// Var calculates the variance (sample variance) of the DataList.
// Returns math.NaN() if the DataList is empty or if not enough valid elements are available.
func (dl *DataList) Var() float64 {
	if len(dl.data) == 0 {
		LogWarning("DataList.Var(): DataList is empty.")
		return math.NaN()
	}

	var sum float64
	var count int

	// First pass: calculate the mean of valid elements
	for _, v := range dl.data {
		xi, ok := ToFloat64Safe(v)
		if !ok {
			LogWarning("DataList.Var(): Element %v is not a numeric type, skipping.", v)
			continue
		}
		sum += xi
		count++
	}

	if count < 2 {
		LogWarning("DataList.Var(): Not enough valid elements to compute variance.")
		return math.NaN()
	}

	mean := sum / float64(count)

	// Second pass: calculate the variance
	var numerator float64
	for _, v := range dl.data {
		xi, ok := ToFloat64Safe(v)
		if !ok {
			// Already logged, skip this element
			continue
		}
		numerator += math.Pow(xi-mean, 2)
	}

	denominator := float64(count - 1)
	variance := numerator / denominator

	return variance
}

// VarP calculates the variance (population variance) of the DataList.
// Returns math.NaN() if the DataList is empty or if no valid elements can be used.
func (dl *DataList) VarP() float64 {
	if len(dl.data) == 0 {
		LogWarning("DataList.VarP(): DataList is empty.")
		return math.NaN()
	}

	// First pass: compute mean over valid elements
	var sum float64
	var count int

	for _, v := range dl.data {
		xi, ok := ToFloat64Safe(v)
		if !ok {
			LogWarning("DataList.VarP(): Element %v is not a numeric type, skipping.", v)
			continue
		}
		sum += xi
		count++
	}

	if count == 0 {
		LogWarning("DataList.VarP(): No valid elements to compute variance.")
		return math.NaN()
	}

	mean := sum / float64(count)

	// Second pass: compute variance
	var numerator float64
	for _, v := range dl.data {
		xi, ok := ToFloat64Safe(v)
		if !ok {
			// Already logged, skip this element
			continue
		}
		numerator += math.Pow(xi-mean, 2)
	}

	variance := numerator / float64(count) // Population variance divides by N

	return variance
}

// Range calculates the range of the DataList.
// Returns math.NaN() if the DataList is empty or if Max or Min cannot be calculated.
func (dl *DataList) Range() float64 {
	if len(dl.data) == 0 {
		LogWarning("DataList.Range(): DataList is empty.")
		return math.NaN()
	}

	max := dl.Max()
	min := dl.Min()

	if math.IsNaN(max) || math.IsNaN(min) {
		LogWarning("DataList.Range(): Max or Min calculation failed.")
		return math.NaN()
	}

	return max - min
}

// Quartile calculates the quartile based on the input value (1 to 3).
// 1 corresponds to the first quartile (Q1), 2 to the median (Q2), and 3 to the third quartile (Q3).
// This implementation uses percentiles to compute quartiles.
func (dl *DataList) Quartile(q int) float64 {
	if len(dl.data) == 0 {
		LogWarning("DataList.Quartile(): DataList is empty.")
		return math.NaN()
	}
	if q < 1 || q > 3 {
		LogWarning("DataList.Quartile(): Invalid quartile value.")
		return math.NaN()
	}

	// Convert the DataList to a slice of float64 for numeric operations, skipping invalid elements
	var numericData []float64
	for _, v := range dl.data {
		vfloat, ok := ToFloat64Safe(v)
		if !ok {
			LogWarning("DataList.Quartile(): Element %v is not a numeric type, skipping.", v)
			continue
		}
		numericData = append(numericData, vfloat)
	}

	if len(numericData) == 0 {
		LogWarning("DataList.Quartile(): No valid elements to compute quartile.")
		return math.NaN()
	}

	// Sort the data
	sort.Float64s(numericData)

	n := len(numericData)
	var p float64

	// Set the percentile based on the quartile
	switch q {
	case 1:
		p = 0.25
	case 2:
		p = 0.5
	case 3:
		p = 0.75
	}

	// Calculate the position using the percentile
	pos := p * float64(n+1)

	// Adjust position if it is outside the range
	if pos < 1.0 {
		pos = 1.0
	} else if pos > float64(n) {
		pos = float64(n)
	}

	// Convert position to indices
	lowerIndex := int(math.Floor(pos)) - 1 // Subtract 1 for zero-based index
	upperIndex := int(math.Ceil(pos)) - 1

	// Ensure indices are within bounds
	if lowerIndex < 0 {
		lowerIndex = 0
	}
	if upperIndex >= n {
		upperIndex = n - 1
	}

	// Handle the case where the position is exactly an integer
	if lowerIndex == upperIndex {
		return numericData[lowerIndex]
	}

	// Interpolate between the lower and upper bounds
	lowerValue := numericData[lowerIndex]
	upperValue := numericData[upperIndex]
	fraction := pos - math.Floor(pos)

	quartile := lowerValue + fraction*(upperValue-lowerValue)

	return quartile
}

// IQR calculates the interquartile range of the DataList.
// Returns math.NaN() if the DataList is empty or if Q1 or Q3 cannot be calculated.
// Returns the interquartile range (Q3 - Q1) as a float64.
func (dl *DataList) IQR() float64 {
	if len(dl.data) == 0 {
		LogWarning("DataList.IQR(): DataList is empty.")
		return math.NaN()
	}

	q1 := dl.Quartile(1)
	q3 := dl.Quartile(3)

	if math.IsNaN(q1) || math.IsNaN(q3) {
		LogWarning("DataList.IQR(): Q1 or Q3 calculation failed.")
		return math.NaN()
	}

	return q3 - q1
}

// Percentile calculates the percentile based on the input value (0 to 100).
// Returns the percentile value as float64, or math.NaN() if the DataList is empty or invalid percentile is provided.
func (dl *DataList) Percentile(p float64) float64 {
	if len(dl.data) == 0 {
		LogWarning("DataList.Percentile(): DataList is empty.")
		return math.NaN()
	}
	if p < 0 || p > 100 {
		LogWarning("DataList.Percentile(): Invalid percentile value.")
		return math.NaN()
	}

	// Convert the DataList to a slice of float64 for numeric operations, skipping invalid elements
	var numericData []float64
	for _, v := range dl.data {
		vfloat, ok := ToFloat64Safe(v)
		if !ok {
			LogWarning("DataList.Percentile(): Element %v cannot be converted to float64, skipping.", v)
			continue
		}
		numericData = append(numericData, vfloat)
	}

	if len(numericData) == 0 {
		LogWarning("DataList.Percentile(): No valid elements to compute percentile.")
		return math.NaN()
	}

	// Sort the data
	sort.Float64s(numericData)

	n := len(numericData)
	if n == 1 {
		return numericData[0]
	}

	// Calculate the position using R's type=7 method
	p /= 100.0
	h := p*(float64(n)-1) + 1

	// Adjust position for zero-based index
	h -= 1

	lowerIndex := int(math.Floor(h))
	upperIndex := int(math.Ceil(h))

	// Ensure indices are within bounds
	if lowerIndex < 0 {
		lowerIndex = 0
	}
	if upperIndex >= n {
		upperIndex = n - 1
	}

	lowerValue := numericData[lowerIndex]
	upperValue := numericData[upperIndex]

	if lowerIndex == upperIndex {
		return lowerValue
	}

	// Interpolate between the lower and upper values
	fraction := h - float64(lowerIndex)
	percentileValue := lowerValue + fraction*(upperValue-lowerValue)

	return percentileValue
}

// Difference calculates the differences between adjacent elements in the DataList.
func (dl *DataList) Difference() *DataList {
	defer func() {
		r := recover()
		if r != nil {
			LogWarning("DataList.Difference(): Data types cannot be compared.")
		}
	}()
	if dl.Len() < 2 {
		LogWarning("DataList.Difference(): DataList is too short to calculate differences, returning nil.")
		return nil
	}

	differenceData := make([]float64, dl.Len()-1)
	for i := 1; i < dl.Len(); i++ {
		differenceData[i-1] = conv.ParseF64(dl.Data()[i]) - conv.ParseF64(dl.Data()[i-1])
	}

	return NewDataList(differenceData)
}

// ======================== Comparison ========================

// IsEqualTo checks if the data of the DataList is equal to another DataList.
func (dl *DataList) IsEqualTo(anotherDl *DataList) bool {
	if dl.Len() != anotherDl.Len() {
		return false
	}

	for i, v := range dl.Data() {
		if v != anotherDl.Data()[i] {
			return false
		}
	}

	return true
}

// IsTheSameAs checks if the DataList is fully the same as another DataList.
// It checks for equality in name, data, creation timestamp, and last modified timestamp.
func (dl *DataList) IsTheSameAs(anotherDl *DataList) bool {
	if dl.Len() != anotherDl.Len() {
		return false
	}

	for i, v := range dl.Data() {
		if v != anotherDl.Data()[i] {
			return false
		}
	}

	if dl.GetName() != anotherDl.GetName() {
		return false
	}

	dl.mu.Lock()
	anotherDl.mu.Lock()
	if dl.creationTimestamp != anotherDl.creationTimestamp || dl.lastModifiedTimestamp.Load() != anotherDl.lastModifiedTimestamp.Load() {
		return false
	}
	dl.mu.Unlock()
	anotherDl.mu.Unlock()

	return true
}

// ======================== Conversion ========================

// ParseNumbers attempts to parse all string elements in the DataList to numeric types.
// If parsing fails, the element is left unchanged.
func (dl *DataList) ParseNumbers() *DataList {
	defer func() {
		dl.mu.Unlock()
		go reorganizeMemory(dl)
		go dl.updateTimestamp()
	}()
	dl.mu.Lock()
	for i, v := range dl.data {
		func() {
			defer func() {
				if r := recover(); r != nil {
					LogWarning("DataList.ParseNumbers(): Failed to parse %v to float64: %v, the element left unchanged.", v, r)
				}
			}()

			dl.data[i] = conv.ParseF64(v)
		}()
	}
	return dl
}

// ParseStrings converts all elements in the DataList to strings.
func (dl *DataList) ParseStrings() *DataList {
	defer func() {
		dl.mu.Unlock()
		go reorganizeMemory(dl)
		go dl.updateTimestamp()
	}()
	dl.mu.Lock()
	for i, v := range dl.data {
		func() {
			defer func() {
				if r := recover(); r != nil {
					LogWarning("DataList.ParseStrings(): Failed to convert %v to string: %v, the element left unchanged.", v, r)
				}
			}()

			dl.data[i] = conv.ToString(v)
		}()
	}
	return dl
}

// ToF64Slice converts the DataList to a float64 slice.
// Returns the float64 slice.
// Returns nil if the DataList is empty.
// ToF64Slice converts the DataList to a float64 slice.
func (dl *DataList) ToF64Slice() []float64 {
	defer func() {
		dl.mu.Unlock()
		go reorganizeMemory(dl)
	}()
	dl.mu.Lock()
	if len(dl.data) == 0 {
		LogWarning("DataList.ToF64Slice(): DataList is empty, returning nil.")
		return nil
	}

	floatData := make([]float64, len(dl.data))
	for i, v := range dl.data {
		floatData[i] = ToFloat64(v)
	}

	return floatData
}

// ToStringSlice converts the DataList to a string slice.
// Returns the string slice.
// Returns nil if the DataList is empty.
func (dl *DataList) ToStringSlice() []string {
	defer func() {
		dl.mu.Unlock()
		go reorganizeMemory(dl)
	}()
	dl.mu.Lock()
	if len(dl.data) == 0 {
		LogWarning("DataList.ToStringSlice(): DataList is empty, returning nil.")
		return nil
	}

	stringData := make([]string, len(dl.data))
	for i, v := range dl.data {
		stringData[i] = conv.ToString(v)
	}

	return stringData
}

// ======================== Timestamp ========================

// GetCreationTimestamp returns the creation time of the DataList in Unix timestamp.
func (dl *DataList) GetCreationTimestamp() int64 {
	return dl.creationTimestamp
}

// GetLastModifiedTimestamp returns the last updated time of the DataList in Unix timestamp.
func (dl *DataList) GetLastModifiedTimestamp() int64 {
	return dl.lastModifiedTimestamp.Load()
}

// updateTimestamp updates the lastModifiedTimestamp to the current Unix time.
func (dl *DataList) updateTimestamp() {
	now := time.Now().Unix()
	oldTimestamp := dl.lastModifiedTimestamp.Load()

	if oldTimestamp < now {
		dl.lastModifiedTimestamp.Store(now)
	}
}

// ======================== Name ========================

// GetName returns the name of the DataList.
func (dl *DataList) GetName() string {
	return dl.name
}

// SetName sets the name of the DataList.
func (dl *DataList) SetName(newName string) *DataList {
	nm := getNameManager()

	// 鎖定 DataList 以確保名稱設置過程的同步性
	dl.mu.Lock()
	defer dl.mu.Unlock()

	// 檢查並註冊新名稱
	if err := nm.registerName(newName); err != nil {
		LogWarning("DataList.SetName(): %v, remaining the old name.", err)
		return dl
	}

	// 解除舊名稱的註冊（如果已有名稱）
	if dl.name != "" {
		nm.unregisterName(dl.name)
	}

	dl.name = newName
	go dl.updateTimestamp()
	return dl
}
