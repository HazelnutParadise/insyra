package env

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"

	insyra "github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
	"github.com/TimLai666/go-decimal/decimal"
)

// unreadableVariable holds a stored variable this build could not decode, such
// as one a newer release wrote, so the next save writes it back as it was.
type unreadableVariable struct {
	stored SerializedVariable
}

const (
	kindTable  = "table"
	kindList   = "list"
	kindScalar = "scalar"
	kindSlice  = "slice"
	// A scaler and a hierarchical tree write their own JSON, kind tag included,
	// so the environment stores the bytes and knows nothing of what is in them.
	kindScaler = "scaler"
	kindHClust = "hclust"
)

// storedTable is how a whole table is written: its columns in order, plus row
// names when the table has any.
type storedTable struct {
	Columns  []storedColumn `json:"columns"`
	RowNames []string       `json:"rowNames,omitempty"`
}

// storedScalar is how a single value is written. Type is empty only for a
// variable that has no value at all.
type storedScalar struct {
	Type  string `json:"type,omitempty"`
	Value any    `json:"value"`
}

// storedSlice is how a slice is written: the element type once, and the elements
// themselves under it.
type storedSlice struct {
	Elem   string `json:"elem"`
	Values []any  `json:"values"`
}

// sliceElemTypes are the element types a slice variable may have, keyed by the
// tag encodeCell gives their values. The keys are read off the types themselves
// rather than written by hand, so the map cannot disagree with the tags
// cell_codec.go produces.
var sliceElemTypes = elementTypes(
	reflect.TypeOf(false),
	reflect.TypeOf(""),
	reflect.TypeOf(int(0)),
	reflect.TypeOf(int8(0)),
	reflect.TypeOf(int16(0)),
	reflect.TypeOf(int32(0)),
	reflect.TypeOf(int64(0)),
	reflect.TypeOf(uint(0)),
	reflect.TypeOf(uint8(0)),
	reflect.TypeOf(uint16(0)),
	reflect.TypeOf(uint32(0)),
	reflect.TypeOf(uint64(0)),
	reflect.TypeOf(float32(0)),
	reflect.TypeOf(float64(0)),
	reflect.TypeOf(time.Time{}),
	reflect.TypeOf(time.Duration(0)),
	reflect.TypeOf(decimal.Decimal{}),
	reflect.TypeOf(json.Number("")),
	reflect.TypeOf([]byte(nil)),
)

// elementTypes keys types by the tag their values are stored under.
func elementTypes(types ...reflect.Type) map[string]reflect.Type {
	byTag := make(map[string]reflect.Type, len(types))
	for _, t := range types {
		byTag[t.String()] = t
	}
	return byTag
}

// encodeVariable returns the stored form of value, or an error saying why the
// environment cannot hold it.
func encodeVariable(value any) (SerializedVariable, error) {
	switch typed := value.(type) {
	case nil:
		return storedNothing(), nil
	case *insyra.DataTable:
		if typed == nil {
			return storedNothing(), nil
		}
		return encodeTableVariable(typed)
	case *insyra.DataList:
		if typed == nil {
			return storedNothing(), nil
		}
		// A list has no column name, so it is stored as a column without one.
		column, err := encodeColumn("", typed.Data())
		if err != nil {
			return SerializedVariable{}, err
		}
		return SerializedVariable{Type: kindList, Name: typed.GetName(), Data: column}, nil
	case *insyra.StandardScaler, *insyra.MinMaxScaler, *insyra.RobustScaler, *insyra.MaxAbsScaler:
		// A scaler marshals itself, kind tag and all, so the file can be read
		// back into the same concrete type it was fitted on.
		if reflect.ValueOf(typed).IsNil() {
			return storedNothing(), nil
		}
		b, err := json.Marshal(typed)
		if err != nil {
			return SerializedVariable{}, fmt.Errorf("scaler: %w", err)
		}
		return SerializedVariable{Type: kindScaler, Data: json.RawMessage(b)}, nil
	case *stats.HierarchicalResult:
		// Every field of a tree is exported, so the standard encoder writes the
		// whole of it and the standard decoder reads it back.
		if reflect.ValueOf(typed).IsNil() {
			return storedNothing(), nil
		}
		b, err := json.Marshal(typed)
		if err != nil {
			return SerializedVariable{}, fmt.Errorf("hierarchical tree: %w", err)
		}
		return SerializedVariable{Type: kindHClust, Data: json.RawMessage(b)}, nil
	case unreadableVariable:
		return typed.stored, nil
	default:
		return encodeOtherVariable(value)
	}
}

// storedNothing is how a variable with no value is stored: a scalar with no
// type, which reads back as nil whatever the file was meant to hold.
func storedNothing() SerializedVariable {
	return SerializedVariable{Type: kindScalar, Data: storedScalar{}}
}

