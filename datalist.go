package insyra

import (
	"fmt"
	"log"
	"math"
	"runtime"
	"sort"
	"strings"
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
}

// IDataList defines the behavior expected from a DataList.
type IDataList interface {
	GetCreationTimestamp() int64
	GetLastModifiedTimestamp() int64
	updateTimestamp()
	GetName() string
	SetName(string)
	Data() []interface{}
	Append(value interface{})
	Get(index int) interface{}
	Update(index int, value interface{})
	InsertAt(index int, value interface{})
	FindFirst(interface{}) interface{}
	FindLast(interface{}) interface{}
	FindAll(interface{}) []int
	Filter(func(interface{}) bool) []interface{}
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
	Max() interface{}
	Min() interface{}
	Mean() interface{}
	GMean() interface{}
	Median() interface{}
	Mode() interface{}
	Stdev() interface{}
	StdevP() interface{}
	Var() interface{}
	VarP() interface{}
	Range() interface{}
	Quartile(int) interface{}
	IQR() interface{}
	Percentile(float64) interface{}
	ToF64Slice() []float64
}

// Data returns the data stored in the DataList.
func (dl *DataList) Data() []interface{} {
	return dl.data
}

// NewDataList creates a new DataList, supporting both slice and variadic inputs,
// and flattens the input before storing it.
func NewDataList(values ...interface{}) *DataList {
	// Flatten the input values using sliceutil.Flatten with the specified generic type
	flatData, err := sliceutil.Flatten[interface{}](values)
	if err != nil {
		log.Printf("[insyra] NewDataList(): Failed to flatten input values:%v\nUsing the original input values.\n", err)
		flatData = values
	}
	return &DataList{
		data:                  flatData,
		creationTimestamp:     time.Now().Unix(),
		lastModifiedTimestamp: time.Now().Unix(),
	}
}

// Append adds a new value to the DataList.
// The value can be of any type.
// The value is appended to the end of the DataList.
func (dl *DataList) Append(value interface{}) {
	dl.data = append(dl.data, value)
	go dl.updateTimestamp()
}

// Get retrieves the value at the specified index in the DataList.
// Supports negative indexing.
// Returns nil if the index is out of bounds.
// Returns the value at the specified index.
func (dl *DataList) Get(index int) interface{} {
	if index < -len(dl.data) || index >= len(dl.data) {
		log.Println("[insyra] DataList.Get(): Index out of bounds, returning nil.")
		return nil
	}
	return dl.data[index]
}

// Update replaces the value at the specified index with the new value.
func (dl *DataList) Update(index int, newValue interface{}) {
	if index < 0 {
		index += len(dl.data)
	}
	if index < 0 || index >= len(dl.data) {
		log.Printf("[insyra] ReplaceAtIndex(): Index %d out of bounds", index)
	}
	dl.data[index] = newValue
	go dl.updateTimestamp()
}

// InsertAt inserts a value at the specified index in the DataList.
// If the index is out of bounds, the value is appended to the end of the list.
func (dl *DataList) InsertAt(index int, value interface{}) {
	// Handle negative index
	if index < 0 {
		index += len(dl.data) + 1
	}

	// If index is out of bounds, append the value to the end
	if index < 0 || index > len(dl.data) {
		log.Println("[insyra] InsertAt(): Index out of bounds, appending value to the end.")
		dl.data = append(dl.data, value)
	} else {
		var err error
		dl.data, err = sliceutil.InsertAt(dl.data, index, value)
		if err != nil {
			log.Println("[insyra] InsertAt(): Failed to insert value at index:", err)
			return
		}
	}

	go dl.updateTimestamp()
}

// FindFirst returns the index of the first occurrence of the specified value in the DataList.
// If the value is not found, it returns nil.
func (dl *DataList) FindFirst(value interface{}) interface{} {
	for i, v := range dl.data {
		if v == value {
			return i
		}
	}
	log.Println("[insyra] FindFirst(): Value not found, returning nil.")
	return nil
}

