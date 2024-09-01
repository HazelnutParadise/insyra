package insyra

import (
	"sync"
	"time"
)

type DataTable struct {
	mu                    sync.Mutex
	columns               map[string]*DataList
	customIndex           []string
	creationTimestamp     int64
	lastModifiedTimestamp int64
}

type IDataTable interface {
	// AddColumns adds columns to the DataTable.
	AddColumns(columns ...*DataList)
	updateTimestamp()
	updateColumnNames()
}

// NewDataTable creates a new DataTable.
func NewDataTable(columns ...*DataList) *DataTable {
	newTable := &DataTable{
		columns:               make(map[string]*DataList),
		creationTimestamp:     time.Now().Unix(),
		lastModifiedTimestamp: time.Now().Unix(),
	}
	if len(columns) > 0 {
		newTable.AddColumns(columns...)
	}
	return newTable
}

// AddColumns adds columns to the DataTable.
func (dt *DataTable) AddColumns(columns ...*DataList) {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	for i, column := range columns {
		columnName := generateColumnName(i)
		dt.columns[columnName] = column
	}
}

// updateTimestamp updates the last modified timestamp of the DataTable.
func (dt *DataTable) updateTimestamp() {
	dt.lastModifiedTimestamp = time.Now().Unix()
}

// updateColumnNames updates the column names (keys in the map) to be in sequential order.
// Auto update timestamp.
func (dt *DataTable) updateColumnNames() {
	defer func() {
		go dt.updateTimestamp()
	}()
	updatedColumns := make(map[string]*DataList)
	i := 0
	for _, column := range dt.columns {
		columnName := generateColumnName(i)
		updatedColumns[columnName] = column
		i++
	}
	dt.columns = updatedColumns
}

func generateColumnName(index int) string {
	name := ""
	for index >= 0 {
		name = string('A'+(index%26)) + name
		index = index/26 - 1
	}
	return name
}
