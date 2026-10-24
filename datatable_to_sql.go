package insyra

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"regexp"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"gorm.io/gorm"
)

// defaultBatchSize is the per-INSERT row count when ToSQLOptions.BatchSize is unset.
const defaultBatchSize = 500

type ToSQLOptions struct {
	IfExists    SQLActionIfTableExists // SQLActionIfTableExistsFail (default), Replace, or Append
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
	if err := nameUnnamedSQLColumns(cols, opts.RowNames); err != nil {
		return err
	}
	if len(rows) == 0 {
		return fmt.Errorf("data is empty")
	}

	fullName := qualifiedTableName(opts.Schema, tableName)
	tx := db.WithContext(ctx)
	if opts.IfExists == SQLActionIfTableExistsReplace && db.Name() == "mysql" {
		cols, rows := dt.collectRowsForSQL(opts.RowNames)
		if err := nameUnnamedSQLColumns(cols, opts.RowNames); err != nil {
			return err
		}
		if len(rows) == 0 {
			return fmt.Errorf("data is empty")
		}
		exists, err := tableExists(tx, tx.Name(), opts.Schema, tableName)
		if err != nil {
			return err
		}
		if exists {
			return replaceThroughStagingTable(tx, fullName, tableName, opts.Schema, cols, rows, opts)
		}
		// A concurrently created table must not be dropped by the ordinary path.
		opts.IfExists = SQLActionIfTableExistsFail
	}
	return saveRowsToDB(tx, fullName, tableName, opts.Schema, cols, rows, opts)
}

const stagingCleanupTimeout = 30 * time.Second

func replaceThroughStagingTable(db *gorm.DB, fullName, plainTable, schema string, cols []string, rows [][]any, opts ToSQLOptions) error {
	dialect := db.Name()
	columnTypes := make(map[string]string, len(cols))
	if len(opts.ColumnTypes) > 0 {
		maps.Copy(columnTypes, opts.ColumnTypes)
	}
	for i, col := range cols {
		if _, ok := columnTypes[col]; ok {
			continue
		}
		var sample any
		for _, row := range rows {
			if row[i] != nil {
				sample = row[i]
				break
			}
		}
		columnTypes[col] = inferSQLType(sample, dialect)
	}
	for _, col := range cols {
		if err := validateSQLType(columnTypes[col]); err != nil {
			return err
		}
	}
	referrer, err := tableReferencedBy(db, dialect, schema, plainTable)
	if err != nil {
		return err
	}
	if referrer != "" {
		return fmt.Errorf("table %s is referenced by a foreign key from %s; nothing was changed", fullName, referrer)
	}
	var random [8]byte
	if _, err := rand.Read(random[:]); err != nil {
		return fmt.Errorf("naming staging table: %w", err)
	}
	suffix := hex.EncodeToString(random[:])
	staging, backup := "insyra_new_"+suffix, "insyra_old_"+suffix
	quotedStaging := quoteQualifiedName(dialect, schema, staging)
	ctx := db.Statement.Context
	if ctx == nil {
		ctx = context.Background()
	}
	cleanup := func(run func(*gorm.DB) error) error {
		c, cancel := context.WithTimeout(context.WithoutCancel(ctx), stagingCleanupTimeout)
		defer cancel()
		return run(db.WithContext(c))
	}
	dropStaging := func(cause error) error {
		if e := cleanup(func(tx *gorm.DB) error { return tx.Exec("DROP TABLE IF EXISTS " + quotedStaging).Error }); e != nil {
			return errors.Join(cause, fmt.Errorf("could not drop staging table: %w", e))
		}
		return cause
	}
	if err := db.Exec(fmt.Sprintf("CREATE TABLE %s (%s)", quotedStaging, sqlColumnDefs(dialect, cols, columnTypes))).Error; err != nil {
		return fmt.Errorf("creating staging table: %w", err)
	}
	if err := db.Transaction(func(tx *gorm.DB) error { return insertSQLRows(tx, dialect, quotedStaging, cols, rows, opts.BatchSize) }); err != nil {
		return dropStaging(err)
	}
	stmt := fmt.Sprintf("RENAME TABLE %s TO %s, %s TO %s", quoteQualifiedName(dialect, schema, plainTable), quoteQualifiedName(dialect, schema, backup), quotedStaging, quoteQualifiedName(dialect, schema, plainTable))
	if err := db.Exec(stmt).Error; err != nil {
		// Check what the rename did on a context of its own: when ctx itself
		// ended during the rename, querying on it would fail and leave the
		// staging table behind.
		var swapped bool
		checkErr := cleanup(func(tx *gorm.DB) error {
			backupExists, err := tableExists(tx, dialect, schema, backup)
			if err != nil {
				return err
			}
			stagingExists, err := tableExists(tx, dialect, schema, staging)
			if err != nil {
				return err
			}
			swapped = backupExists && !stagingExists
			return nil
		})
		if checkErr != nil {
			return errors.Join(fmt.Errorf("replacing table %s: %w", fullName, err), fmt.Errorf("could not verify whether the rename completed: %w", checkErr))
		}
		if !swapped {
			return dropStaging(err)
		}
	}
	if err := cleanup(func(tx *gorm.DB) error {
		return tx.Exec("DROP TABLE " + quoteQualifiedName(dialect, schema, backup)).Error
	}); err != nil {
		return fmt.Errorf("table %s has new rows but old table remains: %w", fullName, err)
	}
	return nil
}

