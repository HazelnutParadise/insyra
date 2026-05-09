package insyra

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

// defaultStreamChunkSize is the per-chunk row count used by ReadSQLStream
// when ReadSQLOptions.ChunkSize is unset.
const defaultStreamChunkSize = 1000

// dateParseLayouts are the time layouts ParseDates tries in order.
var dateParseLayouts = []string{
	time.RFC3339Nano,
	time.RFC3339,
	"2006-01-02 15:04:05.999999999",
	"2006-01-02 15:04:05",
	"2006-01-02T15:04:05",
	"2006-01-02",
}

type ReadSQLOptions struct {
	// RowNameColumn names the column whose values should become DataTable row
	// names. If empty, no column is treated as the row-name column. Defaults
	// to "row_name" when neither RowNameColumn nor IndexCol is set.
	RowNameColumn string
	// IndexCol is an alias for RowNameColumn. When non-empty it overrides
	// RowNameColumn, mirroring pandas' read_sql(index_col=...).
	IndexCol string

	// Query is a custom SQL query. When set, all other query-shape options
	// (Columns, WhereClause, OrderBy, Limit, Offset, Schema) are ignored.
	Query string
	// Params binds positional parameters to Query (or to the auto-built
	// query when supported). Pandas' read_sql(params=...) equivalent.
	Params []any

	// Columns restricts the auto-built SELECT to these columns. Ignored when
	// Query is set.
	Columns []string

	// Schema is an optional schema (PostgreSQL) or database (MySQL) prefix
	// for the auto-built query. SQLite ignores this.
	Schema string

	Limit       int    // Limit the number of rows to read
	Offset      int    // Starting position for reading rows
	WhereClause string // WHERE clause body (without the "WHERE" keyword)
	OrderBy     string // ORDER BY clause body (without the "ORDER BY" keyword)

	// ParseDates names columns whose string/[]byte values should be parsed
	// as time.Time. Several common ISO-style layouts are tried.
	ParseDates []string

	// DType forces the resulting Go type for the named columns. Recognized
	// targets are reflect.TypeFor[int64](), float64, bool, string, time.Time,
	// and []byte. Unknown targets fall back to default handling.
	DType map[string]reflect.Type

	// ChunkSize is the per-chunk row count used by ReadSQLStream. Zero
	// falls back to defaultStreamChunkSize. Ignored by ReadSQL/ReadSQLContext.
	ChunkSize int
}

// ReadSQL reads table data from the database and converts it into a DataTable.
//
// Equivalent to ReadSQLContext(context.Background(), db, tableName, options...).
func ReadSQL(db *gorm.DB, tableName string, options ...ReadSQLOptions) (*DataTable, error) {
	return ReadSQLContext(context.Background(), db, tableName, options...)
}

