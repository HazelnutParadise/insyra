package insyra

import (
	"cmp"
	"fmt"
	"math"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/HazelnutParadise/Go-Utils/asyncutil"
	"github.com/HazelnutParadise/Go-Utils/conv"
	"github.com/HazelnutParadise/Go-Utils/sliceutil"
	"github.com/HazelnutParadise/insyra/internal/utils"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// DataList is a generic dynamic data list.
type DataList struct {
	data                  []any
	name                  string
	creationTimestamp     int64
	lastModifiedTimestamp atomic.Int64
	// mu                    sync.Mutex

	// AtomicDo support
	initOnce sync.Once
	cmdCh    chan func()
	closed   atomic.Bool
}

// Data returns a copy of the data stored in the DataList.
// This prevents external modification of the internal data (Copy-on-Read).
func (dl *DataList) Data() []any {
	var result []any
	dl.AtomicDo(func(dl *DataList) {
		result = make([]any, len(dl.data))
		copy(result, dl.data)
	})
	return result
}

// flattenWithNilSupport flattens a slice of any values, properly handling nil values
// Only flattens slices, not arrays
func flattenWithNilSupport(values []any) []any {
	var result []any

	for _, value := range values {
		if value == nil {
			result = append(result, nil)
			continue
		}

		// Use reflection to check if the value is a slice (but not array)
		rv := reflect.ValueOf(value)
		if rv.Kind() == reflect.Slice {
			// Recursively flatten slice elements
			sliceLen := rv.Len()
			subSlice := make([]any, sliceLen)
			for i := range sliceLen {
				subSlice[i] = rv.Index(i).Interface()
			}
			result = append(result, flattenWithNilSupport(subSlice)...)
		} else {
			result = append(result, value)
		}
	}

	return result
}

// NewDataList creates a new DataList, supporting both slice and variadic inputs,
// and flattens the input before storing it.
func NewDataList(values ...any) *DataList {
	// Use custom flatten function that properly handles nil values
	flatData := flattenWithNilSupport(values)
	// LogDebug("DataList", "NewDataList", "Flattened data: %v", flatData)

	continuousMemData := make([]any, len(flatData))
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
func (dl *DataList) Append(values ...any) {
	dl.AtomicDo(func(dl *DataList) {
		// Append data and update timestamp
		dl.data = append(dl.data, values...)
		go dl.updateTimestamp()
	})
}

// Get retrieves the value at the specified index in the DataList.
// Supports negative indexing.
// Returns nil if the index is out of bounds.
// Returns the value at the specified index.
func (dl *DataList) Get(index int) any {
	var result any
	dl.AtomicDo(func(dl *DataList) {
		// 支持負索引
		if index < 0 {
			index += len(dl.data)
		}
		if index < 0 || index >= len(dl.data) {
			LogWarning("DataList", "Get", "Index out of bounds, returning nil.")
			result = nil
			return
		}
		result = dl.data[index]
	})
	return result
}

// Clone creates a deep copy of the DataList.
func (dl *DataList) Clone() *DataList {
	var newDL *DataList
	dl.AtomicDo(func(dl *DataList) {
		newDL = NewDataList(dl.data)
		newDL.SetName(dl.name)
	})
	return newDL
}

// Count returns the number of occurrences of the specified value in the DataList.
func (dl *DataList) Count(value any) int {
	found := dl.FindAll(value)
	if found == nil {
		return 0
	}
	return len(found)
}

// Counter returns a map of the number of occurrences of each value in the DataList.
func (dl *DataList) Counter() map[any]int {
	counter := make(map[any]int)
	dl.AtomicDo(func(dl *DataList) {
		for _, value := range dl.data {
			counter[value]++
		}
	})
	return counter
}

// Update replaces the value at the specified index with the new value.
// Returns the DataList to support chaining calls.
func (dl *DataList) Update(index int, newValue any) *DataList {
	dl.AtomicDo(func(dl *DataList) {
		if index < 0 {
			index += len(dl.data)
		}
		if index < 0 || index >= len(dl.data) {
			LogWarning("DataList", "ReplaceAtIndex", "Index %d out of bounds", index)
		}
		dl.data[index] = newValue
		go dl.updateTimestamp()
	})
	return dl
}

// InsertAt inserts a value at the specified index in the DataList.
// If the index is out of bounds, the value is appended to the end of the list.
// Returns the DataList to support chaining calls.
func (dl *DataList) InsertAt(index int, value any) *DataList {
	dl.AtomicDo(func(dl *DataList) {
		// Handle negative index
		if index < 0 {
			index += len(dl.data) + 1
		}

		// If index is out of bounds, append the value to the end
		if index < 0 || index > len(dl.data) {
			LogWarning("DataList", "InsertAt", "Index out of bounds, appending value to the end.")
			dl.data = append(dl.data, value)
		} else {
			var err error
			dl.data, err = sliceutil.InsertAt(dl.data, index, value)
			if err != nil {
				LogWarning("DataList", "InsertAt", "Failed to insert value at index: %v", err)
				return
			}
		}

		go dl.updateTimestamp()
	})
	return dl
}

// FindFirst returns the index of the first occurrence of the specified value in the DataList.
// If the value is not found, it returns nil.
func (dl *DataList) FindFirst(value any) any {
	var result any
	dl.AtomicDo(func(dl *DataList) {
		for i, v := range dl.data {
			if v == value {
				result = i
				return
			}
		}
		LogWarning("DataList", "FindFirst", "Value not found, returning nil.")
		result = nil
	})
	return result
}

// FindLast returns the index of the last occurrence of the specified value in the DataList.
// If the value is not found, it returns nil.
func (dl *DataList) FindLast(value any) any {
	var result any
	dl.AtomicDo(func(dl *DataList) {
		for i := len(dl.data) - 1; i >= 0; i-- {
			if dl.data[i] == value {
				result = i
				return
			}
		}
		LogWarning("DataList", "FindLast", "Value not found, returning nil.")
		result = nil
	})
	return result
}

// FindAll returns a slice of all the indices where the specified value is found in the DataList using parallel processing.
// If the value is not found, it returns an empty slice.
func (dl *DataList) FindAll(value any) []int {
	var indices []int
	dl.AtomicDo(func(dl *DataList) {
		length := len(dl.data)
		if length == 0 {
			LogWarning("DataList", "FindAll", "DataList is empty, returning an empty slice.")
			indices = []int{}
			return
		}

		for i, v := range dl.data {
			if v == value {
				indices = append(indices, i)
			}
		}
	})
	return indices
}

// Filter filters the DataList based on a custom filter function provided by the user.
// The filter function should return true for elements that should be included in the result.
func (dl *DataList) Filter(filterFunc func(any) bool) *DataList {
	var filteredData []any
	dl.AtomicDo(func(dl *DataList) {
		filteredData = []any{}

		for _, v := range dl.data {
			if filterFunc(v) {
				filteredData = append(filteredData, v)
			}
		}
	})
	return NewDataList(filteredData...)
}

// ReplaceFirst replaces the first occurrence of oldValue with newValue.
func (dl *DataList) ReplaceFirst(oldValue, newValue any) {
	dl.AtomicDo(func(dl *DataList) {
		for i, v := range dl.data {
			if v == oldValue {
				dl.data[i] = newValue
				go dl.updateTimestamp()
				return
			}
		}
		LogWarning("DataList", "ReplaceFirst", "value not found.")
	})
}

// ReplaceLast replaces the last occurrence of oldValue with newValue.
func (dl *DataList) ReplaceLast(oldValue, newValue any) {
	dl.AtomicDo(func(dl *DataList) {
		for i := len(dl.data) - 1; i >= 0; i-- {
			if dl.data[i] == oldValue {
				dl.data[i] = newValue
				go dl.updateTimestamp()
				return
			}
		}
		LogWarning("DataList", "ReplaceLast", "value not found.")
	})
}

// ReplaceAll replaces all occurrences of oldValue with newValue in the DataList.
// If oldValue is not found, no changes are made.
func (dl *DataList) ReplaceAll(oldValue, newValue any) {
	dl.AtomicDo(func(dl *DataList) {
		length := len(dl.data)
		if length == 0 {
			LogWarning("DataList", "ReplaceAll", "DataList is empty, no replacements made.")
			return
		}

		// 單線程處理資料替換
		for i, v := range dl.data {
			if v == oldValue {
				dl.data[i] = newValue
			}
		}

		go dl.updateTimestamp()
	})
}

// ReplaceOutliers replaces outliers in the DataList with the specified replacement value (e.g., mean, median).
//
// Parameters:
// - stdDevs: Number of standard deviations to use as threshold (e.g., 2.0 means values beyond ±2σ from the mean will be replaced)
// - replacement: Value to replace outliers with
func (dl *DataList) ReplaceOutliers(stdDevs float64, replacement float64) *DataList {
	dl.AtomicDo(func(dl *DataList) {
		mean := dl.Mean()
		stddev := dl.Stdev()
		threshold := stdDevs * stddev

		for i, v := range dl.data {
			val := conv.ParseF64(v)
			if math.Abs(val-mean) > threshold {
				dl.data[i] = replacement
			}
		}
	})
	return dl
}

// ReplaceNaNsWith replaces all NaN values in the DataList with the specified value.
func (dl *DataList) ReplaceNaNsWith(value any) *DataList {
	defer func() {
		go dl.updateTimestamp()
	}()
	dl.AtomicDo(func(dl *DataList) {
		for i, v := range dl.data {
			if val, ok := v.(float64); ok && math.IsNaN(val) {
				dl.data[i] = value
			}
		}
	})
	return dl
}

// ReplaceNilsWith replaces all nil values in the DataList with the specified value.
func (dl *DataList) ReplaceNilsWith(value any) *DataList {
	defer func() {
		go dl.updateTimestamp()
	}()
	dl.AtomicDo(func(dl *DataList) {
		for i, v := range dl.data {
			if v == nil {
				dl.data[i] = value
			}
		}
	})
	return dl
}

// ReplaceNaNsAndNilsWith replaces all NaN and nil values in the DataList with the specified value.
func (dl *DataList) ReplaceNaNsAndNilsWith(value any) *DataList {
	defer func() {
		go dl.updateTimestamp()
	}()
	dl.AtomicDo(func(dl *DataList) {
		for i, v := range dl.data {
			if v == nil {
				dl.data[i] = value
			} else if val, ok := v.(float64); ok && math.IsNaN(val) {
				dl.data[i] = value
			}
		}
	})
	return dl
}

// Pop removes and returns the last element from the DataList.
// Returns the last element.
// Returns nil if the DataList is empty.
func (dl *DataList) Pop() any {
	var result any
	dl.AtomicDo(func(dl *DataList) {
		n, err := sliceutil.Drt_PopFrom(&dl.data)
		if err != nil {
			LogWarning("DataList", "Pop", "DataList is empty, returning nil.")
			result = nil
			return
		}
		result = n
		go dl.updateTimestamp()
	})
	return result
}

// Drop removes the element at the specified index from the DataList and updates the timestamp.
// Returns an error if the index is out of bounds.
func (dl *DataList) Drop(index int) *DataList {
	dl.AtomicDo(func(dl *DataList) {
		if index < 0 {
			index += len(dl.data)
		}
		if index >= len(dl.data) {
			LogWarning("DataList", "Drop", "Index out of bounds, returning")
			return
		}
		dl.data = append(dl.data[:index], dl.data[index+1:]...)
		go dl.updateTimestamp()
	})
	return dl
}

// DropAll removes all occurrences of the specified values from the DataList.
// Supports multiple values to drop.
func (dl *DataList) DropAll(toDrop ...any) *DataList {
	dl.AtomicDo(func(dl *DataList) {
		length := len(dl.data)
		if length == 0 {
			return
		}

		// 決定要開多少個線程，但不超過資料長度
		numGoroutines := runtime.NumCPU()
		if numGoroutines == 0 {
			numGoroutines = 1
		}
		if numGoroutines > length {
			numGoroutines = length
		}

		chunkSize := length / numGoroutines
		remainder := length % numGoroutines

		// 儲存所有的 Awaitable
		var awaitables []*asyncutil.Awaitable

		// 啟動 Awaitables 處理每個部分
		start := 0
		for i := 0; i < numGoroutines; i++ {
			// 計算當前塊的大小，前 remainder 個塊多分配一個元素
			currentChunkSize := chunkSize
			if i < remainder {
				currentChunkSize++
			}

			end := start + currentChunkSize

			// 確保不會超出邊界
			if end > length {
				end = length
			}

			// 只有當 start < end 時才創建任務，避免空塊
			if start < end {
				awaitable := asyncutil.Async(func(dataChunk []any) []any {
					var result []any
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

			start = end
		}

		// 收集所有結果並合併
		var finalResult []any
		for _, awaitable := range awaitables {
			results, err := awaitable.Await()
			if err != nil {
				LogWarning("DataList", "DropAll", "Error in async task: %v", err)
				continue
			}

			if len(results) > 0 {
				finalResult = append(finalResult, results[0].([]any)...)
			}
		}

		// 更新 DataList
		dl.data = finalResult
		go dl.updateTimestamp()
	})
	return dl
}

// DropIfContains removes all elements from the DataList that contain the specified value.
func (dl *DataList) DropIfContains(value any) *DataList {
	dl.AtomicDo(func(dl *DataList) {
		// 創建一個臨時切片存放保留的元素
		var newData []any

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
	})
	return dl
}

// Clear removes all elements from the DataList and updates the timestamp.
func (dl *DataList) Clear() *DataList {
	dl.AtomicDo(func(dl *DataList) {
		dl.data = []any{}
		go dl.updateTimestamp()
	})
	return dl
}

func (dl *DataList) Len() int {
	var l int
	dl.AtomicDo(func(dl *DataList) {
		l = len(dl.data)
	})
	return l
}

// ClearStrings removes all string elements from the DataList and updates the timestamp.
func (dl *DataList) ClearStrings() *DataList {
	dl.AtomicDo(func(dl *DataList) {
		length := len(dl.data)
		if length == 0 {
			return
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

		for i := range numGoroutines {
			start := i * chunkSize
			end := start + chunkSize
			if end > length {
				end = length
			}

			task := asyncutil.Task{
				ID: fmt.Sprintf("Task-%d", i),
				Fn: func(dataChunk []any) []any {
					var result []any
					for _, v := range dataChunk {
						if _, ok := v.(string); !ok {
							result = append(result, v)
						}
					}
					return result
				},
				Args: []any{dl.data[start:end]},
			}

			tasks = append(tasks, task)
		}

		// 使用 ParallelProcess 進行平行處理
		taskResults := asyncutil.ParallelProcess(tasks)

		// 合併所有的結果
		var finalResult []any
		for _, taskResult := range taskResults {
			finalResult = append(finalResult, taskResult.Results[0].([]any)...)
		}

		// 更新 DataList
		dl.data = finalResult
		go dl.updateTimestamp()
	})
	return dl
} // ++++ 此處之後尚未提升性能 ++++

// ClearNumbers removes all numeric elements (int, float, etc.) from the DataList and updates the timestamp.
func (dl *DataList) ClearNumbers() *DataList {
	dl.AtomicDo(func(dl *DataList) {
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
	})
	return dl
}

// ClearNaNs removes all NaN values from the DataList and updates the timestamp.
func (dl *DataList) ClearNaNs() *DataList {
	defer func() {
		go dl.updateTimestamp()
	}()
	dl.AtomicDo(func(dl *DataList) {
		for i := len(dl.data) - 1; i >= 0; i-- {
			if v, ok := dl.data[i].(float64); ok && math.IsNaN(v) {
				dl.data = append(dl.data[:i], dl.data[i+1:]...)
			}
		}
	})
	return dl
}

// ClearNils removes all nil values from the DataList and updates the timestamp.
func (dl *DataList) ClearNils() *DataList {
	defer func() {
		go dl.updateTimestamp()
	}()
	dl.AtomicDo(func(dl *DataList) {
		for i := len(dl.data) - 1; i >= 0; i-- {
			if dl.data[i] == nil {
				dl.data = append(dl.data[:i], dl.data[i+1:]...)
			}
		}
	})
	return dl
}

// ClearNilsAndNaNs removes all nil and NaN values from the DataList and updates the timestamp.
func (dl *DataList) ClearNilsAndNaNs() *DataList {
	defer func() {
		go dl.updateTimestamp()
	}()
	dl.AtomicDo(func(dl *DataList) {
		dl.ClearNaNs().ClearNils()
	})
	return dl
}

// ClearOutliers removes values from the DataList that are outside the specified number of standard deviations.
// This method modifies the original DataList and returns it.
func (dl *DataList) ClearOutliers(stdDevs float64) *DataList {
	defer func() {
		r := recover()
		if r != nil {
			LogWarning("DataList", "ClearOutliers", "Data types cannot be compared")
		}
		go dl.updateTimestamp()
	}()
	dl.AtomicDo(func(dl *DataList) {
		mean := dl.Mean()
		stddev := dl.Stdev()
		threshold := stdDevs * stddev

		// 打印調試信息，確保計算值與 R 相同
		LogDebug("DataList", "ClearOutliers", "Mean: %f", mean)
		LogDebug("DataList", "ClearOutliers", "Standard Deviation: %f", stddev)
		LogDebug("DataList", "ClearOutliers", "Threshold: %f", threshold)

		for i := len(dl.data) - 1; i >= 0; i-- {
			val := conv.ParseF64(dl.data[i])
			LogDebug("DataList", "ClearOutliers", "Checking value: %f", val) // 打印每個檢查的值
			if math.Abs(val-mean) > threshold {
				LogDebug("DataList", "ClearOutliers", "Removing outlier: %f", val) // 打印要移除的異常值
				dl.data = append(dl.data[:i], dl.data[i+1:]...)
			}
		}
	})
	return dl
}

// Normalize normalizes the data in the DataList, skipping NaN values.
// Directly modifies the DataList.
func (dl *DataList) Normalize() *DataList {
	defer func() {
		r := recover()
		if r != nil {
			LogWarning("DataList", "Normalize", "Data types cannot be compared, returning nil")
		}

		go dl.updateTimestamp()
	}()
	isFailed := false
	dl.AtomicDo(func(dl *DataList) {
		min, max := dl.Min(), dl.Max()
		if math.IsNaN(min) || math.IsNaN(max) {
			LogWarning("DataList", "Normalize", "Cannot normalize due to invalid Min/Max values")
			isFailed = true
			return
		}

		for i, v := range dl.data {
			vfloat := conv.ParseF64(v)
			dl.data[i] = (vfloat - min) / (max - min)
		}
	})
	if isFailed {
		return nil
	}
	return dl
}

// Standardize standardizes the data in the DataList.
// Directly modifies the DataList.
func (dl *DataList) Standardize() *DataList {
	defer func() {
		r := recover()
		if r != nil {
			LogWarning("DataList", "Standardize", "Data types cannot be compared, returning nil")
		}
	}()
	dl.AtomicDo(func(dl *DataList) {
		mean := dl.Mean()
		stddev := dl.Stdev()
		for i, v := range dl.data {
			vfloat := conv.ParseF64(v)
			dl.data[i] = (vfloat - mean) / stddev
		}
		go dl.updateTimestamp()
	})
	return dl
}

// FillNaNWithMean replaces all NaN values in the DataList with the mean value.
// Directly modifies the DataList.
func (dl *DataList) FillNaNWithMean() *DataList {
	defer func() {
		r := recover()
		if r != nil {
			LogWarning("DataList", "FillNaNWithMean", "Data types cannot be compared, returning nil")
		}
	}()
	dl.AtomicDo(func(dl *DataList) {
		dlclone := dl.Clone()
		dlNoNaN := dlclone.ClearNaNs()
		mean := dlNoNaN.Mean()
		for i, v := range dl.data {
			vfloat := conv.ParseF64(v)
			if math.IsNaN(vfloat) {
				dl.data[i] = mean
			} else {
				dl.data[i] = vfloat
			}
		}
		go dl.updateTimestamp()
	})
	return dl
}

// MovingAverage calculates the moving average of the DataList using a specified window size.
// Returns a new DataList containing the moving average values.
func (dl *DataList) MovingAverage(windowSize int) *DataList {
	var movingAverageData []float64
	isFailed := false
	dl.AtomicDo(func(dl *DataList) {
		if windowSize <= 0 || windowSize > dl.Len() {
			LogWarning("DataList", "MovingAverage", "Invalid window size")
			isFailed = true
			return
		}
		movingAverageData = make([]float64, len(dl.data)-windowSize+1)
		for i := range movingAverageData {
			windowSum := 0.0
			for j := range windowSize {
				windowSum += dl.data[i+j].(float64)
			}
			movingAverageData[i] = windowSum / float64(windowSize)
		}
	})
	if isFailed {
		return nil
	}
	return NewDataList(movingAverageData)
}

// WeightedMovingAverage applies a weighted moving average to the DataList with a given window size.
// The weights parameter should be a slice or a DataList of the same length as the window size.
// Returns a new DataList containing the weighted moving average values.
func (dl *DataList) WeightedMovingAverage(windowSize int, weights any) *DataList {
	weightsSlice, sliceLen := ProcessData(weights)
	var movingAvgData []float64
	isFailed := false
	dl.AtomicDo(func(dl *DataList) {
		if windowSize <= 0 || windowSize > dl.Len() || sliceLen != windowSize {
			LogWarning("DataList", "WeightedMovingAverage", "Invalid window size or weights length")
			isFailed = true
			return
		}

		// 計算權重總和，避免直接除以 windowSize
		weightsSum := 0.0
		for _, w := range weightsSlice {
			weightsSum += w.(float64)
		}

		movingAvgData = make([]float64, len(dl.data)-windowSize+1)
		for i := 0; i < len(movingAvgData); i++ {
			window := dl.data[i : i+windowSize]
			sum := 0.0
			for j := 0; j < windowSize; j++ {
				sum += window[j].(float64) * weightsSlice[j].(float64)
			}
			movingAvgData[i] = sum / weightsSum // 使用權重總和
		}
	})
	if isFailed {
		return nil
	}
	return NewDataList(movingAvgData)
}

// ExponentialSmoothing applies exponential smoothing to the DataList.
// The alpha parameter controls the smoothing factor.
// Returns a new DataList containing the smoothed values.
func (dl *DataList) ExponentialSmoothing(alpha float64) *DataList {
	if alpha < 0 || alpha > 1 {
		LogWarning("DataList", "ExponentialSmoothing", "Invalid alpha value")
		return nil
	}

	var smoothedData []float64
	dl.AtomicDo(func(dl *DataList) {
		floatData := dl.ToF64Slice()
		smoothedData = make([]float64, dl.Len())
		smoothedData[0] = floatData[0] // 使用初始值作為第一個平滑值
		for i := 1; i < dl.Len(); i++ {
			smoothedData[i] = alpha*floatData[i] + (1-alpha)*smoothedData[i-1]
		}
	})
	return NewDataList(smoothedData)
}

// DoubleExponentialSmoothing applies double exponential smoothing to the DataList.
// The alpha parameter controls the level smoothing, and the beta parameter controls the trend smoothing.
// Returns a new DataList containing the smoothed values.
func (dl *DataList) DoubleExponentialSmoothing(alpha, beta float64) *DataList {
	if alpha < 0 || alpha > 1 || beta < 0 || beta > 1 {
		LogWarning("DataList", "DoubleExponentialSmoothing", "Invalid alpha or beta value")
		return nil
	}
	var smoothedData []float64
	dl.AtomicDo(func(dl *DataList) {
		floatData := dl.ToF64Slice()
		smoothedData = make([]float64, dl.Len())
		trend := 0.0
		level := floatData[0]

		smoothedData[0] = level
		for i := 1; i < dl.Len(); i++ {
			prevLevel := level
			level = alpha*floatData[i] + (1-alpha)*(level+trend)
			trend = beta*(level-prevLevel) + (1-beta)*trend
			smoothedData[i] = level + trend
		}
	})
	return NewDataList(smoothedData)
}

// MovingStdDev calculates the moving standard deviation for the DataList using a specified window size.
func (dl *DataList) MovingStdev(windowSize int) *DataList {
	var movingStdDevData []float64
	isFailed := false
	dl.AtomicDo(func(dl *DataList) {
		if windowSize <= 0 || windowSize > dl.Len() {
			LogWarning("DataList", "MovingStdev", "Invalid window size")
			isFailed = true
			return
		}
		movingStdDevData = make([]float64, len(dl.data)-windowSize+1)
		for i := range movingStdDevData {
			window := NewDataList(dl.data[i : i+windowSize]...)
			movingStdDevData[i] = window.Stdev()
		}
	})
	if isFailed {
		return nil
	}
	return NewDataList(movingStdDevData)
}

// Sort sorts the DataList using a mixed sorting logic.
// It handles string, numeric (including all integer and float types), and time data types.
// If sorting fails, it restores the original order.
func (dl *DataList) Sort(ascending ...bool) *DataList {
	dl.AtomicDo(func(dl *DataList) {
		if len(dl.data) == 0 {
			LogWarning("DataList", "Sort", "DataList is empty, returning")
			return
		}

		// Save the original order
		originalData := make([]any, len(dl.data))
		copy(originalData, dl.data)

		defer func() {
			if r := recover(); r != nil {
				LogWarning("DataList", "Sort", "Sorting failed, restoring original order: %v", r)
				dl.data = originalData
			}
		}()

		ascendingOrder := true
		if len(ascending) > 0 {
			ascendingOrder = ascending[0]
		}
		if len(ascending) > 1 {
			LogWarning("DataList", "Sort", "Too many arguments, returning")
			return
		}

		// Mixed sorting
		order := 1
		if !ascendingOrder {
			order = -1
		}
		utils.ParallelSortStableFunc(dl.data, func(a, b any) int {
			return utils.CompareAny(a, b) * order
		})

		go dl.updateTimestamp()
	})
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
	utils.ParallelSortStableFunc(indexes, func(i, j int) int {
		return cmp.Compare(data[i], data[j])
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
		for j := range count {
			ranked[indexes[i+j]] = avgRank
		}
		i += count
	}

	return NewDataList(ranked)
}

// Reverse reverses the order of the elements in the DataList.
func (dl *DataList) Reverse() *DataList {
	dl.AtomicDo(func(dl *DataList) {
		sliceutil.Reverse(dl.data)
		go dl.updateTimestamp()
	})
	return dl
}

// Upper converts all string elements in the DataList to uppercase.
func (dl *DataList) Upper() *DataList {
	dl.AtomicDo(func(dl *DataList) {
		for i, v := range dl.data {
			if str, ok := v.(string); ok {
				dl.data[i] = strings.ToUpper(str)
			}
		}
		go dl.updateTimestamp()
	})
	return dl
}

// Lower converts all string elements in the DataList to lowercase.
func (dl *DataList) Lower() *DataList {
	dl.AtomicDo(func(dl *DataList) {
		for i, v := range dl.data {
			if str, ok := v.(string); ok {
				dl.data[i] = strings.ToLower(str)
			}
		}
		go dl.updateTimestamp()
	})
	return dl
}

// Capitalize capitalizes the first letter of each string element in the DataList.
func (dl *DataList) Capitalize() *DataList {
	dl.AtomicDo(func(dl *DataList) {
		for i, v := range dl.data {
			if str, ok := v.(string); ok {
				dl.data[i] = cases.Title(language.English, cases.NoLower).String(strings.ToLower(str))
			}
		}
		go dl.updateTimestamp()
	})
	return dl
}

// ======================== Statistics ========================

// Sum calculates the sum of all elements in the DataList.
// Returns math.NaN() if the DataList is empty or if no elements can be converted to float64.
func (dl *DataList) Sum() float64 {
	var sum float64
	var count int
	dl.AtomicDo(func(dl *DataList) {
		if len(dl.data) == 0 {
			LogWarning("DataList", "Sum", "DataList is empty")
			sum = math.NaN()
			return
		}

		sum = 0.0
		count = 0
		for _, v := range dl.data {
			vfloat, ok := ToFloat64Safe(v)
			if !ok {
				LogWarning("DataList", "Sum", "Element %v cannot be converted to float64, skipping.", v)
				continue
			}
			sum += vfloat
			count++
		}

		if count == 0 {
			LogWarning("DataList", "Sum", "No valid elements to compute sum")
			sum = math.NaN()
		}
	})
	return sum
}

// Max returns the maximum value in the DataList.
// Returns math.NaN() if the DataList is empty or if no elements can be converted to float64.
func (dl *DataList) Max() float64 {
	var max float64
	var foundValid bool
	dl.AtomicDo(func(dl *DataList) {
		if len(dl.data) == 0 {
			LogWarning("DataList", "Max", "DataList is empty")
			max = math.NaN()
			return
		}

		max = 0.0
		foundValid = false

		for _, v := range dl.data {
			vfloat, ok := ToFloat64Safe(v)
			if !ok {
				LogWarning("DataList", "Max", "Element %v is not a numeric type, skipping.", v)
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
			LogWarning("DataList", "Max", "No valid elements to compute maximum")
			max = math.NaN()
		}
	})
	return max
}

// Min returns the minimum value in the DataList.
// Returns math.NaN() if the DataList is empty or if no elements can be converted to float64.
func (dl *DataList) Min() float64 {
	var min float64
	var foundValid bool
	dl.AtomicDo(func(dl *DataList) {
		if len(dl.data) == 0 {
			LogWarning("DataList", "Min", "DataList is empty")
			min = math.NaN()
			return
		}

		min = 0.0
		foundValid = false

		for _, v := range dl.data {
			vfloat, ok := ToFloat64Safe(v)
			if !ok {
				LogWarning("DataList", "Min", "Element %v is not a numeric type, skipping.", v)
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
			LogWarning("DataList", "Min", "No valid elements to compute minimum")
			min = math.NaN()
		}
	})
	return min
}

// Mean calculates the arithmetic mean of the DataList.
// Returns math.NaN() if the DataList is empty or if no elements can be converted to float64.
func (dl *DataList) Mean() float64 {
	var mean float64
	dl.AtomicDo(func(dl *DataList) {
		mean = math.NaN()
		if len(dl.data) == 0 {
			LogWarning("DataList", "Mean", "DataList is empty")
			return
		}

		var sum float64
		var count int
		for _, v := range dl.data {
			if val, ok := ToFloat64Safe(v); ok {
				sum += val
				count++
			} else {
				LogWarning("DataList", "Mean", "Element %v is not a numeric type, skipping.", v)
				continue
			}
		}

		if count == 0 {
			LogWarning("DataList", "Mean", "No elements could be converted to float64")
			return
		}

		mean = sum / float64(count)
	})
	return mean
}

// WeightedMean calculates the weighted mean of the DataList using the provided weights.
// The weights parameter should be a slice or a DataList of the same length as the DataList.
// Returns math.NaN() if the DataList is empty, weights are invalid, or if no valid elements can be used.
func (dl *DataList) WeightedMean(weights any) float64 {
	var result float64
	dl.AtomicDo(func(dl *DataList) {
		if dl.Len() == 0 {
			LogWarning("DataList", "WeightedMean", "DataList is empty")
			result = math.NaN()
			return
		}
		weightsSlice, sliceLen := ProcessData(weights)
		if sliceLen != len(dl.data) {
			LogWarning("DataList", "WeightedMean", "Weights length does not match data length")
			result = math.NaN()
			return
		}

		totalWeight := 0.0
		weightedSum := 0.0
		validElements := 0

		for i, v := range dl.data {
			vfloat, ok1 := ToFloat64Safe(v)
			wfloat, ok2 := ToFloat64Safe(weightsSlice[i])
			if !ok1 {
				LogWarning("DataList", "WeightedMean", "Data element at index %d cannot be converted to float64, skipping", i)
				continue
			}
			if !ok2 {
				LogWarning("DataList", "WeightedMean", "Weight at index %d cannot be converted to float64, skipping", i)
				continue
			}
			weightedSum += vfloat * wfloat
			totalWeight += wfloat
			validElements++
		}

		if validElements == 0 {
			LogWarning("DataList", "WeightedMean", "No valid elements to compute weighted mean")
			result = math.NaN()
			return
		}
		if totalWeight == 0 {
			LogWarning("DataList", "WeightedMean", "Total weight is zero, returning NaN")
			result = math.NaN()
			return
		}

		result = weightedSum / totalWeight
	})
	return result
}

// GMean calculates the geometric mean of the DataList.
// Returns the geometric mean.
// Returns math.NaN() if the DataList is empty or if no elements can be converted to float64.
func (dl *DataList) GMean() float64 {
	var result float64
	dl.AtomicDo(func(dl *DataList) {
		result = math.NaN()
		if len(dl.data) == 0 {
			LogWarning("DataList", "GMean", "DataList is empty")
			return
		}

		product := 1.0
		count := 0
		for _, v := range dl.data {
			if val, ok := ToFloat64Safe(v); ok {
				if val <= 0 {
					LogWarning("DataList", "GMean", "Non-positive value encountered, skipping")
					continue
				}
				product *= val
				count++
			} else {
				LogWarning("DataList", "GMean", "Element %v is not a numeric type, skipping", v)
				continue
			}
		}

		if count == 0 {
			LogWarning("DataList", "GMean", "No valid elements to compute geometric mean")
			return
		}

		result = math.Pow(product, 1.0/float64(count))
	})
	return result
}

// Median calculates the median of the DataList.
// Returns math.NaN() if the DataList is empty or if no valid elements can be used.
func (dl *DataList) Median() float64 {
	var result float64
	dl.AtomicDo(func(dl *DataList) {
		if len(dl.data) == 0 {
			LogWarning("DataList", "Median", "DataList is empty")
			result = math.NaN()
			return
		}

		// Convert data to float64 and skip invalid elements
		var validData []float64
		for _, v := range dl.data {
			vfloat, ok := ToFloat64Safe(v)
			if !ok {
				LogWarning("DataList", "Median", "Element %v is not a numeric type, skipping", v)
				continue
			}
			validData = append(validData, vfloat)
		}

		if len(validData) == 0 {
			LogWarning("DataList", "Median", "No valid elements to compute median")
			result = math.NaN()
			return
		}

		// Sort the valid data
		sort.Float64s(validData)

		mid := len(validData) / 2

		if len(validData)%2 == 0 {
			// Even number of elements, return the average of the middle two
			mid1 := validData[mid-1]
			mid2 := validData[mid]
			result = (mid1 + mid2) / 2
		} else {
			// Odd number of elements, return the middle one
			result = validData[mid]
		}
	})
	return result
}

// Mode calculates the mode of the DataList.
// Only works with numeric data types.
// Mode could be a single value or multiple values.
// Returns nil if the DataList is empty or if no valid elements can be used.
func (dl *DataList) Mode() []float64 {
	var result []float64
	dl.AtomicDo(func(dl *DataList) {
		if len(dl.data) == 0 {
			LogWarning("DataList", "Mode", "DataList is empty, returning nil")
			result = nil
			return
		}

		freqMap := make(map[float64]int)
		for _, v := range dl.data {
			vfloat, ok := ToFloat64Safe(v)
			if !ok {
				continue
			}
			freqMap[vfloat]++
		}

		if len(freqMap) == 0 {
			LogWarning("DataList", "Mode", "No valid elements to compute mode, returning nil")
			result = nil
			return
		}

		var modes []float64
		maxFreq := 0
		for _, freq := range freqMap {
			if freq > maxFreq {
				maxFreq = freq
			}
		}

		// Check if all elements have the same frequency
		allSameFrequency := true
		for _, freq := range freqMap {
			if freq != maxFreq {
				allSameFrequency = false
				break
			}
		}

		if allSameFrequency {
			LogWarning("DataList", "Mode", "All elements have the same frequency. No mode exists, returning nil")
			result = nil
			return
		}
		for num, freq := range freqMap {
			if freq == maxFreq {
				modes = append(modes, num)
			}
		}

		result = modes
	})
	return result
}

// MAD calculates the mean absolute deviation of the DataList.
// Returns math.NaN() if the DataList is empty or if no valid elements can be used.
func (dl *DataList) MAD() float64 {
	var sum = 0.0
	var count = 0
	var earlyResult *float64
	dl.AtomicDo(func(dl *DataList) {
		if dl.Len() == 0 {
			LogWarning("DataList", "MAD", "DataList is empty")
			earlyResult = new(float64)
			*earlyResult = math.NaN()
			return
		}

		// Calculate the median using the modified Median function
		median := dl.Median()
		if math.IsNaN(median) {
			LogWarning("DataList", "MAD", "Median calculation failed")
			earlyResult = new(float64)
			*earlyResult = math.NaN()
			return
		}

		// Calculate the mean absolute deviation
		for _, v := range dl.data {
			val, ok := ToFloat64Safe(v)
			if !ok {
				LogWarning("DataList", "MAD", "Element %v is not a numeric type, skipping", v)
				continue
			}
			sum += math.Abs(val - median)
			count++
		}
	})
	if earlyResult != nil {
		return *earlyResult
	}

	if count == 0 {
		LogWarning("DataList", "MAD", "No valid elements to compute MAD")
		return math.NaN()
	}

	return sum / float64(count)
}

// Stdev calculates the standard deviation (sample) of the DataList.
// Returns math.NaN() if the DataList is empty or if no valid elements can be used.
func (dl *DataList) Stdev() float64 {
	var result float64
	dl.AtomicDo(func(dl *DataList) {
		if len(dl.data) == 0 {
			LogWarning("DataList", "Stdev", "DataList is empty")
			result = math.NaN()
			return
		}

		variance := dl.Var()
		if math.IsNaN(variance) {
			LogWarning("DataList", "Stdev", "Variance calculation failed")
			result = math.NaN()
			return
		}

		result = math.Sqrt(variance)
	})
	return result
}

// StdevP calculates the standard deviation (population) of the DataList.
// Returns math.NaN() if the DataList is empty or if no valid elements can be used.
func (dl *DataList) StdevP() float64 {
	var result float64
	dl.AtomicDo(func(dl *DataList) {
		if len(dl.data) == 0 {
			LogWarning("DataList", "StdevP", "DataList is empty")
			result = math.NaN()
			return
		}

		varianceP := dl.VarP()
		if math.IsNaN(varianceP) {
			LogWarning("DataList", "StdevP", "Variance calculation failed")
			result = math.NaN()
			return
		}

		result = math.Sqrt(varianceP)
	})
	return result
}

// Var calculates the variance (sample variance) of the DataList.
// Returns math.NaN() if the DataList is empty or if not enough valid elements are available.
func (dl *DataList) Var() float64 {
	var result float64
	dl.AtomicDo(func(dl *DataList) {
		if len(dl.data) == 0 {
			LogWarning("DataList", "Var", "DataList is empty")
			result = math.NaN()
			return
		}

		var sum float64
		var count int

		// First pass: calculate the mean of valid elements
		for _, v := range dl.data {
			xi, ok := ToFloat64Safe(v)
			if !ok {
				LogWarning("DataList", "Var", "Element %v is not a numeric type, skipping", v)
				continue
			}
			sum += xi
			count++
		}

		if count < 2 {
			LogWarning("DataList", "Var", "Not enough valid elements to compute variance")
			result = math.NaN()
			return
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
			numerator += (xi - mean) * (xi - mean)
		}

		denominator := float64(count - 1)
		result = numerator / denominator
	})
	return result
}

// VarP calculates the variance (population variance) of the DataList.
// Returns math.NaN() if the DataList is empty or if no valid elements can be used.
func (dl *DataList) VarP() float64 {
	var result float64
	dl.AtomicDo(func(dl *DataList) {
		if len(dl.data) == 0 {
			LogWarning("DataList", "VarP", "DataList is empty")
			result = math.NaN()
			return
		}

		// First pass: compute mean over valid elements
		var sum float64
		var count int

		for _, v := range dl.data {
			xi, ok := ToFloat64Safe(v)
			if !ok {
				LogWarning("DataList", "VarP", "Element %v is not a numeric type, skipping", v)
				continue
			}
			sum += xi
			count++
		}

		if count == 0 {
			LogWarning("DataList", "VarP", "No valid elements to compute variance")
			result = math.NaN()
			return
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
			numerator += (xi - mean) * (xi - mean)
		}

		result = numerator / float64(count) // Population variance divides by N
	})
	return result
}

// Range calculates the range of the DataList.
// Returns math.NaN() if the DataList is empty or if Max or Min cannot be calculated.
func (dl *DataList) Range() float64 {
	var result float64
	dl.AtomicDo(func(dl *DataList) {
		if len(dl.data) == 0 {
			LogWarning("DataList", "Range", "DataList is empty")
			result = math.NaN()
			return
		}

		max := dl.Max()
		min := dl.Min()

		if math.IsNaN(max) || math.IsNaN(min) {
			LogWarning("DataList", "Range", "Max or Min calculation failed")
			result = math.NaN()
			return
		}

		result = max - min
	})
	return result
}

// Quartile calculates the quartile based on the input value (1 to 3).
// 1 corresponds to the first quartile (Q1), 2 to the median (Q2), and 3 to the third quartile (Q3).
// This implementation uses percentiles to compute quartiles.
func (dl *DataList) Quartile(q int) float64 {
	var result float64
	dl.AtomicDo(func(dl *DataList) {
		if len(dl.data) == 0 {
			LogWarning("DataList", "Quartile", "DataList is empty")
			result = math.NaN()
			return
		}
		if q < 1 || q > 3 {
			LogWarning("DataList", "Quartile", "Invalid quartile value")
			result = math.NaN()
			return
		}

		// Convert the DataList to a slice of float64 for numeric operations, skipping invalid elements
		var numericData []float64
		for _, v := range dl.data {
			vfloat, ok := ToFloat64Safe(v)
			if !ok {
				LogWarning("DataList", "Quartile", "Element %v is not a numeric type, skipping", v)
				continue
			}
			numericData = append(numericData, vfloat)
		}

		if len(numericData) == 0 {
			LogWarning("DataList", "Quartile", "No valid elements to compute quartile")
			result = math.NaN()
			return
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
			result = numericData[lowerIndex]
			return
		}

		// Interpolate between the lower and upper bounds
		lowerValue := numericData[lowerIndex]
		upperValue := numericData[upperIndex]
		fraction := pos - math.Floor(pos)

		result = lowerValue + fraction*(upperValue-lowerValue)
	})
	return result
}

// IQR calculates the interquartile range of the DataList.
// Returns math.NaN() if the DataList is empty or if Q1 or Q3 cannot be calculated.
// Returns the interquartile range (Q3 - Q1) as a float64.
func (dl *DataList) IQR() float64 {
	var q1, q3 float64
	var earlyResult *float64
	dl.AtomicDo(func(dl *DataList) {
		if dl.Len() == 0 {
			LogWarning("DataList", "IQR", "DataList is empty")
			earlyResult = new(float64)
			*earlyResult = math.NaN()
			return
		}

		q1 = dl.Quartile(1)
		q3 = dl.Quartile(3)
	})
	if earlyResult != nil {
		return *earlyResult
	}

	if math.IsNaN(q1) || math.IsNaN(q3) {
		LogWarning("DataList", "IQR", "Q1 or Q3 calculation failed")
		return math.NaN()
	}

	return q3 - q1
}

// Percentile calculates the percentile based on the input value (0 to 100).
func (dl *DataList) Percentile(p float64) float64 {
	var result float64
	dl.AtomicDo(func(dl *DataList) {
		if len(dl.data) == 0 {
			LogWarning("DataList", "Percentile", "DataList is empty")
			result = math.NaN()
			return
		}
		if p < 0 || p > 100 {
			LogWarning("DataList", "Percentile", "Invalid percentile value")
			result = math.NaN()
			return
		}

		// Convert the DataList to a slice of float64 for numeric operations, skipping invalid elements
		var numericData []float64
		for _, v := range dl.data {
			vfloat, ok := ToFloat64Safe(v)
			if !ok {
				LogWarning("DataList", "Percentile", "Element %v cannot be converted to float64, skipping", v)
				continue
			}
			numericData = append(numericData, vfloat)
		}

		if len(numericData) == 0 {
			LogWarning("DataList", "Percentile", "No valid elements to compute percentile")
			result = math.NaN()
			return
		}

		// Sort the data
		sort.Float64s(numericData)

		n := len(numericData)
		if n == 1 {
			result = numericData[0]
			return
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
			result = lowerValue
			return
		}

		// Interpolate between the lower and upper values
		fraction := h - float64(lowerIndex)
		result = lowerValue + fraction*(upperValue-lowerValue)
	})
	return result
}

// Difference calculates the differences between adjacent elements in the DataList.
func (dl *DataList) Difference() *DataList {
	var result *DataList
	dl.AtomicDo(func(dl *DataList) {
		defer func() {
			if r := recover(); r != nil {
				LogWarning("DataList", "Difference", "Data types cannot be compared")
			}
		}()

		if len(dl.data) < 2 {
			LogWarning("DataList", "Difference", "DataList is too short to calculate differences, returning nil")
			result = nil
			return
		}

		differenceData := make([]float64, dl.Len()-1)
		for i := 1; i < dl.Len(); i++ {
			differenceData[i-1] = conv.ParseF64(dl.data[i]) - conv.ParseF64(dl.data[i-1])
		}

		result = NewDataList(differenceData)
	})
	return result
}

// ======================== Comparison ========================

// IsEqualTo checks if the data of the DataList is equal to another DataList.
func (dl *DataList) IsEqualTo(anotherDl *DataList) bool {
	var result bool
	dl.AtomicDo(func(dl *DataList) {
		anotherDl.AtomicDo(func(anotherDl *DataList) {
			if len(dl.data) != len(anotherDl.data) {
				result = false
				return
			}

			for i, v := range dl.data {
				if v != anotherDl.data[i] {
					result = false
					return
				}
			}

			result = true
		})
	})
	return result
}

// IsTheSameAs checks if the DataList is fully the same as another DataList.
// It checks for equality in name, data, creation timestamp, and last modified timestamp.
func (dl *DataList) IsTheSameAs(anotherDl *DataList) bool {
	var result bool
	dl.AtomicDo(func(dl *DataList) {
		anotherDl.AtomicDo(func(anotherDl *DataList) {
			if dl == anotherDl {
				result = true
				return
			}

			if anotherDl == nil {
				LogWarning("DataList", "IsTheSameAs", "Another DataList is nil, returning false")
				result = false
				return
			}

			if len(dl.data) != len(anotherDl.data) {
				result = false
				return
			}

			for i, v := range dl.data {
				if v != anotherDl.data[i] {
					result = false
					return
				}
			}

			if dl.name != anotherDl.name {
				result = false
				return
			}

			if dl.creationTimestamp != anotherDl.creationTimestamp || dl.lastModifiedTimestamp.Load() != anotherDl.lastModifiedTimestamp.Load() {
				result = false
				return
			}

			result = true
		})
	})
	return result
}

// ======================== Conversion ========================

// ParseNumbers attempts to parse all string elements in the DataList to numeric types.
// If parsing fails, the element is left unchanged.
func (dl *DataList) ParseNumbers() *DataList {
	dl.AtomicDo(func(dl *DataList) {
		for i, v := range dl.data {
			func() {
				defer func() {
					if r := recover(); r != nil {
						LogWarning("DataList", "ParseNumbers", "Failed to parse %v to float64: %v, the element left unchanged", v, r)
					}
				}()

				dl.data[i] = conv.ParseF64(v)
			}()
		}

		go dl.updateTimestamp()
	})
	return dl
}

// ParseStrings converts all elements in the DataList to strings.
func (dl *DataList) ParseStrings() *DataList {
	dl.AtomicDo(func(dl *DataList) {
		for i, v := range dl.data {
			func() {
				defer func() {
					if r := recover(); r != nil {
						LogWarning("DataList", "ParseStrings", "Failed to convert %v to string: %v, the element left unchanged", v, r)
					}
				}()

				dl.data[i] = conv.ToString(v)
			}()
		}

		go dl.updateTimestamp()
	})
	return dl
}

// ToF64Slice converts the DataList to a float64 slice.
// Returns the float64 slice.
// Returns nil if the DataList is empty.
// ToF64Slice converts the DataList to a float64 slice.
func (dl *DataList) ToF64Slice() []float64 {
	var result []float64
	dl.AtomicDo(func(dl *DataList) {
		if len(dl.data) == 0 {
			LogWarning("DataList", "ToF64Slice", "DataList is empty, returning nil")
			result = nil
			return
		}

		floatData := make([]float64, len(dl.data))
		for i, v := range dl.data {
			floatData[i] = ToFloat64(v)
		}

		result = floatData
	})
	return result
}

// ToStringSlice converts the DataList to a string slice.
// Returns the string slice.
// Returns nil if the DataList is empty.
func (dl *DataList) ToStringSlice() []string {
	var result []string
	dl.AtomicDo(func(dl *DataList) {
		if len(dl.data) == 0 {
			LogWarning("DataList", "ToStringSlice", "DataList is empty, returning nil")
			result = nil
			return
		}

		stringData := make([]string, len(dl.data))
		for i, v := range dl.data {
			stringData[i] = conv.ToString(v)
		}

		result = stringData
	})
	return result
}

// ======================== Timestamp ========================

// GetCreationTimestamp returns the creation time of the DataList in Unix timestamp.
func (dl *DataList) GetCreationTimestamp() int64 {
	var ts int64
	dl.AtomicDo(func(dl *DataList) {
		ts = dl.creationTimestamp
	})
	return ts
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
	var name string
	dl.AtomicDo(func(dl *DataList) {
		name = dl.name
	})
	return name
}

// SetName sets the name of the DataList.
func (dl *DataList) SetName(newName string) *DataList {
	dl.AtomicDo(func(dl *DataList) {
		dl.name = newName
		go dl.updateTimestamp()
	})
	return dl
}
