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
)

// defaultBatchSize is the per-INSERT row count when ToSQLOptions.BatchSize is unset.
const defaultBatchSize = 500

type ToSQLOptions struct {
	IfExists    SQLActionIfTableExists // SQLActionIfTableExistsFail (default), Replace, or Append
	HasRowNames bool                   // write the row names as an extra column
	ColumnTypes map[string]string      // 自訂型別

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
	// SQLActionIfTableExistsFail returns an error when the table exists.
	SQLActionIfTableExistsFail SQLActionIfTableExists = iota
	// SQLActionIfTableExistsReplace replaces the table with one holding only
	// the new rows. A write that fails leaves the old table as it was: on
	// SQLite and PostgreSQL the drop and recreate run in one transaction, and
	// on MySQL, whose DDL commits on its own, the rows go into a staging
	// table that RENAME TABLE puts in place. On MySQL a table that another
	// table references by foreign key is refused.
	SQLActionIfTableExistsReplace
	// SQLActionIfTableExistsAppend keeps the table, adds any missing columns
	// and appends the rows.
	SQLActionIfTableExistsAppend
)

// ToSQL writes the DataTable to the given database table.
//
// db is any database/sql connection; a gorm user passes the one gorm.DB()
// returns. dialect names the database behind it and has no default.
//
// Equivalent to ToSQLContext(context.Background(), db, dialect, tableName, options...).
func (dt *DataTable) ToSQL(db *sql.DB, dialect SQLDialect, tableName string, options ...ToSQLOptions) error {
	return dt.ToSQLContext(context.Background(), db, dialect, tableName, options...)
}

// ToSQLContext is the context-aware variant of ToSQL.
//
// All database calls run under ctx, so callers can cancel long writes.
// Rows are inserted with batched multi-value INSERT statements; the batch size
// is controlled by options[0].BatchSize.
func (dt *DataTable) ToSQLContext(ctx context.Context, db *sql.DB, dialect SQLDialect, tableName string, options ...ToSQLOptions) error {
	if dt == nil {
		return fmt.Errorf("dt is nil")
	}
	if db == nil {
		return fmt.Errorf("db cannot be nil")
	}
	if err := checkSQLDialect(dialect); err != nil {
		return err
	}

	if msg := extraOptional("ToSQLOptions", len(options)); msg != "" {
		return errors.New(msg)
	}
	var opts ToSQLOptions
	if len(options) > 0 {
		opts = options[0]
	}
	return dt.toSQL(ctx, db, dialect, tableName, opts, dialect == SQLDialectMySQL)
}

