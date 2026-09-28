package env

import (
	"encoding/base64"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/TimLai666/go-decimal/decimal"
	json "github.com/goccy/go-json"
)

// jsonNumberPattern is the JSON number grammar. encoding/json writes a
// json.Number as a bare literal and refuses one that does not match it, so
// the codec checks the grammar before it stores one.
var jsonNumberPattern = regexp.MustCompile(`^-?(0|[1-9][0-9]*)(\.[0-9]+)?([eE][+-]?[0-9]+)?$`)

func isJSONNumber(s string) bool { return jsonNumberPattern.MatchString(s) }

// storedIntBits is the width each integer tag is restored at. ParseInt range
// checks against it, so a value that no longer fits its type is an error
// rather than a wrapped one.
var storedIntBits = map[string]int{
	"int":   strconv.IntSize,
	"int8":  8,
	"int16": 16,
	"int32": 32,
	"int64": 64,
}

// storedUintBits is storedIntBits for the unsigned tags.
var storedUintBits = map[string]int{
	"uint":   strconv.IntSize,
	"uint8":  8,
	"uint16": 16,
	"uint32": 32,
	"uint64": 64,
}

// encodeCell returns the tag a cell is stored under and its JSON value.
// A nil cell has tag "" and value nil.
func encodeCell(v any) (tag string, value any, err error) {
	if v == nil {
		return "", nil, nil
	}
	tag = fmt.Sprintf("%T", v)
	switch typed := v.(type) {
	case bool:
		return tag, typed, nil
	case string:
		if utf8.ValidString(typed) {
			return tag, typed, nil
		}
		// encoding/json would replace invalid UTF-8 bytes with U+FFFD,
		// changing the content. Store as base64 instead.
		return tag, map[string]any{"base64": base64.StdEncoding.EncodeToString([]byte(typed))}, nil
	case int:
		return tag, json.Number(strconv.FormatInt(int64(typed), 10)), nil
	case int8:
		return tag, json.Number(strconv.FormatInt(int64(typed), 10)), nil
	case int16:
		return tag, json.Number(strconv.FormatInt(int64(typed), 10)), nil
	case int32:
		return tag, json.Number(strconv.FormatInt(int64(typed), 10)), nil
	case int64:
		return tag, json.Number(strconv.FormatInt(typed, 10)), nil
	case uint:
		return tag, json.Number(strconv.FormatUint(uint64(typed), 10)), nil
	case uint8:
		return tag, json.Number(strconv.FormatUint(uint64(typed), 10)), nil
	case uint16:
		return tag, json.Number(strconv.FormatUint(uint64(typed), 10)), nil
	case uint32:
		return tag, json.Number(strconv.FormatUint(uint64(typed), 10)), nil
	case uint64:
		return tag, json.Number(strconv.FormatUint(typed, 10)), nil
	case float64:
		if name, special := specialFloatName(typed); special {
			return tag, name, nil
		}
		return tag, json.Number(strconv.FormatFloat(typed, 'g', -1, 64)), nil
	case float32:
		as64 := float64(typed)
		if name, special := specialFloatName(as64); special {
			return tag, name, nil
		}
		return tag, json.Number(strconv.FormatFloat(as64, 'g', -1, 32)), nil
	case time.Time:
		// RFC 3339 writes offsets to the minute. If the offset has seconds
		// (e.g., pre-1883 America/New_York at -04:56:02), storing it directly
		// would lose those seconds on read-back. Convert to UTC first to
		// preserve the instant; the year check is then done on the UTC time.
		if _, offset := typed.Zone(); offset%60 != 0 {
			typed = typed.UTC()
		}
		if year := typed.Year(); year < 0 || year > 9999 {
			return "", nil, fmt.Errorf("time in year %d is outside the years RFC 3339 can write", year)
		}
		return tag, typed.Format(time.RFC3339Nano), nil
	case time.Duration:
		return tag, json.Number(strconv.FormatInt(int64(typed), 10)), nil
	case []byte:
		return tag, base64.StdEncoding.EncodeToString(typed), nil
	case decimal.Decimal:
		// A negative scale has no plain-decimal spelling, so String cannot
		// render it and the value would not survive the file.
		if typed.Scale() < 0 {
			return "", nil, fmt.Errorf("decimal with scale %d has no plain-decimal spelling", typed.Scale())
		}
		return tag, typed.String(), nil
	case json.Number:
		if !isJSONNumber(string(typed)) {
			return "", nil, fmt.Errorf("json.Number %q is not a JSON number", string(typed))
		}
		return tag, string(typed), nil
	case map[string]any:
		if err := checkStorableLeaves(typed, 1); err != nil {
			return "", nil, fmt.Errorf("%T holding %w", typed, err)
		}
		return tag, typed, nil
	case []any:
		if err := checkStorableLeaves(typed, 1); err != nil {
			return "", nil, fmt.Errorf("%T holding %w", typed, err)
		}
		return tag, typed, nil
	default:
		return "", nil, fmt.Errorf("%T is not a value the environment can store", v)
	}
}

