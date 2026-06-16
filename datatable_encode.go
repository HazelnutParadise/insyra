package insyra

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// NaNPolicy controls how missing (nil or NaN) source values are encoded.
type NaNPolicy int

const (
	// NaNAsCategory makes missing values their own category.
	NaNAsCategory NaNPolicy = iota
	// NaNError returns an error when a missing value is present.
	NaNError
	// NaNSkip skips missing values in the encoded output.
	NaNSkip
)

// UnknownPolicy controls unseen categories encountered during Transform.
type UnknownPolicy int

const (
	// UnknownIgnore emits all-zero indicators or nil cells for unseen values.
	UnknownIgnore UnknownPolicy = iota
	// UnknownError returns an error for the first unseen value.
	UnknownError
	// UnknownAsNew extends the encoder with unseen values.
	UnknownAsNew
)

// LabelSort controls id assignment order for LabelEncode.
type LabelSort int

const (
	// LabelSortFirstSeen assigns ids in first-appearance order.
	LabelSortFirstSeen LabelSort = iota
	// LabelSortLexicographic assigns ids by sorted string form.
	LabelSortLexicographic
	// LabelSortByFrequency assigns ids by descending frequency.
	LabelSortByFrequency
)

// OneHotOptions configures DataTable one-hot encoding.
type OneHotOptions struct {
	Columns        []string
	DropFirst      bool
	HandleNaN      NaNPolicy
	Unknown        UnknownPolicy
	Prefix         string
	Separator      string
	KeepOriginal   bool
	SortCategories bool
}

// LabelEncodeOptions configures DataTable label encoding.
type LabelEncodeOptions struct {
	Column       string
	NewColumn    string
	SortBy       LabelSort
	HandleNaN    NaNPolicy
	Unknown      UnknownPolicy
	KeepOriginal bool
}

// OrdinalEncodeOptions configures DataTable ordinal encoding.
type OrdinalEncodeOptions struct {
	Column       string
	Order        []any
	NewColumn    string
	HandleNaN    NaNPolicy
	Unknown      UnknownPolicy
	KeepOriginal bool
}

// Encoder is the minimal shared surface for fitted categorical encoders.
type Encoder interface {
	Transform(dt *DataTable) (*DataTable, error)
	InverseTransform(dt *DataTable) (*DataTable, error)
	Kind() string
}

// OneHotEncoder stores fitted one-hot category mappings.
type OneHotEncoder struct {
	opts          OneHotOptions
	columns       []oneHotColumnState
	outputColumns []string
}

type oneHotColumnState struct {
	sourceRef     string
	sourceName    string
	sourceIndex   int
	prefix        string
	categories    []any
	keyToIndex    map[string]int
	outputColumns []string
}

// LabelEncoder stores a fitted label encoding.
type LabelEncoder struct {
	opts        LabelEncodeOptions
	sourceRef   string
	sourceName  string
	encodedName string
	classes     []any
	keyToID     map[string]int
}

// OrdinalEncoder stores a fitted ordinal encoding.
type OrdinalEncoder struct {
	opts        OrdinalEncodeOptions
	sourceRef   string
	sourceName  string
	encodedName string
	classes     []any
	keyToID     map[string]int
}

// OneHotEncode fits one-hot indicators for selected columns and returns a new table.
func (dt *DataTable) OneHotEncode(opts OneHotOptions) (*DataTable, *OneHotEncoder, error) {
	enc, err := fitOneHotEncoder(dt, opts)
	if err != nil {
		return nil, nil, err
	}
	out, err := enc.Transform(dt)
	if err != nil {
		return nil, nil, err
	}
	return out, enc, nil
}

// LabelEncode fits integer ids for a column and returns a new table.
func (dt *DataTable) LabelEncode(opts LabelEncodeOptions) (*DataTable, *LabelEncoder, error) {
	enc, err := fitLabelEncoder(dt, opts)
	if err != nil {
		return nil, nil, err
	}
	out, err := enc.Transform(dt)
	if err != nil {
		return nil, nil, err
	}
	return out, enc, nil
}