// FindLast returns the index of the last occurrence of the specified value in the DataList.
// If the value is not found, it returns nil.
func (dl *DataList) FindLast(value interface{}) interface{} {
	for i := len(dl.data) - 1; i >= 0; i-- {
		if dl.data[i] == value {
			return i
		}
	}
	log.Println("[insyra] FindLast(): Value not found, returning nil.")
	return nil
}

// FindAll returns a slice of all the indices where the specified value is found in the DataList using parallel processing.
// If the value is not found, it returns an empty slice.
func (dl *DataList) FindAll(value interface{}) []int {
	length := len(dl.data)
	if length == 0 {
		log.Println("[insyra] FindAll(): DataList is empty, returning an empty slice.")
		return []int{}
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
			Fn: func(dataChunk []interface{}, startIndex int) []int {
				var localIndices []int
				for j, v := range dataChunk {
					if v == value {
						localIndices = append(localIndices, startIndex+j)
					}
				}
				return localIndices
			},
			Args: []interface{}{dl.data[start:end], start},
		}

		tasks = append(tasks, task)
	}

	// 使用 ParallelProcess 來處理所有任務
	taskResults := asyncutil.ParallelProcess(tasks)

	var indices []int
	for _, result := range taskResults {
		if len(result.Results) > 0 {
			indices = append(indices, result.Results[0].([]int)...)
		}
	}

	if len(indices) == 0 {
		log.Println("[insyra] FindAll(): Value not found, returning an empty slice.")
	}

	return indices
}

// Filter filters the DataList based on a custom filter function provided by the user.
// The filter function should return true for elements that should be included in the result.
func (dl *DataList) Filter(filterFunc func(interface{}) bool) *DataList {
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
	for i, v := range dl.data {
		if v == oldValue {
			dl.data[i] = newValue
			go dl.updateTimestamp()
		}
	}
	log.Printf("[insyra] ReplaceFirst(): value not found.")
}

// ReplaceLast replaces the last occurrence of oldValue with newValue.
func (dl *DataList) ReplaceLast(oldValue, newValue interface{}) {
	for i := len(dl.data) - 1; i >= 0; i-- {
		if dl.data[i] == oldValue {
			dl.data[i] = newValue
			go dl.updateTimestamp()
		}
	}
	log.Printf("[insyra] ReplaceLast(): value not found.")
}