func encodeTableVariable(table *insyra.DataTable) (SerializedVariable, error) {
	// Columns is a non-nil slice even for a table with none, so the file always
	// carries a "columns" array for a reader to find.
	count := table.NumCols()
	columns := make([]storedColumn, 0, count)
	for i := 0; i < count; i++ {
		col := table.GetColByNumber(i)
		column, err := encodeColumn(col.GetName(), col.Data())
		if err != nil {
			// An unnamed column can only be found by its position, so the
			// error names the column the way the user would look for it.
			if name := col.GetName(); name != "" {
				return SerializedVariable{}, fmt.Errorf("column %q: %w", name, err)
			}
			return SerializedVariable{}, fmt.Errorf("column %d: %w", i, err)
		}
		columns = append(columns, column)
	}
	stored := storedTable{Columns: columns}
	// Row names are written only when some row has one; a table with none restores with none.
	for _, name := range table.RowNames() {
		if name != "" {
			stored.RowNames = table.RowNames()
			break
		}
	}
	return SerializedVariable{Type: kindTable, Name: table.GetName(), Data: stored}, nil
}

// encodeOtherVariable stores anything that is not a table or a list: one value
// on its own, or a slice whose element type the environment holds.
func encodeOtherVariable(value any) (SerializedVariable, error) {
	tag, encoded, err := encodeCell(value)
	if err == nil {
		return SerializedVariable{Type: kindScalar, Data: storedScalar{Type: tag, Value: encoded}}, nil
	}
	// A []byte is a slice whose elements the environment holds too, but it is
	// one value, so the cell branch above claims it first.
	rv := reflect.ValueOf(value)
	if rv.Kind() == reflect.Slice {
		// The slice has to be exactly []T for a T the map holds: a named slice
		// type, or a slice of a named element type, would come back as []T, which
		// is a different type than the one the caller holds.
		if elem := rv.Type().Elem().String(); sliceElemTypes[elem] != nil && rv.Type() == reflect.SliceOf(sliceElemTypes[elem]) {
			return encodeSliceVariable(rv, elem)
		}
	}
	// Not a slice either, so the cell error is the whole story.
	return SerializedVariable{}, err
}

func encodeSliceVariable(rv reflect.Value, elem string) (SerializedVariable, error) {
	// Values is a non-nil slice so that an empty slice writes "values":[] rather
	// than a null a reader could not tell from a missing array.
	values := make([]any, rv.Len())
	for i := range values {
		_, value, err := encodeCell(rv.Index(i).Interface())
		if err != nil {
			return SerializedVariable{}, fmt.Errorf("element %d: %w", i, err)
		}
		values[i] = value
	}
	return SerializedVariable{Type: kindSlice, Data: storedSlice{Elem: elem, Values: values}}, nil
}

// decodeVariable rebuilds a variable stored by encodeVariable. ok is false when
// sv.Type is not one of the kinds above, so the caller can try the legacy reader.
func decodeVariable(sv SerializedVariable) (value any, ok bool, err error) {
	switch sv.Type {
	case kindTable:
		value, err = decodeStoredTable(sv)
	case kindList:
		value, err = decodeStoredList(sv)
	case kindScalar:
		value, err = decodeStoredScalar(sv)
	case kindSlice:
		value, err = decodeStoredSlice(sv)
	case kindScaler:
		value, err = decodeStoredScaler(sv)
	case kindHClust:
		value, err = decodeStoredHClust(sv)
	default:
		return nil, false, nil
	}
	if err != nil {
		return nil, true, err
	}
	return value, true, nil
}

func decodeStoredTable(sv SerializedVariable) (*insyra.DataTable, error) {
	object, isObject := sv.Data.(map[string]any)
	if !isObject {
		return nil, fmt.Errorf("table is a %T, not a JSON object", sv.Data)
	}
	storedColumns, hasColumns := object["columns"]
	if !hasColumns {
		return nil, errors.New("table has no columns")
	}
	columns, isArray := storedColumns.([]any)
	if !isArray {
		return nil, fmt.Errorf("table columns is a %T, not an array", storedColumns)
	}
	table := insyra.NewDataTable()
	if len(columns) > 0 {
		lists := make([]*insyra.DataList, 0, len(columns))
		for i, raw := range columns {
			name, cells, err := decodeColumn(raw)
			if err != nil {
				return nil, fmt.Errorf("column %d: %w", i, err)
			}
			// Append, not the NewDataList constructor: the constructor
			// flattens a slice cell such as []byte into one cell per element.
			lists = append(lists, insyra.NewDataList().Append(cells...).SetName(name))
		}
		table.AppendCols(lists...)
	}
	if storedNames, hasRowNames := object["rowNames"]; hasRowNames {
		names, err := decodeRowNames(storedNames)
		if err != nil {
			return nil, err
		}
		table.SetRowNames(names)
	}
	table.SetName(sv.Name)
	return table, nil
}

