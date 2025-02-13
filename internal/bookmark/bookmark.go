package bookmark

import (
	"errors"
	"fmt"
	"strconv"
)

var (
	ErrDuplicate    = errors.New("bookmark already exists")
	ErrInvalid      = errors.New("bookmark invalid")
	ErrInvalidID    = errors.New("invalid bookmark id")
	ErrInvalidInput = errors.New("invalid input")
	ErrNotFound     = errors.New("no bookmark found")
	ErrNotSelected  = errors.New("no bookmark selected")
	ErrTagsEmpty    = errors.New("TAGS cannot be empty")
	ErrURLEmpty     = errors.New("URL cannot be empty")
	ErrUnknownField = errors.New("bookmark field unknown")
)

// Bookmark represents a bookmark.
type Bookmark struct {
	CreatedAt string `db:"created_at" json:"created_at"`
	URL       string `db:"url"        json:"url"`
	Tags      string `db:"tags"       json:"tags"`
	Title     string `db:"title"      json:"title"`
	Desc      string `db:"desc"       json:"desc"`
	ID        int    `db:"id"         json:"id"`
}

// Field returns the value of a field.
func (b *Bookmark) Field(f string) (string, error) {
	var s string
	switch f {
	case "id", "i", "1":
		s = strconv.Itoa(b.ID)
	case "url", "u", "2":
		s = b.URL
	case "title", "t", "3":
		s = b.Title
	case "tags", "T", "4":
		s = b.Tags
	case "desc", "d", "5":
		s = b.Desc
	default:
		return "", fmt.Errorf("%w: '%s'", ErrUnknownField, f)
	}

	return s, nil
}

func (b *Bookmark) Buffer() []byte {
	return []byte(fmt.Sprintf(`# URL: (required)
%s
# Title: (leave an empty line for web fetch)
%s
# Tags: (comma separated)
%s
# Description: (leave an empty line for web fetch)
%s
# end ------------------------------------------------------------------------
`, b.URL, b.Title, b.Tags, b.Desc))
}

// New creates a new bookmark.
func New() *Bookmark {
	return &Bookmark{}
}
