package insyra

// Map applies a function to all elements in the DataList and returns a new DataList with the results.
// The mapFunc should take an element of any type and its index, then return a transformed value of any type.
func (dl *DataList) Map(mapFunc func(int, any) any) *DataList {
	defer func() {
		dl.mu.Unlock()
	}()
	dl.mu.Lock()

	if len(dl.data) == 0 {
		LogWarning("DataList", "Map", "DataList is empty, returning empty DataList")
		return NewDataList()
	}

	mappedData := make([]any, len(dl.data))

	for i, v := range dl.data {
		func() {
			defer func() {
				if r := recover(); r != nil {
					LogWarning("DataList", "Map", "Error applying function to element at index %d: %v, keeping original value", i, r)
					mappedData[i] = v // 保留原始值
				}
			}()

			mappedData[i] = mapFunc(i, v)
		}()
	}

	return NewDataList(mappedData...)
}