// ReplaceAll replaces all occurrences of oldValue with newValue in the DataList.
// If oldValue is not found, no changes are made.
func (dl *DataList) ReplaceAll(oldValue, newValue interface{}) {
	length := len(dl.data)
	if length == 0 {
		log.Println("[insyra] ReplaceAll(): DataList is empty, no replacements made.")
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
	n, err := sliceutil.Drt_PopFrom(&dl.data)
	if err != nil {
		log.Println("[insyra] DataList.Pop(): DataList is empty, returning nil.")
		return nil
	}
	go dl.updateTimestamp()
	return n
}

// Drop removes the element at the specified index from the DataList and updates the timestamp.
// Returns an error if the index is out of bounds.
func (dl *DataList) Drop(index int) {
	if index < 0 {
		index += len(dl.data)
	}
	if index >= len(dl.data) {
		log.Println("[insyra] DataList.Drop(): Index out of bounds, returning.")
		return
	}
	dl.data = append(dl.data[:index], dl.data[index+1:]...)
	go dl.updateTimestamp()
}

// DropAll removes all occurrences of the specified values from the DataList.
// Supports multiple values to drop.
func (dl *DataList) DropAll(toDrop ...interface{}) {
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
			log.Println("[insyra] DropAll(): Error in async task:", err)
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
	dl.data = []interface{}{}
	go dl.updateTimestamp()
}

func (dl *DataList) Len() int {
	return len(dl.data)
}

// ClearStrings removes all string elements from the DataList and updates the timestamp.
func (dl *DataList) ClearStrings() {
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
	if len(dl.data) == 0 {
		log.Println("[insyra] DataList.Sort(): DataList is empty, returning.")
		return
	}

	// Save the original order
	originalData := make([]interface{}, len(dl.data))
	copy(originalData, dl.data)

	defer func() {
		if r := recover(); r != nil {
			log.Println("[insyra] DataList.Sort(): Sorting failed, restoring original order:", r)
			dl.data = originalData
		}
	}()

	ascendingOrder := true
	if len(ascending) > 0 {
		ascendingOrder = ascending[0]
	}
	if len(ascending) > 1 {
		log.Println("[insyra] DataList.Sort(): Too many arguments, returning.")
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
	sliceutil.Reverse(dl.data)
}

// Upper converts all string elements in the DataList to uppercase.
func (dl *DataList) Upper() {
	for i, v := range dl.data {
		if str, ok := v.(string); ok {
			dl.data[i] = strings.ToUpper(str)
		}
	}
	go dl.updateTimestamp()
}

// Lower converts all string elements in the DataList to lowercase.
func (dl *DataList) Lower() {
	for i, v := range dl.data {
		if str, ok := v.(string); ok {
			dl.data[i] = strings.ToLower(str)
		}
	}
	go dl.updateTimestamp()
}

// Capitalize capitalizes the first letter of each string element in the DataList.
func (dl *DataList) Capitalize() {
	for i, v := range dl.data {
		if str, ok := v.(string); ok {
			dl.data[i] = strings.Title(strings.ToLower(str))
		}
	}
	go dl.updateTimestamp()
}

// ======================== Statistics ========================

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
				log.Println("[insyra] DataList.Max(): Data types cannot be compared, returning nil.")
				return nil
			}
		case float64:
			if val, ok := v.(float64); ok && val > maxVal {
				max = val
			} else if intVal, ok := v.(int); ok && float64(intVal) > maxVal {
				max = float64(intVal)
			} else if !ok {
				log.Println("[insyra] DataList.Max(): Data types cannot be compared, returning nil.")
				return nil
			}
		case string:
			if val, ok := v.(string); ok {
				valF64, ok := ToFloat64Safe(val)
				if !ok {
					log.Println("[insyra] DataList.Max(): Data types cannot be compared, returning nil.")
					return nil
				}
				maxValF64, ok := ToFloat64Safe(maxVal)
				if !ok {
					log.Println("[insyra] DataList.Max(): Data types cannot be compared, returning nil.")
					return nil
				}
				if valF64 > maxValF64 {
					max = val
				}
			} else if !ok {
				log.Println("[insyra] DataList.Max(): Data types cannot be compared, returning nil.")
				return nil
			}
		default:
			log.Println("[insyra] DataList.Max(): Data types cannot be compared, returning nil.")
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
		log.Println("[insyra] DataList.Min(): DataList is empty, returning nil.")
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
				log.Println("[insyra] DataList.Min(): Data types cannot be compared, returning nil.")
				return nil
			}
		case float64:
			if val, ok := v.(float64); ok && val < minVal {
				min = val
			} else if intVal, ok := v.(int); ok && float64(intVal) < minVal {
				min = float64(intVal)
			} else if !ok {
				log.Println("[insyra] DataList.Min(): Data types cannot be compared, returning nil.")
				return nil
			}
		case string:
			if val, ok := v.(string); ok && conv.ParseF64(val) < conv.ParseF64(minVal) {
				min = val
			} else if !ok {
				log.Println("[insyra] DataList.Min(): Data types cannot be compared, returning nil.")
				return nil
			}
		default:
			log.Println("[insyra] DataList.Min(): Data types cannot be compared, returning nil.")
			return nil
		}
	}

	return min
}

