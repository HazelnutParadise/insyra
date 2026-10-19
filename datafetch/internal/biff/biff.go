// Package biff reads the cells of the BIFF8 workbooks (.xls, Excel 97-2003)
// that some data providers still publish. It reads only what a table of values
// needs: sheet names, numbers, text and the cached results of formulas. Styles,
// dates, merged ranges and everything written after the first workbook
// substream are ignored.
package biff

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"unicode/utf16"
)

// Cell is one value of a sheet: a number or a string. Blank cells are not
// stored.
type Cell struct {
	Number   float64
	Text     string
	IsNumber bool
}

// Sheet is one worksheet.
type Sheet struct {
	Name  string
	cells map[[2]int]Cell
	rows  int
}

// Rows is one past the last row that holds a value.
func (s *Sheet) Rows() int { return s.rows }

// Cell returns the cell at the zero-based row and column and whether it holds
// a value.
func (s *Sheet) Cell(row, col int) (Cell, bool) {
	c, ok := s.cells[[2]int{row, col}]
	return c, ok
}

// Workbook is the sheets of a file.
type Workbook struct {
	Sheets []*Sheet
}

// Read parses a BIFF8 workbook held in an OLE2 compound file.
func Read(data []byte) (wb *Workbook, err error) {
	defer func() {
		if r := recover(); r != nil {
			wb, err = nil, fmt.Errorf("biff: malformed file: %v", r)
		}
	}()
	stream, err := workbookStream(data)
	if err != nil {
		return nil, err
	}
	return parseWorkbook(stream)
}

const (
	recBOF        = 0x0809
	recEOF        = 0x000a
	recBoundSheet = 0x0085
	recSST        = 0x00fc
	recContinue   = 0x003c
	recLabelSST   = 0x00fd
	recLabel      = 0x0204
	recNumber     = 0x0203
	recRK         = 0x027e
	recMulRK      = 0x00bd
	recFormula    = 0x0006
	recString     = 0x0207
)

type record struct {
	id   uint16
	data []byte
}

// records splits a stream into its records, joining the CONTINUE records that
// follow an SST into one record.
func records(stream []byte, from int) []record {
	var out []record
	for pos := from; pos+4 <= len(stream); {
		id := binary.LittleEndian.Uint16(stream[pos:])
		n := int(binary.LittleEndian.Uint16(stream[pos+2:]))
		if pos+4+n > len(stream) {
			break
		}
		out = append(out, record{id, stream[pos+4 : pos+4+n]})
		pos += 4 + n
	}
	return out
}

