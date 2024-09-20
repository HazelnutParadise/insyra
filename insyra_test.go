package insyra

import (
	"fmt"
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