func tableReferencedBy(db *gorm.DB, dialect, schema, table string) (string, error) {
	if dialect != "mysql" {
		return "", nil
	}
	var names []string
	query := "SELECT CONCAT(TABLE_SCHEMA, '.', TABLE_NAME) FROM information_schema.KEY_COLUMN_USAGE WHERE REFERENCED_TABLE_NAME = ? AND REFERENCED_TABLE_SCHEMA = "
	args := []any{table}
	if schema != "" {
		query += "?"
		args = append(args, schema)
	} else {
		query += "DATABASE()"
	}
	// A table referencing itself moves with the rename; the same name in
	// another database is a different table and does block it.
	query += " AND NOT (TABLE_NAME = ? AND TABLE_SCHEMA = REFERENCED_TABLE_SCHEMA) LIMIT 1"
	args = append(args, table)
	if err := db.Raw(query, args...).Scan(&names).Error; err != nil {
		return "", err
	}
	if len(names) == 0 {
		return "", nil
	}
	return names[0], nil
}

func sqlColumnDefs(dialect string, cols []string, types map[string]string) string {
	defs := make([]string, 0, len(cols))
	for _, col := range cols {
		defs = append(defs, fmt.Sprintf("%s %s", quoteSQLIdent(dialect, col), types[col]))
	}
	return strings.Join(defs, ", ")
}

