package insyra

import (
	"math"
)

func (dl *DataList) replaceAll_notAtomic(oldValue, newValue any) {
	defer func() {
		go dl.updateTimestamp()
	}()

	isOldValueNaN := false
	if val, ok := oldValue.(float64); ok && math.IsNaN(val) {
		isOldValueNaN = true
	}

	// 單線程處理資料替換
	for i, v := range dl.data {
		if isOldValueNaN {
			if val, ok := v.(float64); ok && math.IsNaN(val) {
				dl.data[i] = newValue
			}
		} else if v == oldValue {
			dl.data[i] = newValue
		}
	}
}

func (dl *DataList) replaceFirst_notAtomic(oldValue, newValue any) {
	defer func() {
		go dl.updateTimestamp()
	}()

	isOldValueNaN := false
	if val, ok := oldValue.(float64); ok && math.IsNaN(val) {
		isOldValueNaN = true
	}
	// 單線程處理資料替換
	for i, v := range dl.data {
		if !isOldValueNaN && v == oldValue {
			dl.data[i] = newValue
			return
		}
		if isOldValueNaN {
			if val, ok := v.(float64); ok && math.IsNaN(val) {
				dl.data[i] = newValue
				return
			}
		}
	}
}

func (dl *DataList) replaceLast_notAtomic(oldValue, newValue any) {
	defer func() {
		go dl.updateTimestamp()
	}()
	isOldValueNaN := false
	if val, ok := oldValue.(float64); ok && math.IsNaN(val) {
		isOldValueNaN = true
	}
	// 單線程處理資料替換
	for i := len(dl.data) - 1; i >= 0; i-- {
		if !isOldValueNaN && dl.data[i] == oldValue {
			dl.data[i] = newValue
			return
		} else if isOldValueNaN {
			if val, ok := dl.data[i].(float64); ok && math.IsNaN(val) {
				dl.data[i] = newValue
				return
			}
		}
	}
}

func (dl *DataList) replaceNaNsAndNilsWith_notAtomic(value any) {
	defer func() {
		go dl.updateTimestamp()
	}()

	for i, v := range dl.data {
		if v == nil {
			dl.data[i] = value
		} else if val, ok := v.(float64); ok && math.IsNaN(val) {
			dl.data[i] = value
		}
	}
}
