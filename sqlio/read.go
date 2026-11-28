package sqlio

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"iter"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/internal/prep"
)

// ReadOptions are the settings of Read and Stream. The zero value reads every
// column and row of the table and takes a column named row_name, when the
// table has one, as the row names.
type ReadOptions struct {
	// RowNameColumn names the column whose values become the table's row
	// names, matched without regard to case; that column is not a column of
	// the result. Empty means "row_name".
	RowNameColumn string

	// Columns restricts the SELECT to these columns, in this order.
	Columns []string

	// Schema is a schema (PostgreSQL) or database (MySQL) to read the table
	// from. SQLite ignores it.
	Schema string

	Limit  int // at most this many rows; 0 means no limit
	Offset int // skip this many rows first

	// WhereClause is the body of a WHERE clause, without the keyword. It is
	// written into the SQL as it is, so never place untrusted input in it:
	// write placeholders and bind the values through Params, or use Query.
	WhereClause string
	// OrderBy is the body of an ORDER BY clause, without the keywords.
	OrderBy string
	// Params binds the placeholders written in WhereClause, in the driver's
	// own form ($1 on PostgreSQL, ? on SQLite and MySQL).
	Params []any

	// ParseDates names columns whose text should become time.Time, matched
	// without regard to case. Several ISO-style layouts are tried.
	ParseDates []string

	// DType forces the Go type of the named columns. Recognised targets are
	// reflect.TypeFor of int64, float64, bool, string, time.Time and []byte;
	// another target leaves the column as the driver returned it.
	DType map[string]reflect.Type
}

// QueryOptions are the settings of Query and StreamQuery. The zero value binds
// no parameters and takes a column named row_name, when the result has one,
// as the row names.
type QueryOptions struct {
	// Params binds the query's placeholders, in the driver's own form ($1 on
	// PostgreSQL, ? on SQLite and MySQL).
	Params []any

	// RowNameColumn, ParseDates and DType are as in ReadOptions.
	RowNameColumn string
	ParseDates    []string
	DType         map[string]reflect.Type
}

// readSettings is what a read runs with, from either options type.
type readSettings struct {
	query         string
	params        []any
	rowNameColumn string
	columns       []string
	schema        string
	limit         int
	offset        int
	whereClause   string
	orderBy       string
	parseDates    []string
	dtype         map[string]reflect.Type
}

func tableSettings(o ReadOptions) readSettings {
	return readSettings{
		params:        o.Params,
		rowNameColumn: rowNameColumnOrDefault(o.RowNameColumn),
		columns:       o.Columns,
		schema:        o.Schema,
		limit:         o.Limit,
		offset:        o.Offset,
		whereClause:   o.WhereClause,
		orderBy:       o.OrderBy,
		parseDates:    o.ParseDates,
		dtype:         o.DType,
	}
}

func querySettings(query string, o QueryOptions) readSettings {
	return readSettings{
		query:         query,
		params:        o.Params,
		rowNameColumn: rowNameColumnOrDefault(o.RowNameColumn),
		parseDates:    o.ParseDates,
		dtype:         o.DType,
	}
}

func rowNameColumnOrDefault(name string) string {
	if name == "" {
		return "row_name"
	}
	return name
}

// Read reads a table into a new DataTable. db is any database/sql connection;
// a gorm user passes the one gorm.DB() returns. dialect names the database
// behind it and has no default. The query and the scan run under ctx. A table
// that does not exist is an error naming it. At most one ReadOptions may be
// given.
func Read(ctx context.Context, db *sql.DB, dialect Dialect, table string, opts ...ReadOptions) (*insyra.DataTable, error) {
	o, err := oneOption("ReadOptions", opts)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(table) == "" {
		return nil, errors.New("the table name is empty; to run a query of your own, use Query")
	}
	return readAll(ctx, db, dialect, table, tableSettings(o))
}

// Query runs a query of your own and reads its result into a new DataTable.
// The query and its Params go to the driver as written, so they use the
// driver's own placeholders ($1 on PostgreSQL, ? on SQLite and MySQL). At most
// one QueryOptions may be given.
func Query(ctx context.Context, db *sql.DB, dialect Dialect, query string, opts ...QueryOptions) (*insyra.DataTable, error) {
	o, err := oneOption("QueryOptions", opts)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(query) == "" {
		return nil, errors.New("the query is empty")
	}
	return readAll(ctx, db, dialect, "", querySettings(query, o))
}

