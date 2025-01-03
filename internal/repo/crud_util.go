package repo

import (
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"

	"github.com/haaag/gm/internal/config"
	"github.com/haaag/gm/internal/slice"
)

// reorderIDs reorders the IDs in the specified table.
func (r *SQLiteRepository) reorderIDs(t Table) error {
	// FIX: Every time we re-order IDs, the db's size gets bigger
	// It's a bad implementation? (but it works)
	// Maybe use 'VACUUM' command? it is safe?
	bs := slice.New[Row]()
	if err := r.Records(t, bs); err != nil {
		return err
	}

	if bs.Empty() {
		return nil
	}

	log.Printf("reordering IDs in table: %s", t)
	tempTable := "temp_" + t
	if err := r.TableCreate(tempTable, tableMainSchema); err != nil {
		return err
	}

	if err := r.insertBulk(tempTable, bs); err != nil {
		return err
	}

	if err := r.tableDrop(t); err != nil {
		return err
	}

	return r.tableRename(tempTable, t)
}

// maintenance performs maintenance tasks on the SQLite repository.
func (r *SQLiteRepository) maintenance(_ *SQLiteConfig) error {
	if err := r.checkSize(config.DB.MaxBytesSize); err != nil {
		return fmt.Errorf("%w", err)
	}

	return nil
}

// HasRecord checks if a record exists in the specified table and column.
func (r *SQLiteRepository) HasRecord(t Table, c, target string) bool {
	var recordCount int

	sqlQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s=?", t, c)

	if err := r.DB.QueryRow(sqlQuery, target).Scan(&recordCount); err != nil {
		log.Fatal(err)
	}

	return recordCount > 0
}

// IsDatabaseInitialized returns true if the database is initialized.
func (r *SQLiteRepository) IsDatabaseInitialized(t Table) bool {
	tExists, _ := r.tableExists(t)
	return tExists
}

// tableExists checks whether a table with the specified name exists in the SQLite database.
func (r *SQLiteRepository) tableExists(t Table) (bool, error) {
	query := "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name = ?"

	var count int
	if err := r.DB.QueryRow(query, t).Scan(&count); err != nil {
		log.Printf("table %s does not exist", t)
		return false, fmt.Errorf("%w: checking if table exists", err)
	}

	log.Printf("table '%s' exists: %v", t, count > 0)

	return count > 0, nil
}

// tableRename renames the temporary table to the specified main table name.
func (r *SQLiteRepository) tableRename(tempTable, mainTable Table) error {
	log.Printf("renaming table %s to %s", tempTable, mainTable)

	_, err := r.DB.Exec(fmt.Sprintf("ALTER TABLE %s RENAME TO %s", tempTable, mainTable))
	if err != nil {
		return fmt.Errorf("%w: renaming table from '%s' to '%s'", err, tempTable, mainTable)
	}

	log.Printf("renamed table %s to %s\n", tempTable, mainTable)

	return nil
}

// TableCreate creates a new table with the specified name in the SQLite database.
func (r *SQLiteRepository) TableCreate(s Table, schema string) error {
	log.Printf("creating table: %s", s)
	tableSchema := fmt.Sprintf(schema, s)

	_, err := r.DB.Exec(tableSchema)
	if err != nil {
		return fmt.Errorf("error creating table: %w", err)
	}

	return nil
}

// tableDrop drops the specified table from the SQLite database.
func (r *SQLiteRepository) tableDrop(t Table) error {
	log.Printf("dropping table: %s", t)

	_, err := r.DB.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", t))
	if err != nil {
		return fmt.Errorf("%w: dropping table '%s'", err, t)
	}

	log.Printf("dropped table: %s\n", t)

	return nil
}

// resetSQLiteSequence resets the SQLite sequence for the given table.
func (r *SQLiteRepository) resetSQLiteSequence(t Table) error {
	if _, err := r.DB.Exec("DELETE FROM sqlite_sequence WHERE name=?", t); err != nil {
		return fmt.Errorf("resetting sqlite sequence: %w", err)
	}

	return nil
}

// vacuum rebuilds the database file, repacking it into a minimal amount of
// disk space.
func (r *SQLiteRepository) vacuum() error {
	log.Println("vacuuming database")
	_, err := r.DB.Exec("VACUUM")
	if err != nil {
		return fmt.Errorf("vacuum: %w", err)
	}

	return nil
}

// size returns the size of the database.
func (r *SQLiteRepository) size() (int64, error) {
	var size int64
	err := r.DB.QueryRow("SELECT page_count * page_size FROM pragma_page_count(), pragma_page_size()").
		Scan(&size)
	if err != nil {
		return 0, fmt.Errorf("size: %w", err)
	}

	log.Printf("size of the database: %d bytes\n", size)

	return size, nil
}

// IsEmpty returns true if the database is empty.
func (r *SQLiteRepository) IsEmpty(m, d Table) bool {
	return r.maxID(m) == 0 && r.maxID(d) == 0
}

// checkSize checks the size of the database.
func (r *SQLiteRepository) checkSize(n int64) error {
	size, err := r.size()
	if err != nil {
		return fmt.Errorf("size: %w", err)
	}
	if size > n {
		return r.vacuum()
	}

	return nil
}

// DropSecure removes all records database.
func (r *SQLiteRepository) DropSecure() error {
	if err := r.deleteAll(r.Cfg.TableMain); err != nil {
		return fmt.Errorf("%w", err)
	}
	if err := r.deleteAll(r.Cfg.TableDeleted); err != nil {
		return fmt.Errorf("%w", err)
	}
	if err := r.resetSQLiteSequence(r.Cfg.TableMain); err != nil {
		return fmt.Errorf("%w", err)
	}
	if err := r.resetSQLiteSequence(r.Cfg.TableDeleted); err != nil {
		return fmt.Errorf("%w", err)
	}
	if err := r.vacuum(); err != nil {
		return fmt.Errorf("%w", err)
	}

	return nil
}

// Restore restores record/s from deleted tabled.
func (r *SQLiteRepository) Restore(bs *Slice) error {
	return r.insertBulk(Table(config.DB.MainTable), bs)
}

// IsClosed checks if the database connection is closed.
func (r *SQLiteRepository) IsClosed() bool {
	return connClosed
}
