// Package repo provides the model of the configuration for a database.
package repo

import (
	"path/filepath"

	"github.com/haaag/gm/internal/config"
	"github.com/haaag/gm/internal/sys/files"
)

// SQLiteConfig represents the configuration for a SQLite database.
type SQLiteConfig struct {
	Name         string       `json:"name"`
	Path         string       `json:"path"`
	TableMain    Table        `json:"table_main"`
	TableDeleted Table        `json:"table_deleted"`
	Backup       SQLiteBackup `json:"backup"`
	MaxBytesSize int64        `json:"max_bytes_size"`
}

type SQLiteBackup struct {
	Path    string   `json:"path"`
	Files   []string `json:"files"`
	Limit   int      `json:"limit"`
	Enabled bool     `json:"enabled"`
}

func (b *SQLiteBackup) SetLimit(n int) {
	b.Limit = n
	b.Enabled = n > 0
}

func newSQLiteBackup(p string) *SQLiteBackup {
	return &SQLiteBackup{
		Path:    filepath.Join(p, "backup"),
		Files:   []string{},
		Enabled: false,
		Limit:   0,
	}
}

func (c *SQLiteConfig) Fullpath() string {
	return filepath.Join(c.Path, c.Name)
}

func (c *SQLiteConfig) SetPath(p string) *SQLiteConfig {
	c.Path = p
	return c
}

func (c *SQLiteConfig) SetName(s string) *SQLiteConfig {
	c.Name = files.EnsureExt(s, ".sqlite")
	return c
}

func (c *SQLiteConfig) Exists() bool {
	return files.Exists(c.Fullpath())
}

// NewSQLiteCfg returns the default settings for the database.
func NewSQLiteCfg(p string) *SQLiteConfig {
	// WARN: do not like Table(*)
	return &SQLiteConfig{
		TableMain:    Table(config.DB.MainTable),
		TableDeleted: Table(config.DB.DeletedTable),
		MaxBytesSize: config.DB.MaxBytesSize,
		Path:         p,
		Backup:       *newSQLiteBackup(p),
	}
}