func decodeRowNames(stored any) ([]string, error) {
	perRow, isArray := stored.([]any)
	if !isArray {
		return nil, fmt.Errorf("table rowNames is a %T, not an array", stored)
	}
	names := make([]string, len(perRow))
	for i, entry := range perRow {
		name, isString := entry.(string)
		if !isString {
			return nil, fmt.Errorf("table rowName %d is a %T, not a string", i, entry)
		}
		names[i] = name
	}
	return names, nil
}

func decodeStoredList(sv SerializedVariable) (*insyra.DataList, error) {
	// The column's own name is ignored: a list is named by the variable, which
	// is the only name the DSL will look it up by.
	_, cells, err := decodeColumn(sv.Data)
	if err != nil {
		return nil, err
	}
	return insyra.NewDataList().Append(cells...).SetName(sv.Name), nil
}

func decodeStoredScalar(sv SerializedVariable) (any, error) {
	object, isObject := sv.Data.(map[string]any)
	if !isObject {
		return nil, fmt.Errorf("scalar is a %T, not a JSON object", sv.Data)
	}
	// No type at all is only readable when the value is null, which decodeCell
	// already answers: a null has no type to restore.
	tag := ""
	if stored, hasType := object["type"]; hasType {
		typed, isString := stored.(string)
		if !isString {
			return nil, fmt.Errorf("scalar type is a %T, not a string", stored)
		}
		tag = typed
	}
	return decodeCell(tag, object["value"])
}

func decodeStoredSlice(sv SerializedVariable) (any, error) {
	object, isObject := sv.Data.(map[string]any)
	if !isObject {
		return nil, fmt.Errorf("slice is a %T, not a JSON object", sv.Data)
	}
	storedElem, hasElem := object["elem"]
	if !hasElem {
		return nil, errors.New("slice has no element type")
	}
	elem, isString := storedElem.(string)
	if !isString {
		return nil, fmt.Errorf("slice element type is a %T, not a string", storedElem)
	}
	elemType, known := sliceElemTypes[elem]
	if !known {
		return nil, fmt.Errorf("%q is not an element type a slice can be restored from", elem)
	}
	storedValues, hasValues := object["values"]
	if !hasValues {
		return nil, errors.New("slice has no values")
	}
	values, isArray := storedValues.([]any)
	if !isArray {
		return nil, fmt.Errorf("slice values is a %T, not an array", storedValues)
	}
	out := reflect.MakeSlice(reflect.SliceOf(elemType), len(values), len(values))
	for i, value := range values {
		// A null element is refused rather than filled in: the slice the
		// caller gets back holds elemType, which has no nil to put there.
		if value == nil {
			return nil, fmt.Errorf("element %d: a stored null has no %s to restore it as", i, elem)
		}
		cell, err := decodeCell(elem, value)
		if err != nil {
			return nil, fmt.Errorf("element %d: %w", i, err)
		}
		out.Index(i).Set(reflect.ValueOf(cell))
	}
	return out.Interface(), nil
}

// decodeStoredScaler rebuilds the fitted scaler MarshalJSON wrote. The concrete
// type is not in the fields, so the kind tag the scaler wrote is what says which
// of the four to build — and each of them refuses JSON written by another kind,
// so the file cannot be read into the wrong type.
func decodeStoredScaler(sv SerializedVariable) (any, error) {
	object, isObject := sv.Data.(map[string]any)
	if !isObject {
		return nil, fmt.Errorf("scaler is a %T, not a JSON object", sv.Data)
	}
	kind, _ := object["kind"].(string)
	var scaler any
	switch kind {
	case "standard":
		scaler = new(insyra.StandardScaler)
	case "minmax":
		scaler = new(insyra.MinMaxScaler)
	case "robust":
		scaler = new(insyra.RobustScaler)
	case "maxabs":
		scaler = new(insyra.MaxAbsScaler)
	default:
		return nil, fmt.Errorf("unknown scaler kind %q", kind)
	}
	// The numbers a decoder with UseNumber leaves at the leaves marshal back out
	// as the same literals the file held, which is what a float64 needs to come
	// back bit for bit.
	raw, err := json.Marshal(sv.Data)
	if err != nil {
		return nil, fmt.Errorf("scaler: %w", err)
	}
	if err := json.Unmarshal(raw, scaler); err != nil {
		return nil, fmt.Errorf("scaler: %w", err)
	}
	return scaler, nil
}

// decodeStoredHClust rebuilds a hierarchical tree. Every field of it is
// exported, so writing the stored object out and reading it back is all it takes.
func decodeStoredHClust(sv SerializedVariable) (any, error) {
	raw, err := json.Marshal(sv.Data)
	if err != nil {
		return nil, fmt.Errorf("hierarchical tree: %w", err)
	}
	tree := new(stats.HierarchicalResult)
	if err := json.Unmarshal(raw, tree); err != nil {
		return nil, fmt.Errorf("hierarchical tree: %w", err)
	}
	return tree, nil
}
