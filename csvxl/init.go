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
// - UTF-8 (most common; used as fallback for certain detected encodings)
// - Big5 (Traditional Chinese)
// - ASCII
// - Various other encodings. Unknown or unrecognized encodings will be reported as errors by `DetectEncoding`.
//
// You can still explicitly specify an encoding using the constants:
// - csvxl.UTF8 for UTF-8 encoding
// - csvxl.Big5 for Big5 encoding
// - csvxl.Auto for automatic detection (default)
//
// Example usage:
//
//	// Convert CSV files to Excel with auto-detection (default)
//	if err := csvxl.CsvToExcel([]string{"file1.csv", "file2.csv"}, nil, "output.xlsx"); err != nil {
//	    panic(err)
//	}
//
//	// Convert with explicit encoding
//	if err := csvxl.CsvToExcel([]string{"file1.csv"}, nil, "output.xlsx", csvxl.UTF8); err != nil {
//	    panic(err)
//	}
//
//	// Detect encoding of a specific file
//	encoding, err := csvxl.DetectEncoding("myfile.csv")
//	if err != nil {
//		// Unknown encoding or detection error; handle appropriately
//	}
package csvxl

func init() {}
