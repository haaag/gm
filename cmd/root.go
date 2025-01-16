package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/haaag/gm/internal/bookmark"
	"github.com/haaag/gm/internal/config"
	"github.com/haaag/gm/internal/handler"
	"github.com/haaag/gm/internal/repo"
	"github.com/haaag/gm/internal/slice"
	"github.com/haaag/gm/internal/sys/terminal"
)

type (
	Bookmark = bookmark.Bookmark
	Slice    = slice.Slice[Bookmark]
)

var (
	// SQLiteCfg holds the configuration for the database and backups.
	Cfg *repo.SQLiteConfig

	// Main database name.
	DBName string
)

// handleData processes records based on user input and filtering criteria.
func handleData(r *repo.SQLiteRepository, args []string) (*Slice, error) {
	bs := slice.New[Bookmark]()
	if err := handler.Records(r, bs, args); err != nil {
		return nil, fmt.Errorf("getting records: %w", err)
	}

	switch {
	case len(Tags) > 0:
		if err := handler.ByTags(r, Tags, bs); err != nil {
			return nil, fmt.Errorf("records from tags: %w", err)
		}
	case Head > 0 || Tail > 0:
		if err := handler.ByHeadAndTail(bs, Head, Tail); err != nil {
			return nil, fmt.Errorf("records from head and tail: %w", err)
		}
	case Menu:
		if err := handler.Menu(bs, Multiline); err != nil {
			return nil, fmt.Errorf("records from menu: %w", err)
		}
	}

	return bs, nil
}

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:          config.App.Cmd,
	Short:        config.App.Info.Title,
	Long:         config.App.Info.Desc,
	Args:         cobra.MinimumNArgs(0),
	SilenceUsage: true,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return verifyDatabase(Cfg)
	},
	RunE: func(_ *cobra.Command, args []string) error {
		r, err := repo.New(Cfg)
		if err != nil {
			return fmt.Errorf("%w", err)
		}
		defer r.Close()

		terminal.ReadPipedInput(&args)
		bs, err := handleData(r, args)
		if err != nil {
			return err
		}

		// actions
		switch {
		case Status:
			return handler.CheckStatus(bs)
		case Remove:
			return handler.Remove(r, bs)
		case Edit:
			return handler.Edition(r, bs)
		case Copy:
			return handler.Copy(bs)
		case Open && !QR:
			return handler.Open(bs)
		}

		// display
		switch {
		case JSON:
			return handler.JSON(bs)
		case Oneline:
			return handler.Oneline(bs)
		case Field != "":
			return handler.ByField(bs, Field)
		case QR:
			return handler.QR(bs, Open)
		default:
			return handler.Print(bs)
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		handler.ErrAndExit(err)
	}
}
