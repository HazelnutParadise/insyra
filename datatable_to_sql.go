package insyra

import (
	"context"
	"database/sql"
	"fmt"
	"maps"
	"reflect"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"gorm.io/gorm"
)

// defaultBatchSize is the per-INSERT row count when ToSQLOptions.BatchSize is unset.
const defaultBatchSize = 500

type ToSQLOptions struct {
	IfExists    SQLActionIfTableExists // "fail", "replace", "append"
	RowNames    bool
	ColumnTypes map[string]string // 自訂型別

	// Schema is an optional schema (PostgreSQL) or database (MySQL) name to
	// prefix the table reference with. SQLite ignores this. The caller is
	// responsible for any required quoting; the value is passed through as-is.
	Schema string

	// BatchSize controls how many rows are bundled into a single multi-row
	// INSERT. Zero falls back to defaultBatchSize. Note that the total number
	// of bind parameters per batch (BatchSize * column-count) must stay below
	// the driver limit (PostgreSQL/MySQL: 65535).
	BatchSize int
}

type SQLActionIfTableExists int

const (
	SQLActionIfTableExistsFail SQLActionIfTableExists = iota
	SQLActionIfTableExistsReplace
	SQLActionIfTableExistsAppend
)

// ToSQL writes the DataTable to the given database table.
//
// Equivalent to ToSQLContext(context.Background(), db, tableName, options...).
func (dt *DataTable) ToSQL(db *gorm.DB, tableName string, options ...ToSQLOptions) error {
	return dt.ToSQLContext(context.Background(), db, tableName, options...)
}

// ToSQLContext is the context-aware variant of ToSQL.
//
// All database calls run under ctx, so callers can cancel long writes.
// Rows are inserted with batched multi-value INSERT statements; the batch size
// is controlled by options[0].BatchSize.
func (dt *DataTable) ToSQLContext(ctx context.Context, db *gorm.DB, tableName string, options ...ToSQLOptions) error {
	if dt == nil {
		return fmt.Errorf("dt is nil")
	}
	if db == nil {
		return fmt.Errorf("db cannot be nil")
	}

	var opts ToSQLOptions
	if len(options) > 0 {
		opts = options[0]
	}
	if opts.BatchSize <= 0 {
		opts.BatchSize = defaultBatchSize
	}

	cols, rows := dt.collectRowsForSQL(opts.RowNames)
	if len(rows) == 0 {
		return fmt.Errorf("data is empty")
	}

	fullName := qualifiedTableName(opts.Schema, tableName)
	tx := db.WithContext(ctx)
	return saveRowsToDB(tx, fullName, tableName, opts.Schema, cols, rows, opts)
}

// collectRowsForSQL extracts the canonical column ordering and row values
// from the DataTable. When includeRowName is true, the first column is
// "row_name" populated from GetRowNameByIndex.
func (dt *DataTable) collectRowsForSQL(includeRowName bool) (cols []string, rows [][]any) {
	dt.AtomicDo(func(dt *DataTable) {
		numRow, numCol := dt.Size()
		if numRow == 0 {
			return
		}
		if includeRowName {
			cols = append(cols, "row_name")
		}
		for j := range numCol {
			cols = append(cols, dt.columns[j].GetName())
		}
		rows = make([][]any, 0, numRow)
		for i := range numRow {
			row := make([]any, 0, len(cols))
			if includeRowName {
				rn, ok := dt.GetRowNameByIndex(i)
				if !ok {
					rn = ""
				}
				row = append(row, rn)
			}
			r := dt.GetRow(i)
			for j := range numCol {
				row = append(row, r.Get(j))
			}
			rows = append(rows, row)
		}
	})
	return cols, rows
}

func qualifiedTableName(schema, table string) string {
	if schema == "" {
		return table
	}
	return schema + "." + table
}

