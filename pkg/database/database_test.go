package database

import (
	"database/sql"
	"fmt"
	"log"
	"testing"

	"gomarks/pkg/errs"

	"gomarks/pkg/bookmark"
	c "gomarks/pkg/constants"
)

var tempTableName = "test_table"

func setupTestDB(t *testing.T) (*sql.DB, *SQLiteRepository) {
	t.Helper()
	// Set up an in-memory database for testing
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}

	// test_table
	sqlQuery := fmt.Sprintf(
		"CREATE TABLE %s (id INTEGER PRIMARY KEY, url TEXT, title TEXT, tags TEXT, desc TEXT, created_at TEXT)",
		tempTableName,
	)
	_, err = db.Exec(sqlQuery)

	if err != nil {
		t.Fatal(err)
	}

	r := newSQLiteRepository(db)

	return db, r
}

func teardownTestDB(db *sql.DB) {
	if err := db.Close(); err != nil {
		log.Printf("Error closing rows: %v", err)
	}
}

func getValidBookmark() bookmark.Bookmark {
	return bookmark.Bookmark{
		URL:   "https://www.example.com",
		Title: bookmark.NullString{NullString: sql.NullString{String: "Title", Valid: true}},
		Tags:  "test,testme,go",
		Desc: bookmark.NullString{
			NullString: sql.NullString{String: "Description", Valid: true},
		},
		CreatedAt: "2023-01-01 12:00:00",
	}
}

func TestDropTable(t *testing.T) {
	db, r := setupTestDB(t)
	defer teardownTestDB(db)

	err := r.dropTable(tempTableName)
	if err != nil {
		t.Errorf("Error dropping table: %v", err)
	}

	_, err = db.Exec(fmt.Sprintf("SELECT * FROM %s", c.DBMainTableName))
	if err == nil {
		t.Errorf("DBMainTable still exists after calling HandleDropDB")
	}

	_, err = db.Exec(fmt.Sprintf("SELECT * FROM %s", c.DBDeletedTableName))
	if err == nil {
		t.Errorf("DBDeletedTable still exists after calling HandleDropDB")
	}
}

func TestTableExists(t *testing.T) {
	db, r := setupTestDB(t)
	defer teardownTestDB(db)

	exists, err := r.tableExists(tempTableName)
	if err != nil {
		t.Fatal(err)
	}

	if !exists {
		t.Error("TableExists returned false for an existing table")
	}
}

func TestTableDoesNotExists(t *testing.T) {
	db, r := setupTestDB(t)
	defer teardownTestDB(db)

	exists, err := r.tableExists("non_existent_table")
	if err != nil {
		t.Fatal(err)
	}

	if exists {
		t.Error("TableExists returned true for a non-existent table")
	}
}

func TestInsertRecord(t *testing.T) {
	db, r := setupTestDB(t)
	defer teardownTestDB(db)

	// Insert a valid record
	b := &bookmark.Bookmark{
		URL:   "https://example.com",
		Title: bookmark.NullString{NullString: sql.NullString{String: "Title", Valid: true}},
		Tags:  "test",
		Desc: bookmark.NullString{
			NullString: sql.NullString{String: "Description", Valid: true},
		},
		CreatedAt: "2023-01-01 12:00:00",
	}

	inserted, err := r.InsertRecord(b, tempTableName)
	if err != nil {
		t.Fatal(err)
	}

	if inserted.ID == 0 {
		t.Error("InsertRecord did not return a valid ID")
	}

	// Insert a duplicate record
	duplicate := &bookmark.Bookmark{
		URL: "https://example.com",
	}

	_, err = r.InsertRecord(duplicate, tempTableName)
	if err == nil {
		t.Error("InsertRecord did not return an error for a duplicate record")
	}

	// Insert an invalid record
	invalidBookmark := &bookmark.Bookmark{}

	_, err = r.InsertRecord(invalidBookmark, tempTableName)
	if err == nil {
		t.Error("InsertRecord did not return an error for an invalid record")
	}
}

func TestDeleteRecord(t *testing.T) {
	db, r := setupTestDB(t)
	defer teardownTestDB(db)

	// Insert a valid record
	b := getValidBookmark()

	_, err := r.InsertRecord(&b, tempTableName)
	if err != nil {
		t.Fatal(err)
	}

	// Test the deletion of a valid record
	err = r.DeleteRecord(&b, tempTableName)
	if err != nil {
		t.Errorf("DeleteRecord returned an error: %v", err)
	}

	// Test the deletion of a non-existent record
	err = r.DeleteRecord(&b, tempTableName)
	if err == nil {
		t.Error("DeleteRecord did not return an error for a non-existent record")
	}
}

func TestIsRecordExists(t *testing.T) {
	db, r := setupTestDB(t)
	defer teardownTestDB(db)

	b := &bookmark.Bookmark{
		URL: "https://example.com",
	}

	query := fmt.Sprintf("INSERT INTO %s (url) VALUES (?)", tempTableName)

	_, err := db.Exec(query, b.URL)
	if err != nil {
		t.Fatal(err)
	}

	exists := r.RecordExists(b.URL, tempTableName)
	if !exists {
		t.Errorf("isRecordExists returned false for an existing record")
	}

	nonExistentBookmark := &bookmark.Bookmark{
		URL: "https://non_existent.com",
	}

	exists = r.RecordExists(nonExistentBookmark.URL, tempTableName)
	if exists {
		t.Errorf("isRecordExists returned true for a non-existent record")
	}
}

