package insyra

import (
	"os"

	json "github.com/goccy/go-json"
)

// ToJSON converts the DataTable to JSON format and writes it to the provided file path.
// The function accepts two parameters:
// - filePath: the file path to write the JSON file to.
// - useColName: if true, the column names will be used as keys in the JSON object, otherwise the column index(A, B, C...) will be used.
// Every row will be a JSON object with the column names as keys and the row values as values.
// The function returns an error if the file cannot be created or the JSON data cannot be written to the file.
func (dt *DataTable) ToJSON(filePath string, useColNames bool) error {
	data := dt.Data(useColNames)
	columns := []string{}
	for col := range data {
		columns = append(columns, col)
	}

	rows := []map[string]any{}

	maxColLength := 0
	for _, colData := range data {
		if len(colData) > maxColLength {
			maxColLength = len(colData)
		}
	}

	for i := 0; i < maxColLength; i++ {
		row := make(map[string]any)
		for _, col := range columns {
			key := col
			if i < len(data[col]) {
				row[key] = data[col][i]
			} else {
				row[key] = nil
			}
		}
		rows = append(rows, row)
	}

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
	data := dt.Data(useColNames)
	columns := []string{}
	for col := range data {
		columns = append(columns, col)
	}

	rows := []map[string]any{}

	maxColLength := 0
	for _, colData := range data {
		if len(colData) > maxColLength {
			maxColLength = len(colData)
		}
	}

	for i := range maxColLength {
		row := make(map[string]any)
		for _, col := range columns {
			key := col
			if i < len(data[col]) {
				row[key] = data[col][i]
			} else {
				row[key] = nil
			}
		}
		rows = append(rows, row)
	}

	jsonData, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		LogWarning("DataTable", "ToJSON_Byte", "%v", err)
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
