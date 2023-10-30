package database

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	bm "gomarks/pkg/bookmark"
	c "gomarks/pkg/constants"
	u "gomarks/pkg/util"

	_ "github.com/mattn/go-sqlite3"
)

var (
	ErrDuplicate    = errors.New("sql: record already exists")
	ErrNotExists    = errors.New("sql: row not exists")
	ErrUpdateFailed = errors.New("sql: update failed")
	ErrDeleteFailed = errors.New("sql: delete failed")
)

type SQLiteRepository struct {
	DB *sql.DB
}

func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{
		DB: db,
	}
}

func GetDB() *SQLiteRepository {
	dbPath := u.GetDBPath()
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal("Error opening database:", err)
	}

	r := NewSQLiteRepository(db)
	if exists, _ := r.TableExists(c.DBMainTableName); !exists {
		r.initDB()
		s := r.GetDBInfo()
		fmt.Println(s)
	}
	return r
}

func (r *SQLiteRepository) initDB() {
	err := r.CreateTable(c.DBMainTableName)
	if err != nil {
		log.Fatal(err)
	}

	err = r.CreateTable(c.DBDeletedTableName)
	if err != nil {
		log.Fatal(err)
	}

	if _, err := r.InsertRecord(&bm.InitBookmark, c.DBMainTableName); err != nil {
		return
	}
}

func (r *SQLiteRepository) DropTable(t string) error {
	log.Printf("Dropping table: %s", t)
	_, err := r.DB.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", t))
	if err != nil {
		return err
	}
	log.Printf("Dropped table: %s\n", t)
	return nil
}

func (r *SQLiteRepository) InsertRecord(b *bm.Bookmark, tableName string) (*bm.Bookmark, error) {
	if !b.IsValid() {
		return nil, fmt.Errorf("invalid bookmark: %s", ErrNotExists)
	}

	if r.RecordExists(b.URL, tableName) {
		return nil, fmt.Errorf(
			"bookmark already exists in %s table %s: %s",
			tableName,
			ErrDuplicate,
			b.URL,
		)
	}

	currentTime := time.Now()
	sqlQuery := fmt.Sprintf(
		`INSERT INTO %s(
      url, title, tags, desc, created_at)
      VALUES(?, ?, ?, ?, ?)`, tableName)
	result, err := r.DB.Exec(
		sqlQuery,
		b.URL,
		b.Title,
		b.Tags,
		b.Desc,
		currentTime.Format("2006-01-02 15:04:05"),
	)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	b.ID = int(id)

	log.Printf("Inserted bookmark: %s (table: %s)\n", b.URL, tableName)
	return b, nil
}

func (r *SQLiteRepository) UpdateRecord(b *bm.Bookmark, t string) (bm.Bookmark, error) {
	if !r.RecordExists(b.URL, t) {
		return *b, fmt.Errorf("error updating bookmark %s: %s", ErrNotExists, b.URL)
	}
	sqlQuery := fmt.Sprintf(
		"UPDATE %s SET url = ?, title = ?, tags = ?, desc = ?, created_at = ? WHERE id = ?",
		t,
	)
	_, err := r.DB.Exec(sqlQuery, b.URL, b.Title, b.Tags, b.Desc, b.Created_at, b.ID)
	if err != nil {
		return *b, fmt.Errorf("error updating bookmark %s: %s", ErrUpdateFailed, err)
	}
	log.Printf("Updated bookmark %s (table: %s)\n", b.URL, t)
	return *b, nil
}

func (r *SQLiteRepository) DeleteRecord(b *bm.Bookmark, tableName string) error {
	log.Printf("Deleting bookmark %s (table: %s)\n", b.URL, tableName)
	if !r.RecordExists(b.URL, tableName) {
		return fmt.Errorf("error removing bookmark %s: %s", ErrNotExists, b.URL)
	}
	_, err := r.DB.Exec(fmt.Sprintf("DELETE FROM %s WHERE id = ?", tableName), b.ID)
	if err != nil {
		return fmt.Errorf("error removing bookmark %s: %s", ErrDeleteFailed, err)
	}
	log.Printf("Deleted bookmark %s (table: %s)\n", b.URL, tableName)
	return nil
}

