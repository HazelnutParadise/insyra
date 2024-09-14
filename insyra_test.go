package insyra

import (
	"fmt"
	"reflect"
	"testing"
)

func TestProcessData(t *testing.T) {
	dl := NewDataList(1, 2, 3)
	_, ok := interface{}(dl).(IDataList)
	if !ok {
		fmt.Println("DataList does not implement IDataList")
	} else {
		fmt.Println("DataList implements IDataList")
	}
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
func TestAppend(t *testing.T) {
	dl := NewDataList()
	dl.Append(1)
	dl.Append(2)

	expected := []interface{}{1, 2}
	if !reflect.DeepEqual(dl.Data(), expected) {
		t.Errorf("Expected data %v, got %v", expected, dl.Data())
	}
}

// 測試 Get 函數
func TestGet(t *testing.T) {
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
func TestUpdate(t *testing.T) {
	dl := NewDataList(1, 2, 3)
	dl.Update(1, 5)

	expected := []interface{}{1, 5, 3}
	if !reflect.DeepEqual(dl.Data(), expected) {
		t.Errorf("Expected data %v, got %v", expected, dl.Data())
	}
}

// 測試 InsertAt 函數
func TestInsertAt(t *testing.T) {
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
func TestFindFirst(t *testing.T) {
	dl := NewDataList(1, 2, 3, 2)
	index := dl.FindFirst(2)

	if index != 1 {
		t.Errorf("Expected first index 1, got %v", index)
	}
}

// 測試 FindLast 函數
func TestFindLast(t *testing.T) {
	dl := NewDataList(1, 2, 3, 2)
	index := dl.FindLast(2)

	if index != 3 {
		t.Errorf("Expected last index 3, got %v", index)
	}
}

// 測試 Drop 函數
func TestDrop(t *testing.T) {
	dl := NewDataList(1, 2, 3, 4)
	dl.Drop(2)

	expected := []interface{}{1, 2, 4}
	if !reflect.DeepEqual(dl.Data(), expected) {
		t.Errorf("Expected data %v, got %v", expected, dl.Data())
	}
}

// 測試 Clear 函數
func TestClear(t *testing.T) {
	dl := NewDataList(1, 2, 3, 4)
	dl.Clear()

	if dl.Len() != 0 {
		t.Errorf("Expected length 0, got %d", dl.Len())
	}
}

func float64Equal(a, b float64) bool {
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff < 1e-5
}

// 測試 Max 函數
func TestMax(t *testing.T) {
	dl := NewDataList(1, 2, 3, 4)
	max := dl.Max()

	if !float64Equal(max, 4) {
		t.Errorf("Expected max 4, got %v", max)
	}
}

// 測試 Min 函數
func TestMin(t *testing.T) {
	dl := NewDataList(1, 2, 3, 4)
	min := dl.Min()

	if !float64Equal(min, 1) {
		t.Errorf("Expected min 1, got %v", min)
	}
}

// 測試 Sum 函數
func TestSum(t *testing.T) {
	dl := NewDataList(1, 2, 3, 4, "J")
	sum := dl.Sum()

	if !float64Equal(sum, 10) {
		t.Errorf("Expected sum 10, got %v", sum)
	}
}

// 測試 Mean 函數
func TestMean(t *testing.T) {
	dl := NewDataList(1, 2, 3, int(4), "5")
	mean := dl.Mean()

	if !float64Equal(mean, 2.5) {
		t.Errorf("Expected mean 2.5, got %v", mean)
	}
}

// 測試 WeightedMean 函數
func TestWeightedMean(t *testing.T) {
	dl := NewDataList(1, 2, 3, 4)
	weights := NewDataList(1, 2, 3, 4)
	mean := dl.WeightedMean(weights)

	if !float64Equal(mean, 3) {
		t.Errorf("Expected weighted mean 3, got %v", mean)
	}
}

// 測試 GMean 函數
func TestGMean(t *testing.T) {
	dl := NewDataList(1, 2, 3, "J", 4)
	gmean := dl.GMean()

	if !float64Equal(gmean, 2.213363839400643) {
		t.Errorf("Expected geometric mean 2.213363839400643, got %v", gmean)
	}
}

// 測試 Median 函數
func TestMedian(t *testing.T) {
	dl := NewDataList(1, 2, 3, 4)
	median := dl.Median()

	if !float64Equal(median, 2.5) {
		t.Errorf("Expected median 2.5, got %v", median)
	}
}

// 測試 Mode 函數
func TestMode(t *testing.T) {
	dl := NewDataList(1, 2, 3, 2, 4)
	mode := dl.Mode()

	if !float64Equal(mode, 2) {
		t.Errorf("Expected mode 2, got %v", mode)
	}
}

// 測試 MAD 函數
func TestMAD(t *testing.T) {
	dl := NewDataList(1, 2, 3, 4)
	mad := dl.MAD()

	if !float64Equal(mad, 1) {
		t.Errorf("Expected MAD 1, got %v", mad)
	}
}

// 測試 Stdev 函數
func TestStdev(t *testing.T) {
	dl := NewDataList(1, 2, 3, 4)
	stdev := dl.Stdev()

	if !float64Equal(stdev, 1.2909944487358056) {
		t.Errorf("Expected standard deviation 1.2909944487358056, got %v", stdev)
	}
}

// 測試 StdevP 函數
func TestStdevP(t *testing.T) {
	dl := NewDataList(1, 2, 3, 4)
	stdev := dl.StdevP().(float64)

	if !float64Equal(stdev, 1.118033988749895) {
		t.Errorf("Expected population standard deviation 1.118033988749895, got %v", stdev)
	}
}

// 測試 Var 函數
func TestVar(t *testing.T) {
	dl := NewDataList(1, 2, 3, 4)
	variance := dl.Var().(float64)

	if !float64Equal(variance, 1.6666666666666667) {
		t.Errorf("Expected variance 1.6666666666666667, got %v", variance)
	}
}

// 測試 VarP 函數
func TestVarP(t *testing.T) {
	dl := NewDataList(1, 2, 3, 4)
	variance := dl.VarP().(float64)

	if !float64Equal(variance, 1.25) {
		t.Errorf("Expected population variance 1.25, got %v", variance)
	}
}

// 測試 SetName 和 GetName 函數
func TestSetName(t *testing.T) {
	dl := NewDataList(1, 2, 3, 4)
	dl.SetName("TestName")

	if dl.GetName() != "TestName" {
		t.Errorf("Expected name TestName, got %v", dl.GetName())
	}
}
