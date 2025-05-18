package insyra

// ======================== Name ========================

// GetName returns the name of the DataTable.
func (dt *DataTable) GetName() string {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	return dt.name
}

// SetName sets the name of the DataTable.
func (dt *DataTable) SetName(name string) *DataTable {
	dt.mu.Lock()
	defer func() {
		dt.mu.Unlock()
		go dt.updateTimestamp()
	}()
	dt.name = name
	return dt
}
