package insyra

// ======================== Name ========================

// GetName returns the name of the DataTable.
func (dt *DataTable) GetName() string {
	var result string
	dt.AtomicDo(func(dt *DataTable) {
		result = dt.name
	})
	return result
}

// SetName sets the name of the DataTable.
func (dt *DataTable) SetName(name string) *DataTable {
	var result *DataTable
	dt.AtomicDo(func(dt *DataTable) {
		dt.name = name
		result = dt
		go dt.updateTimestamp()
	})
	return result
}
