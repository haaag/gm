package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/haaag/gm/internal/bookmark"
	"github.com/haaag/gm/internal/bookmark/scraper"
	"github.com/haaag/gm/internal/config"
	"github.com/haaag/gm/internal/format"
	"github.com/haaag/gm/internal/format/color"
	"github.com/haaag/gm/internal/format/frame"
	"github.com/haaag/gm/internal/handler"
	"github.com/haaag/gm/internal/repo"
	"github.com/haaag/gm/internal/sys"
	"github.com/haaag/gm/internal/sys/files"
	"github.com/haaag/gm/internal/sys/spinner"
	"github.com/haaag/gm/internal/sys/terminal"
)

// addCmd represents the add command.
var addCmd = &cobra.Command{
	Use:   "add",
	Short: "add a new bookmark",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return handler.ValidateDB(cmd, Cfg)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := repo.New(Cfg)
		if err != nil {
			return fmt.Errorf("%w", err)
		}
		defer r.Close()

		return add(r, args)
	},
}

// add adds a new bookmark.
func add(r *repo.SQLiteRepository, args []string) error {
	// setup terminal and interrupt func handler (ctrl+c handler)
	t := terminal.New(terminal.WithInterruptFn(func(err error) {
		r.Close()
		sys.ErrAndExit(err)
	}))

	if terminal.IsPiped() && len(args) < 2 {
		return fmt.Errorf("%w: URL or TAGS cannot be empty", bookmark.ErrInvalid)
	}

	// header
	f := frame.New(frame.WithColorBorder(color.Gray))
	header := color.BrightYellow("Add Bookmark").String()
	q := color.Gray(" (ctrl+c to exit)").Italic().String()
	f.Header(header + q).Ln().Render()

	b := bookmark.New()
	if err := addParseNewBookmark(t, r, b, args); err != nil {
		return err
	}

	if !Force {
		if err := addHandleConfirmation(t, b); err != nil {
			if !errors.Is(err, bookmark.ErrBufferUnchanged) {
				return fmt.Errorf("%w", err)
			}
		}
	}

	// insert new bookmark
	if err := r.Insert(b); err != nil {
		return fmt.Errorf("%w", err)
	}

	if !Force {
		t.ClearLine(1)
	}
	success := color.BrightGreen("Successfully").Italic().String()
	f.Clean().Success(success + " bookmark created").Render()

	return nil
}

// addHandleConfirmation confirms if the user wants to save the bookmark.
func addHandleConfirmation(t *terminal.Term, b *Bookmark) error {
	fmt.Println()
	f := frame.New(frame.WithColorBorder(color.Gray), frame.WithNoNewLine())
	save := color.BrightGreen("save").String()
	s := f.Success(save).String() + " bookmark?"
	opt := t.Choose(s, []string{"yes", "no", "edit"}, "y")
	opt = strings.ToLower(opt)

	switch opt {
	case "n", "no":
		return fmt.Errorf("%w", handler.ErrActionAborted)
	case "e", "edit":
		if err := bookmarkEdition(b); err != nil {
			return fmt.Errorf("%w", err)
		}
	}

	return nil
}

// addHandleURL retrieves a URL from args or prompts the user for input.
func addHandleURL(t *terminal.Term, args *[]string) string {
	f := frame.New(frame.WithColorBorder(color.Gray), frame.WithNoNewLine())
	f.Header(color.BrightMagenta("URL\t:").String())

	// Checks if URL is provided
	if len(*args) > 0 {
		url := strings.TrimRight((*args)[0], "\n")
		f.Text(" " + color.Gray(url).String()).Ln().Render()
		*args = (*args)[1:]

		return url
	}

	// Checks clipboard
	c := addHandleClipboard(t)
	if c != "" {
		return c
	}

	f.Ln().Render()
	url := t.Input(f.Border.Mid)

	f.Clean().Mid(color.BrightMagenta("URL\t:").String()).
		Text(" " + color.Gray(url).String()).Ln()

	t.ClearLine(format.CountLines(f.String()))
	f.Render()

	return url
}

