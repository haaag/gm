package errs

import "errors"

var (
	// bookmark
	ErrActionCancelled     = errors.New("action cancelled")
	ErrBookmarkDuplicate   = errors.New("bookmark already exists")
	ErrBookmarkEdition     = errors.New("")
	ErrBookmarkInvalid     = errors.New("bookmark invalid")
	ErrBookmarkNotFound    = errors.New("no bookmark found")
	ErrBookmarkNotSelected = errors.New("no bookmark selected")
	ErrBookmarkUnchaged    = errors.New("buffer unchanged")
	ErrEditorNotFound      = errors.New("editor not found")
	ErrItemNotFound        = errors.New("item not found")
	ErrOptionInvalid       = errors.New("invalid option")
	ErrTagsEmpty           = errors.New("tags cannot be empty")
	ErrURLEmpty            = errors.New("URL cannot be empty")

	// database
	ErrRecordNotExists = errors.New("row not exists")
	ErrRecordDelete    = errors.New("error delete record")
	ErrRecordDuplicate = errors.New("record already exists")
	ErrRecordInsert    = errors.New("error inserting record")
	ErrRecordScan      = errors.New("error scan record")
	ErrRecordUpdate    = errors.New("error update failed")
	ErrSQLQuery        = errors.New("error executing query")
)