func parseWorkbook(stream []byte) (*Workbook, error) {
	recs := records(stream, 0)
	if len(recs) == 0 || recs[0].id != recBOF {
		return nil, errors.New("biff: the workbook stream does not start with a BOF record")
	}
	type sheetInfo struct {
		name   string
		offset int
	}
	var infos []sheetInfo
	var sst []string
	for i := 0; i < len(recs); i++ {
		r := recs[i]
		switch r.id {
		case recBoundSheet:
			if len(r.data) < 8 {
				return nil, errors.New("biff: short BOUNDSHEET record")
			}
			offset := int(binary.LittleEndian.Uint32(r.data))
			cch := int(r.data[6])
			flags := r.data[7]
			name, _ := decodeChars(r.data[8:], cch, flags&1 != 0)
			infos = append(infos, sheetInfo{name, offset})
		case recSST:
			// The SST and the CONTINUE records after it form one string table.
			chunks := [][]byte{r.data}
			for i+1 < len(recs) && recs[i+1].id == recContinue {
				i++
				chunks = append(chunks, recs[i].data)
			}
			var err error
			if sst, err = readSST(chunks); err != nil {
				return nil, err
			}
		case recEOF:
			i = len(recs) // the globals end at the first EOF
		}
	}
	if len(infos) == 0 {
		return nil, errors.New("biff: no sheets found")
	}

	wb := &Workbook{}
	for _, info := range infos {
		sheet := &Sheet{Name: info.name, cells: map[[2]int]Cell{}}
		wb.Sheets = append(wb.Sheets, sheet)
		if info.offset <= 0 || info.offset >= len(stream) {
			continue
		}
		sheetRecs := records(stream, info.offset)
		for i := 1; i < len(sheetRecs); i++ {
			r := sheetRecs[i]
			if r.id == recEOF {
				break
			}
			switch r.id {
			case recLabelSST:
				row, col, idx := int(u16(r.data, 0)), int(u16(r.data, 2)), int(u32(r.data, 6))
				if idx < len(sst) {
					sheet.set(row, col, Cell{Text: sst[idx]})
				}
			case recLabel:
				row, col := int(u16(r.data, 0)), int(u16(r.data, 2))
				cch := int(u16(r.data, 6))
				text, _ := decodeChars(r.data[9:], cch, r.data[8]&1 != 0)
				sheet.set(row, col, Cell{Text: text})
			case recNumber:
				sheet.set(int(u16(r.data, 0)), int(u16(r.data, 2)), Cell{Number: math.Float64frombits(binary.LittleEndian.Uint64(r.data[6:])), IsNumber: true})
			case recRK:
				sheet.set(int(u16(r.data, 0)), int(u16(r.data, 2)), Cell{Number: rkValue(u32(r.data, 6)), IsNumber: true})
			case recMulRK:
				row, first := int(u16(r.data, 0)), int(u16(r.data, 2))
				for k := 0; 4+k*6+6 <= len(r.data)-2; k++ {
					sheet.set(row, first+k, Cell{Number: rkValue(u32(r.data, 4+k*6+2)), IsNumber: true})
				}
			case recFormula:
				row, col := int(u16(r.data, 0)), int(u16(r.data, 2))
				if r.data[12] == 0xff && r.data[13] == 0xff {
					// A cached string, boolean or error instead of a number. A
					// string comes in the STRING record that follows.
					if r.data[6] == 0 && i+1 < len(sheetRecs) && sheetRecs[i+1].id == recString {
						i++
						next := sheetRecs[i].data
						cch := int(u16(next, 0))
						text, _ := decodeChars(next[3:], cch, next[2]&1 != 0)
						sheet.set(row, col, Cell{Text: text})
					}
					continue
				}
				sheet.set(row, col, Cell{Number: math.Float64frombits(binary.LittleEndian.Uint64(r.data[6:])), IsNumber: true})
			}
		}
	}
	return wb, nil
}

func (s *Sheet) set(row, col int, c Cell) {
	s.cells[[2]int{row, col}] = c
	if row+1 > s.rows {
		s.rows = row + 1
	}
}

func u16(b []byte, off int) uint16 { return binary.LittleEndian.Uint16(b[off:]) }
func u32(b []byte, off int) uint32 { return binary.LittleEndian.Uint32(b[off:]) }

// rkValue decodes an RK number: 30 bits of either an integer or the top of a
// double, optionally divided by 100.
func rkValue(rk uint32) float64 {
	var v float64
	if rk&2 != 0 {
		v = float64(int32(rk) >> 2)
	} else {
		v = math.Float64frombits(uint64(rk&0xfffffffc) << 32)
	}
	if rk&1 != 0 {
		v /= 100
	}
	return v
}

// decodeChars reads cch characters stored one byte each (Latin-1) or two bytes
// each (UTF-16LE), and returns the text and the bytes used.
func decodeChars(b []byte, cch int, wide bool) (string, int) {
	if wide {
		n := cch * 2
		if n > len(b) {
			n = len(b) &^ 1
		}
		u := make([]uint16, n/2)
		for i := range u {
			u[i] = binary.LittleEndian.Uint16(b[2*i:])
		}
		return string(utf16.Decode(u)), n
	}
	if cch > len(b) {
		cch = len(b)
	}
	runes := make([]rune, cch)
	for i := 0; i < cch; i++ {
		runes[i] = rune(b[i])
	}
	return string(runes), cch
}

