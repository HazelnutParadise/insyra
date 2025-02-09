package insyra

// ======================== Memory Compaction Mechanism ========================

// reorganizeMemory compacts the memory for a given DataList.
func reorganizeMemory(dl *DataList) {
	if !dl.isFragmented() {
		return
	}
	dl.mu.Lock()
	defer dl.mu.Unlock()

	newData := make([]any, len(dl.data))
	copy(newData, dl.data)
	dl.data = newData
	LogDebug("MemoryManager.ReorganizeMemory(): DataList reorganized with len=%d, cap=%d\n", len(dl.data), cap(dl.data))
}

func (dl *DataList) isFragmented() bool {
	LogDebug("DataList.isFragmented(): len=%d, cap=%d\n", len(dl.data), cap(dl.data))
	return len(dl.data) < cap(dl.data)/2
}