func saveRowsToDB(db *gorm.DB, fullName, plainTable, schema string, cols []string, rows [][]any, opts ToSQLOptions) error {
	if len(rows) == 0 {
		return fmt.Errorf("data is empty")
	}
	return db.Transaction(func(tx *gorm.DB) error {
		dialect := tx.Name()

		// Resolve column types: caller-provided overrides take precedence;
		// otherwise infer from the first non-nil sample in each column.
		columnTypes := make(map[string]string, len(cols))
		if len(opts.ColumnTypes) > 0 {
			maps.Copy(columnTypes, opts.ColumnTypes)
		}
		for ci, col := range cols {
			if _, ok := columnTypes[col]; ok {
				continue
			}
			var sample any
			for _, row := range rows {
				if row[ci] != nil {
					sample = row[ci]
					break
				}
			}
			columnTypes[col] = inferSQLType(sample, dialect)
		}
		if opts.RowNames {
			if _, ok := columnTypes["row_name"]; !ok {
				columnTypes["row_name"] = textType(dialect)
			}
		}

		exists, err := tableExists(tx, dialect, schema, plainTable)
		if err != nil {
			return err
		}

		if exists {
			switch opts.IfExists {
			case SQLActionIfTableExistsFail:
				return fmt.Errorf("table %s already exists", fullName)
			case SQLActionIfTableExistsReplace:
				if err := tx.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s;", fullName)).Error; err != nil {
					return err
				}
				exists = false
			case SQLActionIfTableExistsAppend:
				// keep existing table
			}
		}

		if !exists {
			colDefs := make([]string, 0, len(cols))
			for _, c := range cols {
				colDefs = append(colDefs, fmt.Sprintf("%s %s", c, columnTypes[c]))
			}
			createSQL := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s);", fullName, strings.Join(colDefs, ", "))
			if err := tx.Exec(createSQL).Error; err != nil {
				return err
			}
		} else if opts.IfExists == SQLActionIfTableExistsAppend {
			// Hoist the schema check out of the per-row loop. Existing column
			// names are fetched once; missing columns are added once.
			existing, err := fetchTableColumns(tx, dialect, schema, plainTable)
			if err != nil {
				return err
			}
			existingMap := make(map[string]bool, len(existing))
			for _, c := range existing {
				existingMap[strings.ToLower(c)] = true
			}
			for _, c := range cols {
				if !existingMap[strings.ToLower(c)] {
					typ := columnTypes[c]
					if typ == "" {
						typ = textType(dialect)
					}
					alterSQL := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s;", fullName, c, typ)
					if err := tx.Exec(alterSQL).Error; err != nil {
						return err
					}
					LogInfo("DataTable", "ToSQL", "Added column %s to table %s", c, fullName)
				}
			}
		}

		// Batched multi-value INSERT.
		ph := placeholderFormat(dialect)
		for start := 0; start < len(rows); start += opts.BatchSize {
			end := start + opts.BatchSize
			if end > len(rows) {
				end = len(rows)
			}
			builder := sq.Insert(fullName).PlaceholderFormat(ph).Columns(cols...)
			for _, row := range rows[start:end] {
				builder = builder.Values(row...)
			}
			sqlStr, args, err := builder.ToSql()
			if err != nil {
				return err
			}
			if err := tx.Exec(sqlStr, args...).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func placeholderFormat(dialect string) sq.PlaceholderFormat {
	switch dialect {
	case "postgres":
		return sq.Dollar
	default:
		return sq.Question
	}
}

func textType(dialect string) string {
	_ = dialect
	return "TEXT"
}

// tableExists checks whether the given table is present, using a
// dialect-appropriate query. Errors are swallowed and treated as "not
// existing" to preserve historical behaviour.
func tableExists(tx *gorm.DB, dialect, schema, table string) (bool, error) {
	var count int
	var q *gorm.DB
	switch dialect {
	case "mysql":
		if schema != "" {
			q = tx.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_name = ? AND table_schema = ?", table, schema)
		} else {
			q = tx.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_name = ? AND table_schema = (SELECT DATABASE())", table)
		}
	case "postgres":
		if schema != "" {
			q = tx.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_name = ? AND table_schema = ?", table, schema)
		} else {
			q = tx.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_name = ? AND table_schema = current_schema()", table)
		}
	default: // sqlite or unknown
		q = tx.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table)
	}
	if err := q.Scan(&count).Error; err != nil {
		return false, nil
	}
	return count > 0, nil
}

// fetchTableColumns returns the existing column names of the given table.
func fetchTableColumns(tx *gorm.DB, dialect, schema, table string) ([]string, error) {
	var (
		rows *sql.Rows
		err  error
	)
	switch dialect {
	case "mysql":
		if schema != "" {
			rows, err = tx.Raw("SELECT COLUMN_NAME FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_NAME = ? AND TABLE_SCHEMA = ?", table, schema).Rows()
		} else {
			rows, err = tx.Raw("SELECT COLUMN_NAME FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_NAME = ? AND TABLE_SCHEMA = (SELECT DATABASE())", table).Rows()
		}
	case "postgres":
		if schema != "" {
			rows, err = tx.Raw("SELECT column_name FROM information_schema.columns WHERE table_name = ? AND table_schema = ?", table, schema).Rows()
		} else {
			rows, err = tx.Raw("SELECT column_name FROM information_schema.columns WHERE table_name = ? AND table_schema = current_schema()", table).Rows()
		}
	default: // sqlite or unknown
		rows, err = tx.Raw(fmt.Sprintf("PRAGMA table_info(%s)", table)).Rows()
	}
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []string
	switch dialect {
	case "mysql", "postgres":
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				return nil, err
			}
			out = append(out, name)
		}
	default: // PRAGMA table_info: cid, name, type, notnull, dflt_value, pk
		for rows.Next() {
			var (
				cid       int
				name      string
				typeName  string
				notnull   int
				dfltValue *string
				pk        int
			)
			if err := rows.Scan(&cid, &name, &typeName, &notnull, &dfltValue, &pk); err != nil {
				return nil, err
			}
			out = append(out, name)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// inferSQLType returns a dialect-appropriate column type for the given sample
// value. nil samples (e.g. an entirely-nil column) fall back to TEXT.
func inferSQLType(v any, dialect string) string {
	if v == nil {
		return textType(dialect)
	}
	return inferSQLTypeFromReflect(reflect.TypeOf(v), dialect)
}

func inferSQLTypeFromReflect(t reflect.Type, dialect string) string {
	if t == nil {
		return textType(dialect)
	}
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t == reflect.TypeFor[time.Time]() {
		switch dialect {
		case "postgres":
			return "TIMESTAMP"
		default:
			return "DATETIME"
		}
	}
	if t.Kind() == reflect.Slice && t.Elem().Kind() == reflect.Uint8 {
		switch dialect {
		case "postgres":
			return "BYTEA"
		default:
			return "BLOB"
		}
	}
	// sql.Null* and similar: a struct with a Valid bool plus one value field.
	if t.Kind() == reflect.Struct {
		if vf, ok := t.FieldByName("Valid"); ok && vf.Type.Kind() == reflect.Bool {
			for i := 0; i < t.NumField(); i++ {
				f := t.Field(i)
				if f.Name == "Valid" {
					continue
				}
				return inferSQLTypeFromReflect(f.Type, dialect)
			}
		}
	}
	switch dialect {
	case "mysql":
		switch t.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return "BIGINT"
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return "BIGINT UNSIGNED"
		case reflect.Float32, reflect.Float64:
			return "DOUBLE"
		case reflect.Bool:
			return "BOOLEAN"
		case reflect.String:
			return "VARCHAR(255)"
		default:
			return "TEXT"
		}
	case "postgres":
		switch t.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return "BIGINT"
		case reflect.Float32, reflect.Float64:
			return "DOUBLE PRECISION"
		case reflect.Bool:
			return "BOOLEAN"
		case reflect.String:
			return "TEXT"
		default:
			return "TEXT"
		}
	default: // sqlite / unknown
		switch t.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return "INTEGER"
		case reflect.Float32, reflect.Float64:
			return "REAL"
		case reflect.Bool:
			return "BOOLEAN"
		case reflect.String:
			return "TEXT"
		default:
			return "TEXT"
		}
	}
}