// toSQL writes the table with options already resolved. ddlCommits says the
// database commits every DDL statement at once, as MySQL does, so a CREATE,
// DROP or ALTER inside a transaction cannot be rolled back. There a new or
// replacing table is written under a staging name that a rename puts in
// place, and columns an append adds are dropped again if the rows fail.
// SQLite and PostgreSQL run DDL inside a transaction, where a failed write
// already undoes everything, and a rename would carry the views that read
// the table over to the old copy.
func (dt *DataTable) toSQL(ctx context.Context, db *sql.DB, sqlDialect SQLDialect, tableName string, opts ToSQLOptions, ddlCommits bool) error {
	if opts.BatchSize <= 0 {
		opts.BatchSize = defaultBatchSize
	}
	cols, rows := dt.collectRowsForSQL(opts.HasRowNames)
	if err := nameUnnamedSQLColumns(cols, opts.HasRowNames); err != nil {
		return err
	}
	if len(rows) == 0 {
		return fmt.Errorf("data is empty")
	}

	fullName := qualifiedTableName(opts.Schema, tableName)
	dialect := string(sqlDialect)
	if opts.IfExists == SQLActionIfTableExistsReplace && ddlCommits {
		exists, err := tableExists(ctx, db, dialect, opts.Schema, tableName)
		if err != nil {
			return err
		}
		if exists {
			return replaceThroughStagingTable(ctx, db, dialect, fullName, tableName, opts.Schema, cols, rows, opts)
		}
		// Nothing to replace, so the table is created. Should another session
		// create it meanwhile, refuse rather than fall back to dropping it,
		// which is what this path avoids.
		opts.IfExists = SQLActionIfTableExistsFail
	}
	if ddlCommits {
		return saveRowsWithCommittedDDL(ctx, db, dialect, fullName, tableName, opts.Schema, cols, rows, opts)
	}
	return saveRowsToDB(ctx, db, dialect, fullName, tableName, opts.Schema, cols, rows, opts)
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
			return fmt.Errorf("column %d has no name", i-first+1)
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

// quoteSQLIdent quotes a single SQL identifier for the given dialect,
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

func saveRowsToDB(ctx context.Context, db *sql.DB, dialect, fullName, plainTable, schema string, cols []string, rows [][]any, opts ToSQLOptions) error {
	if len(rows) == 0 {
		return fmt.Errorf("data is empty")
	}
	return inSQLTx(ctx, db, func(tx *sql.Tx) error {

		// All identifiers below are quoted per-dialect to prevent SQL injection
		// through table/schema/column names (which may originate from untrusted
		// CSV/JSON headers), and column types are validated against a whitelist.
		columnTypes, err := resolveSQLColumnTypes(dialect, cols, rows, opts)
		if err != nil {
			return err
		}
		quotedTable := quoteQualifiedName(dialect, schema, plainTable)

		exists, err := tableExists(ctx, tx, dialect, schema, plainTable)
		if err != nil {
			return err
		}

		if exists {
			switch opts.IfExists {
			case SQLActionIfTableExistsFail:
				return fmt.Errorf("table %s already exists", fullName)
			case SQLActionIfTableExistsReplace:
				if _, err := tx.ExecContext(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s;", quotedTable)); err != nil {
					return err
				}
				exists = false
			case SQLActionIfTableExistsAppend:
				// keep existing table
			}
		}

		if !exists {
			createSQL := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s);", quotedTable, sqlColumnDefs(dialect, cols, columnTypes))
			if _, err := tx.ExecContext(ctx, createSQL); err != nil {
				return err
			}
		} else if opts.IfExists == SQLActionIfTableExistsAppend {
			// Hoist the schema check out of the per-row loop. Existing column
			// names are fetched once; missing columns are added once.
			existing, err := fetchTableColumns(ctx, tx, dialect, schema, plainTable)
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
					if _, err := tx.ExecContext(ctx, alterSQL); err != nil {
						return err
					}
					LogInfo("DataTable", "ToSQL", "Added column %s to table %s", c, fullName)
				}
			}
		}

		return insertSQLRows(ctx, tx, dialect, quotedTable, cols, rows, opts.BatchSize)
	})
}