// Mean calculates the arithmetic mean of the DataList.
// Returns the arithmetic mean.
// Returns nil if the DataList is empty.
// Mean returns the arithmetic mean of the DataList.
func (dl *DataList) Mean() interface{} {
	if len(dl.data) == 0 {
		log.Println("[insyra] DataList.Mean(): DataList is empty, returning nil.")
		return nil
	}

	var sum float64
	for _, v := range dl.data {
		if val, ok := ToFloat64Safe(v); ok {
			sum += val
		} else {
			log.Println("[insyra] DataList.Mean(): Data types cannot be compared, returning nil.")
			return nil
		}
	}

	return sum / float64(len(dl.data))
}

// GMean calculates the geometric mean of the DataList.
// Returns the geometric mean.
// Returns nil if the DataList is empty.
// GMean returns the geometric mean of the DataList.
func (dl *DataList) GMean() interface{} {
	if len(dl.data) == 0 {
		log.Println("[insyra] DataList.GMean(): DataList is empty, returning nil.")
		return nil
	}

	product := 1.0
	for _, v := range dl.data {
		if val, ok := ToFloat64Safe(v); ok {
			product *= val
		} else {
			log.Println("[insyra] DataList.GMean(): Data types cannot be compared, returning nil.")
			return nil
		}
	}

	return math.Pow(product, 1.0/float64(len(dl.data)))
}

// Median calculates the median of the DataList.
// Returns the median.
// Returns nil if the DataList is empty.
// Median returns the median of the DataList.
func (dl *DataList) Median() interface{} {
	if len(dl.data) == 0 {
		log.Println("[insyra] DataList.Median(): DataList is empty, returning nil.")
		return nil
	}

	sortedData := make([]interface{}, len(dl.data))
	copy(sortedData, dl.data)
	dl.Sort()

	mid := len(sortedData) / 2
	if len(sortedData)%2 == 0 {
		return (ToFloat64(sortedData[mid-1]) + ToFloat64(sortedData[mid])) / 2
	}

	return ToFloat64(sortedData[mid])
}