// addHandleTags retrieves the Tags from args or prompts the user for input.
func addHandleTags(t *terminal.Term, r *repo.SQLiteRepository, args *[]string) string {
	f := frame.New(frame.WithColorBorder(color.Gray), frame.WithNoNewLine())
	f.Header(color.BrightBlue("Tags\t:").String())

	// this checks if tags are provided and returns them
	if len(*args) > 0 {
		tags := strings.TrimRight((*args)[0], "\n")
		tags = strings.Join(strings.Fields(tags), ",")
		tags = bookmark.ParseTags(tags)

		f.Text(" " + color.Gray(tags).String()).Ln().Render()

		*args = (*args)[1:]

		return tags
	}

	f.Text(color.Gray(" (spaces|comma separated)").Italic().String()).Ln().Render()

	mTags, _ := repo.CounterTags(r)
	tags := t.ChooseTags(f.Border.Mid, mTags)
	tags = bookmark.ParseTags(tags)

	f.Clean().Mid(color.BrightBlue("Tags\t:").String()).
		Text(" " + color.Gray(tags).String()).Ln()

	t.ClearLine(format.CountLines(f.String()))
	f.Render()

	return tags
}

// addParseNewBookmark fetch metadata and parses the new bookmark.
func addParseNewBookmark(
	t *terminal.Term,
	r *repo.SQLiteRepository,
	b *Bookmark,
	args []string,
) error {
	// retrieve url
	url, err := addParseURL(t, r, &args)
	if err != nil {
		return err
	}

	// retrieve tags
	tags := addHandleTags(t, r, &args)
	// fetch title and description
	title, desc := addTitleAndDesc(url, true)

	b.URL = url
	b.Title = title
	b.Tags = bookmark.ParseTags(tags)
	b.Desc = strings.Join(format.Split(desc, terminal.MinWidth), "\n")

	return nil
}

// addTitleAndDesc fetch and display title and description.
func addTitleAndDesc(url string, verbose bool) (title, desc string) {
	sp := spinner.New(
		spinner.WithMesg(color.Yellow("scraping webpage...").String()),
		spinner.WithColor(color.BrightMagenta),
	)
	sp.Start()

	sc := scraper.New(url)
	_ = sc.Scrape()

	title = sc.Title()
	desc = sc.Desc()

	sp.Stop()

	if verbose {
		const indentation int = 10
		f := frame.New(frame.WithColorBorder(color.Gray), frame.WithNoNewLine())
		width := terminal.MinWidth - len(f.Border.Row)
		titleColor := color.Gray(format.SplitAndAlign(title, width, indentation)).String()
		descColor := color.Gray(format.SplitAndAlign(desc, width, indentation)).String()

		f.Mid(color.BrightCyan("Title\t: ").String()).Text(titleColor).Ln().
			Mid(color.BrightOrange("Desc\t: ").String()).Text(descColor).Ln().
			Render()
	}

	return title, desc
}

// addParseURL parse URL from args.
func addParseURL(t *terminal.Term, r *repo.SQLiteRepository, args *[]string) (string, error) {
	url := addHandleURL(t, args)
	if url == "" {
		return url, bookmark.ErrURLEmpty
	}

	// WARN: do we need this trim? why?
	url = strings.TrimRight(url, "/")

	if r.HasRecord(r.Cfg.Tables.Main, "url", url) {
		item, _ := r.ByURL(r.Cfg.Tables.Main, url)
		return "", fmt.Errorf("%w with id: %d", bookmark.ErrDuplicate, item.ID)
	}

	return url, nil
}

// addHandleClipboard checks if there a valid URL in the clipboard.
func addHandleClipboard(t *terminal.Term) string {
	c := sys.ReadClipboard()
	if !handler.URLValid(c) {
		return ""
	}

	f := frame.New(frame.WithColorBorder(color.Gray), frame.WithNoNewLine())
	f.Mid(color.BrightCyan("found valid URL in clipboard").Italic().String()).Ln()
	f.Render()
	lines := format.CountLines(f.String()) + 1

	bURL := f.Clean().
		Mid(color.BrightMagenta("URL\t:").String()).
		Text(" " + color.Gray(c).String()).String()

	fmt.Print(bURL)

	f.Clean().Ln().Row().Ln().Render().Clean()
	lines += format.CountLines(f.Mid("continue?").String())
	opt := t.Choose(f.String(), []string{"yes", "no"}, "y")
	switch opt {
	case "n", "no":
		t.ClearLine(lines)
		return ""
	default:
		t.ClearLine(lines)
		fmt.Println(bURL)
		return c
	}
}

// bookmarkEdition edits a bookmark with a text editor.
func bookmarkEdition(b *Bookmark) error {
	te, err := files.Editor(config.App.Env.Editor)
	if err != nil {
		return fmt.Errorf("%w", err)
	}

	if err := bookmark.Edit(te, bookmark.Buffer(b), b); err != nil {
		return fmt.Errorf("%w", err)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(addCmd)
}