// saveRowsWithCommittedDDL is saveRowsToDB for a database whose DDL commits on
// its own. A missing table is created under a staging name and renamed in
// once every row is written; a rename to a name that is taken fails, so a
// table another session creates meanwhile is refused, not overwritten. An
// append adds its missing columns first, writes the rows in one transaction,
// and drops the columns it added if the rows fail.
func saveRowsWithCommittedDDL(ctx context.Context, db *sql.DB, dialect, fullName, plainTable, schema string, cols []string, rows [][]any, opts ToSQLOptions) error {
	columnTypes, err := resolveSQLColumnTypes(dialect, cols, rows, opts)
	if err != nil {
		return err
	}
	exists, err := tableExists(ctx, db, dialect, schema, plainTable)
	if err != nil {
		return err
	}
	if !exists {
		return createThroughStagingTable(ctx, db, dialect, fullName, plainTable, schema, cols, rows, columnTypes, opts)
	}
	if opts.IfExists != SQLActionIfTableExistsAppend {
		return fmt.Errorf("table %s already exists", fullName)
	}

	quotedTable := quoteQualifiedName(dialect, schema, plainTable)
	existing, err := fetchTableColumns(ctx, db, dialect, schema, plainTable)
	if err != nil {
		return err
	}
	have := make(map[string]bool, len(existing))
	for _, c := range existing {
		have[strings.ToLower(c)] = true
	}
	var added []string
	dropAdded := func(cause error) error {
		var left []string
		for _, c := range added {
			if err := runSQLCleanup(ctx, func(ctx context.Context) error {
				_, err := db.ExecContext(ctx, fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s;", quotedTable, quoteSQLIdent(dialect, c)))
				return err
			}); err != nil {
				left = append(left, c)
				cause = errors.Join(cause, err)
			}
		}
		if len(left) > 0 {
			return errors.Join(cause, fmt.Errorf("the columns %s added to table %s could not be dropped again", strings.Join(left, ", "), fullName))
		}
		return cause
	}
	for _, c := range cols {
		if have[strings.ToLower(c)] {
			continue
		}
		typ := columnTypes[c]
		if typ == "" {
			typ = textType(dialect)
		}
		if err := validateSQLType(typ); err != nil {
			return dropAdded(err)
		}
		if _, err := db.ExecContext(ctx, fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s;", quotedTable, quoteSQLIdent(dialect, c), typ)); err != nil {
			return dropAdded(err)
		}
		added = append(added, c)
	}
	if err := inSQLTx(ctx, db, func(tx *sql.Tx) error {
		return insertSQLRows(ctx, tx, dialect, quotedTable, cols, rows, opts.BatchSize)
	}); err != nil {
		return dropAdded(err)
	}
	for _, c := range added {
		LogInfo("DataTable", "ToSQL", "Added column %s to table %s", c, fullName)
	}
	return nil
}

// createThroughStagingTable creates a missing table without ever leaving an
// incomplete one under its name: the rows go into a staging table, which a
// rename then puts under the target's name.
func createThroughStagingTable(ctx context.Context, db *sql.DB, dialect, fullName, plainTable, schema string, cols []string, rows [][]any, columnTypes map[string]string, opts ToSQLOptions) error {
	suffix, err := stagingSuffix()
	if err != nil {
		return err
	}
	staging := "insyra_new_" + suffix
	quotedStaging := quoteQualifiedName(dialect, schema, staging)
	dropStaging := func(cause error) error {
		if err := runSQLCleanup(ctx, func(ctx context.Context) error {
			_, err := db.ExecContext(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s;", quotedStaging))
			return err
		}); err != nil {
			return errors.Join(cause, fmt.Errorf("could not drop the staging table %s: %w", qualifiedTableName(schema, staging), err))
		}
		return cause
	}
	if _, err := db.ExecContext(ctx, fmt.Sprintf("CREATE TABLE %s (%s);", quotedStaging, sqlColumnDefs(dialect, cols, columnTypes))); err != nil {
		return fmt.Errorf("creating the staging table %s: %w (creating a table on MySQL writes it under a new name first, which needs the CREATE, INSERT, ALTER and DROP privileges on the database)",
			qualifiedTableName(schema, staging), err)
	}
	if err := inSQLTx(ctx, db, func(tx *sql.Tx) error {
		return insertSQLRows(ctx, tx, dialect, quotedStaging, cols, rows, opts.BatchSize)
	}); err != nil {
		return dropStaging(err)
	}
	renameErr := func() error {
		_, err := db.ExecContext(ctx, renameStatement(dialect, schema, staging, plainTable))
		return err
	}()
	if renameErr == nil {
		return nil
	}
	// The rename may have run even though no answer came back. It did if the
	// staging table is gone; if it is still there and the name is taken,
	// another session created the table meanwhile.
	var stagingLeft, taken bool
	if err := runSQLCleanup(ctx, func(ctx context.Context) error {
		var err error
		if stagingLeft, err = tableExists(ctx, db, dialect, schema, staging); err != nil {
			return err
		}
		taken, err = tableExists(ctx, db, dialect, schema, plainTable)
		return err
	}); err != nil {
		return errors.Join(fmt.Errorf("creating table %s: %w", fullName, renameErr),
			fmt.Errorf("could not check whether the rename happened: %w; if %s exists, it holds the new rows", err, qualifiedTableName(schema, staging)))
	}
	if !stagingLeft {
		return nil
	}
	if taken {
		return dropStaging(fmt.Errorf("table %s already exists", fullName))
	}
	return dropStaging(fmt.Errorf("creating table %s: %w", fullName, renameErr))
}

// renameStatement puts the table from under the name to.
func renameStatement(dialect, schema, from, to string) string {
	if dialect == "mysql" {
		return fmt.Sprintf("RENAME TABLE %s TO %s", quoteQualifiedName(dialect, schema, from), quoteQualifiedName(dialect, schema, to))
	}
	return fmt.Sprintf("ALTER TABLE %s RENAME TO %s", quoteQualifiedName(dialect, schema, from), quoteSQLIdent(dialect, to))
}

// stagingSuffix is the random part of a staging table's name.
func stagingSuffix() (string, error) {
	var random [8]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", fmt.Errorf("naming the staging table: %w", err)
	}
	return hex.EncodeToString(random[:]), nil
}

// runSQLCleanup runs one cleanup statement with a context the caller cannot
// cancel, bounded by stagingCleanupTimeout, so that cancelling a write does
// not also cancel undoing it.
func runSQLCleanup(ctx context.Context, run func(ctx context.Context) error) error {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), stagingCleanupTimeout)
	defer cancel()
	return run(ctx)
}