// Mode calculates the mode of the DataList.
// Returns the mode.
// Returns nil if the DataList is empty.
// Mode returns the mode of the DataList.
func (dl *DataList) Mode() interface{} {
	if len(dl.data) == 0 {
		log.Println("[insyra] DataList.Mode(): DataList is empty, returning nil.")
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

	return mode
}

// Stdev calculates the standard deviation(sample) of the DataList.
// Returns the standard deviation.
// Returns nil if the DataList is empty.
// Stdev returns the standard deviation of the DataList.
func (dl *DataList) Stdev() interface{} {
	if len(dl.data) == 0 {
		log.Println("[insyra] DataList.Stdev(): DataList is empty, returning nil.")
		return nil
	}
	variance := dl.Var()
	if variance == nil {
		log.Println("[insyra] DataList.Stdev(): Variance calculation failed, returning nil.")
		return nil
	}
	return math.Sqrt(ToFloat64(variance))
}

// StdevP calculates the standard deviation(population) of the DataList.
// Returns the standard deviation.
// Returns nil if the DataList is empty or the standard deviation cannot be calculated.
func (dl *DataList) StdevP() interface{} {
	if len(dl.data) == 0 {
		log.Println("[insyra] DataList.StdevP(): DataList is empty, returning nil.")
		return nil
	}
	varianceP := dl.VarP()
	if varianceP == nil {
		log.Println("[insyra] DataList.StdevP(): Variance calculation failed, returning nil.")
		return nil
	}
	return math.Sqrt(ToFloat64(varianceP))
}

// Var calculates the variance(sample) of the DataList.
// Returns the variance.
// Returns nil if the DataList is empty or the variance cannot be calculated.
func (dl *DataList) Var() interface{} {
	n := float64(dl.Len())
	if n == 0.0 {
		log.Println("[insyra] DataList.Var(): DataList is empty, returning nil.")
		return nil
	}
	m := dl.Mean()
	mean, ok := ToFloat64Safe(m)
	if !ok {
		log.Println("[insyra] DataList.Var(): Mean is not a float64, returning nil.")
		return nil
	}

	denominator := n - 1
	if denominator == 0 {
		log.Println("[insyra] DataList.Var(): Denominator is 0, returning nil.")
		return nil
	}
	numerator := 0.0
	for i := 0; i < len(dl.data); i++ {
		xi, ok := ToFloat64Safe(dl.data[i])
		if !ok {
			log.Println("[insyra] DataList.Var(): Element is not a float64, returning nil.")
			return nil
		}
		numerator += math.Pow(xi-mean, 2)
	}
	return numerator / denominator
}

// VarP calculates the variance(population) of the DataList.
// Returns the variance.
// Returns nil if the DataList is empty or the variance cannot be calculated.
func (dl *DataList) VarP() interface{} {
	n := float64(dl.Len())
	if n == 0.0 {
		log.Println("[insyra] DataList.VarP(): DataList is empty, returning nil.")
		return nil
	}
	m := dl.Mean()
	mean, ok := ToFloat64Safe(m)
	if !ok {
		log.Println("[insyra] DataList.VarP(): Mean is not a float64, returning nil.")
		return nil
	}
	numerator := 0.0
	for i := 0; i < len(dl.data); i++ {
		xi, ok := ToFloat64Safe(dl.data[i])
		if !ok {
			log.Println("[insyra] DataList.VarP(): Element is not a float64, returning nil.")
			return nil
		}
		numerator += math.Pow(xi-mean, 2)
	}
	return numerator / n
}

// Range calculates the range of the DataList.
// Returns the range.
// Returns nil if the DataList is empty.
// Range returns the range of the DataList.
func (dl *DataList) Range() interface{} {
	if len(dl.data) == 0 {
		log.Println("[insyra] DataList.Range(): DataList is empty, returning nil.")
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
		log.Println("[insyra] DataList.Quartile(): DataList is empty, returning nil.")
		return nil
	}
	if q < 1 || q > 3 {
		log.Println("[insyra] DataList.Quartile(): Invalid quartile value, returning nil.")
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
		log.Println("[insyra] DataList.IQR(): DataList is empty, returning nil.")
		return nil
	}
	q3, ok := ToFloat64Safe(dl.Quartile(3))
	if !ok {
		log.Println("[insyra] DataList.IQR(): Q3 is not a float64, returning nil.")
		return nil
	}
	q1, ok := ToFloat64Safe(dl.Quartile(1))
	if !ok {
		log.Println("[insyra] DataList.IQR(): Q1 is not a float64, returning nil.")
		return nil
	}
	return q3 - q1
}

// Percentile calculates the percentile based on the input value (0 to 100).
// Returns the percentile value, or nil if the DataList is empty.
func (dl *DataList) Percentile(p float64) interface{} {
	if len(dl.data) == 0 {
		log.Println("[insyra] DataList.Percentile(): DataList is empty, returning nil.")
		return nil
	}
	if p < 0 || p > 100 {
		log.Println("[insyra] DataList.Percentile(): Invalid percentile value, returning nil.")
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

// ToF64Slice converts the DataList to a float64 slice.
// Returns the float64 slice.
// Returns nil if the DataList is empty.
// ToF64Slice converts the DataList to a float64 slice.
func (dl *DataList) ToF64Slice() []float64 {
	if len(dl.data) == 0 {
		log.Println("[insyra] DataList.ToF64Slice(): DataList is empty, returning nil.")
		return nil
	}

	floatData := make([]float64, len(dl.data))
	for i, v := range dl.data {
		floatData[i] = ToFloat64(v)
	}

	return floatData
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

func (dl *DataList) SetName(name string) {
	// 未來可限制名稱
	dl.name = name
	go dl.updateTimestamp()
}