// ReadSQLContext is the context-aware variant of ReadSQL. The query and row
// scanning run under ctx, so callers can cancel long-running reads.
func ReadSQLContext(ctx context.Context, db *gorm.DB, tableName string, options ...ReadSQLOptions) (*DataTable, error) {
	if db == nil {
		return nil, fmt.Errorf("db cannot be nil")
	}

	opts := normalizeReadSQLOptions(options)
	tx := db.WithContext(ctx)

	query, params, err := buildReadSQLQuery(tx, tableName, opts)
	if err != nil {
		return nil, err
	}

	LogDebug("core", "ReadSQL", "Executing SQL query: %s", query)
	rows, err := tx.Raw(query, params...).Rows()
	if err != nil {
		return nil, fmt.Errorf("error executing query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	dt, _, err := scanRowsToDataTable(rows, opts, 0)
	return dt, err
}

// ReadSQLChunk is a streamed slice of rows produced by ReadSQLStream. Exactly
// one of Table or Err is set per chunk.
type ReadSQLChunk struct {
	Table *DataTable
	Err   error
}

// ReadSQLStream reads a (potentially huge) query result in chunks, emitting
// each chunk as a DataTable on the returned channel. The channel is closed
// when the stream completes, when ctx is cancelled, or after a fatal error.
//
// Chunk size is controlled by options[0].ChunkSize; zero means use the package
// default (1000 rows). The reader goroutine respects ctx cancellation between
// rows.
func ReadSQLStream(ctx context.Context, db *gorm.DB, tableName string, options ...ReadSQLOptions) (<-chan ReadSQLChunk, error) {
	if db == nil {
		return nil, fmt.Errorf("db cannot be nil")
	}

	opts := normalizeReadSQLOptions(options)
	chunkSize := opts.ChunkSize
	if chunkSize <= 0 {
		chunkSize = defaultStreamChunkSize
	}

	tx := db.WithContext(ctx)
	query, params, err := buildReadSQLQuery(tx, tableName, opts)
	if err != nil {
		return nil, err
	}

	LogDebug("core", "ReadSQLStream", "Executing SQL query: %s (chunkSize=%d)", query, chunkSize)
	rows, err := tx.Raw(query, params...).Rows()
	if err != nil {
		return nil, fmt.Errorf("error executing query: %w", err)
	}

	out := make(chan ReadSQLChunk)
	go func() {
		defer close(out)
		defer func() { _ = rows.Close() }()
		for {
			select {
			case <-ctx.Done():
				out <- ReadSQLChunk{Err: ctx.Err()}
				return
			default:
			}
			dt, done, err := scanRowsToDataTable(rows, opts, chunkSize)
			if err != nil {
				out <- ReadSQLChunk{Err: err}
				return
			}
			if dt != nil {
				select {
				case <-ctx.Done():
					out <- ReadSQLChunk{Err: ctx.Err()}
					return
				case out <- ReadSQLChunk{Table: dt}:
				}
			}
			if done {
				return
			}
		}
	}()
	return out, nil
}

// normalizeReadSQLOptions resolves option defaults and IndexCol / RowNameColumn
// aliasing.
func normalizeReadSQLOptions(options []ReadSQLOptions) ReadSQLOptions {
	var opts ReadSQLOptions
	if len(options) > 0 {
		opts = options[0]
	}
	if opts.IndexCol != "" {
		opts.RowNameColumn = opts.IndexCol
	} else if opts.RowNameColumn == "" {
		opts.RowNameColumn = "row_name"
	}
	return opts
}

// buildReadSQLQuery resolves either the user-supplied Query or constructs one
// from the table name and the shape options. When a custom Query is set, the
// table-existence pre-check is skipped (the database will report any error).
func buildReadSQLQuery(tx *gorm.DB, tableName string, opts ReadSQLOptions) (string, []any, error) {
	if opts.Query != "" {
		return opts.Query, opts.Params, nil
	}

	dialect := tx.Name()
	if err := requireTableExists(tx, dialect, opts.Schema, tableName); err != nil {
		return "", nil, err
	}

	full := qualifiedTableName(opts.Schema, tableName)
	selectList := "*"
	if len(opts.Columns) > 0 {
		selectList = strings.Join(opts.Columns, ", ")
	}
	q := fmt.Sprintf("SELECT %s FROM %s", selectList, full)
	if opts.WhereClause != "" {
		q += " WHERE " + opts.WhereClause
	}
	if opts.OrderBy != "" {
		q += " ORDER BY " + opts.OrderBy
	}
	if opts.Limit > 0 {
		q += fmt.Sprintf(" LIMIT %d", opts.Limit)
	}
	if opts.Offset > 0 {
		q += fmt.Sprintf(" OFFSET %d", opts.Offset)
	}
	return q, opts.Params, nil
}

func requireTableExists(tx *gorm.DB, dialect, schema, tableName string) error {
	exists, err := tableExists(tx, dialect, schema, tableName)
	if err != nil {
		return fmt.Errorf("error checking existence of table %s: %w", tableName, err)
	}
	if !exists {
		return fmt.Errorf("table %s does not exist", tableName)
	}
	return nil
}

// scanRowsToDataTable consumes up to maxRows rows from the cursor and returns
// them as a DataTable. maxRows == 0 means "consume all remaining rows".
//
// The returned `done` flag is true when the underlying cursor has been
// exhausted (rows.Next() returned false). When done is true, callers must not
// reuse the cursor — driver implementations typically auto-close it. The
// returned table is nil when no rows were read in this call.
func scanRowsToDataTable(rows *sql.Rows, opts ReadSQLOptions, maxRows int) (*DataTable, bool, error) {
	columnNames, err := rows.Columns()
	if err != nil {
		return nil, true, fmt.Errorf("error getting column names: %w", err)
	}
	colTypes, err := rows.ColumnTypes()
	if err != nil {
		// non-fatal: fall back to no type info
		colTypes = nil
	}

	parseDateSet := make(map[string]bool, len(opts.ParseDates))
	for _, c := range opts.ParseDates {
		parseDateSet[strings.ToLower(c)] = true
	}

	rowNameColIndex := -1
	if opts.RowNameColumn != "" {
		for i, colName := range columnNames {
			if strings.EqualFold(colName, opts.RowNameColumn) {
				rowNameColIndex = i
				break
			}
		}
	}

	dataLists := make([]*DataList, len(columnNames))
	for i, colName := range columnNames {
		if i == rowNameColIndex {
			continue
		}
		dl := NewDataList()
		dl.SetName(colName)
		dataLists[i] = dl
	}

	rowValues := make([]any, len(columnNames))
	scanArgs := make([]any, len(columnNames))
	for i := range rowValues {
		scanArgs[i] = &rowValues[i]
	}

	rowNames := make(map[int]string)
	var rowIndex = 0
	read := 0

	hitLimit := false
	for rows.Next() {
		if err := rows.Scan(scanArgs...); err != nil {
			return nil, true, fmt.Errorf("error scanning row data: %w", err)
		}

		if rowNameColIndex >= 0 && rowValues[rowNameColIndex] != nil {
			rowNames[rowIndex] = fmt.Sprintf("%v", rowValues[rowNameColIndex])
		}

		for i, val := range rowValues {
			if i == rowNameColIndex {
				continue
			}
			dl := dataLists[i]
			if dl == nil {
				continue
			}
			var ct *sql.ColumnType
			if colTypes != nil && i < len(colTypes) {
				ct = colTypes[i]
			}
			converted := convertSQLValue(val, ct, columnNames[i], opts, parseDateSet)
			dl.Append(converted)
		}
		rowIndex++
		read++
		if maxRows > 0 && read >= maxRows {
			hitLimit = true
			break
		}
	}
	// done == true when the cursor was exhausted naturally (i.e. we did not
	// stop because of maxRows). In that case the underlying driver typically
	// auto-closes rows, so callers must not reuse the cursor.
	done := !hitLimit
	if err := rows.Err(); err != nil {
		return nil, true, fmt.Errorf("error iterating over rows: %w", err)
	}
	if read == 0 {
		return nil, done, nil
	}

	dt := NewDataTable()
	validColumns := make([]*DataList, 0, len(dataLists))
	for i, dl := range dataLists {
		if i == rowNameColIndex || dl == nil {
			continue
		}
		if len(dl.data) > 0 {
			validColumns = append(validColumns, dl)
		}
	}
	if len(validColumns) > 0 {
		dt.AppendCols(validColumns...)
		if rowNameColIndex >= 0 {
			for i, name := range rowNames {
				dt.SetRowNameByIndex(i, name)
			}
		}
	}
	return dt, done, nil
}

// convertSQLValue applies DType coercion, ParseDates parsing, and the
// historical []byte → string fallback while preserving binary blobs.
func convertSQLValue(val any, ct *sql.ColumnType, colName string, opts ReadSQLOptions, parseDateSet map[string]bool) any {
	if val == nil {
		return nil
	}

	if target, ok := opts.DType[colName]; ok {
		if v, ok := coerceToType(val, target); ok {
			return v
		}
	}

	if parseDateSet[strings.ToLower(colName)] {
		if t, ok := tryParseTime(val); ok {
			return t
		}
	}

	if b, isBytes := val.([]byte); isBytes {
		if isBinaryColumn(ct) {
			return b
		}
		s := string(b)
		if ct != nil {
			if v, ok := coerceStringByDBType(s, ct); ok {
				return v
			}
		}
		return s
	}

	return val
}

func isBinaryColumn(ct *sql.ColumnType) bool {
	if ct == nil {
		return false
	}
	switch strings.ToUpper(ct.DatabaseTypeName()) {
	case "BLOB", "BYTEA", "BINARY", "VARBINARY", "LONGBLOB", "MEDIUMBLOB", "TINYBLOB":
		return true
	}
	return false
}

// coerceStringByDBType converts a stringified driver value to the most
// appropriate Go type when the column is numeric or boolean. Non-matching
// columns return ok=false so the caller keeps the string as-is.
func coerceStringByDBType(s string, ct *sql.ColumnType) (any, bool) {
	switch strings.ToUpper(ct.DatabaseTypeName()) {
	case "INT", "INTEGER", "BIGINT", "SMALLINT", "TINYINT", "MEDIUMINT", "INT2", "INT4", "INT8":
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n, true
		}
	case "FLOAT", "DOUBLE", "REAL", "DECIMAL", "NUMERIC", "FLOAT4", "FLOAT8":
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return f, true
		}
	case "BOOL", "BOOLEAN":
		if b, err := strconv.ParseBool(s); err == nil {
			return b, true
		}
	}
	return nil, false
}

