package insyra

import (
	"fmt"
	"math"
	"math/big"
	"runtime"
	"sort"
	"strings"
	"sync"
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
	lastModifiedTimestamp int64
	// 記憶體管理
	mu sync.Mutex
}

// IDataList defines the behavior expected from a DataList.
type IDataList interface {
	isFragmented() bool
	GetCreationTimestamp() int64
	GetLastModifiedTimestamp() int64
	updateTimestamp()
	GetName() string
	SetName(string)
	Data() []interface{}
	Append(values ...interface{})
	Get(index int) interface{}
	Count(interface{}) int
	Update(index int, value interface{})
	InsertAt(index int, value interface{})
	FindFirst(interface{}) interface{}
	FindLast(interface{}) interface{}
	FindAll(interface{}) []int
	Filter(func(interface{}) bool) *DataList
	ReplaceFirst(interface{}, interface{})
	ReplaceLast(interface{}, interface{})
	ReplaceAll(interface{}, interface{})
	Pop() interface{}
	Drop(index int)
	DropAll(...interface{})
	Clear()
	ClearStrings()
	ClearNumbers()
	Len() int
	Sort(acending ...bool)
	Reverse()
	Upper()
	Lower()
	Capitalize()
	Sum() interface{}
	Max() interface{}
	Min() interface{}
	Mean(highPrecision ...bool) interface{}
	GMean() interface{}
	Median(highPrecision ...bool) interface{}
	Mode() interface{}
	Mad() interface{}
	Stdev(highPrecision ...bool) interface{}
	StdevP(highPrecision ...bool) interface{}
	Var(highPrecision ...bool) interface{}
	VarP(highPrecision ...bool) interface{}
	Range() interface{}
	Quartile(int) interface{}
	IQR() interface{}
	Percentile(float64) interface{}
	ParseNumbers()
	ParseStrings()
	ToF64Slice() []float64
	ToStringSlice() []string
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

	dl := &DataList{
		data:                  continuousMemData,
		creationTimestamp:     time.Now().Unix(),
		lastModifiedTimestamp: time.Now().Unix(),
	}
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

// Count returns the number of occurrences of the specified value in the DataList.
func (dl *DataList) Count(value interface{}) int {
	found := dl.FindAll(value)
	if found == nil {
		return 0
	}
	return len(found)
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

	// 獲取可用的 CPU 核心數量
	numGoroutines := runtime.NumCPU()

	// 決定每個線程處理的數據量
	chunkSize := length / numGoroutines
	if length%numGoroutines != 0 {
		chunkSize++
	}

	var tasks []asyncutil.Task

	// 創建並行任務
	for i := 0; i < numGoroutines; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if end > length {
			end = length
		}

		task := asyncutil.Task{
			ID: fmt.Sprintf("task-%d", i),
			Fn: func(dataChunk []interface{}) []interface{} {
				for j, v := range dataChunk {
					if v == oldValue {
						dataChunk[j] = newValue
					}
				}
				return dataChunk
			},
			Args: []interface{}{dl.data[start:end]},
		}

		tasks = append(tasks, task)
	}

	// 使用 ParallelProcess 來處理所有任務
	taskResults := asyncutil.ParallelProcess(tasks)

	// 合併結果
	for i, result := range taskResults {
		start := i * chunkSize
		end := start + chunkSize
		if end > length {
			end = length
		}
		copy(dl.data[start:end], result.Results[0].([]interface{}))
	}

	go dl.updateTimestamp()
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
func (dl *DataList) Drop(index int) {
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
		return
	}
	dl.data = append(dl.data[:index], dl.data[index+1:]...)
	go dl.updateTimestamp()
}