func (r *SQLiteRepository) GetRecordByID(n int, t string) (bm.Bookmark, error) {
	log.Printf("Getting bookmark by ID %d (table: %s)\n", n, t)
	row := r.DB.QueryRow(fmt.Sprintf("SELECT * FROM %s WHERE id = ?", t), n)

	var b bm.Bookmark
	err := row.Scan(&b.ID, &b.URL, &b.Title, &b.Tags, &b.Desc, &b.Created_at)
	if err != nil {
		if err == sql.ErrNoRows {
			return bm.Bookmark{}, fmt.Errorf("bookmark with ID %d not found", n)
		}
		return bm.Bookmark{}, err
	}
	log.Printf("Got bookmark by ID %d (table: %s)\n", n, t)
	return b, nil
}

func (r *SQLiteRepository) getRecordsBySQL(q string, args ...interface{}) ([]bm.Bookmark, error) {
	rows, err := r.DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var all []bm.Bookmark
	for rows.Next() {
		var b bm.Bookmark
		if err := rows.Scan(&b.ID, &b.URL, &b.Title, &b.Tags, &b.Desc, &b.Created_at); err != nil {
			return nil, err
		}
		all = append(all, b)
	}
	return all, nil
}

func (r *SQLiteRepository) GetRecordsAll(t string) ([]bm.Bookmark, error) {
	log.Printf("Getting all records from table: '%s'", t)
	sqlQuery := fmt.Sprintf("SELECT * FROM %s ORDER BY id ASC", t)
	bookmarks, err := r.getRecordsBySQL(sqlQuery)
	if err != nil {
		log.Fatal(err)
	}
	if len(bookmarks) == 0 {
		log.Printf("No records found in table: '%s'", t)
		return []bm.Bookmark{}, fmt.Errorf("no bookmarks found")
	}
	log.Printf("Got %d records from table: '%s'", len(bookmarks), t)
	return bookmarks, nil
}

func (r *SQLiteRepository) GetRecordsByQuery(q, t string) ([]bm.Bookmark, error) {
	log.Printf("Getting records by query: %s", q)
	sqlQuery := fmt.Sprintf(
		`SELECT 
        id, url, title, tags, desc, created_at
      FROM %s 
      WHERE id LIKE ? 
        OR title LIKE ? 
        OR url LIKE ? 
        OR tags LIKE ? 
        OR desc LIKE ?
      ORDER BY id ASC
    `,
		t,
	)
	queryValue := "%" + q + "%"
	bs, err := r.getRecordsBySQL(
		sqlQuery,
		queryValue,
		queryValue,
		queryValue,
		queryValue,
		queryValue,
	)
	if err != nil {
		return nil, err
	}
	if len(bs) == 0 {
		log.Printf("No records found by query: '%s'", q)
		return []bm.Bookmark{}, fmt.Errorf("no bookmarks found")
	}
	log.Printf("Got %d records by query: '%s'", len(bs), q)
	return bs, err
}

func (r *SQLiteRepository) RecordExists(url, tableName string) bool {
	sqlQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE url=?", tableName)
	var recordCount int
	err := r.DB.QueryRow(sqlQuery, url).Scan(&recordCount)
	if err != nil {
		log.Fatal(err)
	}
	return recordCount > 0
}

func (r *SQLiteRepository) getMaxID() int {
	sqlQuery := fmt.Sprintf("SELECT COALESCE(MAX(id), 0) FROM %s", c.DBMainTableName)
	var lastIndex int
	err := r.DB.QueryRow(sqlQuery).Scan(&lastIndex)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Max ID: %d", lastIndex)
	return lastIndex
}

func (r *SQLiteRepository) TableExists(t string) (bool, error) {
	log.Printf("Checking if table '%s' exists", t)
	query := "SELECT name FROM sqlite_master WHERE type='table' AND name = ?"
	rows, err := r.DB.Query(query, t)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	if !rows.Next() {
		return false, nil
	}
	log.Printf("Table '%s' exists", t)
	return true, nil
}