// OrdinalEncode fits explicit ordered ids for a column and returns a new table.
func (dt *DataTable) OrdinalEncode(opts OrdinalEncodeOptions) (*DataTable, *OrdinalEncoder, error) {
	enc, err := fitOrdinalEncoder(dt, opts)
	if err != nil {
		return nil, nil, err
	}
	out, err := enc.Transform(dt)
	if err != nil {
		return nil, nil, err
	}
	return out, enc, nil
}

// Transform applies this fitted one-hot encoder to a new table.
func (e *OneHotEncoder) Transform(dt *DataTable) (*DataTable, error) {
	if e == nil {
		return nil, fmt.Errorf("OneHotEncoder.Transform: encoder is nil")
	}
	out := NewDataTable()
	var err error
	dt.AtomicDo(func(t *DataTable) {
		var cols []*DataList
		cols, err = e.transformNotAtomic(t)
		if err != nil {
			return
		}
		out.AppendCols(cols...)
		copyRowNamesNotAtomic(out, t)
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// InverseTransform rebuilds source columns from one-hot indicators.
func (e *OneHotEncoder) InverseTransform(dt *DataTable) (*DataTable, error) {
	if e == nil {
		return nil, fmt.Errorf("OneHotEncoder.InverseTransform: encoder is nil")
	}
	out := NewDataTable()
	var err error
	dt.AtomicDo(func(t *DataTable) {
		var cols []*DataList
		cols, err = e.inverseTransformNotAtomic(t)
		if err != nil {
			return
		}
		out.AppendCols(cols...)
		copyRowNamesNotAtomic(out, t)
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Kind returns the encoder family name.
func (e *OneHotEncoder) Kind() string { return "onehot" }

// Categories returns source-column categories in encoded order.
func (e *OneHotEncoder) Categories() map[string][]any {
	out := make(map[string][]any, len(e.columns))
	for _, col := range e.columns {
		out[col.sourceName] = append([]any(nil), col.categories...)
	}
	return out
}

// OutputColumns returns generated indicator column names.
func (e *OneHotEncoder) OutputColumns() []string {
	return append([]string(nil), e.outputColumns...)
}

// Transform applies this fitted label encoder to a new table.
func (e *LabelEncoder) Transform(dt *DataTable) (*DataTable, error) {
	if e == nil {
		return nil, fmt.Errorf("LabelEncoder.Transform: encoder is nil")
	}
	return transformScalarEncoder(dt, e.sourceRef, e.sourceName, e.encodedName, e.opts.NewColumn != "", e.opts.KeepOriginal, func(v any) (any, error) {
		return e.encodeValue(v)
	})
}

// InverseTransform decodes the label column back to its source values.
func (e *LabelEncoder) InverseTransform(dt *DataTable) (*DataTable, error) {
	if e == nil {
		return nil, fmt.Errorf("LabelEncoder.InverseTransform: encoder is nil")
	}
	return inverseScalarEncoder(dt, e.sourceName, e.encodedName, e.opts.NewColumn != "", e.opts.KeepOriginal, func(v any) (any, error) {
		return e.inverseOne(v)
	})
}

// Kind returns the encoder family name.
func (e *LabelEncoder) Kind() string { return "label" }

// Classes returns category values by id.
func (e *LabelEncoder) Classes() []any {
	return append([]any(nil), e.classes...)
}

// Inverse maps label ids back to category values.
func (e *LabelEncoder) Inverse(values ...any) ([]any, error) {
	return inverseIDs(values, e.inverseOne)
}

// Transform applies this fitted ordinal encoder to a new table.
func (e *OrdinalEncoder) Transform(dt *DataTable) (*DataTable, error) {
	if e == nil {
		return nil, fmt.Errorf("OrdinalEncoder.Transform: encoder is nil")
	}
	return transformScalarEncoder(dt, e.sourceRef, e.sourceName, e.encodedName, e.opts.NewColumn != "", e.opts.KeepOriginal, func(v any) (any, error) {
		return e.encodeValue(v)
	})
}

// InverseTransform decodes the ordinal column back to its source values.
func (e *OrdinalEncoder) InverseTransform(dt *DataTable) (*DataTable, error) {
	if e == nil {
		return nil, fmt.Errorf("OrdinalEncoder.InverseTransform: encoder is nil")
	}
	return inverseScalarEncoder(dt, e.sourceName, e.encodedName, e.opts.NewColumn != "", e.opts.KeepOriginal, func(v any) (any, error) {
		return e.inverseOne(v)
	})
}

// Kind returns the encoder family name.
func (e *OrdinalEncoder) Kind() string { return "ordinal" }

// Classes returns category values by id.
func (e *OrdinalEncoder) Classes() []any {
	return append([]any(nil), e.classes...)
}

// Inverse maps ordinal ids back to category values.
func (e *OrdinalEncoder) Inverse(values ...any) ([]any, error) {
	return inverseIDs(values, e.inverseOne)
}

func fitOneHotEncoder(dt *DataTable, opts OneHotOptions) (*OneHotEncoder, error) {
	opts = normalizeOneHotOptions(opts)
	if len(opts.Columns) == 0 {
		return nil, fmt.Errorf("OneHotEncode: Columns requires at least one column")
	}

	enc := &OneHotEncoder{opts: opts}
	var err error
	dt.AtomicDo(func(t *DataTable) {
		seenCols := map[int]struct{}{}
		for _, ref := range opts.Columns {
			idx, label, ok := resolveEncodingColumn(t, ref)
			if !ok {
				err = fmt.Errorf("OneHotEncode: column %q not found", ref)
				return
			}
			if _, exists := seenCols[idx]; exists {
				err = fmt.Errorf("OneHotEncode: column %q listed more than once", ref)
				return
			}
			seenCols[idx] = struct{}{}
			prefix := opts.Prefix
			if prefix == "" {
				prefix = label
			}
			state := oneHotColumnState{
				sourceRef:   label,
				sourceName:  label,
				sourceIndex: idx,
				prefix:      prefix,
				keyToIndex:  map[string]int{},
			}
			if t.columns[idx].name != "" {
				state.sourceName = t.columns[idx].name
				state.sourceRef = t.columns[idx].name
			}
			if err = collectOneHotCategories(&state, t.columns[idx].data, opts); err != nil {
				return
			}
			if opts.SortCategories {
				sortCategoriesByString(state.categories, state.keyToIndex)
			}
			state.outputColumns = oneHotOutputNames(state, opts)
			enc.columns = append(enc.columns, state)
			enc.outputColumns = append(enc.outputColumns, state.outputColumns...)
		}
	})
	if err != nil {
		return nil, err
	}
	return enc, nil
}

func fitLabelEncoder(dt *DataTable, opts LabelEncodeOptions) (*LabelEncoder, error) {
	if strings.TrimSpace(opts.Column) == "" {
		return nil, fmt.Errorf("LabelEncode: Column is required")
	}
	enc := &LabelEncoder{opts: opts}
	var err error
	dt.AtomicDo(func(t *DataTable) {
		idx, label, ok := resolveEncodingColumn(t, opts.Column)
		if !ok {
			err = fmt.Errorf("LabelEncode: column %q not found", opts.Column)
			return
		}
		enc.sourceRef = label
		enc.sourceName = label
		if t.columns[idx].name != "" {
			enc.sourceRef = t.columns[idx].name
			enc.sourceName = t.columns[idx].name
		}
		enc.encodedName = enc.sourceName
		if opts.NewColumn != "" {
			enc.encodedName = opts.NewColumn
		}
		enc.classes, enc.keyToID, err = collectLabelClasses(t.columns[idx].data, opts)
	})
	if err != nil {
		return nil, err
	}
	return enc, nil
}

func fitOrdinalEncoder(dt *DataTable, opts OrdinalEncodeOptions) (*OrdinalEncoder, error) {
	if strings.TrimSpace(opts.Column) == "" {
		return nil, fmt.Errorf("OrdinalEncode: Column is required")
	}
	if len(opts.Order) == 0 {
		return nil, fmt.Errorf("OrdinalEncode: Order requires at least one category")
	}
	enc := &OrdinalEncoder{opts: opts}
	var err error
	dt.AtomicDo(func(t *DataTable) {
		idx, label, ok := resolveEncodingColumn(t, opts.Column)
		if !ok {
			err = fmt.Errorf("OrdinalEncode: column %q not found", opts.Column)
			return
		}
		enc.sourceRef = label
		enc.sourceName = label
		if t.columns[idx].name != "" {
			enc.sourceRef = t.columns[idx].name
			enc.sourceName = t.columns[idx].name
		}
		enc.encodedName = enc.sourceName
		if opts.NewColumn != "" {
			enc.encodedName = opts.NewColumn
		}
		enc.classes, enc.keyToID, err = collectOrdinalClasses(t.columns[idx].data, opts)
	})
	if err != nil {
		return nil, err
	}
	return enc, nil
}

func collectOneHotCategories(state *oneHotColumnState, values []any, opts OneHotOptions) error {
	for _, raw := range values {
		v, skip, err := normalizeCategoryValue(raw, opts.HandleNaN, "OneHotEncode")
		if err != nil {
			return err
		}
		if skip {
			continue
		}
		addCategory(&state.categories, state.keyToIndex, v)
	}
	return nil
}

func collectLabelClasses(values []any, opts LabelEncodeOptions) ([]any, map[string]int, error) {
	type info struct {
		value     any
		firstSeen int
		count     int
	}
	infos := map[string]*info{}
	order := []string{}
	for _, raw := range values {
		v, skip, err := normalizeCategoryValue(raw, opts.HandleNaN, "LabelEncode")
		if err != nil {
			return nil, nil, err
		}
		if skip {
			continue
		}
		key := labelKey(v)
		if existing, ok := infos[key]; ok {
			existing.count++
			continue
		}
		infos[key] = &info{value: v, firstSeen: len(order), count: 1}
		order = append(order, key)
	}
	switch opts.SortBy {
	case LabelSortFirstSeen:
	case LabelSortLexicographic:
		sort.SliceStable(order, func(i, j int) bool {
			return fmt.Sprint(infos[order[i]].value) < fmt.Sprint(infos[order[j]].value)
		})
	case LabelSortByFrequency:
		sort.SliceStable(order, func(i, j int) bool {
			a, b := infos[order[i]], infos[order[j]]
			if a.count != b.count {
				return a.count > b.count
			}
			return a.firstSeen < b.firstSeen
		})
	default:
		return nil, nil, fmt.Errorf("LabelEncode: unknown LabelSort %d", opts.SortBy)
	}
	classes := make([]any, 0, len(order))
	keyToID := make(map[string]int, len(order))
	for _, key := range order {
		keyToID[key] = len(classes)
		classes = append(classes, infos[key].value)
	}
	return classes, keyToID, nil
}

func collectOrdinalClasses(values []any, opts OrdinalEncodeOptions) ([]any, map[string]int, error) {
	classes := []any{}
	keyToID := map[string]int{}
	for _, raw := range opts.Order {
		v := raw
		if isNilOrNaN(v) {
			switch opts.HandleNaN {
			case NaNError:
				return nil, nil, fmt.Errorf("OrdinalEncode: Order contains missing value")
			case NaNSkip, NaNAsCategory:
				v = nil
			}
		}
		if !addCategory(&classes, keyToID, v) {
			return nil, nil, fmt.Errorf("OrdinalEncode: duplicate category %v in Order", raw)
		}
	}
	for _, raw := range values {
		v, skip, err := normalizeCategoryValue(raw, opts.HandleNaN, "OrdinalEncode")
		if err != nil {
			return nil, nil, err
		}
		if skip {
			continue
		}
		key := labelKey(v)
		if _, ok := keyToID[key]; ok {
			continue
		}
		if opts.HandleNaN == NaNAsCategory && v == nil {
			addCategory(&classes, keyToID, nil)
			continue
		}
		switch opts.Unknown {
		case UnknownIgnore:
			continue
		case UnknownAsNew:
			addCategory(&classes, keyToID, v)
		case UnknownError:
			return nil, nil, fmt.Errorf("OrdinalEncode: value %v is not in Order", raw)
		default:
			return nil, nil, fmt.Errorf("OrdinalEncode: unknown UnknownPolicy %d", opts.Unknown)
		}
	}
	return classes, keyToID, nil
}

func (e *OneHotEncoder) transformNotAtomic(t *DataTable) ([]*DataList, error) {
	stateByIndex := map[int]*oneHotColumnState{}
	for i := range e.columns {
		idx, _, ok := resolveEncodingColumn(t, e.columns[i].sourceRef)
		if !ok {
			return nil, fmt.Errorf("OneHotEncoder.Transform: column %q not found", e.columns[i].sourceRef)
		}
		stateByIndex[idx] = &e.columns[i]
	}

	outCols := []*DataList{}
	for idx, col := range t.columns {
		state, encoded := stateByIndex[idx]
		if !encoded {
			outCols = append(outCols, col.Clone())
			continue
		}
		if e.opts.KeepOriginal {
			outCols = append(outCols, col.Clone())
		}
		encodedCols, err := e.encodeOneHotColumn(state, col.data)
		if err != nil {
			return nil, err
		}
		outCols = append(outCols, encodedCols...)
	}
	e.refreshOutputColumns()
	if err := rejectDuplicateOutputNames(outCols, "OneHotEncoder.Transform"); err != nil {
		return nil, err
	}
	return outCols, nil
}

func (e *OneHotEncoder) encodeOneHotColumn(state *oneHotColumnState, values []any) ([]*DataList, error) {
	start := 0
	if e.opts.DropFirst && len(state.categories) > 0 {
		start = 1
	}
	for _, raw := range values {
		v, skip, err := normalizeCategoryValue(raw, e.opts.HandleNaN, "OneHotEncoder.Transform")
		if err != nil {
			return nil, err
		}
		if skip {
			continue
		}
		key := labelKey(v)
		if _, ok := state.keyToIndex[key]; ok {
			continue
		}
		switch e.opts.Unknown {
		case UnknownIgnore:
		case UnknownError:
			return nil, fmt.Errorf("OneHotEncoder.Transform: unknown category %v", raw)
		case UnknownAsNew:
			state.keyToIndex[key] = len(state.categories)
			state.categories = append(state.categories, v)
			name := oneHotCategoryColumnName(state.prefix, e.opts.Separator, v)
			state.outputColumns = append(state.outputColumns, name)
		default:
			return nil, fmt.Errorf("OneHotEncoder.Transform: unknown UnknownPolicy %d", e.opts.Unknown)
		}
	}

	cols := make([]*DataList, 0, len(state.categories)-start)
	for _, name := range state.outputColumns {
		cols = append(cols, NewDataList().SetName(name))
	}
	for _, raw := range values {
		row := make([]int, len(cols))
		v, skip, err := normalizeCategoryValue(raw, e.opts.HandleNaN, "OneHotEncoder.Transform")
		if err != nil {
			return nil, err
		}
		if !skip {
			if id, ok := state.keyToIndex[labelKey(v)]; ok && id >= start {
				row[id-start] = 1
			}
		}
		for i, value := range row {
			cols[i].Append(value)
		}
	}
	return cols, nil
}

func (e *OneHotEncoder) inverseTransformNotAtomic(t *DataTable) ([]*DataList, error) {
	generated := map[string]struct{}{}
	insertions := map[int][]*oneHotColumnState{}
	for i := range e.columns {
		state := &e.columns[i]
		if e.opts.KeepOriginal {
			if idx, _, ok := resolveEncodingColumn(t, state.sourceRef); ok {
				for _, name := range state.outputColumns {
					generated[name] = struct{}{}
				}
				_ = idx
				continue
			}
		}
		first := -1
		for _, name := range state.outputColumns {
			idx, _, ok := resolveEncodingColumn(t, name)
			if ok && (first < 0 || idx < first) {
				first = idx
			}
			generated[name] = struct{}{}
		}
		if first >= 0 {
			insertions[first] = append(insertions[first], state)
			continue
		}
		if len(state.outputColumns) == 0 {
			insertAt := state.sourceIndex
			if insertAt < 0 {
				insertAt = 0
			}
			if insertAt > len(t.columns) {
				insertAt = len(t.columns)
			}
			insertions[insertAt] = append(insertions[insertAt], state)
		}
	}

	outCols := []*DataList{}
	for idx, col := range t.columns {
		if states, ok := insertions[idx]; ok {
			for _, state := range states {
				decoded, err := e.decodeOneHotColumn(t, state)
				if err != nil {
					return nil, err
				}
				outCols = append(outCols, decoded)
			}
		}
		if _, isGenerated := generated[col.name]; isGenerated {
			continue
		}
		outCols = append(outCols, col.Clone())
	}
	if states, ok := insertions[len(t.columns)]; ok {
		for _, state := range states {
			decoded, err := e.decodeOneHotColumn(t, state)
			if err != nil {
				return nil, err
			}
			outCols = append(outCols, decoded)
		}
	}
	if err := rejectDuplicateOutputNames(outCols, "OneHotEncoder.InverseTransform"); err != nil {
		return nil, err
	}
	return outCols, nil
}

func (e *OneHotEncoder) decodeOneHotColumn(t *DataTable, state *oneHotColumnState) (*DataList, error) {
	nRows := t.getMaxColLength()
	out := NewDataList()
	out.SetName(state.sourceName)
	start := 0
	if e.opts.DropFirst && len(state.categories) > 0 {
		start = 1
	}
	indicatorCols := make([]*DataList, len(state.outputColumns))
	for i, name := range state.outputColumns {
		idx, _, ok := resolveEncodingColumn(t, name)
		if !ok {
			return nil, fmt.Errorf("OneHotEncoder.InverseTransform: indicator column %q not found", name)
		}
		indicatorCols[i] = t.columns[idx]
	}
	for row := 0; row < nRows; row++ {
		found := -1
		for j, col := range indicatorCols {
			if row >= len(col.data) {
				continue
			}
			if isOne(col.data[row]) {
				found = j + start
				break
			}
		}
		if found >= 0 && found < len(state.categories) {
			out.Append(state.categories[found])
			continue
		}
		if e.opts.DropFirst && len(state.categories) > 0 {
			out.Append(state.categories[0])
			continue
		}
		if e.opts.HandleNaN == NaNError {
			return nil, fmt.Errorf("OneHotEncoder.InverseTransform: all-zero row %d", row)
		}
		out.Append(nil)
	}
	return out, nil
}

func (e *LabelEncoder) encodeValue(raw any) (any, error) {
	v, skip, err := normalizeCategoryValue(raw, e.opts.HandleNaN, "LabelEncoder.Transform")
	if err != nil {
		return nil, err
	}
	if skip {
		return nil, nil
	}
	key := labelKey(v)
	if id, ok := e.keyToID[key]; ok {
		return id, nil
	}
	switch e.opts.Unknown {
	case UnknownIgnore:
		return nil, nil
	case UnknownError:
		return nil, fmt.Errorf("LabelEncoder.Transform: unknown category %v", raw)
	case UnknownAsNew:
		id := len(e.classes)
		e.classes = append(e.classes, v)
		e.keyToID[key] = id
		return id, nil
	default:
		return nil, fmt.Errorf("LabelEncoder.Transform: unknown UnknownPolicy %d", e.opts.Unknown)
	}
}

func (e *LabelEncoder) inverseOne(v any) (any, error) {
	if v == nil {
		return nil, nil
	}
	id, ok := integerID(v)
	if !ok {
		return nil, fmt.Errorf("LabelEncoder.Inverse: invalid id %v", v)
	}
	if id < 0 || id >= len(e.classes) {
		return nil, fmt.Errorf("LabelEncoder.Inverse: id %d out of range", id)
	}
	return e.classes[id], nil
}

func (e *OrdinalEncoder) encodeValue(raw any) (any, error) {
	v, skip, err := normalizeCategoryValue(raw, e.opts.HandleNaN, "OrdinalEncoder.Transform")
	if err != nil {
		return nil, err
	}
	if skip {
		return nil, nil
	}
	key := labelKey(v)
	if id, ok := e.keyToID[key]; ok {
		return id, nil
	}
	switch e.opts.Unknown {
	case UnknownIgnore:
		return nil, nil
	case UnknownError:
		return nil, fmt.Errorf("OrdinalEncoder.Transform: unknown category %v", raw)
	case UnknownAsNew:
		id := len(e.classes)
		e.classes = append(e.classes, v)
		e.keyToID[key] = id
		return id, nil
	default:
		return nil, fmt.Errorf("OrdinalEncoder.Transform: unknown UnknownPolicy %d", e.opts.Unknown)
	}
}

func (e *OrdinalEncoder) inverseOne(v any) (any, error) {
	if v == nil {
		return nil, nil
	}
	id, ok := integerID(v)
	if !ok {
		return nil, fmt.Errorf("OrdinalEncoder.Inverse: invalid id %v", v)
	}
	if id < 0 || id >= len(e.classes) {
		return nil, fmt.Errorf("OrdinalEncoder.Inverse: id %d out of range", id)
	}
	return e.classes[id], nil
}

func transformScalarEncoder(dt *DataTable, sourceRef, sourceName, encodedName string, hasNewColumn, keepOriginal bool, encode func(any) (any, error)) (*DataTable, error) {
	out := NewDataTable()
	var err error
	dt.AtomicDo(func(t *DataTable) {
		idx, _, ok := resolveEncodingColumn(t, sourceRef)
		if !ok {
			err = fmt.Errorf("encoder Transform: column %q not found", sourceRef)
			return
		}
		outCols := []*DataList{}
		for colIdx, col := range t.columns {
			if colIdx != idx {
				outCols = append(outCols, col.Clone())
				continue
			}
			encoded := NewDataList()
			if hasNewColumn {
				encoded.SetName(encodedName)
				if keepOriginal {
					outCols = append(outCols, col.Clone(), encoded)
				} else {
					outCols = append(outCols, encoded)
				}
			} else {
				encoded.SetName(sourceName)
				outCols = append(outCols, encoded)
			}
			for _, v := range col.data {
				got, encErr := encode(v)
				if encErr != nil {
					err = encErr
					return
				}
				encoded.Append(got)
			}
		}
		if err != nil {
			return
		}
		if dupErr := rejectDuplicateOutputNames(outCols, "encoder Transform"); dupErr != nil {
			err = dupErr
			return
		}
		out.AppendCols(outCols...)
		copyRowNamesNotAtomic(out, t)
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func inverseScalarEncoder(dt *DataTable, sourceName, encodedName string, hasNewColumn, keepOriginal bool, decode func(any) (any, error)) (*DataTable, error) {
	out := NewDataTable()
	var err error
	dt.AtomicDo(func(t *DataTable) {
		encodedIdx, _, ok := resolveEncodingColumn(t, encodedName)
		if !ok {
			err = fmt.Errorf("encoder InverseTransform: column %q not found", encodedName)
			return
		}
		_, sourceExists := t.getColNumberByName_notAtomic(sourceName)
		outCols := []*DataList{}
		for idx, col := range t.columns {
			if hasNewColumn && keepOriginal && sourceExists {
				if idx == encodedIdx {
					continue
				}
				outCols = append(outCols, col.Clone())
				continue
			}
			if idx != encodedIdx {
				if hasNewColumn && col.name == sourceName {
					continue
				}
				outCols = append(outCols, col.Clone())
				continue
			}
			decoded := NewDataList()
			decoded.SetName(sourceName)
			for _, v := range col.data {
				got, decErr := decode(v)
				if decErr != nil {
					err = decErr
					return
				}
				decoded.Append(got)
			}
			outCols = append(outCols, decoded)
		}
		if err != nil {
			return
		}
		if dupErr := rejectDuplicateOutputNames(outCols, "encoder InverseTransform"); dupErr != nil {
			err = dupErr
			return
		}
		out.AppendCols(outCols...)
		copyRowNamesNotAtomic(out, t)
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func normalizeOneHotOptions(opts OneHotOptions) OneHotOptions {
	if opts.Separator == "" {
		opts.Separator = "_"
	}
	return opts
}

func resolveEncodingColumn(dt *DataTable, ref string) (int, string, bool) {
	if idx, ok := dt.getColNumberByName_notAtomic(ref); ok {
		label := dt.columns[idx].name
		if label == "" {
			label = fallbackEncodingColumnName(idx)
		}
		return idx, label, true
	}
	if idx, ok := ParseColIndex(ref); ok && idx >= 0 && idx < len(dt.columns) {
		label := dt.columns[idx].name
		if label == "" {
			label = fallbackEncodingColumnName(idx)
		}
		return idx, label, true
	}
	return -1, "", false
}

func fallbackEncodingColumnName(idx int) string {
	if name, ok := alphaColIndex(idx); ok {
		return name
	}
	return fmt.Sprintf("col_%d", idx)
}

func normalizeCategoryValue(v any, policy NaNPolicy, method string) (any, bool, error) {
	if !isNilOrNaN(v) {
		return v, false, nil
	}
	switch policy {
	case NaNAsCategory:
		return nil, false, nil
	case NaNError:
		return nil, false, fmt.Errorf("%s: missing value found", method)
	case NaNSkip:
		return nil, true, nil
	default:
		return nil, false, fmt.Errorf("%s: unknown NaNPolicy %d", method, policy)
	}
}

func addCategory(classes *[]any, keyToID map[string]int, v any) bool {
	key := labelKey(v)
	if _, ok := keyToID[key]; ok {
		return false
	}
	keyToID[key] = len(*classes)
	*classes = append(*classes, v)
	return true
}

func sortCategoriesByString(classes []any, keyToID map[string]int) {
	firstSeen := map[string]int{}
	for i, v := range classes {
		firstSeen[labelKey(v)] = i
	}
	sort.SliceStable(classes, func(i, j int) bool {
		a, b := fmt.Sprint(classes[i]), fmt.Sprint(classes[j])
		if a != b {
			return a < b
		}
		return firstSeen[labelKey(classes[i])] < firstSeen[labelKey(classes[j])]
	})
	for k := range keyToID {
		delete(keyToID, k)
	}
	for i, v := range classes {
		keyToID[labelKey(v)] = i
	}
}

func oneHotOutputNames(state oneHotColumnState, opts OneHotOptions) []string {
	start := 0
	if opts.DropFirst && len(state.categories) > 0 {
		start = 1
	}
	names := make([]string, 0, len(state.categories)-start)
	for _, v := range state.categories[start:] {
		names = append(names, oneHotCategoryColumnName(state.prefix, opts.Separator, v))
	}
	return names
}

func oneHotCategoryColumnName(prefix, sep string, v any) string {
	if sep == "" {
		sep = "_"
	}
	return prefix + sep + fmt.Sprint(v)
}

func (e *OneHotEncoder) refreshOutputColumns() {
	e.outputColumns = e.outputColumns[:0]
	for _, col := range e.columns {
		e.outputColumns = append(e.outputColumns, col.outputColumns...)
	}
}

func labelKey(v any) string {
	return fmt.Sprintf("%T:%#v", v, v)
}

func copyRowNamesNotAtomic(out, src *DataTable) {
	if src.rowNames != nil {
		out.rowNames = src.rowNames.Clone()
	}
}

func rejectDuplicateOutputNames(cols []*DataList, method string) error {
	seen := map[string]struct{}{}
	for _, col := range cols {
		name := col.name
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			return fmt.Errorf("%s: duplicate output column name %q", method, name)
		}
		seen[name] = struct{}{}
	}
	return nil
}

func isOne(v any) bool {
	if v == nil {
		return false
	}
	if f, ok := ToFloat64Safe(v); ok {
		return f == 1
	}
	return false
}

func integerID(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int8:
		return int(n), true
	case int16:
		return int(n), true
	case int32:
		return int(n), true
	case int64:
		return int(n), true
	case uint:
		return int(n), true
	case uint8:
		return int(n), true
	case uint16:
		return int(n), true
	case uint32:
		return int(n), true
	case uint64:
		return int(n), true
	case float32:
		f := float64(n)
		i := int(f)
		return i, f == float64(i)
	case float64:
		i := int(n)
		return i, n == float64(i)
	default:
		return 0, false
	}
}

func inverseIDs(values []any, inverse func(any) (any, error)) ([]any, error) {
	flat := flattenInverseArgs(values)
	out := make([]any, 0, len(flat))
	for _, v := range flat {
		got, err := inverse(v)
		if err != nil {
			return nil, err
		}
		out = append(out, got)
	}
	return out, nil
}

func flattenInverseArgs(values []any) []any {
	if len(values) != 1 || values[0] == nil {
		return values
	}
	rv := reflect.ValueOf(values[0])
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return values
	}
	out := make([]any, 0, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		out = append(out, rv.Index(i).Interface())
	}
	return out
}
