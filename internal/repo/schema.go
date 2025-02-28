package repo

// tables.
const (
	tableMainName     = "bookmarks"
	tableTagsName     = "tags"
	tableRelationName = "bookmark_tags"
	tableTempName     = "temp_bookmarks"
)

// schemaMain is the schema for the main table.
var schemaMain = tableSchema{
	name:  tableMainName,
	sql:   tableMainSchema,
	index: tableMainIndex,
}

// schemaTags is the schema for the tags table.
var schemaTags = tableSchema{
	name:  tableTagsName,
	sql:   tableTagsSchema,
	index: tableTagsIndex,
}

// schemaRelation is the schema for the relation table.
var schemaRelation = tableSchema{
	name:    tableRelationName,
	sql:     tableRelationSchema,
	index:   tableRelationIndex,
	trigger: tableRelationTriggerCleanup,
}

// schemaTemp is used for reordering the IDs in the main table.
var schemaTemp = tableSchema{
	name:    tableTempName,
	sql:     tableTempSchema,
	trigger: tableRelationTriggerCleanup,
	index:   tableMainIndex,
}

// main table.
const (
	tableMainSchema = `
    PRAGMA foreign_keys = ON;

    CREATE TABLE IF NOT EXISTS bookmarks (
        id          INTEGER PRIMARY KEY AUTOINCREMENT,
        url         TEXT    NOT NULL UNIQUE,
        title       TEXT    DEFAULT "",
        desc        TEXT    DEFAULT "",
        created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        last_visit  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        visit_count INTEGER DEFAULT 0,
        favorite    BOOLEAN DEFAULT FALSE
    );`

	tableMainIndex = `
    CREATE UNIQUE INDEX IF NOT EXISTS idx_bookmarks_url
    ON bookmarks(url);`
)

// temp table.
const (
	tableTempSchema = `
    PRAGMA foreign_keys = ON;

    CREATE TABLE IF NOT EXISTS temp_bookmarks (
        id          INTEGER PRIMARY KEY AUTOINCREMENT,
        url         TEXT    NOT NULL UNIQUE,
        title       TEXT    DEFAULT "",
        desc        TEXT    DEFAULT "",
        created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        last_visit  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        visit_count INTEGER DEFAULT 0,
        favorite    BOOLEAN DEFAULT FALSE
    );`
)

// tags table.
const (
	tableTagsSchema = `
    CREATE TABLE IF NOT EXISTS tags (
        id          INTEGER PRIMARY KEY AUTOINCREMENT,
        name        TEXT    NOT NULL UNIQUE
    );`

	tableTagsIndex = `
    CREATE UNIQUE INDEX IF NOT EXISTS idx_tags_name
    ON tags(name);`
)

// relation table.
const (
	tableRelationSchema = `
    CREATE TABLE IF NOT EXISTS bookmark_tags (
        bookmark_url TEXT NOT NULL,
        tag_id      INTEGER NOT NULL,
        FOREIGN KEY (bookmark_url) REFERENCES bookmarks(url) ON DELETE CASCADE,
        FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE,
        PRIMARY KEY (bookmark_url, tag_id)
    );`

	tableRelationIndex = `
    CREATE INDEX IF NOT EXISTS idx_bookmark_tags
    ON bookmark_tags(bookmark_url, tag_id);`

	tableRelationTriggerCleanup = `
  CREATE TRIGGER IF NOT EXISTS cleanup_bookmark_and_tags
  AFTER DELETE ON bookmark_tags
  BEGIN
      -- Eliminate the bookmark if you no longer have associations in bookmark_tags.
      DELETE FROM bookmarks
      WHERE url = OLD.bookmark_url
        AND NOT EXISTS (
            SELECT 1 FROM bookmark_tags WHERE bookmark_url = OLD.bookmark_url
        );

      -- Clean the tags that are no longer associated with any bookmark.
      DELETE FROM tags
      WHERE id NOT IN (
          SELECT DISTINCT tag_id FROM bookmark_tags
      );
  END;`
)
