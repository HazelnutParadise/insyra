package insyra

import (
	"fmt"
	"reflect"
	"slices"
	"sort"
	"testing"
	"time"
)

func float64Equal(a, b float64) bool {
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff < 1e-5
}

func IDataListTest(dl IDataList) {
	return
}

func TestIDataList(t *testing.T) {
	defer func() {
		r := recover()
		if r != nil {
			t.Errorf("IDataListTest() panicked with: %v", r)
		}
	}()
	dl := NewDataList()
	IDataListTest(dl)
}

// 測試 NewDataList 函數
func TestNewDataList(t *testing.T) {
	dl := NewDataList(1, 2, 3)
	if dl.Len() != 3 {
		t.Errorf("Expected length 3, got %d", dl.Len())
	}

	expected := []interface{}{1, 2, 3}
	if !reflect.DeepEqual(dl.Data(), expected) {
		t.Errorf("Expected data %v, got %v", expected, dl.Data())
	}
}

// 測試 Append 函數
func TestDataListAppend(t *testing.T) {
	dl := NewDataList()
	dl.Append(1)
	dl.Append(2)

	expected := []interface{}{1, 2}
	if !reflect.DeepEqual(dl.Data(), expected) {
		t.Errorf("Expected data %v, got %v", expected, dl.Data())
	}
}

// 測試 Get 函數
func TestDataListGet(t *testing.T) {
	dl := NewDataList(1, 2, 3)
	val := dl.Get(1)
	if val != 2 {
		t.Errorf("Expected value 2, got %v", val)
	}

	val = dl.Get(-1)
	if val != 3 {
		t.Errorf("Expected value 3 for negative index, got %v", val)
	}

	val = dl.Get(5)
	if val != nil {
		t.Errorf("Expected nil for out of bounds index, got %v", val)
	}
}

// 測試 Update 函數
func TestDataListUpdate(t *testing.T) {
	dl := NewDataList(1, 2, 3)
	dl.Update(1, 5)

	expected := []interface{}{1, 5, 3}
	if !reflect.DeepEqual(dl.Data(), expected) {
		t.Errorf("Expected data %v, got %v", expected, dl.Data())
	}
}

// 測試 InsertAt 函數
func TestDataListInsertAt(t *testing.T) {
	dl := NewDataList(1, 2, 3)
	dl.InsertAt(1, 5)

	expected := []interface{}{1, 5, 2, 3}
	if !reflect.DeepEqual(dl.Data(), expected) {
		t.Errorf("Expected data %v, got %v", expected, dl.Data())
	}

	dl.InsertAt(-1, 7)
	expected = []interface{}{1, 5, 2, 3, 7}
	if !reflect.DeepEqual(dl.Data(), expected) {
		t.Errorf("Expected data %v, got %v", expected, dl.Data())
	}
}

// 測試 FindFirst 函數
func TestDataListFindFirst(t *testing.T) {
	dl := NewDataList(1, 2, 3, 2)
	index := dl.FindFirst(2)

	if index != 1 {
		t.Errorf("Expected first index 1, got %v", index)
	}
}

// 測試 FindLast 函數
func TestDataListFindLast(t *testing.T) {
	dl := NewDataList(1, 2, 3, 2)
	index := dl.FindLast(2)

	if index != 3 {
		t.Errorf("Expected last index 3, got %v", index)
	}
}

func TestDataListFinalAll(t *testing.T) {
	dl := NewDataList(1, 2, 3, 2, 2, 1)
	expected := []int{1, 3, 4}
	indexList := dl.FindAll(2)
	sort.Ints(indexList)
	if !slices.Equal(expected, indexList) {
		t.Errorf("Expected %v, got %v", expected, indexList)
	}
}

func TestDataListFilter(t *testing.T) {
	dl := NewDataList(1, 2, 3, 2, 2, 1)
	dl = dl.Filter(func(value interface{}) bool {
		return value != 2
	})

	expected := []interface{}{1, 3, 1}
	if !reflect.DeepEqual(dl.Data(), expected) {
		t.Errorf("Expected data %v, got %v", expected, dl.Data())
	}
}

func TestDataListReplaceFirst(t *testing.T) {
	dl := NewDataList(1, 2, 3, 2, 2, 1)
	dl.ReplaceFirst(2, 10)

	expected := []interface{}{1, 10, 3, 2, 2, 1}
	if !reflect.DeepEqual(dl.Data(), expected) {
		t.Errorf("Expected data %v, got %v", expected, dl.Data())
	}
}

