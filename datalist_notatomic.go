package insyra

import (
	"math"
)

// The replace helpers unwrap a Cell mark on the new value here, not only in
// the public methods, because the DataTable replace methods call them
// directly. valueMatcher unwraps the old value.

func (dl *DataList) replaceAll_notAtomic(oldValue, newValue any) {
	defer dl.updateTimestamp()
	newValue = unwrapCell(newValue)

	matches := valueMatcher(oldValue)
	for i, v := range dl.data {
		if matches(v) {
			dl.data[i] = newValue
		}
	}
}

func (dl *DataList) replaceFirst_notAtomic(oldValue, newValue any) {
	defer dl.updateTimestamp()
	newValue = unwrapCell(newValue)

	matches := valueMatcher(oldValue)
	for i, v := range dl.data {
		if matches(v) {
			dl.data[i] = newValue
			return
		}
	}
}

func (dl *DataList) replaceLast_notAtomic(oldValue, newValue any) {
	defer dl.updateTimestamp()
	newValue = unwrapCell(newValue)
	matches := valueMatcher(oldValue)
	for i := len(dl.data) - 1; i >= 0; i-- {
		if matches(dl.data[i]) {
			dl.data[i] = newValue
			return
		}
	}
}

func (dl *DataList) replaceNaNsAndNilsWith_notAtomic(value any) {
	defer dl.updateTimestamp()
	value = unwrapCell(value)

	for i, v := range dl.data {
		if v == nil {
			dl.data[i] = value
		} else if val, ok := v.(float64); ok && math.IsNaN(val) {
			dl.data[i] = value
		}
	}
}
