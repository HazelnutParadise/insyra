package pd

import (
	"fmt"

	"github.com/HazelnutParadise/insyra"
	gpdc "github.com/apoplexi24/gpandas/utils/collection"
)

type Series struct {
	gpdc.Series
}

func FromDataList(dl insyra.IDataList) (*Series, error) {
	if dl == nil {
		return nil, fmt.Errorf("nil DataList")
	}
	if dl.Len() == 0 {
		return nil, fmt.Errorf("empty DataList")
	}

	listType := ""
	int64List := []int64{}
	float64List := []float64{}
	stringList := []string{}
	for _, val := range dl.Data() {
		tryAppendValue(val, &listType, &int64List, &float64List, &stringList)
	}

	switch listType {
	case "int":
		gpds, err := gpdc.NewInt64SeriesFromData(int64List, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create int Series: %v", err)
		}
		return &Series{gpds}, nil
	case "float":
		gpds, err := gpdc.NewFloat64SeriesFromData(float64List, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create float Series: %v", err)
		}
		return &Series{gpds}, nil
	case "string":
		gpds, err := gpdc.NewStringSeriesFromData(stringList, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create string Series: %v", err)
		}
		return &Series{gpds}, nil
	default:
		gpds, err := gpdc.NewAnySeriesFromData(dl.Data(), nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create any Series: %v", err)
		}
		return &Series{gpds}, nil
	}
}

func tryAppendValue(val any, listType *string, int64List *[]int64, float64List *[]float64, stringList *[]string) {
	switch v := val.(type) {
	case int:
		if *listType != "" && *listType != "int" {
			*listType = "any"
			return
		}
		*listType = "int"
		*int64List = append(*int64List, int64(v))
	case int8:
		if *listType != "" && *listType != "int" {
			*listType = "any"
			return
		}
		*listType = "int"
		*int64List = append(*int64List, int64(v))
	case int16:
		if *listType != "" && *listType != "int" {
			*listType = "any"
			return
		}
		*listType = "int"
		*int64List = append(*int64List, int64(v))
	case int32:
		if *listType != "" && *listType != "int" {
			*listType = "any"
			return
		}
		*listType = "int"
		*int64List = append(*int64List, int64(v))
	case int64:
		if *listType != "" && *listType != "int" {
			*listType = "any"
			return
		}
		*listType = "int"
		*int64List = append(*int64List, v)
	case float32:
		if *listType != "" && *listType != "float" {
			*listType = "any"
			return
		}
		*listType = "float"
		*float64List = append(*float64List, float64(v))
	case float64:
		if *listType != "" && *listType != "float" {
			*listType = "any"
			return
		}
		*listType = "float"
		*float64List = append(*float64List, v)
	case string:
		if *listType != "" && *listType != "string" {
			*listType = "any"
			return
		}
		*listType = "string"
		*stringList = append(*stringList, v)
	default:
		*listType = "any"
	}
}

func FromGPandasSeries(gpds gpdc.Series) (*Series, error) {
	if gpds == nil {
		return nil, fmt.Errorf("nil gpandas Series")
	}
	return &Series{gpds}, nil
}

// ToDataList converts the Series into an `insyra.DataList`, copying values
// and preserving `nil` entries. Returns an error if the receiver or the
// underlying gpandas Series is nil.
func (s *Series) ToDataList() (*insyra.DataList, error) {
	if s == nil || s.Series == nil {
		return nil, fmt.Errorf("nil Series")
	}
	data := s.ValuesCopy()
	dl := insyra.NewDataList(data)
	return dl, nil
}