// resolveSQLColumnTypes gives every column its SQL type: the caller's
// ColumnTypes first, otherwise one inferred from the column's first non-nil
// value. Every type is checked against the safe character set.
func resolveSQLColumnTypes(dialect string, cols []string, rows [][]any, opts ToSQLOptions) (map[string]string, error) {
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
	if opts.HasRowNames {
		if _, ok := columnTypes["row_name"]; !ok {
			columnTypes["row_name"] = textType(dialect)
		}
	}
	for _, c := range cols {
		if err := validateSQLType(columnTypes[c]); err != nil {
			return nil, err
		}
	}
	return columnTypes, nil
}

// sqlColumnDefs is the column list of a CREATE TABLE.
func sqlColumnDefs(dialect string, cols []string, columnTypes map[string]string) string {
	colDefs := make([]string, 0, len(cols))
	for _, c := range cols {
		colDefs = append(colDefs, fmt.Sprintf("%s %s", quoteSQLIdent(dialect, c), columnTypes[c]))
	}
	return strings.Join(colDefs, ", ")
}

// insertSQLRows writes rows with batched multi-value INSERTs. Table and column
// identifiers are quoted; only VALUES are parameterized.
func insertSQLRows(ctx context.Context, tx sqlConn, dialect, quotedTable string, cols []string, rows [][]any, batchSize int) error {
	quotedCols := make([]string, len(cols))
	for i, c := range cols {
		quotedCols[i] = quoteSQLIdent(dialect, c)
	}
	head := "INSERT INTO " + quotedTable + " (" + strings.Join(quotedCols, ",") + ") VALUES "
	rowMarks := "(" + strings.TrimSuffix(strings.Repeat("?,", len(cols)), ",") + ")"
	for start := 0; start < len(rows); start += batchSize {
		end := min(start+batchSize, len(rows))
		var b strings.Builder
		b.WriteString(head)
		args := make([]any, 0, (end-start)*len(cols))
		for i, row := range rows[start:end] {
			if len(row) != len(cols) {
				return fmt.Errorf("row %d has %d values for %d columns", start+i+1, len(row), len(cols))
			}
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(rowMarks)
			args = append(args, row...)
		}
		if _, err := tx.ExecContext(ctx, bindVars(dialect, b.String()), args...); err != nil {
			return err
		}
	}
	return nil
}

// stagingCleanupTimeout bounds each statement that cleans up after a staged
// replace. The cleanup outlives the caller's context, so that cancelling the
// write does not leave the staging table behind, but it must not wait without
// end on a lock that an interrupted INSERT still holds.
const stagingCleanupTimeout = 30 * time.Second

