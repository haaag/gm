package repo

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/jmoiron/sqlx"
)

// GetOrCreateTag returns the tag ID.
func (r *SQLiteRepository) GetOrCreateTag(tx *sqlx.Tx, s string) (int64, error) {
	if s == "" {
		// no tag to process
		return 0, nil
	}
	// try to get the tag within the transaction
	tagID, err := getTag(tx, s)
	if err != nil {
		return 0, fmt.Errorf("getting tag: error retrieving tag: %w", err)
	}
	// if the tag doesn't exist, create it within the transaction
	if tagID == 0 {
		tagID, err = createTag(tx, s)
		if err != nil {
			return 0, fmt.Errorf("creating tag: error creating tag: %w", err)
		}
	}

	return tagID, nil
}

// associateTags associates tags to the given record.
func (r *SQLiteRepository) associateTags(tx *sqlx.Tx, b *Row) error {
	tags := strings.Split(b.Tags, ",")
	log.Printf("associating tags: %v with URL: %s\n", tags, b.URL)
	for _, tag := range tags {
		if tag == "" || tag == " " {
			continue
		}
		tagID, err := r.GetOrCreateTag(tx, tag)
		if err != nil {
			return err
		}
		log.Printf("processing tag: '%s' with id: %d\n", tag, tagID)
		_ = tx.MustExec(
			"INSERT OR IGNORE INTO bookmark_tags (bookmark_url, tag_id) VALUES (?, ?)",
			b.URL,
			tagID,
		)
	}

	return nil
}

// getTag returns the tag ID.
func getTag(tx *sqlx.Tx, tag string) (int64, error) {
	var tagID int64
	err := tx.QueryRowx("SELECT id FROM tags WHERE name = ?", tag).Scan(&tagID)
	if errors.Is(err, sql.ErrNoRows) {
		// tag not found
		return 0, nil
	} else if err != nil {
		return 0, fmt.Errorf("getTag: error querying tag: %w", err)
	}

	return tagID, nil
}

// createTag creates a new tag.
func createTag(tx *sqlx.Tx, tag string) (int64, error) {
	result := tx.MustExec("INSERT INTO tags (name) VALUES (?)", tag)
	tagID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("CreateTag: error getting last insert ID: %w", err)
	}

	return tagID, nil
}

// CounterTags returns a map with tag as key and count as value.
func CounterTags(r *SQLiteRepository) (map[string]int, error) {
	q := `
		SELECT
      t.name,
      COUNT(bt.tag_id) AS tag_count
    FROM
      tags t
      LEFT JOIN bookmark_tags bt ON t.id = bt.tag_id
    GROUP BY
      t.id,
      t.name;`

	var results []struct {
		Name  string `db:"name"`
		Count int    `db:"tag_count"`
	}
	if err := r.DB.Select(&results, q); err != nil {
		return nil, fmt.Errorf("error querying tags count: %w", err)
	}
	tagCounts := make(map[string]int, len(results))
	for _, row := range results {
		tagCounts[row.Name] = row.Count
	}

	return tagCounts, nil
}
