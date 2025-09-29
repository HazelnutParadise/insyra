// `csvxl` package provides functions for converting between CSV and Excel formats.
//
// Key Features:
// - Convert multiple CSV files to a single Excel workbook with separate sheets
// - Split Excel workbooks into multiple CSV files
// - Automatic encoding detection for CSV files
// - Support for UTF-8, Big5, and other common text encodings
// - Batch processing of files in directories
//
// Encoding Support:
// The package now supports automatic encoding detection by default. When no encoding
// is explicitly specified, the package will automatically detect the encoding of CSV files
// using charset detection algorithms. Supported encodings include:
// - UTF-8 (default fallback)
// - Big5 (Traditional Chinese)
// - ASCII
// - Various other encodings with automatic fallback to UTF-8
//
// You can still explicitly specify an encoding using the constants:
// - csvxl.UTF8 for UTF-8 encoding
// - csvxl.Big5 for Big5 encoding  
// - csvxl.Auto for automatic detection (default)
//
// Example usage:
//   // Convert CSV files to Excel with auto-detection (default)
//   csvxl.CsvToExcel([]string{"file1.csv", "file2.csv"}, nil, "output.xlsx")
//
//   // Convert with explicit encoding
//   csvxl.CsvToExcel([]string{"file1.csv"}, nil, "output.xlsx", csvxl.UTF8)
//
//   // Detect encoding of a specific file
//   encoding, err := csvxl.DetectEncoding("myfile.csv")
package csvxl

func init() {}