// readSST reads the shared string table out of the SST record's data and the
// CONTINUE records after it. A string's characters may be split between two
// chunks; the second chunk then starts with a byte that says whether the rest
// is stored in one byte or two per character.
func readSST(chunks [][]byte) ([]string, error) {
	r := &sstReader{chunks: chunks}
	if len(chunks[0]) < 8 {
		return nil, errors.New("biff: short SST record")
	}
	r.pos = 8
	unique := int(u32(chunks[0], 4))
	out := make([]string, 0, unique)
	for i := 0; i < unique; i++ {
		if r.atEnd() {
			break
		}
		cch := int(r.u16())
		flags := r.byte()
		runs, ext := 0, 0
		if flags&0x08 != 0 {
			runs = int(r.u16())
		}
		if flags&0x04 != 0 {
			ext = int(r.u32())
		}
		out = append(out, r.chars(cch, flags&1 != 0))
		r.skip(runs*4 + ext)
	}
	return out, nil
}

type sstReader struct {
	chunks [][]byte
	chunk  int
	pos    int
}

func (r *sstReader) atEnd() bool {
	for r.chunk < len(r.chunks) && r.pos >= len(r.chunks[r.chunk]) {
		r.chunk++
		r.pos = 0
	}
	return r.chunk >= len(r.chunks)
}

func (r *sstReader) byte() byte {
	r.atEnd()
	b := r.chunks[r.chunk][r.pos]
	r.pos++
	return b
}

func (r *sstReader) u16() uint16 { return uint16(r.byte()) | uint16(r.byte())<<8 }

func (r *sstReader) u32() uint32 { return uint32(r.u16()) | uint32(r.u16())<<16 }

func (r *sstReader) skip(n int) {
	for n > 0 && !r.atEnd() {
		take := min(n, len(r.chunks[r.chunk])-r.pos)
		r.pos += take
		n -= take
	}
}

// chars reads cch characters, switching encoding at a chunk boundary.
func (r *sstReader) chars(cch int, wide bool) string {
	var u []uint16
	for cch > 0 {
		if r.pos >= len(r.chunks[r.chunk]) {
			// A string continues in the next chunk, which starts with the
			// encoding flag of the remaining characters.
			r.chunk++
			r.pos = 0
			wide = r.byte()&1 != 0
		}
		avail := len(r.chunks[r.chunk]) - r.pos
		size := 1
		if wide {
			size = 2
		}
		n := min(cch, avail/size)
		if n == 0 {
			// Not enough bytes left in this chunk for one character.
			r.pos = len(r.chunks[r.chunk])
			continue
		}
		data := r.chunks[r.chunk][r.pos : r.pos+n*size]
		for i := 0; i < n; i++ {
			if wide {
				u = append(u, binary.LittleEndian.Uint16(data[2*i:]))
			} else {
				u = append(u, uint16(data[i]))
			}
		}
		r.pos += n * size
		cch -= n
	}
	return string(utf16.Decode(u))
}