func tryParseTime(val any) (time.Time, bool) {
	switch v := val.(type) {
	case time.Time:
		return v, true
	case []byte:
		return tryParseTimeString(string(v))
	case string:
		return tryParseTimeString(v)
	}
	return time.Time{}, false
}

func tryParseTimeString(s string) (time.Time, bool) {
	for _, layout := range dateParseLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// coerceToType performs a best-effort conversion of val to target. Returns
// ok=false when no safe conversion is available.
func coerceToType(val any, target reflect.Type) (any, bool) {
	if val == nil {
		return nil, true
	}
	if target == nil {
		return val, true
	}

	// Direct type match.
	rv := reflect.ValueOf(val)
	if rv.Type() == target {
		return val, true
	}

	// time.Time target: try parsing strings/bytes.
	if target == reflect.TypeFor[time.Time]() {
		if t, ok := tryParseTime(val); ok {
			return t, true
		}
		return nil, false
	}

	// []byte target: stringify then convert.
	if target == reflect.TypeFor[[]byte]() {
		switch v := val.(type) {
		case []byte:
			return v, true
		case string:
			return []byte(v), true
		default:
			return fmt.Append(nil, v), true
		}
	}

	// Convert via reflect when assignable.
	if rv.Type().ConvertibleTo(target) {
		return rv.Convert(target).Interface(), true
	}

	// Stringify, then re-parse based on target kind.
	s := stringifyForCoerce(val)
	switch target.Kind() {
	case reflect.String:
		return s, true
	case reflect.Bool:
		if b, err := strconv.ParseBool(s); err == nil {
			return b, true
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return reflect.ValueOf(n).Convert(target).Interface(), true
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if n, err := strconv.ParseUint(s, 10, 64); err == nil {
			return reflect.ValueOf(n).Convert(target).Interface(), true
		}
	case reflect.Float32, reflect.Float64:
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return reflect.ValueOf(f).Convert(target).Interface(), true
		}
	}
	return nil, false
}

func stringifyForCoerce(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case []byte:
		return string(t)
	default:
		return fmt.Sprint(v)
	}
}