func TestUpdateRecordSuccess(t *testing.T) {
	db, r := setupTestDB(t)
	defer teardownTestDB(db)

	validB := getValidBookmark()

	q := fmt.Sprintf("INSERT INTO %s (url) VALUES (?)", tempTableName)

	result, err := db.Exec(q, validB.URL)
	if err != nil {
		t.Errorf("Error inserting bookmark: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Errorf("Error getting last insert ID: %v", err)
	}

	validB.ID = int(id)

	_, err = r.UpdateRecord(&validB, tempTableName)
	if err != nil {
		t.Error(err)
	}

	q = fmt.Sprintf("SELECT * FROM %s WHERE id = ?", tempTableName)
	row := db.QueryRow(q, validB.ID)

	var b bookmark.Bookmark

	err = row.Scan(&b.ID, &b.URL, &b.Title, &b.Tags, &b.Desc, &b.CreatedAt)
	if err != nil {
		t.Errorf("Error scanning row: %v", err)
	}

	if b.ID != validB.ID {
		t.Errorf("Error updating bookmark: %v", errs.ErrRecordUpdate)
	}
}

func TestUpdateRecordError(t *testing.T) {
	db, r := setupTestDB(t)
	defer teardownTestDB(db)

	_, err := r.UpdateRecord(&bookmark.Bookmark{}, tempTableName)
	if err == nil {
		t.Error("UpdateRecord did not return an error for an invalid record")
	}
}

func TestGetRecordByID(t *testing.T) {
	db, r := setupTestDB(t)
	defer teardownTestDB(db)

	b := getValidBookmark()

	inserted, err := r.InsertRecord(&b, tempTableName)
	if err != nil {
		t.Fatal(err)
	}

	record, err := r.GetRecordByID(inserted.ID, tempTableName)
	if err != nil {
		t.Errorf("Error getting bookmark by ID: %v", err)
	}

	if record.ID != inserted.ID || record.URL != inserted.URL {
		t.Errorf("Unexpected bookmark retrieved: got %+v, expected %+v", record, inserted)
	}
}

func TestGetRecordsByQuery(t *testing.T) {
	expectedRecords := 2
	db, r := setupTestDB(t)

	defer teardownTestDB(db)

	b := getValidBookmark()
	_, _ = r.InsertRecord(&b, tempTableName)
	b.URL = "https://www.example2.com"
	_, _ = r.InsertRecord(&b, tempTableName)
	b.URL = "https://www.another.com"

	records, err := r.GetRecordsByQuery("example", tempTableName)
	if err != nil {
		t.Errorf("Error getting bookmarks by query: %v", err)
	}

	if records.Len() != expectedRecords {
		t.Errorf("Unexpected number of bookmarks retrieved: got %d, expected %d", records.Len(), 2)
	}

	var count int
	err = db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", tempTableName)).Scan(&count)

	if err != nil {
		t.Fatal(err)
	}

	if count != records.Len() {
		t.Errorf("Expected %d records, got %d", records.Len(), count)
	}
}

func TestInsertRecordsBulk(t *testing.T) {
	db, r := setupTestDB(t)
	defer teardownTestDB(db)

	// Crear una lista de marcadores de posición de prueba
	bookmarks := bookmark.Slice{
		{
			URL: "url1",
			Title: bookmark.NullString{
				NullString: sql.NullString{String: "title1", Valid: true},
			},
			Tags: "tag1",
			Desc: bookmark.NullString{
				NullString: sql.NullString{String: "desc1", Valid: true},
			},
			CreatedAt: "2023-01-01 12:00:00",
		},
		{
			URL: "url2",
			Title: bookmark.NullString{
				NullString: sql.NullString{String: "title2", Valid: true},
			},
			Tags: "tag2",
			Desc: bookmark.NullString{
				NullString: sql.NullString{String: "desc2", Valid: true},
			},
			CreatedAt: "2023-01-01 12:00:00",
		},
		{
			URL: "url3",
			Title: bookmark.NullString{
				NullString: sql.NullString{String: "title2", Valid: true},
			},
			Tags: "tag3",
			Desc: bookmark.NullString{
				NullString: sql.NullString{String: "desc2", Valid: true},
			},
			CreatedAt: "2023-01-01 12:00:00",
		},
	}

	err := r.insertRecordsBulk(tempTableName, &bookmarks)
	if err != nil {
		t.Fatal(err)
	}

	var count int

	err = db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", tempTableName)).Scan(&count)
	if err != nil {
		t.Fatal(err)
	}

	if count != len(bookmarks) {
		t.Errorf("Expected %d records, got %d", len(bookmarks), count)
	}
}

func TestRenameTable(t *testing.T) {
	db, r := setupTestDB(t)
	defer teardownTestDB(db)

	err := r.renameTable(tempTableName, c.DBDeletedTableName)
	if err != nil {
		t.Errorf("Error renaming table: %v", err)
	}

	exists, err := r.tableExists(c.DBDeletedTableName)
	if err != nil {
		t.Errorf("Error checking if table exists: %v", err)
	}

	if !exists {
		t.Errorf("Table %s does not exist", c.DBDeletedTableName)
	}
}

func TestBookmarkIsValid(t *testing.T) {
	validBookmark := bookmark.Bookmark{
		Title: bookmark.NullString{NullString: sql.NullString{String: "Example", Valid: true}},
		URL:   "https://www.example.com",
		Tags:  "tag1,tag2",
	}

	if !validBookmark.IsValid() {
		t.Errorf("TestBookmarkIsValid: expected valid bookmark to be valid")
	}

	invalidBookmark := bookmark.Bookmark{
		Title: bookmark.NullString{NullString: sql.NullString{String: "", Valid: false}},
		URL:   "",
	}

	if invalidBookmark.IsValid() {
		t.Errorf("TestBookmarkIsValid: expected invalid bookmark to be invalid")
	}
}