func TestDataListReplaceLast(t *testing.T) {
	dl := NewDataList(1, 2, 3, 2, 2, 1)
	dl.ReplaceLast(2, 10)

	expected := []interface{}{1, 2, 3, 2, 10, 1}
	if !reflect.DeepEqual(dl.Data(), expected) {
		t.Errorf("Expected data %v, got %v", expected, dl.Data())
	}
}

func TestDataListReplaceAll(t *testing.T) {
	dl := NewDataList(1, 4, 4, 5, 4, 9)
	dl.ReplaceAll(4, 10)

	expected := []interface{}{1, 10, 10, 5, 10, 9}
	if !reflect.DeepEqual(dl.Data(), expected) {
		t.Errorf("Expected data %v, got %v", expected, dl.Data())
	}

	dlstr := NewDataList("a", "b", "c", "a")
	dlstr.ReplaceAll("a", "d")

	expected = []interface{}{"d", "b", "c", "d"}
	if !reflect.DeepEqual(dlstr.Data(), expected) {
		t.Errorf("Expected data %v, got %v", expected, dlstr.Data())
	}
}

func TestDataListReplaceOutliers(t *testing.T) {
	// TODO
}

func TestDataListPop(t *testing.T) {
	dl := NewDataList(1, 2, 3, 4)
	last := dl.Pop()

	if last != 4 {
		t.Errorf("Expected last element 4, got %v", last)
	}

	expected := []interface{}{1, 2, 3}
	if !reflect.DeepEqual(dl.Data(), expected) {
		t.Errorf("Expected data %v, got %v", expected, dl.Data())
	}
}

// 測試 Drop 函數
func TestDataListDrop(t *testing.T) {
	dl := NewDataList(1, 2, 3, 4)
	dl.Drop(2)

	expected := []interface{}{1, 2, 4}
	if !reflect.DeepEqual(dl.Data(), expected) {
		t.Errorf("Expected data %v, got %v", expected, dl.Data())
	}
}

func TestDataListDropAll(t *testing.T) {
	// TODO
}

func TestDataListDropIfContains(t *testing.T) {
	// TODO
}

// 測試 Clear 函數
func TestDataListClear(t *testing.T) {
	dl := NewDataList(1, 2, 3, 4)
	dl.Clear()

	if dl.Len() != 0 {
		t.Errorf("Expected length 0, got %d", dl.Len())
	}
}

func TestDataListClearStrings(t *testing.T) {
	// for small test
	dl := NewDataList("a", "b", "c", 0, 1, 2)

	dl.ClearStrings()
	if dl.Len() != 3 {
		t.Errorf("Expected length 3, got %d", dl.Len())
	}

	var visited [3]bool
	for i := 0; i < 3; i++ {
		visited[dl.Get(i).(int)] = true
	}
	for i := 0; i < 3; i++ {
		if !visited[i] {
			t.Errorf("Expected %d in dataList", i)
		}
	}

	// for large test
	const n = 5000
	dl = NewDataList()
	for i := 0; i < n; i++ {
		dl.Append(fmt.Sprintf("string-%d", i))
	}
	for i := 0; i < n; i++ {
		dl.Append(i)
	}
	dl.ClearStrings()
	var visited2 [n]bool
	for i := 0; i < n; i++ {
		visited2[i] = true
	}
	for i := 0; i < n; i++ {
		if !visited2[i] {
			t.Errorf("Expected %d in dataList", i)
		}
	}
}

func TestDataListClearNumbers(t *testing.T) {
	// TODO
}

func TestDataListClearNaNs(t *testing.T) {
	// TODO
}

func TestDataListClearOutliers(t *testing.T) {
	// TODO
}

func TestDataListNormalize(t *testing.T) {
	// TODO
}

func TestDataListStandardize(t *testing.T) {
	// TODO
}

func TestDataListFillNaNWithMean(t *testing.T) {
	// TODO
}

func TestDataListMovingAverage(t *testing.T) {
	// TODO
}

func TestDataListWeightedMovingAverage(t *testing.T) {
	// TODO
}

func TestDataListExponentialSmoothing(t *testing.T) {
	// TODO
}

func TestDataListDoubleExponentialSmoothing(t *testing.T) {
	// TODO
}

func TestDataListMovingStdev(t *testing.T) {
	// TODO
}

func TestDataListSort(t *testing.T) {
	// TODO
}

func TestDataListRank(t *testing.T) {
	// TODO
}

func TestDataListReverse(t *testing.T) {
	// TODO
}

func TestDataListUpper(t *testing.T) {
	// TODO
}

func TestDataListLower(t *testing.T) {
	// TODO
}

func TestDataListCapitalize(t *testing.T) {
	// TODO
}