// decodeCell rebuilds a cell from its tag and the value a json.Decoder with
// UseNumber produced for it.
func decodeCell(tag string, value any) (any, error) {
	// A stored null is a nil cell, whatever the file said it was: the tag
	// cannot describe a value that is not there.
	if value == nil {
		return nil, nil
	}
	if tag == "" {
		return nil, fmt.Errorf("a stored cell with a value has no type tag")
	}
	if bits, ok := storedIntBits[tag]; ok {
		number, isNumber := value.(json.Number)
		if !isNumber {
			return nil, cellTypeError(tag, value)
		}
		parsed, err := strconv.ParseInt(string(number), 10, bits)
		if err != nil {
			return nil, err
		}
		return narrowInt(tag, parsed), nil
	}
	if bits, ok := storedUintBits[tag]; ok {
		number, isNumber := value.(json.Number)
		if !isNumber {
			return nil, cellTypeError(tag, value)
		}
		parsed, err := strconv.ParseUint(string(number), 10, bits)
		if err != nil {
			return nil, err
		}
		return narrowUint(tag, parsed), nil
	}
	switch tag {
	case "bool":
		typed, ok := value.(bool)
		if !ok {
			return nil, cellTypeError(tag, value)
		}
		return typed, nil
	case "string":
		// Valid UTF-8 strings are stored as JSON strings.
		if typed, ok := value.(string); ok {
			return typed, nil
		}
		// Invalid UTF-8 strings are stored as {"base64": "..."}.
		if m, ok := value.(map[string]any); ok {
			if len(m) == 1 {
				if b64, ok := m["base64"].(string); ok {
					decoded, err := base64.StdEncoding.DecodeString(b64)
					if err != nil {
						return nil, fmt.Errorf("base64 decode: %w", err)
					}
					return string(decoded), nil
				}
			}
		}
		return nil, cellTypeError(tag, value)
	case "float64":
		return parseStoredFloat(tag, value, 64)
	case "float32":
		parsed, err := parseStoredFloat(tag, value, 32)
		if err != nil {
			return nil, err
		}
		return float32(parsed), nil
	case "time.Time":
		typed, ok := value.(string)
		if !ok {
			return nil, cellTypeError(tag, value)
		}
		parsed, err := time.Parse(time.RFC3339Nano, typed)
		if err != nil {
			return nil, err
		}
		return parsed, nil
	case "time.Duration":
		number, isNumber := value.(json.Number)
		if !isNumber {
			return nil, cellTypeError(tag, value)
		}
		parsed, err := strconv.ParseInt(string(number), 10, 64)
		if err != nil {
			return nil, err
		}
		return time.Duration(parsed), nil
	case "[]uint8":
		typed, ok := value.(string)
		if !ok {
			return nil, cellTypeError(tag, value)
		}
		// An empty string decodes to an empty non-nil slice, so a stored
		// []byte{} does not come back as a nil slice.
		return base64.StdEncoding.DecodeString(typed)
	case "decimal.Decimal":
		typed, ok := value.(string)
		if !ok {
			return nil, cellTypeError(tag, value)
		}
		return decimal.ParseExact(typed)
	case "json.Number":
		typed, ok := value.(string)
		if !ok {
			return nil, cellTypeError(tag, value)
		}
		if !isJSONNumber(typed) {
			return nil, fmt.Errorf("json.Number %q is not a JSON number", typed)
		}
		return json.Number(typed), nil
	case "map[string]interface {}":
		typed, ok := value.(map[string]any)
		if !ok {
			return nil, cellTypeError(tag, value)
		}
		return typed, nil
	case "[]interface {}":
		typed, ok := value.([]any)
		if !ok {
			return nil, cellTypeError(tag, value)
		}
		return typed, nil
	default:
		return nil, fmt.Errorf("unknown stored type %q", tag)
	}
}

