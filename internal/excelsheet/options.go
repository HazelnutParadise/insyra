package excelsheet

import "github.com/xuri/excelize/v2"

// unzipSizeLimit bounds how much an Excel file may expand to while it is
// being read. excelize's default is 16 GB, which is no protection at all
// against a zip bomb: a few kilobytes on disk can decompress into more memory
// than the host has. 512 MB is far above any real spreadsheet.
const unzipSizeLimit = 512 << 20

// Options is the excelize configuration every workbook insyra opens is read
// with: the excel package and csvxl alike.
func Options() excelize.Options {
	return excelize.Options{UnzipSizeLimit: unzipSizeLimit}
}
