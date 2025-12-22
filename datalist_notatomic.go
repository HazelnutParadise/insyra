package insyra

func (dl *DataList) replaceAll_notAtomic(oldValue, newValue any) *DataList {
	length := len(dl.data)
	if length == 0 {
		LogWarning("DataList", "ReplaceAll", "DataList is empty, no replacements made.")
		return dl
	}

	// 單線程處理資料替換
	for i, v := range dl.data {
		if v == oldValue {
			dl.data[i] = newValue
		}
	}

	go dl.updateTimestamp()
	return dl
}