// workbookStream returns the "Workbook" (or "Book") stream of an OLE2 compound
// file.
func workbookStream(data []byte) ([]byte, error) {
	if len(data) < 512 || binary.LittleEndian.Uint64(data) != 0xe11ab1a1e011cfd0 {
		return nil, errors.New("biff: not an OLE2 compound file")
	}
	sectorSize := 1 << binary.LittleEndian.Uint16(data[0x1e:])
	miniSize := 1 << binary.LittleEndian.Uint16(data[0x20:])
	cutoff := int(binary.LittleEndian.Uint32(data[0x38:]))
	firstDir := int(binary.LittleEndian.Uint32(data[0x30:]))
	firstMiniFAT := int(binary.LittleEndian.Uint32(data[0x3c:]))
	firstDIFAT := int(binary.LittleEndian.Uint32(data[0x44:]))
	numDIFAT := int(binary.LittleEndian.Uint32(data[0x48:]))

	sector := func(i int) ([]byte, error) {
		start := (i + 1) * sectorSize
		if i < 0 || start+sectorSize > len(data) {
			return nil, fmt.Errorf("biff: sector %d is outside the file", i)
		}
		return data[start : start+sectorSize], nil
	}
	// The FAT sector numbers: 109 in the header, then those in the DIFAT chain.
	var fatSectors []int
	for i := 0; i < 109; i++ {
		if s := int32(binary.LittleEndian.Uint32(data[0x4c+4*i:])); s >= 0 {
			fatSectors = append(fatSectors, int(s))
		}
	}
	for next, n := firstDIFAT, 0; n < numDIFAT && next >= 0; n++ {
		sec, err := sector(next)
		if err != nil {
			return nil, err
		}
		for i := 0; i < sectorSize/4-1; i++ {
			if s := int32(binary.LittleEndian.Uint32(sec[4*i:])); s >= 0 {
				fatSectors = append(fatSectors, int(s))
			}
		}
		next = int(int32(binary.LittleEndian.Uint32(sec[sectorSize-4:])))
	}
	var fat []int32
	for _, s := range fatSectors {
		sec, err := sector(s)
		if err != nil {
			return nil, err
		}
		for i := 0; i < sectorSize/4; i++ {
			fat = append(fat, int32(binary.LittleEndian.Uint32(sec[4*i:])))
		}
	}
	chain := func(start int, table []int32) []int {
		var out []int
		for s, n := start, 0; s >= 0 && s < len(table) && n <= len(table); n++ {
			out = append(out, s)
			s = int(table[s])
		}
		return out
	}
	readChain := func(start int) ([]byte, error) {
		var out []byte
		for _, s := range chain(start, fat) {
			sec, err := sector(s)
			if err != nil {
				return nil, err
			}
			out = append(out, sec...)
		}
		return out, nil
	}

	dir, err := readChain(firstDir)
	if err != nil {
		return nil, err
	}
	type entry struct {
		name  string
		start int
		size  int
		kind  byte
	}
	var entries []entry
	for off := 0; off+128 <= len(dir); off += 128 {
		nameLen := int(binary.LittleEndian.Uint16(dir[off+0x40:]))
		if nameLen < 2 || nameLen > 64 {
			entries = append(entries, entry{})
			continue
		}
		u := make([]uint16, nameLen/2-1)
		for i := range u {
			u[i] = binary.LittleEndian.Uint16(dir[off+2*i:])
		}
		entries = append(entries, entry{
			name:  string(utf16.Decode(u)),
			kind:  dir[off+0x42],
			start: int(int32(binary.LittleEndian.Uint32(dir[off+0x74:]))),
			size:  int(binary.LittleEndian.Uint32(dir[off+0x78:])),
		})
	}
	var book, root *entry
	for i := range entries {
		switch {
		case entries[i].kind == 5:
			root = &entries[i]
		case entries[i].kind == 2 && (entries[i].name == "Workbook" || entries[i].name == "Book"):
			book = &entries[i]
		}
	}
	if book == nil {
		return nil, errors.New("biff: the file has no Workbook stream")
	}
	if book.size >= cutoff {
		raw, err := readChain(book.start)
		if err != nil {
			return nil, err
		}
		if book.size > len(raw) {
			return nil, errors.New("biff: the Workbook stream is cut short")
		}
		return raw[:book.size], nil
	}
	// A small stream lives in the mini stream, which is the root entry's
	// stream, cut into mini sectors that the mini FAT chains.
	if root == nil {
		return nil, errors.New("biff: the file has no root entry")
	}
	miniStream, err := readChain(root.start)
	if err != nil {
		return nil, err
	}
	miniFATBytes, err := readChain(firstMiniFAT)
	if err != nil {
		return nil, err
	}
	miniFAT := make([]int32, len(miniFATBytes)/4)
	for i := range miniFAT {
		miniFAT[i] = int32(binary.LittleEndian.Uint32(miniFATBytes[4*i:]))
	}
	var out []byte
	for _, s := range chain(book.start, miniFAT) {
		start := s * miniSize
		if start+miniSize > len(miniStream) {
			return nil, errors.New("biff: a mini sector is outside the mini stream")
		}
		out = append(out, miniStream[start:start+miniSize]...)
	}
	if book.size > len(out) {
		return nil, errors.New("biff: the Workbook stream is cut short")
	}
	return out[:book.size], nil
}