// narrowInt puts a parsed int64 back into the type its tag names.
func narrowInt(tag string, parsed int64) any {
	switch tag {
	case "int":
		return int(parsed)
	case "int8":
		return int8(parsed)
	case "int16":
		return int16(parsed)
	case "int32":
		return int32(parsed)
	default:
		return parsed
	}
}

// narrowUint puts a parsed uint64 back into the type its tag names.
func narrowUint(tag string, parsed uint64) any {
	switch tag {
	case "uint":
		return uint(parsed)
	case "uint8":
		return uint8(parsed)
	case "uint16":
		return uint16(parsed)
	case "uint32":
		return uint32(parsed)
	default:
		return parsed
	}
}

// parseStoredFloat reads a stored float: a JSON number, or one of the three
// names a value encoding/json cannot write gets stored under.
func parseStoredFloat(tag string, value any, bits int) (float64, error) {
	switch typed := value.(type) {
	case json.Number:
		return strconv.ParseFloat(string(typed), bits)
	case string:
		parsed, ok := specialFloatValue(typed)
		if !ok {
			return 0, fmt.Errorf("%q is not a name a %s can be stored under", typed, tag)
		}
		return parsed, nil
	default:
		return 0, cellTypeError(tag, value)
	}
}

// cellTypeError names the tag and the type that was actually stored, which are
// the two things a reader of the file has to check.
func cellTypeError(tag string, value any) error {
	return fmt.Errorf("stored value of type %s is a %T, which the tag cannot restore", tag, value)
}

// specialFloatName is the name a non-finite float is stored under, the same
// three specialFloatKey marks spell.
func specialFloatName(f float64) (string, bool) {
	switch {
	case math.IsNaN(f):
		return "NaN", true
	case math.IsInf(f, 1):
		return "+Inf", true
	case math.IsInf(f, -1):
		return "-Inf", true
	default:
		return "", false
	}
}

// specialFloatValue is the inverse of specialFloatName, and the only other
// reading of a stored string it accepts: a finite float is stored as a number.
func specialFloatValue(name string) (float64, bool) {
	switch name {
	case "NaN":
		return math.NaN(), true
	case "+Inf":
		return math.Inf(1), true
	case "-Inf":
		return math.Inf(-1), true
	default:
		return 0, false
	}
}

// maxStoredNesting is the maximum nesting depth for values stored in the
// environment. A depth of 1 is the outermost map or slice.
const maxStoredNesting = 64

