package excelsheet

import (
	"encoding/xml"
	"path"
	"strings"

	"github.com/xuri/excelize/v2"
)

type relsDoc struct {
	Relationships []struct {
		ID         string `xml:"Id,attr"`
		Target     string `xml:"Target,attr"`
		TargetMode string `xml:"TargetMode,attr"`
	} `xml:"Relationship"`
}

// relsPath is where the relationships of the part at partPath are stored.
func relsPath(partPath string) string {
	return path.Join(path.Dir(partPath), "_rels", path.Base(partPath)+".rels")
}

// relTargets reads the relationships stored for partPath and returns the parts
// they point to, as package paths. A part with no stored relationships, or one
// that cannot be read, has none.
func relTargets(f *excelize.File, partPath string) []string {
	raw, ok := f.Pkg.Load(relsPath(partPath))
	if !ok {
		return nil
	}
	data, ok := raw.([]byte)
	if !ok {
		return nil
	}
	var doc relsDoc
	if err := xml.Unmarshal(data, &doc); err != nil {
		return nil
	}
	var out []string
	for _, rel := range doc.Relationships {
		if strings.EqualFold(rel.TargetMode, "External") {
			continue
		}
		target := rel.Target
		if strings.HasPrefix(target, "/") {
			out = append(out, strings.TrimPrefix(target, "/"))
		} else {
			out = append(out, path.Join(path.Dir(partPath), target))
		}
	}
	return out
}

// sheetPartPath returns the package path of the worksheet at idx, or "" when
// the workbook's stored relationships do not say.
func sheetPartPath(f *excelize.File, idx int) string {
	wb := f.WorkBook
	if wb == nil || idx < 0 || idx >= len(wb.Sheets.Sheet) {
		return ""
	}
	raw, ok := f.Pkg.Load("xl/_rels/workbook.xml.rels")
	if !ok {
		return ""
	}
	data, ok := raw.([]byte)
	if !ok {
		return ""
	}
	var doc relsDoc
	if err := xml.Unmarshal(data, &doc); err != nil {
		return ""
	}
	id := wb.Sheets.Sheet[idx].ID
	for _, rel := range doc.Relationships {
		if rel.ID != id {
			continue
		}
		if strings.HasPrefix(rel.Target, "/") {
			return strings.TrimPrefix(rel.Target, "/")
		}
		return path.Join("xl", rel.Target)
	}
	return ""
}

// ownedParts lists every part the worksheet at idx reaches through its
// relationships, transitively: its comments, the legacy drawing that holds the
// comment boxes, its tables, its drawing and the charts and pictures that
// drawing uses.
func ownedParts(f *excelize.File, idx int) []string {
	sheetPath := sheetPartPath(f, idx)
	if sheetPath == "" {
		return nil
	}
	var out []string
	seen := map[string]bool{sheetPath: true}
	queue := []string{sheetPath}
	for len(queue) > 0 {
		part := queue[0]
		queue = queue[1:]
		for _, target := range relTargets(f, part) {
			if seen[target] {
				continue
			}
			seen[target] = true
			out = append(out, target)
			queue = append(queue, target)
		}
	}
	return out
}

// dropParts removes the parts the replaced worksheet owned, unless another
// part of the workbook still points at them. DeleteSheet removes the worksheet
// and its own relationships but leaves what they pointed to in the file,
// unreferenced, so the old comments, tables and drawings stay in the saved
// workbook and a new table cannot take an old table's name.
func dropParts(f *excelize.File, parts []string) {
	if len(parts) == 0 {
		return
	}
	referenced := map[string]bool{}
	f.Pkg.Range(func(key, _ any) bool {
		name, _ := key.(string)
		if !strings.HasSuffix(name, ".rels") {
			return true
		}
		// "xl/worksheets/_rels/sheet1.xml.rels" belongs to "xl/worksheets/sheet1.xml".
		owner := path.Join(path.Dir(path.Dir(name)), strings.TrimSuffix(path.Base(name), ".rels"))
		for _, target := range relTargets(f, owner) {
			referenced[target] = true
		}
		return true
	})
	for _, part := range parts {
		if referenced[part] {
			continue
		}
		f.Pkg.Delete(part)
		f.Pkg.Delete(relsPath(part))
		f.Relationships.Delete(relsPath(part))
		f.Drawings.Delete(part)
		f.Sheet.Delete(part)
		delete(f.Comments, part)
		delete(f.VMLDrawing, part)
		delete(f.DecodeVMLDrawing, part)
		if f.ContentTypes != nil {
			kept := f.ContentTypes.Overrides[:0]
			for _, o := range f.ContentTypes.Overrides {
				if o.PartName != "/"+part {
					kept = append(kept, o)
				}
			}
			f.ContentTypes.Overrides = kept
		}
	}
}