// replaceThroughStagingTable replaces an existing table without ever leaving
// the caller without it, for a database whose DDL commits on its own.
//
// A table that another table references by foreign key is refused before
// anything changes: a rename would move that reference to the old copy. The
// rows then go into a new staging table, inside a transaction of their own.
// Only once they are all written does a rename put the staging table under the
// target's name and the old table under a backup name, which is then dropped.
// A failure before the rename drops the staging table and leaves the target
// untouched.
func replaceThroughStagingTable(ctx context.Context, db *sql.DB, dialect, fullName, plainTable, schema string, cols []string, rows [][]any, opts ToSQLOptions) error {
	columnTypes, err := resolveSQLColumnTypes(dialect, cols, rows, opts)
	if err != nil {
		return err
	}
	referrer, err := tableReferencedBy(ctx, db, dialect, schema, plainTable)
	if err != nil {
		return err
	}
	if referrer != "" {
		return fmt.Errorf("table %s is referenced by a foreign key from %s, and replacing it would leave that reference on the old copy; nothing was changed: drop the constraint first, or append", fullName, referrer)
	}
	suffix, err := stagingSuffix()
	if err != nil {
		return err
	}
	staging, backup := "insyra_new_"+suffix, "insyra_old_"+suffix
	quotedStaging := quoteQualifiedName(dialect, schema, staging)

	cleanup := func(run func(ctx context.Context) error) error { return runSQLCleanup(ctx, run) }
	dropStaging := func(cause error) error {
		if err := cleanup(func(ctx context.Context) error {
			_, err := db.ExecContext(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s;", quotedStaging))
			return err
		}); err != nil {
			return errors.Join(cause, fmt.Errorf("could not drop the staging table %s: %w", qualifiedTableName(schema, staging), err))
		}
		return cause
	}

	createSQL := fmt.Sprintf("CREATE TABLE %s (%s);", quotedStaging, sqlColumnDefs(dialect, cols, columnTypes))
	if _, err := db.ExecContext(ctx, createSQL); err != nil {
		// Nothing was created, so there is nothing to drop.
		return fmt.Errorf("creating the staging table %s: %w (replacing a table on MySQL creates tables under new names, which needs the CREATE, INSERT, ALTER and DROP privileges on the database)",
			qualifiedTableName(schema, staging), err)
	}
	if err := inSQLTx(ctx, db, func(tx *sql.Tx) error {
		return insertSQLRows(ctx, tx, dialect, quotedStaging, cols, rows, opts.BatchSize)
	}); err != nil {
		return dropStaging(err)
	}

	stmts := renameSwapStatements(dialect, schema, plainTable, staging, backup)
	var swapErr error
	if len(stmts) == 1 {
		// MySQL's RENAME TABLE is atomic on its own and commits on its own,
		// so a transaction around it would only add a COMMIT that can fail
		// after the rename has happened.
		_, swapErr = db.ExecContext(ctx, stmts[0])
	} else {
		// Two statements: one transaction, so a failed second one undoes
		// the first.
		swapErr = inSQLTx(ctx, db, func(tx *sql.Tx) error {
			for _, stmt := range stmts {
				if _, err := tx.ExecContext(ctx, stmt); err != nil {
					return err
				}
			}
			return nil
		})
	}
	if swapErr != nil {
		// The rename may have run even though no answer came back, over a
		// dropped connection or a context cancelled at that moment. Look.
		var swapped bool
		checkErr := cleanup(func(ctx context.Context) error {
			var err error
			swapped, err = renameSwapHappened(ctx, db, dialect, schema, staging, backup)
			return err
		})
		if checkErr != nil {
			return errors.Join(fmt.Errorf("replacing table %s: %w", fullName, swapErr),
				fmt.Errorf("could not check whether the rename happened: %w; if %s exists, it holds the old rows and %s holds the new ones, otherwise %s holds the new rows",
					checkErr, qualifiedTableName(schema, backup), fullName, qualifiedTableName(schema, staging)))
		}
		if !swapped {
			return dropStaging(fmt.Errorf("replacing table %s: %w", fullName, swapErr))
		}
	}
	if err := cleanup(func(ctx context.Context) error {
		_, err := db.ExecContext(ctx, fmt.Sprintf("DROP TABLE %s;", quoteQualifiedName(dialect, schema, backup)))
		return err
	}); err != nil {
		return fmt.Errorf("table %s now holds the new rows, but its previous contents remain in %s, which could not be dropped: %w",
			fullName, qualifiedTableName(schema, backup), err)
	}
	return nil
}

// renameSwapHappened reports whether the rename of a staged replace took
// effect: the staging table is gone and the backup exists.
func renameSwapHappened(ctx context.Context, db sqlConn, dialect, schema, staging, backup string) (bool, error) {
	stagingLeft, err := tableExists(ctx, db, dialect, schema, staging)
	if err != nil {
		return false, err
	}
	backupMade, err := tableExists(ctx, db, dialect, schema, backup)
	if err != nil {
		return false, err
	}
	return !stagingLeft && backupMade, nil
}

// tableReferencedBy names a table, other than table itself, that references
// table by foreign key, or returns "" when none does or the dialect is one the
// staged replace does not run on.
func tableReferencedBy(ctx context.Context, db sqlConn, dialect, schema, table string) (string, error) {
	var query string
	var args []any
	switch dialect {
	case "mysql":
		const refs = "SELECT TABLE_NAME FROM information_schema.REFERENTIAL_CONSTRAINTS WHERE REFERENCED_TABLE_NAME = ? AND NOT (TABLE_NAME = REFERENCED_TABLE_NAME AND CONSTRAINT_SCHEMA = UNIQUE_CONSTRAINT_SCHEMA) AND UNIQUE_CONSTRAINT_SCHEMA = "
		if schema != "" {
			query, args = refs+"? LIMIT 1", []any{table, schema}
		} else {
			query, args = refs+"(SELECT DATABASE()) LIMIT 1", []any{table}
		}
	case "sqlite":
		query, args = `SELECT m.name FROM sqlite_master AS m JOIN pragma_foreign_key_list(m.name) AS f WHERE m.type = 'table' AND f."table" = ? COLLATE NOCASE AND m.name <> ? COLLATE NOCASE LIMIT 1`, []any{table, table}
	default:
		return "", nil
	}
	var name string
	err := db.QueryRowContext(ctx, bindVars(dialect, query), args...).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return name, nil
}

// renameSwapStatements puts staging under the name table and table under the
// name backup. MySQL does it in one RENAME TABLE statement, which it runs
// atomically, so no session sees a moment with no table under the name.
// Elsewhere it is two ALTER TABLE statements.
func renameSwapStatements(dialect, schema, table, staging, backup string) []string {
	q := func(name string) string { return quoteQualifiedName(dialect, schema, name) }
	if dialect == "mysql" {
		return []string{fmt.Sprintf("RENAME TABLE %s TO %s, %s TO %s", q(table), q(backup), q(staging), q(table))}
	}
	return []string{
		fmt.Sprintf("ALTER TABLE %s RENAME TO %s", q(table), quoteSQLIdent(dialect, backup)),
		fmt.Sprintf("ALTER TABLE %s RENAME TO %s", q(staging), quoteSQLIdent(dialect, table)),
	}
}

func textType(dialect string) string {
	_ = dialect
	return "TEXT"
}

// tableExists checks whether the given table is present, using a
// dialect-appropriate query. Errors are swallowed and treated as "not
// existing" to preserve historical behaviour.
func tableExists(ctx context.Context, tx sqlConn, dialect, schema, table string) (bool, error) {
	var count int
	var query string
	var args []any
	switch dialect {
	case "mysql":
		if schema != "" {
			query, args = "SELECT COUNT(*) FROM information_schema.tables WHERE table_name = ? AND table_schema = ?", []any{table, schema}
		} else {
			query, args = "SELECT COUNT(*) FROM information_schema.tables WHERE table_name = ? AND table_schema = (SELECT DATABASE())", []any{table}
		}
	case "postgres":
		if schema != "" {
			query, args = "SELECT COUNT(*) FROM information_schema.tables WHERE table_name = ? AND table_schema = ?", []any{table, schema}
		} else {
			query, args = "SELECT COUNT(*) FROM information_schema.tables WHERE table_name = ? AND table_schema = current_schema()", []any{table}
		}
	default: // sqlite or unknown
		query, args = "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", []any{table}
	}
	if err := tx.QueryRowContext(ctx, bindVars(dialect, query), args...).Scan(&count); err != nil {
		// A COUNT against information_schema/sqlite_master failing is a real DB
		// error (connection/permission/etc.), not "table absent"; propagate it
		// instead of silently reporting the table as non-existent.
		return false, err
	}
	return count > 0, nil
}

// fetchTableColumns returns the existing column names of the given table.
func fetchTableColumns(ctx context.Context, tx sqlConn, dialect, schema, table string) ([]string, error) {
	var (
		rows *sql.Rows
		err  error
	)
	switch dialect {
	case "mysql":
		if schema != "" {
			rows, err = tx.QueryContext(ctx, bindVars(dialect, "SELECT COLUMN_NAME FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_NAME = ? AND TABLE_SCHEMA = ?"), table, schema)
		} else {
			rows, err = tx.QueryContext(ctx, bindVars(dialect, "SELECT COLUMN_NAME FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_NAME = ? AND TABLE_SCHEMA = (SELECT DATABASE())"), table)
		}
	case "postgres":
		if schema != "" {
			rows, err = tx.QueryContext(ctx, bindVars(dialect, "SELECT column_name FROM information_schema.columns WHERE table_name = ? AND table_schema = ?"), table, schema)
		} else {
			rows, err = tx.QueryContext(ctx, bindVars(dialect, "SELECT column_name FROM information_schema.columns WHERE table_name = ? AND table_schema = current_schema()"), table)
		}
	default: // sqlite or unknown
		// Quote like every other statement here: an unquoted identifier breaks
		// on an ordinary name containing a space, and leaves the statement open
		// to whatever the name contains.
		rows, err = tx.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s)", quoteSQLIdent(dialect, table)))
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