func readAll(ctx context.Context, db *sql.DB, dialect Dialect, table string, s readSettings) (*insyra.DataTable, error) {
	if db == nil {
		return nil, fmt.Errorf("db cannot be nil")
	}
	if err := checkDialect(dialect); err != nil {
		return nil, err
	}
	query, params, err := buildReadSQLQuery(ctx, db, string(dialect), table, s)
	if err != nil {
		return nil, err
	}
	insyra.LogDebug("sqlio", "Read", "Executing SQL query: %s", query)
	rows, err := db.QueryContext(ctx, query, params...)
	if err != nil {
		return nil, fmt.Errorf("error executing query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	dt, _, err := scanRowsToDataTable(rows, s, 0)
	if err != nil {
		return nil, err
	}
	if dt == nil {
		dt = insyra.NewDataTable()
	}
	return dt, nil
}

// Stream reads a table batchSize rows at a time. Range over it:
//
//	for dt, err := range sqlio.Stream(ctx, db, sqlio.Postgres, "orders", 10000) {
//		if err != nil {
//			return err
//		}
//		// use dt
//	}
//
// Each batch arrives as a DataTable with a nil error. A failure (building or
// running the query, scanning a row, ctx being cancelled) arrives once, as a
// nil table and the error, and ends the loop. The query runs when the loop
// starts and batches are read in the caller's own goroutine, so leaving the
// loop early closes the rows and returns the connection to the pool at once.
// batchSize must be positive. At most one ReadOptions may be given.
func Stream(ctx context.Context, db *sql.DB, dialect Dialect, table string, batchSize int, opts ...ReadOptions) iter.Seq2[*insyra.DataTable, error] {
	o, err := oneOption("ReadOptions", opts)
	if err == nil && strings.TrimSpace(table) == "" {
		err = errors.New("the table name is empty; to stream a query of your own, use StreamQuery")
	}
	return stream(ctx, db, dialect, table, batchSize, tableSettings(o), err)
}

// StreamQuery runs a query of your own and reads its result batchSize rows at
// a time, the way Stream reads a table. The query and its Params go to the
// driver as written. batchSize must be positive. At most one QueryOptions may
// be given.
func StreamQuery(ctx context.Context, db *sql.DB, dialect Dialect, query string, batchSize int, opts ...QueryOptions) iter.Seq2[*insyra.DataTable, error] {
	o, err := oneOption("QueryOptions", opts)
	if err == nil && strings.TrimSpace(query) == "" {
		err = errors.New("the query is empty")
	}
	return stream(ctx, db, dialect, "", batchSize, querySettings(query, o), err)
}

func stream(ctx context.Context, db *sql.DB, dialect Dialect, table string, batchSize int, s readSettings, setupErr error) iter.Seq2[*insyra.DataTable, error] {
	return func(yield func(*insyra.DataTable, error) bool) {
		if setupErr != nil {
			yield(nil, setupErr)
			return
		}
		if db == nil {
			yield(nil, fmt.Errorf("db cannot be nil"))
			return
		}
		if err := checkDialect(dialect); err != nil {
			yield(nil, err)
			return
		}
		if batchSize <= 0 {
			yield(nil, fmt.Errorf("batchSize must be positive, got %d", batchSize))
			return
		}

		query, params, err := buildReadSQLQuery(ctx, db, string(dialect), table, s)
		if err != nil {
			yield(nil, err)
			return
		}

		insyra.LogDebug("sqlio", "Stream", "Executing SQL query: %s (batchSize=%d)", query, batchSize)
		rows, err := db.QueryContext(ctx, query, params...)
		if err != nil {
			yield(nil, fmt.Errorf("error executing query: %w", err))
			return
		}
		// However the loop ends (the last batch, a failure, or the caller
		// breaking out) the rows are closed and the connection goes back.
		defer func() { _ = rows.Close() }()

		for {
			if err := ctx.Err(); err != nil {
				yield(nil, err)
				return
			}
			dt, done, err := scanRowsToDataTable(rows, s, batchSize)
			if err != nil {
				yield(nil, err)
				return
			}
			if dt != nil && !yield(dt, nil) {
				return
			}
			if done {
				return
			}
		}
	}
}

// buildReadSQLQuery resolves either the caller's query or constructs one
// from the table name and the shape options. When a custom Query is set, the
// table-existence pre-check is skipped (the database will report any error).
func buildReadSQLQuery(ctx context.Context, db sqlConn, dialect, tableName string, opts readSettings) (string, []any, error) {
	if opts.query != "" {
		return opts.query, opts.params, nil
	}

	if err := requireTableExists(ctx, db, dialect, opts.schema, tableName); err != nil {
		return "", nil, err
	}

	// Quote table/schema/column identifiers per-dialect to prevent SQL injection
	// through names that may originate from untrusted data (e.g. column names read
	// from a CSV/JSON header). WhereClause and OrderBy are raw SQL fragments the
	// caller writes explicitly and are treated as trusted (not quotable safely).
	full := quoteQualifiedName(dialect, opts.schema, tableName)
	selectList := "*"
	if len(opts.columns) > 0 {
		quoted := make([]string, len(opts.columns))
		for i, c := range opts.columns {
			quoted[i] = quoteSQLIdent(dialect, c)
		}
		selectList = strings.Join(quoted, ", ")
	}
	q := fmt.Sprintf("SELECT %s FROM %s", selectList, full)
	if opts.whereClause != "" {
		q += " WHERE " + opts.whereClause
	}
	if opts.orderBy != "" {
		q += " ORDER BY " + opts.orderBy
	}
	if opts.limit > 0 {
		q += fmt.Sprintf(" LIMIT %d", opts.limit)
	}
	if opts.offset > 0 {
		q += fmt.Sprintf(" OFFSET %d", opts.offset)
	}
	return q, opts.params, nil
}

func requireTableExists(ctx context.Context, db sqlConn, dialect, schema, tableName string) error {
	exists, err := tableExists(ctx, db, dialect, schema, tableName)
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
func scanRowsToDataTable(rows *sql.Rows, opts readSettings, maxRows int) (*insyra.DataTable, bool, error) {
	columnNames, err := rows.Columns()
	if err != nil {
		return nil, true, fmt.Errorf("error getting column names: %w", err)
	}
	colTypes, err := rows.ColumnTypes()
	if err != nil {
		// non-fatal: fall back to no type info
		colTypes = nil
	}

	rowNameColIndex := -1
	if opts.rowNameColumn != "" {
		for i, colName := range columnNames {
			if strings.EqualFold(colName, opts.rowNameColumn) {
				rowNameColIndex = i
				break
			}
		}
	}

	dataLists := make([]*insyra.DataList, len(columnNames))
	for i, colName := range columnNames {
		if i == rowNameColIndex {
			continue
		}
		dl := insyra.NewDataList()
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
			converted := convertSQLValue(val, ct, columnNames[i], opts)
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

	dt := insyra.NewDataTable()
	validColumns := make([]*insyra.DataList, 0, len(dataLists))
	for i, dl := range dataLists {
		if i == rowNameColIndex || dl == nil {
			continue
		}
		if dl.Len() > 0 {
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
		if cols := parseDateColumns(columnNames, rowNameColIndex, opts); len(cols) > 0 {
			// The names come from the query's own column list, so they are
			// names, not indices.
			selectors := make([]any, len(cols))
			for i, col := range cols {
				selectors[i] = insyra.Name(col)
			}
			dt.ParseDatesCols(selectors)
		}
	}
	return dt, done, nil
}

// parseDateColumns resolves ParseDates (matched case-insensitively,
// as it always has been) to the actual column names present in the result. The
// row-name column is not a column of the table, and a column whose type DType
// already forces is left to DType.
func parseDateColumns(columnNames []string, rowNameColIndex int, opts readSettings) []string {
	if len(opts.parseDates) == 0 {
		return nil
	}
	wanted := make(map[string]bool, len(opts.parseDates))
	for _, c := range opts.parseDates {
		wanted[strings.ToLower(c)] = true
	}
	cols := make([]string, 0, len(opts.parseDates))
	for i, name := range columnNames {
		if i == rowNameColIndex || !wanted[strings.ToLower(name)] {
			continue
		}
		if _, forced := opts.dtype[name]; forced {
			continue
		}
		cols = append(cols, name)
	}
	return cols
}

// convertSQLValue applies DType coercion and the historical []byte → string
// fallback while preserving binary blobs. ParseDates runs afterwards, once the
// whole table exists, through DataTable.ParseDatesCols.
func convertSQLValue(val any, ct *sql.ColumnType, colName string, opts readSettings) any {
	if val == nil {
		return nil
	}

	if target, ok := opts.dtype[colName]; ok {
		if v, ok := coerceToType(val, target); ok {
			return v
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
	return prep.ParseTimeWithLayouts(s, prep.DateParseLayouts)
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
