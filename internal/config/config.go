package config

import "os"

const (
	AppName string = "gomarks"
	Command string = "gm"
)

var Version string = "0.0.7"

type database struct {
	Name             string // Default name of the SQLite database.
	MainTable        string // Name of the main bookmarks table.
	DeletedTable     string // Name of the deleted bookmarks table.
	DateFormat       string // Database date format
	BackupDateFormat string // Database backup date format
	MaxBytesSize     int64  // Maximum size in bytes of the SQLite database before vacuum.
	BackupMaxBackups int    // Maximum number of backups allowed.
}

type app struct {
	Name    string      `json:"name"`
	Cmd     string      `json:"cmd"`
	Version string      `json:"version"`
	Info    information `json:"data"`
	Env     environment `json:"env"`
	Path    path        `json:"path"`
}

// path represents the application path.
type path struct {
	Backup string `json:"backup"`
	Config string `json:"home"`
	Data   string `json:"data"`
}

// information represents the application information.
type information struct {
	URL   string `json:"url"`
	Title string `json:"title"`
	Tags  string `json:"tags"`
	Desc  string `json:"desc"`
}

// environment represents the application environment.
type environment struct {
	Home      string `json:"home"`
	Editor    string `json:"editor"`
	BackupMax string `json:"max_backups"`
}

type files struct {
	DirPermissions  os.FileMode
	FilePermissions os.FileMode
}

var DB = database{
	Name:             "bookmarks.db",
	MainTable:        "bookmarks",
	DeletedTable:     "deleted_bookmarks",
	DateFormat:       "2006-01-02 15:04:05",
	BackupDateFormat: "2006-01-02_15-04",
	MaxBytesSize:     1000000,
	BackupMaxBackups: 3,
}

var Files = files{
	// DirPermissions the default permissions for new directories.
	DirPermissions: 0o755,
	// FilePermissions the default permissions for new files.
	FilePermissions: 0o644,
}

var App = app{
	Name:    AppName,
	Cmd:     Command,
	Version: Version,
	Info: information{
		URL:   "https://github.com/haaag/gomarks#readme",
		Title: "Gomarks: A bookmark manager",
		Tags:  "golang,awesome,bookmarks,cli",
		Desc:  "Simple yet powerful bookmark manager for your terminal",
	},
	Env: environment{
		Home:      "GOMARKS_HOME",
		Editor:    "GOMARKS_EDITOR",
		BackupMax: "GOMARKS_BACKUP_MAX",
	},
}