// 測試 Sum 函數
func TestDataListSum(t *testing.T) {
	dl := NewDataList(1, 2, 3, 4, "J")
	sum := dl.Sum()

	if !float64Equal(sum, 10) {
		t.Errorf("Expected sum 10, got %v", sum)
	}
}

// 測試 Max 函數
func TestDataListMax(t *testing.T) {
	dl := NewDataList(1, 2, 3, 4)
	max := dl.Max()

	if !float64Equal(max, 4) {
		t.Errorf("Expected max 4, got %v", max)
	}
}

// 測試 Min 函數
func TestDataListMin(t *testing.T) {
	dl := NewDataList(1, 2, 3, 4)
	min := dl.Min()

	if !float64Equal(min, 1) {
		t.Errorf("Expected min 1, got %v", min)
	}
}

// 測試 Mean 函數
func TestDataListMean(t *testing.T) {
	dl := NewDataList(1, 2, 3, int(4), "5")
	mean := dl.Mean()

	if !float64Equal(mean, 2.5) {
		t.Errorf("Expected mean 2.5, got %v", mean)
	}
}

// 測試 WeightedMean 函數
func TestDataListWeightedMean(t *testing.T) {
	dl := NewDataList(1, 2, 3, 4)
	weights := NewDataList(1, 2, 3, 4)
	mean := dl.WeightedMean(weights)

	if !float64Equal(mean, 3) {
		t.Errorf("Expected weighted mean 3, got %v", mean)
	}
}

// 測試 GMean 函數
func TestDataListGMean(t *testing.T) {
	dl := NewDataList(1, 2, 3, "J", 4)
	gmean := dl.GMean()

	if !float64Equal(gmean, 2.213363839400643) {
		t.Errorf("Expected geometric mean 2.213363839400643, got %v", gmean)
	}
}

// 測試 Median 函數
func TestDataListMedian(t *testing.T) {
	dl := NewDataList(1, 2, 3, 4)
	median := dl.Median()

	if !float64Equal(median, 2.5) {
		t.Errorf("Expected median 2.5, got %v", median)
	}
}

// 測試 Mode 函數
func TestDataListMode(t *testing.T) {
	dl := NewDataList(1, 2, 3, 2, 4)
	mode := dl.Mode()

	if !float64Equal(mode, 2) {
		t.Errorf("Expected mode 2, got %v", mode)
	}
}

// 測試 MAD 函數
func TestDataListMAD(t *testing.T) {
	dl := NewDataList(1, 2, 3, 4)
	mad := dl.MAD()

	if !float64Equal(mad, 1) {
		t.Errorf("Expected MAD 1, got %v", mad)
	}
}

// 測試 Stdev 函數
func TestDataListStdev(t *testing.T) {
	dl := NewDataList(1, 2, 3, 4)
	stdev := dl.Stdev()

	if !float64Equal(stdev, 1.2909944487358056) {
		t.Errorf("Expected standard deviation 1.2909944487358056, got %v", stdev)
	}
}

// 測試 StdevP 函數
func TestDataListStdevP(t *testing.T) {
	dl := NewDataList(1, 2, 3, 4)
	stdev := dl.StdevP()

	if !float64Equal(stdev, 1.118033988749895) {
		t.Errorf("Expected population standard deviation 1.118033988749895, got %v", stdev)
	}
}

// 測試 Var 函數
func TestDataListVar(t *testing.T) {
	dl := NewDataList(1, 2, 3, 4)
	variance := dl.Var()

	if !float64Equal(variance, 1.6666666666666667) {
		t.Errorf("Expected variance 1.6666666666666667, got %v", variance)
	}
}

// 測試 VarP 函數
func TestDataListVarP(t *testing.T) {
	dl := NewDataList(1, 2, 3, 4)
	variance := dl.VarP()

	if !float64Equal(variance, 1.25) {
		t.Errorf("Expected population variance 1.25, got %v", variance)
	}
}

// 測試 Range 函數
func TestDataListRange(t *testing.T) {
	dl := NewDataList(1, 2, 3, 4)
	r := dl.Range()

	if !float64Equal(r, 3) {
		t.Errorf("Expected range 3, got %v", r)
	}
}

// 測試 Quantile 函數
func TestDataListQuartile(t *testing.T) {
	dl := NewDataList(6, 47, 49, 15, 42, 41, 7, 39, 43, 40, 36)
	q := dl.Quartile(3)

	if !float64Equal(q, 43) {
		t.Errorf("Expected quartile 43, got %v", q)
	}
}

// 測試 IQR 函數
func TestDataListIQR(t *testing.T) {
	dl := NewDataList(6, 47, 49, 15, 42, 41, 7, 39, 43, 40, 36)
	iqr := dl.IQR()

	if !float64Equal(iqr, 28) {
		t.Errorf("Expected IQR 28, got %v", iqr)
	}
}

