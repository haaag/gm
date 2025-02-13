package cmd

import (
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/haaag/gm/internal/config"
	"github.com/haaag/gm/internal/format/color"
	"github.com/haaag/gm/internal/handler"
	"github.com/haaag/gm/internal/menu"
	"github.com/haaag/gm/internal/repo"
	"github.com/haaag/gm/internal/sys"
	"github.com/haaag/gm/internal/sys/terminal"
)

var (
	Copy bool
	Open bool
	Tags []string
	QR   bool

	Menu   bool
	Edit   bool
	Head   int
	Remove bool
	Tail   int

	Field     string
	JSON      bool
	Oneline   bool
	Multiline bool
	WithColor string

	Force   bool
	Status  bool
	Verbose bool
)

func initConfig() {
	// set logging level
	handler.LoggingLevel(&Verbose)
	// set force
	handler.Force(&Force)
	// enable color
	config.App.Color = WithColor != "never" && !terminal.IsPiped()
	menu.WithColor(&config.App.Color)
	// set terminal defaults
	terminal.NoColor(&config.App.Color)
	terminal.LoadMaxWidth()
	// enable color output
	color.Enable(&config.App.Color)
	// load data home path for the app.
	dataHomePath, err := loadDataPath()
	if err != nil {
		sys.ErrAndExit(err)
	}
	config.App.Path.Data = dataHomePath                            // Home
	config.App.Path.Backup = filepath.Join(dataHomePath, "backup") // Backups
	// set database settings/paths
	Cfg = repo.NewSQLiteCfg(dataHomePath)
	Cfg.SetName(DBName)
	Cfg.Backup.SetLimit(backupGetLimit())
}

// init sets the config for the root command.
func init() {
	cobra.OnInitialize(initConfig)
	// global
	pf := rootCmd.PersistentFlags()
	pf.StringVarP(&DBName, "name", "n", config.DB.Name, "database name")
	pf.StringVar(&WithColor, "color", "always", "output with pretty colors [always|never]")
	pf.BoolVarP(&Verbose, "verbose", "v", false, "verbose mode")
	pf.BoolVar(&Force, "force", false, "force action | don't ask confirmation")
	// local
	f := rootCmd.Flags()
	// prints
	f.BoolVarP(&JSON, "json", "j", false, "output in JSON format")
	f.BoolVarP(&Multiline, "multiline", "M", false, "output in formatted multiline (fzf)")
	f.BoolVarP(&Oneline, "oneline", "O", false, "output in formatted oneline (fzf)")
	f.StringVarP(&Field, "field", "f", "", "output by field [id|url|title|tags]")
	// actions
	f.BoolVarP(&Copy, "copy", "c", false, "copy bookmark to clipboard")
	f.BoolVarP(&Open, "open", "o", false, "open bookmark in default browser")
	f.BoolVarP(&QR, "qr", "q", false, "generate qr-code")
	f.BoolVarP(&Remove, "remove", "r", false, "remove a bookmarks by query or id")
	f.StringSliceVarP(&Tags, "tag", "t", nil, "list by tag")
	// experimental
	f.BoolVarP(&Menu, "menu", "m", false, "menu mode (fzf)")
	f.BoolVarP(&Edit, "edit", "e", false, "edit with preferred text editor")
	f.BoolVarP(&Status, "status", "s", false, "check bookmarks status")
	// modifiers
	f.IntVarP(&Head, "head", "H", 0, "the <int> first part of bookmarks")
	f.IntVarP(&Tail, "tail", "T", 0, "the <int> last part of bookmarks")
	// others
	rootCmd.CompletionOptions.HiddenDefaultCmd = true
	rootCmd.SilenceErrors = true
	rootCmd.DisableSuggestions = true
	rootCmd.SuggestionsMinimumDistance = 1
}

// isSubCmdCalled returns true if the subcommand was called.
func isSubCmdCalled(cmd *cobra.Command, cmdName string) bool {
	p := cmd.Parent()
	if p == nil {
		return false
	}
	for _, subCmd := range p.Commands() {
		if subCmd.CalledAs() == cmdName {
			return true
		}
	}

	return false
}