// checkStorableLeaves walks a map or slice the environment will store whole,
// and refuses a leaf it could not restore. insyra.ReadJSON leaves json.Number
// at the leaves, so that is the shape a value read from a JSON file has; a
// leaf typed as float64 or int would come back as a json.Number instead.
// depth is the current nesting level of the container being checked, starting
// at 1 for the outermost container. The depth limit applies only to containers.
func checkStorableLeaves(value any, depth int) error {
	switch typed := value.(type) {
	case nil, bool:
		return nil
	case string:
		if !utf8.ValidString(typed) {
			return fmt.Errorf("string that is not valid UTF-8")
		}
		return nil
	case json.Number:
		if !isJSONNumber(string(typed)) {
			return fmt.Errorf("json.Number %q is not a JSON number", string(typed))
		}
		return nil
	case map[string]any:
		if depth > maxStoredNesting {
			return fmt.Errorf("value nested more than %d levels deep", maxStoredNesting)
		}
		for k, leaf := range typed {
			if !utf8.ValidString(k) {
				return fmt.Errorf("map key that is not valid UTF-8")
			}
			if err := checkStorableLeaves(leaf, depth+1); err != nil {
				return err
			}
		}
		return nil
	case []any:
		if depth > maxStoredNesting {
			return fmt.Errorf("value nested more than %d levels deep", maxStoredNesting)
		}
		for _, leaf := range typed {
			if err := checkStorableLeaves(leaf, depth+1); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("%T is not a value the environment can store", typed)
	}
}

// storedColumn is how one column of cells is written. Type is set when every
// non-nil cell has the same tag; otherwise Types holds one tag per cell, ""
// for a nil cell. A column with no non-nil cell has neither.
type storedColumn struct {
	Name   string   `json:"name,omitempty"`
	Type   string   `json:"type,omitempty"`
	Types  []string `json:"types,omitempty"`
	Values []any    `json:"values"`
}

// encodeColumn stores cells under name.
func encodeColumn(name string, cells []any) (storedColumn, error) {
	tags := make([]string, len(cells))
	values := make([]any, len(cells))
	// A column whose cells share one tag stores that tag once, in Type, so the
	// per-cell list is only written out once two tags have disagreed.
	oneTag := ""
	mixed := false
	for i, cell := range cells {
		tag, value, err := encodeCell(cell)
		if err != nil {
			return storedColumn{}, fmt.Errorf("row %d: %w", i, err)
		}
		tags[i], values[i] = tag, value
		if tag == "" {
			// A nil cell has no type, so it can neither agree nor disagree
			// with the one the rest of the column shares.
			continue
		}
		switch {
		case oneTag == "":
			oneTag = tag
		case oneTag != tag:
			mixed = true
		}
	}
	column := storedColumn{Name: name, Values: values}
	if mixed {
		column.Types = tags
	} else {
		// No non-nil cell at all leaves oneTag "", which is also what the
		// file then says: there is no type to claim for a column of nothing.
		column.Type = oneTag
	}
	return column, nil
}

// decodeColumn rebuilds a column from the value a json.Decoder with UseNumber
// produced for a storedColumn.
func decodeColumn(raw any) (name string, cells []any, err error) {
	object, isObject := raw.(map[string]any)
	if !isObject {
		return "", nil, errors.New("column is not a JSON object")
	}
	if stored, hasName := object["name"]; hasName {
		typed, isString := stored.(string)
		if !isString {
			return "", nil, fmt.Errorf("column name is a %T, not a string", stored)
		}
		name = typed
	}
	storedValues, hasValues := object["values"]
	if !hasValues {
		return "", nil, errors.New("column has no values")
	}
	values, isArray := storedValues.([]any)
	if !isArray {
		return "", nil, fmt.Errorf("column values is a %T, not an array", storedValues)
	}
	tags := make([]string, len(values))
	if storedTypes, hasTypes := object["types"]; hasTypes {
		perCell, isArray := storedTypes.([]any)
		if !isArray {
			return "", nil, fmt.Errorf("column types is a %T, not an array", storedTypes)
		}
		if len(perCell) != len(values) {
			return "", nil, fmt.Errorf("column has %d types for %d values", len(perCell), len(values))
		}
		for i, entry := range perCell {
			tag, isString := entry.(string)
			if !isString {
				return "", nil, fmt.Errorf("column type of row %d is a %T, not a string", i, entry)
			}
			tags[i] = tag
		}
	} else {
		// No per-cell types: one tag for the whole column, and none at all
		// when the file carries no "type" either.
		tag := ""
		if storedType, hasType := object["type"]; hasType {
			typed, isString := storedType.(string)
			if !isString {
				return "", nil, fmt.Errorf("column type is a %T, not a string", storedType)
			}
			tag = typed
		}
		for i := range tags {
			tags[i] = tag
		}
	}
	cells = make([]any, len(values))
	for i, value := range values {
		cell, err := decodeCell(tags[i], value)
		if err != nil {
			return "", nil, fmt.Errorf("row %d: %w", i, err)
		}
		cells[i] = cell
	}
	return name, cells, nil
}