func (r *SQLiteRepository) InsertRecordsBulk(tempTable string, bookmarks []bm.Bookmark) error {
	log.Printf("Inserting %d bookmarks into table: %s", len(bookmarks), tempTable)
	tx, err := r.DB.Begin()
	if err != nil {
		return err
	}

	sqlQuery := fmt.Sprintf(
		"INSERT INTO %s (url, title, tags, desc, created_at) VALUES (?, ?, ?, ?, ?)",
		tempTable,
	)
	stmt, err := tx.Prepare(sqlQuery)
	if err != nil {
		err = tx.Rollback()
		return err
	}

	for _, b := range bookmarks {
		_, err = stmt.Exec(b.URL, b.Title, b.Tags, b.Desc, b.Created_at)
		if err != nil {
			err = tx.Rollback()
			return err
		}
	}

	if err := stmt.Close(); err != nil {
		err = tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	log.Printf("Inserted %d bookmarks into table: %s", len(bookmarks), tempTable)
	return nil
}

func (r *SQLiteRepository) RenameTable(tempTable string, mainTable string) error {
	log.Printf("Renaming table %s to %s", tempTable, mainTable)
	_, err := r.DB.Exec(fmt.Sprintf("ALTER TABLE %s RENAME TO %s", tempTable, mainTable))
	if err != nil {
		return err
	}
	log.Printf("Renamed table %s to %s\n", tempTable, mainTable)
	return nil
}

func (r *SQLiteRepository) CreateTable(s string) error {
	log.Printf("Creating table: %s", s)
	schema := fmt.Sprintf(c.MainTableSchema, s)
	_, err := r.DB.Exec(schema)
	if err != nil {
		return fmt.Errorf("error creating table: %s", err)
	}
	log.Printf("Created table: %s\n", s)
	return err
}

func (r *SQLiteRepository) ReorderIDs() error {
	log.Printf("Reordering IDs in table: %s", c.DBMainTableName)

	if r.getMaxID() == 0 {
		return nil
	}

	tempTable := fmt.Sprintf("temp_%s", c.DBMainTableName)
	if err := r.CreateTable(tempTable); err != nil {
		return err
	}

	bookmarks, err := r.GetRecordsAll(c.DBMainTableName)
	if err != nil {
		return err
	}

	if err := r.InsertRecordsBulk(tempTable, bookmarks); err != nil {
		return err
	}

	if err := r.DropTable(c.DBMainTableName); err != nil {
		return err
	}

	if err := r.RenameTable(tempTable, c.DBMainTableName); err != nil {
		return err
	}

	return nil
}

func (r *SQLiteRepository) TagsWithCount() (u.Counter, error) {
	tagCounter := make(u.Counter)

	bookmarks, err := r.GetRecordsAll(c.DBMainTableName)
	if err != nil {
		return nil, err
	}
	for _, bookmark := range bookmarks {
		tagCounter.Add(bookmark.Tags, ",")
	}
	return tagCounter, nil
}

func (r *SQLiteRepository) GetRecordsByTag(t string) ([]bm.Bookmark, error) {
	bookmarks, err := r.getRecordsBySQL(
		fmt.Sprintf("SELECT * FROM %s WHERE tags LIKE ?", c.DBMainTableName),
		fmt.Sprintf("%%%s%%", t),
	)
	if err != nil {
		return nil, err
	}
	if len(bookmarks) == 0 {
		return []bm.Bookmark{}, fmt.Errorf("no bookmarks found")
	}
	return bookmarks, nil
}

func (r *SQLiteRepository) GetLastRecords(n int, t string) ([]bm.Bookmark, error) {
	bookmarks, err := r.getRecordsBySQL(
		fmt.Sprintf("SELECT * FROM %s ORDER BY id DESC LIMIT ?", t),
		n,
	)
	if err != nil {
		return nil, err
	}
	if len(bookmarks) == 0 {
		return []bm.Bookmark{}, fmt.Errorf("no bookmarks found")
	}
	return bookmarks, nil
}
