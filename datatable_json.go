package insyra

import (
	"fmt"
	"os"

	"github.com/HazelnutParadise/insyra/internal/utils"
	json "github.com/goccy/go-json"
)

// buildJSONRows builds the row-oriented representation used by the ToJSON
// family directly from the ordered column slice, so columns that share a name
// are not silently dropped (dt.Data collapses duplicate names into one map
// entry). Duplicate keys are disambiguated with a numeric suffix.
func (dt *DataTable) buildJSONRows(useColNames bool) []map[string]any {
	var rows []map[string]any
	dt.AtomicDo(func(dt *DataTable) {
		n := len(dt.columns)
		keys := make([]string, n)
		used := make(map[string]int, n)
		for i, col := range dt.columns {
			key := ""
			if useColNames && col.name != "" {
				key = col.name
			} else {
				key, _ = utils.CalcColIndex(i)
			}
			base := key
			for suffix := 2; used[key] > 0; suffix++ {
				key = fmt.Sprintf("%s_%d", base, suffix)
			}
			used[key] = 1
			keys[i] = key
		}
		maxLen := dt.getMaxColLength()
		rows = make([]map[string]any, maxLen)
		for r := 0; r < maxLen; r++ {
			row := make(map[string]any, n)
			for i, col := range dt.columns {
				if r < len(col.data) {
					row[keys[i]] = col.data[r]
				} else {
					row[keys[i]] = nil
				}
			}
			rows[r] = row
		}
	})
	return rows
}

// ToJSON converts the DataTable to JSON format and writes it to the provided file path.
// The function accepts two parameters:
// - filePath: the file path to write the JSON file to.
// - useColName: if true, the column names will be used as keys in the JSON object, otherwise the column index(A, B, C...) will be used.
// Every row will be a JSON object with the column names as keys and the row values as values.
// The function returns an error if the file cannot be created or the JSON data cannot be written to the file.
func (dt *DataTable) ToJSON(filePath string, useColNames bool) error {
	rows := dt.buildJSONRows(useColNames)

	jsonData, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return err
	}

	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	_, err = file.Write(jsonData)
	if err != nil {
		return err
	}

	return nil
}

// ToJSON_Byte converts the DataTable to JSON format and returns it as a byte slice.
// The function accepts one parameter:
// - useColName: if true, the column names will be used as keys in the JSON object, otherwise the column index(A, B, C...) will be used.
// Every row will be a JSON object with the column names as keys and the row values as values.
// The function returns the JSON data as a byte slice.
func (dt *DataTable) ToJSON_Bytes(useColNames bool) []byte {
	rows := dt.buildJSONRows(useColNames)

	jsonData, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		dt.warn("ToJSON_Byte", "%v", err)
		return nil
	}

	return jsonData
}

// ToJSON_String converts the DataTable to JSON format and returns it as a string.
// The function accepts one parameter:
// - useColName: if true, the column names will be used as keys in the JSON object, otherwise the column index(A, B, C...) will be used.
// Every row will be a JSON object with the column names as keys and the row values as values.
// The function returns the JSON data as a string.
func (dt *DataTable) ToJSON_String(useColNames bool) string {
	jsonBytes := dt.ToJSON_Bytes(useColNames)
	return string(jsonBytes)
}