// 測試 Percentile 函數
func TestDataListPercentile(t *testing.T) {
	dl := NewDataList(10, 2, 38, 23, 38, 23, 21, 234)
	p := dl.Percentile(15)

	if !float64Equal(p, 10.55) {
		t.Errorf("Expected percentile 10.55, got %v", p)
	}
}

func TestDataListDifference(t *testing.T) {
	// Todo
}

func TestDataListIsEqualTo(t *testing.T) {
	dl1 := NewDataList("M", 2, "3", 4.9)
	dl2 := NewDataList("M", 2, "3", 4.9)

	if !dl1.IsEqualTo(dl2) {
		t.Errorf("Expected dl1 and dl2 to be equal, but they are not")
	}
}

func TestDataListIsTheSameAs(t *testing.T) {
	dl1 := NewDataList("M", 2, "3", 4.9)
	dl2 := NewDataList("M", 2, "3", 8)
	dl2.lastModifiedTimestamp.Add(1)
	dl2.ReplaceLast(8, 4.9)

	if dl1.IsTheSameAs(dl2) {
		t.Errorf("Expected dl1 and dl2 not to be the same, but they are")
	}
}

func TestDataListParseNumbers(t *testing.T) {
	dl := NewDataList("1", 2, "3", 8)
	dl = dl.ParseNumbers()

	if !reflect.DeepEqual(dl.Data(), []interface{}{1.0, 2.0, 3.0, 8.0}) {
		t.Errorf("Expected data %v, got %v", []interface{}{1.0, 2.0, 3.0, 8.0}, dl.Data())
	}
}

func TestDataListParseStrings(t *testing.T) {
	dl := NewDataList("1", 2, "3", 8)
	dl = dl.ParseStrings()

	if !reflect.DeepEqual(dl.Data(), []interface{}{"1", "2", "3", "8"}) {
		t.Errorf("Expected data %v, got %v", []interface{}{"1", "2", "3", "8"}, dl.Data())
	}
}

func TestDataListToF64Slice(t *testing.T) {
	dl := NewDataList(1.9, 2, 3, 4)
	slice := dl.ToF64Slice()

	if !reflect.DeepEqual(slice, []float64{1.9, 2, 3, 4}) {
		t.Errorf("Expected float64 slice %v, got %v", []float64{1.9, 2, 3, 4}, slice)
	}
}

func TestDataListToStringSlice(t *testing.T) {
	dl := NewDataList(1, 2, 3, 4)
	slice := dl.ToStringSlice()

	if !reflect.DeepEqual(slice, []string{"1", "2", "3", "4"}) {
		t.Errorf("Expected string slice %v, got %v", []string{"1", "2", "3", "4"}, slice)
	}
}

func TestDataListGetCreationTimestamp(t *testing.T) {
	dl := NewDataList(1, 2, 3, 4)
	time.Sleep(1 * time.Second)
	newTime := time.Now().Unix()
	dl.SetName("TestName2")

	if dl.GetCreationTimestamp() == newTime || dl.GetCreationTimestamp() == 0 {
		t.Errorf("Creation timestamp wrong, got %v", dl.GetCreationTimestamp())
	}
}

func TestDataListGetLastModifiedTimestamp(t *testing.T) {
	// TODO
}

func TestDataListGetName(t *testing.T) {
	dl := NewDataList(1, 2, 3, 4)
	dl.SetName("TestName")

	if dl.GetName() != "TestName" {
		t.Errorf("Expected name TestName, got %v", dl.GetName())
	}
}

// 測試 SetName 和 GetName 函數
func TestDataListSetName(t *testing.T) {
	dl := NewDataList(1, 2, 3, 4)
	dl.SetName("TestName")

	if dl.GetName() != "TestName" {
		t.Errorf("Expected name TestName, got %v", dl.GetName())
	}
}

func TestDataListCounter(t *testing.T) {
	dl := NewDataList(1, "k", "k", 4, 4, 6, 7, "9", "9", 10, "4")
	counter := dl.Counter()

	if counter[1] != 1 || counter["k"] != 2 || counter[4] != 2 || counter["9"] != 2 || counter[10] != 1 || counter["4"] != 1 {
		t.Errorf("Expected counter %v, got %v", map[interface{}]int{1: 1, "k": 2, 4: 2, "9": 2, 10: 1, "4": 1}, counter)
	}
}

func TestDataListCount(t *testing.T) {
	dl := NewDataList(1, "k", "k", 4, 4, 6, 7, "9", "9", 10, "4")
	count := dl.Count(4)

	if count != 2 {
		t.Errorf("Expected count 2, got %v", count)
	}
}
