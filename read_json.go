package insyra

import (
	"fmt"
	"os"
)

// ReadJSON reads a JSON file and loads the data into a DataTable and returns it.
func ReadJSON(filePath string) (*DataTable, error) {
	dt := NewDataTable()

	// 讀取檔案
	buf, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	// 解析 JSON
	var rows []map[string]any
	err = json.Unmarshal(buf, &rows)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	// 將資料加入 DataTable
	for _, row := range rows {
		dt.AppendRowsByColName(row)
	}

	return dt, nil
}

func ReadJSON_Bytes(data []byte) (*DataTable, error) {
	dt := NewDataTable()

	// 解析 JSON
	var rows []map[string]any
	err := json.Unmarshal(data, &rows)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	// 將資料加入 DataTable
	for _, row := range rows {
		dt.AppendRowsByColName(row)
	}

	return dt, nil
}
