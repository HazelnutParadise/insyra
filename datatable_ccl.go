package insyra

func (dt *DataTable) AddColUsingCCL(newColName, ccl string) *DataTable {
	slice, err := ApplyCCLOnDataTable(dt, ccl)
	if err != nil {
		LogWarning("DataTable.AddColUsingCCL: Failed to apply CCL on DataTable: %v", err)
		return dt
	}
	dt.AppendCols(NewDataList(slice...).SetName(newColName))
	return dt
}