// DropAll removes all occurrences of the specified values from the DataList.
// Supports multiple values to drop.
func (dl *DataList) DropAll(toDrop ...interface{}) {
	defer func() {
		dl.mu.Unlock()
		go reorganizeMemory(dl)
	}()
	dl.mu.Lock()
	length := len(dl.data)
	if length == 0 {
		return
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
}

// Clear removes all elements from the DataList and updates the timestamp.
func (dl *DataList) Clear() {
	defer func() {
		dl.mu.Unlock()
		go reorganizeMemory(dl)
	}()
	dl.mu.Lock()
	dl.data = []interface{}{}
	go dl.updateTimestamp()
}

func (dl *DataList) Len() int {
	return len(dl.data)
}

// ClearStrings removes all string elements from the DataList and updates the timestamp.
func (dl *DataList) ClearStrings() {
	defer func() {
		dl.mu.Unlock()
		go reorganizeMemory(dl)
	}()
	dl.mu.Lock()
	length := len(dl.data)
	if length == 0 {
		return
	}

	// 獲取可用的 CPU 核心數量
	numGoroutines := runtime.NumCPU()

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
}

// ++++ 此處之後尚未提升性能 ++++

// ClearNumbers removes all numeric elements (int, float, etc.) from the DataList and updates the timestamp.
func (dl *DataList) ClearNumbers() {
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
}

// Sort sorts the DataList using a mixed sorting logic.
// It handles string, numeric (including all integer and float types), and time data types.
// If sorting fails, it restores the original order.
func (dl *DataList) Sort(ascending ...bool) {
	defer func() {
		dl.mu.Unlock()
		go reorganizeMemory(dl)
	}()
	dl.mu.Lock()
	if len(dl.data) == 0 {
		LogWarning("DataList.Sort(): DataList is empty, returning.")
		return
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
		return
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
}

// Reverse reverses the order of the elements in the DataList.
func (dl *DataList) Reverse() {
	defer func() {
		dl.mu.Unlock()
		go reorganizeMemory(dl)
	}()
	dl.mu.Lock()
	sliceutil.Reverse(dl.data)
}

// Upper converts all string elements in the DataList to uppercase.
func (dl *DataList) Upper() {
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
}

// Lower converts all string elements in the DataList to lowercase.
func (dl *DataList) Lower() {
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
}

// Capitalize capitalizes the first letter of each string element in the DataList.
func (dl *DataList) Capitalize() {
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
}

// ======================== Statistics ========================

func (dl *DataList) Sum() interface{} {
	if len(dl.data) == 0 {
		LogWarning("DataList.Sum(): DataList is empty, returning nil.")
		return nil
	}
	var sum float64
	for _, v := range dl.data {
		vfloat, ok := ToFloat64Safe(v)
		if !ok {
			LogWarning("DataList.Sum(): Data types cannot be compared, returning nil.")
			return nil
		}
		sum += vfloat
	}

	return sum
}

// Max returns the maximum value in the DataList.
// Returns the maximum value.
// Returns nil if the DataList is empty.
// Max returns the maximum value in the DataList, or nil if the data types cannot be compared.
func (dl *DataList) Max() interface{} {
	if len(dl.data) == 0 {
		return nil
	}

	var max interface{}

	for _, v := range dl.data {
		if max == nil {
			max = v
			continue
		}

		switch maxVal := max.(type) {
		case int:
			if val, ok := v.(int); ok && val > maxVal {
				max = val
			} else if !ok {
				LogWarning("DataList.Max(): Data types cannot be compared, returning nil.")
				return nil
			}
		case float64:
			if val, ok := v.(float64); ok && val > maxVal {
				max = val
			} else if intVal, ok := v.(int); ok && float64(intVal) > maxVal {
				max = float64(intVal)
			} else if !ok {
				LogWarning("DataList.Max(): Data types cannot be compared, returning nil.")
				return nil
			}
		case string:
			if val, ok := v.(string); ok {
				valF64, ok := ToFloat64Safe(val)
				if !ok {
					LogWarning("DataList.Max(): Data types cannot be compared, returning nil.")
					return nil
				}
				maxValF64, ok := ToFloat64Safe(maxVal)
				if !ok {
					LogWarning("DataList.Max(): Data types cannot be compared, returning nil.")
					return nil
				}
				if valF64 > maxValF64 {
					max = val
				}
			} else if !ok {
				LogWarning("DataList.Max(): Data types cannot be compared, returning nil.")
				return nil
			}
		default:
			LogWarning("DataList.Max(): Data types cannot be compared, returning nil.")
			return nil
		}
	}

	return max
}

// Min returns the minimum value in the DataList.
// Returns the minimum value.
// Returns nil if the DataList is empty.
// Min returns the minimum value in the DataList, or nil if the data types cannot be compared.
func (dl *DataList) Min() interface{} {
	if len(dl.data) == 0 {
		LogWarning("DataList.Min(): DataList is empty, returning nil.")
		return nil
	}

	var min interface{}
	for _, v := range dl.data {
		if min == nil {
			min = v
			continue
		}

		switch minVal := min.(type) {
		case int:
			if val, ok := v.(int); ok && val < minVal {
				min = val
			} else if !ok {
				LogWarning("DataList.Min(): Data types cannot be compared, returning nil.")
				return nil
			}
		case float64:
			if val, ok := v.(float64); ok && val < minVal {
				min = val
			} else if intVal, ok := v.(int); ok && float64(intVal) < minVal {
				min = float64(intVal)
			} else if !ok {
				LogWarning("DataList.Min(): Data types cannot be compared, returning nil.")
				return nil
			}
		case string:
			if val, ok := v.(string); ok && conv.ParseF64(val) < conv.ParseF64(minVal) {
				min = val
			} else if !ok {
				LogWarning("DataList.Min(): Data types cannot be compared, returning nil.")
				return nil
			}
		default:
			LogWarning("DataList.Min(): Data types cannot be compared, returning nil.")
			return nil
		}
	}

	return min
}

// Mean calculates the arithmetic mean of the DataList.
// If highPrecision is true, it will calculate using big.Rat for high precision.
// Otherwise, it calculates using float64.
// Returns nil if the DataList is empty or if an invalid number of parameters is provided.
func (dl *DataList) Mean(highPrecision ...bool) interface{} {
	if len(dl.data) == 0 {
		LogWarning("DataList.Mean(): DataList is empty, returning nil.")
		return nil
	}

	// 檢查參數數量
	if len(highPrecision) > 1 {
		LogWarning("DataList.Mean(): Too many arguments, returning nil.")
		return nil
	}

	// 默認使用普通模式（float64），若有參數則使用參數設定
	highPrecisionMode := false
	if len(highPrecision) == 1 {
		highPrecisionMode = highPrecision[0]
	}

	if highPrecisionMode {
		sum := new(big.Rat)
		count := big.NewRat(int64(len(dl.data)), 1)

		for _, v := range dl.data {
			if val, ok := ToFloat64Safe(v); ok {
				ratValue := new(big.Rat).SetFloat64(val)
				sum.Add(sum, ratValue)
			} else {
				LogWarning("DataList.Mean(): Data types cannot be compared, returning nil.")
				return nil
			}
		}

		mean := new(big.Rat).Quo(sum, count)
		return mean
	}

	// 普通模式（float64）
	var sum float64
	for _, v := range dl.data {
		if val, ok := ToFloat64Safe(v); ok {
			sum += val
		} else {
			LogWarning("DataList.Mean(): Data types cannot be compared, returning nil.")
			return nil
		}
	}
	mean := sum / float64(len(dl.data))
	return mean
}

// GMean calculates the geometric mean of the DataList.
// Returns the geometric mean.
// Returns nil if the DataList is empty.
// GMean returns the geometric mean of the DataList.
func (dl *DataList) GMean() interface{} {
	if len(dl.data) == 0 {
		LogWarning("DataList.GMean(): DataList is empty, returning nil.")
		return nil
	}

	product := 1.0
	for _, v := range dl.data {
		if val, ok := ToFloat64Safe(v); ok {
			product *= val
		} else {
			LogWarning("DataList.GMean(): Data types cannot be compared, returning nil.")
			return nil
		}
	}

	return math.Pow(product, 1.0/float64(len(dl.data)))
}

// Median calculates the median of the DataList.
// Returns the median.
// Returns nil if the DataList is empty.
// Turn on highPrecision to return a big.Rat instead of a float64.
func (dl *DataList) Median(highPrecision ...bool) interface{} {
	if len(dl.data) == 0 {
		LogWarning("DataList.Median(): DataList is empty, returning nil.")
		return nil
	}

	// 檢查參數數量
	if len(highPrecision) > 1 {
		LogWarning("DataList.Median(): Too many arguments, returning nil.")
		return nil
	}

	// 對數據進行排序
	sortedData := make([]interface{}, len(dl.data))
	copy(sortedData, dl.data)
	dl.Sort()

	mid := len(sortedData) / 2
	useHighPrecision := len(highPrecision) == 1 && highPrecision[0]

	if len(sortedData)%2 == 0 {
		// 當元素個數為偶數時，返回中間兩個數的平均值
		mid1 := ToFloat64(sortedData[mid-1])
		mid2 := ToFloat64(sortedData[mid])

		if useHighPrecision {
			// 使用 big.Rat 進行精確的有理數運算
			ratMid1 := new(big.Rat).SetFloat64(mid1)
			ratMid2 := new(big.Rat).SetFloat64(mid2)

			// 計算平均值
			sum := new(big.Rat).Add(ratMid1, ratMid2)
			mean := new(big.Rat).Quo(sum, big.NewRat(2, 1))

			// 返回高精度結果
			return mean
		} else {
			// 使用 float64 計算並返回
			return (mid1 + mid2) / 2
		}
	}

	// 當元素個數為奇數時，返回中間的那個數
	midValue := ToFloat64(sortedData[mid])
	if useHighPrecision {
		return new(big.Rat).SetFloat64(midValue)
	}

	return midValue
}

// Mode calculates the mode of the DataList.
// Returns the mode.
// Returns nil if the DataList is empty.
// Mode returns the mode of the DataList.
func (dl *DataList) Mode() interface{} {
	if len(dl.data) == 0 {
		LogWarning("DataList.Mode(): DataList is empty, returning nil.")
		return nil
	}

	freqMap := make(map[interface{}]int)
	for _, v := range dl.data {
		freqMap[v]++
	}

	var mode interface{}
	maxFreq := 0
	for k, v := range freqMap {
		if v > maxFreq {
			mode = k
			maxFreq = v
		}
	}

	return mode.(float64)
}

// Mad calculates the mean absolute deviation of the DataList.
// Returns the mean absolute deviation.
// Returns nil if the DataList is empty.
func (dl *DataList) Mad() interface{} {
	if len(dl.data) == 0 {
		LogWarning("DataList.Mad(): DataList is empty, returning nil.")
		return nil
	}

	median := dl.Median()
	if median == nil {
		LogWarning("DataList.Mad(): Median calculation failed, returning nil.")
		return nil
	}

	// Calculate the mean absolute deviation
	var sum float64
	for _, v := range dl.data {
		val := ToFloat64(v)
		sum += math.Abs(val - median.(float64))
	}

	return sum / float64(len(dl.data))
}

// Stdev calculates the standard deviation(sample) of the DataList.
// Returns the standard deviation.
// Returns nil if the DataList is empty.
// Stdev returns the standard deviation of the DataList.
func (dl *DataList) Stdev(highPrecision ...bool) interface{} {
	if len(dl.data) == 0 {
		LogWarning("DataList.Stdev(): DataList is empty, returning nil.")
		return nil
	}

	// 判斷是否使用高精度模式
	useHighPrecision := len(highPrecision) == 1 && highPrecision[0]
	if len(highPrecision) > 1 {
		LogWarning("DataList.Stdev(): Too many arguments, returning nil.")
		return nil
	}

	var variance interface{}
	if useHighPrecision {
		variance = dl.Var(true)
	} else {
		variance = dl.Var(false)
	}

	if variance == nil {
		LogWarning("DataList.Stdev(): Variance calculation failed, returning nil.")
		return nil
	}

	if useHighPrecision {
		// 高精度模式下使用 SqrtRat 進行開方運算
		varianceRat := variance.(*big.Rat)
		sqrtVariance := SqrtRat(varianceRat)
		return sqrtVariance
	}

	// 普通模式下使用 float64 計算
	return math.Sqrt(ToFloat64(variance))
}

// StdevP calculates the standard deviation(population) of the DataList.
// Returns the standard deviation.
// Returns nil if the DataList is empty or the standard deviation cannot be calculated.
func (dl *DataList) StdevP(highPrecision ...bool) interface{} {
	if len(dl.data) == 0 {
		LogWarning("DataList.StdevP(): DataList is empty, returning nil.")
		return nil
	}
	if len(highPrecision) > 1 {
		LogWarning("DataList.StdevP(): Too many arguments, returning nil.")
		return nil
	}
	var varianceP interface{}
	if len(highPrecision) == 1 && highPrecision[0] {
		// 使用 big.Rat 進行高精度計算
		varianceP = dl.VarP(true)
	} else {
		varianceP = dl.VarP()
	}

	if varianceP == nil {
		LogWarning("DataList.StdevP(): Variance calculation failed, returning nil.")
		return nil
	}

	if !highPrecision[0] {
		return math.Sqrt(ToFloat64(varianceP))
	} else {
		return SqrtRat(varianceP.(*big.Rat))
	}
}

// Var calculates the variance(sample) of the DataList.
// Returns the variance.
// Returns nil if the DataList is empty or the variance cannot be calculated.
func (dl *DataList) Var(highPrecision ...bool) interface{} {
	n := float64(dl.Len())
	if n == 0.0 {
		LogWarning("DataList.Var(): DataList is empty, returning nil.")
		return nil
	}

	// 判斷是否使用高精度模式
	useHighPrecision := len(highPrecision) == 1 && highPrecision[0]
	if len(highPrecision) > 1 {
		LogWarning("DataList.Var(): Too many arguments, returning nil.")
		return nil
	}

	if useHighPrecision {
		// 使用 big.Rat 進行高精度計算
		mean := dl.Mean(true).(*big.Rat)
		denominator := new(big.Rat).SetFloat64(n - 1)
		if denominator.Cmp(big.NewRat(0, 1)) == 0 {
			LogWarning("DataList.Var(): Denominator is 0, returning nil.")
			return nil
		}
		numerator := new(big.Rat)
		for i := 0; i < len(dl.data); i++ {
			xi, ok := ToFloat64Safe(dl.data[i])
			if !ok {
				LogWarning("DataList.Var(): Element is not a float64, returning nil.")
				return nil
			}
			ratXi := new(big.Rat).SetFloat64(xi)
			diff := new(big.Rat).Sub(ratXi, mean)
			squareDiff := new(big.Rat).Mul(diff, diff)
			numerator.Add(numerator, squareDiff)
		}
		variance := new(big.Rat).Quo(numerator, denominator)
		return variance
	}

	// 普通模式使用 float64 計算
	mean := dl.Mean(false).(float64)
	denominator := n - 1
	if denominator == 0 {
		LogWarning("DataList.Var(): Denominator is 0, returning nil.")
		return nil
	}
	numerator := 0.0
	for i := 0; i < len(dl.data); i++ {
		xi, ok := ToFloat64Safe(dl.data[i])
		if !ok {
			LogWarning("DataList.Var(): Element is not a float64, returning nil.")
			return nil
		}
		numerator += math.Pow(xi-mean, 2)
	}
	return numerator / denominator
}

// VarP calculates the variance(population) of the DataList.
// Returns the variance.
// Returns nil if the DataList is empty or the variance cannot be calculated.
func (dl *DataList) VarP(highPrecision ...bool) interface{} {
	if len(highPrecision) > 1 {
		LogWarning("VarP(): More than one highPrecision argument, returning nil.")
		return nil
	}

	useHighPrecision := len(highPrecision) == 1 && highPrecision[0]

	n := float64(dl.Len())
	if n == 0.0 {
		LogWarning("DataList.VarP(): DataList is empty, returning nil.")
		return nil
	}

	if useHighPrecision {
		// 使用高精度计算
		mean := dl.Mean(true).(*big.Rat)
		numerator := new(big.Rat)
		for i := 0; i < len(dl.data); i++ {
			xi := new(big.Rat).SetFloat64(ToFloat64(dl.data[i]))
			diff := new(big.Rat).Sub(xi, mean)
			diffSquared := new(big.Rat).Mul(diff, diff)
			numerator.Add(numerator, diffSquared)
		}
		denominator := new(big.Rat).SetFloat64(n)
		variance := new(big.Rat).Quo(numerator, denominator)
		return variance
	} else {
		// 使用普通精度计算
		mean, ok := ToFloat64Safe(dl.Mean())
		if !ok {
			LogWarning("DataList.VarP(): Mean is not a float64, returning nil.")
			return nil
		}
		numerator := 0.0
		for i := 0; i < len(dl.data); i++ {
			xi, ok := ToFloat64Safe(dl.data[i])
			if !ok {
				LogWarning("DataList.VarP(): Element is not a float64, returning nil.")
				return nil
			}
			numerator += math.Pow(xi-mean, 2)
		}
		return numerator / n
	}
}

// Range calculates the range of the DataList.
// Returns the range.
// Returns nil if the DataList is empty.
// Range returns the range of the DataList.
func (dl *DataList) Range() interface{} {
	if len(dl.data) == 0 {
		LogWarning("DataList.Range(): DataList is empty, returning nil.")
		return nil
	}

	max := ToFloat64(dl.Max())
	min := ToFloat64(dl.Min())

	return max - min
}

// Quartile calculates the quartile based on the input value (1 to 3).
// 1 corresponds to the first quartile (Q1), 2 to the median (Q2), and 3 to the third quartile (Q3).
func (dl *DataList) Quartile(q int) interface{} {
	if len(dl.data) == 0 {
		LogWarning("DataList.Quartile(): DataList is empty, returning nil.")
		return nil
	}
	if q < 1 || q > 3 {
		LogWarning("DataList.Quartile(): Invalid quartile value, returning nil.")
		return nil
	}

	// Convert the DataList to a slice of float64 for numeric operations
	numericData := dl.ToF64Slice()

	// Sort the data
	sort.Float64s(numericData)

	n := len(numericData)

	var pos float64
	var lower, upper float64

	switch q {
	case 1:
		pos = 0.25 * float64(n-1)
	case 2:
		pos = 0.5 * float64(n-1)
	case 3:
		pos = 0.75 * float64(n-1)
	}

	// Get the index for lower and upper bounds
	lowerIndex := int(math.Floor(pos))
	upperIndex := int(math.Ceil(pos))

	// Handle the case where the position is exactly an integer
	if lowerIndex == upperIndex {
		return numericData[lowerIndex]
	}

	// Interpolate between the lower and upper bounds
	lower = numericData[lowerIndex]
	upper = numericData[upperIndex]

	return lower + (pos-float64(lowerIndex))*(upper-lower)
}

// IQR calculates the interquartile range of the DataList.
// Returns the interquartile range.
// Returns nil if the DataList is empty.
func (dl *DataList) IQR() interface{} {
	if len(dl.data) == 0 {
		LogWarning("DataList.IQR(): DataList is empty, returning nil.")
		return nil
	}
	q3, ok := ToFloat64Safe(dl.Quartile(3))
	if !ok {
		LogWarning("DataList.IQR(): Q3 is not a float64, returning nil.")
		return nil
	}
	q1, ok := ToFloat64Safe(dl.Quartile(1))
	if !ok {
		LogWarning("DataList.IQR(): Q1 is not a float64, returning nil.")
		return nil
	}
	return q3 - q1
}

// Percentile calculates the percentile based on the input value (0 to 100).
// Returns the percentile value, or nil if the DataList is empty.
func (dl *DataList) Percentile(p float64) interface{} {
	if len(dl.data) == 0 {
		LogWarning("DataList.Percentile(): DataList is empty, returning nil.")
		return nil
	}
	if p < 0 || p > 100 {
		LogWarning("DataList.Percentile(): Invalid percentile value, returning nil.")
		return nil
	}

	// Convert the DataList to a slice of float64 for numeric operations
	numericData := dl.ToF64Slice()

	// Sort the data
	sort.Float64s(numericData)

	// Calculate the index
	pos := p / 100 * float64(len(numericData)-1)
	lowerIndex := int(math.Floor(pos))
	upperIndex := int(math.Ceil(pos))

	// Handle the case where the position is exactly an integer
	if lowerIndex == upperIndex {
		return numericData[lowerIndex]
	}

	// Interpolate between the lower and upper bounds
	lower := numericData[lowerIndex]
	upper := numericData[upperIndex]

	return lower + (pos-float64(lowerIndex))*(upper-lower)
}

// ======================== Conversion ========================

// ParseNumbers attempts to parse all string elements in the DataList to numeric types.
// If parsing fails, the element is left unchanged.
func (dl *DataList) ParseNumbers() {
	defer func() {
		dl.mu.Unlock()
		go reorganizeMemory(dl)
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
}

// ParseStrings converts all elements in the DataList to strings.
func (dl *DataList) ParseStrings() {
	defer func() {
		dl.mu.Unlock()
		go reorganizeMemory(dl)
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
	return dl.lastModifiedTimestamp
}

// updateTimestamp updates the lastModifiedTimestamp to the current Unix time.
func (dl *DataList) updateTimestamp() {
	dl.lastModifiedTimestamp = time.Now().Unix()
}

// ======================== Name ========================
func (dl *DataList) GetName() string {
	return dl.name
}

func (dl *DataList) SetName(newName string) {
	nm := getNameManager()

	// 鎖定 DataList 以確保名稱設置過程的同步性
	dl.mu.Lock()
	defer dl.mu.Unlock()

	// 檢查並註冊新名稱
	if err := nm.registerName(newName); err != nil {
		LogWarning("DataList.SetName(): %v, remaining the old name.", err)
		return
	}

	// 解除舊名稱的註冊（如果已有名稱）
	if dl.name != "" {
		nm.unregisterName(dl.name)
	}

	dl.name = newName
	go dl.updateTimestamp()
}