func insertSQLRows(tx *gorm.DB, dialect, table string, cols []string, rows [][]any, batchSize int) error {
	if batchSize <= 0 {
		batchSize = defaultBatchSize
	}
	quoted := make([]string, len(cols))
	for i, col := range cols {
		quoted[i] = quoteSQLIdent(dialect, col)
	}
	for start := 0; start < len(rows); start += batchSize {
		end := min(start+batchSize, len(rows))
		builder := sq.Insert(table).PlaceholderFormat(placeholderFormat(dialect)).Columns(quoted...)
		for _, row := range rows[start:end] {
			builder = builder.Values(row...)
		}
		s, args, err := builder.ToSql()
		if err != nil {
			return err
		}
		if err := tx.Exec(s, args...).Error; err != nil {
			return err
		}
	}
	return nil
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

// nameUnnamedSQLColumns gives a column with no name its Excel-style letter,
// the name insyra already uses for it elsewhere: MySQL and PostgreSQL refuse
// an empty identifier. A letter another column already goes by is refused,
// compared without case because MySQL and SQLite compare column names so.
func nameUnnamedSQLColumns(cols []string, hasRowNames bool) error {
	first := 0
	if hasRowNames {
		first = 1
	}
	taken := make(map[string]bool, len(cols))
	for _, c := range cols {
		if c != "" {
			taken[strings.ToLower(c)] = true
		}
	}
	for i := first; i < len(cols); i++ {
		if cols[i] != "" {
			continue
		}
		letter, ok := ColNumberToIndex(i - first)
		if !ok {
			return fmt.Errorf("column index %d has no name", i-first)
		}
		if taken[strings.ToLower(letter)] {
			return fmt.Errorf("column %s has no name, and another column is already named %s; name the column before writing it", letter, letter)
		}
		cols[i] = letter
		taken[strings.ToLower(letter)] = true
	}
	return nil
}

func qualifiedTableName(schema, table string) string {
	if schema == "" {
		return table
	}
	return schema + "." + table
}

// quoteSQLIdent quotes a single SQL identifier for the given gorm dialect,
// escaping the quote character by doubling it. This prevents SQL injection
// through table/column/schema names that originate from untrusted data
// (e.g. column names read from a CSV/JSON header).
func quoteSQLIdent(dialect, name string) string {
	if dialect == "mysql" {
		return "`" + strings.ReplaceAll(name, "`", "``") + "`"
	}
	// postgres / sqlite / sqlserver and others use the SQL-standard double quote.
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

// quoteQualifiedName quotes an optional schema and a table name, joining them
// with a dot, so the result is a safe fully-qualified identifier.
func quoteQualifiedName(dialect, schema, table string) string {
	if schema == "" {
		return quoteSQLIdent(dialect, table)
	}
	return quoteSQLIdent(dialect, schema) + "." + quoteSQLIdent(dialect, table)
}

// sqlTypeAllowed restricts column-type declarations to a safe character set so
// that caller-supplied ColumnTypes cannot smuggle SQL. It permits identifiers,
// spaces, digits, parentheses and commas (e.g. "VARCHAR(255)", "DECIMAL(10,2)",
// "DOUBLE PRECISION") but rejects quotes, semicolons and other punctuation.
var sqlTypeAllowed = regexp.MustCompile(`^[A-Za-z0-9_ (),]+$`)

// validateSQLType returns an error if a column type declaration contains
// characters outside the safe whitelist.
func validateSQLType(typ string) error {
	if !sqlTypeAllowed.MatchString(strings.TrimSpace(typ)) {
		return fmt.Errorf("invalid SQL column type %q: only letters, digits, spaces, underscores, parentheses and commas are allowed", typ)
	}
	return nil
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

		// All identifiers below are quoted per-dialect to prevent SQL injection
		// through table/schema/column names (which may originate from untrusted
		// CSV/JSON headers), and column types are validated against a whitelist.
		quotedTable := quoteQualifiedName(dialect, schema, plainTable)
		for _, c := range cols {
			if err := validateSQLType(columnTypes[c]); err != nil {
				return err
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
				if err := tx.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s;", quotedTable)).Error; err != nil {
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
				colDefs = append(colDefs, fmt.Sprintf("%s %s", quoteSQLIdent(dialect, c), columnTypes[c]))
			}
			createSQL := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s);", quotedTable, strings.Join(colDefs, ", "))
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
					if err := validateSQLType(typ); err != nil {
						return err
					}
					alterSQL := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s;", quotedTable, quoteSQLIdent(dialect, c), typ)
					if err := tx.Exec(alterSQL).Error; err != nil {
						return err
					}
					LogInfo("DataTable", "ToSQL", "Added column %s to table %s", c, fullName)
				}
			}
		}

		// Batched multi-value INSERT. Table and column identifiers are quoted
		// (squirrel emits them verbatim); only VALUES are parameterized.
		quotedCols := make([]string, len(cols))
		for i, c := range cols {
			quotedCols[i] = quoteSQLIdent(dialect, c)
		}
		ph := placeholderFormat(dialect)
		for start := 0; start < len(rows); start += opts.BatchSize {
			end := start + opts.BatchSize
			if end > len(rows) {
				end = len(rows)
			}
			builder := sq.Insert(quotedTable).PlaceholderFormat(ph).Columns(quotedCols...)
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
		// A COUNT against information_schema/sqlite_master failing is a real DB
		// error (connection/permission/etc.), not "table absent"; propagate it
		// instead of silently reporting the table as non-existent.
		return false, err
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
		// Quote like every other statement here: an unquoted identifier breaks
		// on an ordinary name containing a space, and leaves the statement open
		// to whatever the name contains.
		rows, err = tx.Raw(fmt.Sprintf("PRAGMA table_info(%s)", quoteSQLIdent(dialect, table))).Rows()
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
