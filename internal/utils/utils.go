package utils

func ParseColIndex(colName string) int {
	result := 0
	for _, char := range colName {
		result = result*26 + int(char-'A') + 1
	}
	return result - 1
}
