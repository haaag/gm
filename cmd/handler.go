package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/haaag/gm/internal/bookmark"
	"github.com/haaag/gm/internal/bookmark/qr"
	"github.com/haaag/gm/internal/config"
	"github.com/haaag/gm/internal/format"
	"github.com/haaag/gm/internal/format/color"
	"github.com/haaag/gm/internal/format/frame"
	"github.com/haaag/gm/internal/menu"
	"github.com/haaag/gm/internal/repo"
	"github.com/haaag/gm/internal/slice"
	"github.com/haaag/gm/internal/sys/files"
	"github.com/haaag/gm/internal/sys/terminal"
)

var (
	ErrActionAborted = errors.New("action aborted")
	ErrInvalidOption = errors.New("invalid option")
)

// handleDataSlice retrieves records from the database based on either
// an ID or a query string.
func handleDataSlice(r *repo.SQLiteRepository, args []string) (*Slice, error) {
	bs := slice.New[Bookmark]()
	if err := handleRecords(r, bs, args); err != nil {
		return nil, err
	}

	return bs, nil
}

// handleByField prints the selected field.
func handleByField(bs *Slice) error {
	if Field == "" {
		return nil
	}

	Exit = true

	printer := func(b Bookmark) error {
		f, err := b.Field(Field)
		if err != nil {
			return fmt.Errorf("%w", err)
		}

		fmt.Println(f)

		return nil
	}

	if err := bs.ForEachErr(printer); err != nil {
		return fmt.Errorf("%w", err)
	}

	return nil
}

// handlePrintOut prints the bookmarks in different formats.
func handlePrintOut(bs *Slice) error {
	if Exit {
		return nil
	}

	n := terminal.MinWidth
	lastIdx := bs.Len() - 1

	bs.ForEachIdx(func(i int, b Bookmark) {
		fmt.Print(bookmark.Frame(&b, n))
		if i != lastIdx {
			fmt.Println()
		}
	})

	return nil
}

// handleOneline formats the bookmarks in oneline.
func handleOneline(bs *Slice) error {
	if !Oneline {
		return nil
	}

	Exit = true

	bs.ForEach(func(b Bookmark) {
		fmt.Print(bookmark.Oneline(&b, terminal.MaxWidth))
	})

	return nil
}

// handleJSONFormat formats the bookmarks in JSON.
func handleJSONFormat(bs *Slice) error {
	if !JSON {
		return nil
	}

	Exit = true

	if bs.Empty() {
		fmt.Println(string(format.ToJSON(config.App)))
		return nil
	}

	fmt.Println(string(format.ToJSON(bs.Items())))

	return nil
}

// handleHeadAndTail returns a slice of bookmarks with limited elements.
func handleHeadAndTail(bs *Slice) error {
	if Head == 0 && Tail == 0 {
		return nil
	}

	if Head < 0 || Tail < 0 {
		return fmt.Errorf("%w: head=%d tail=%d", ErrInvalidOption, Head, Tail)
	}

	bs.Head(Head)
	bs.Tail(Tail)

	return nil
}

// handleByQuery executes a search query on the given repository based on
// provided arguments.
func handleByQuery(r *repo.SQLiteRepository, bs *Slice, args []string) error {
	if bs.Len() != 0 || len(args) == 0 {
		return nil
	}

	query := strings.Join(args, "%")
	if err := r.ByQuery(r.Cfg.TableMain, query, bs); err != nil {
		return fmt.Errorf("%w: '%s'", err, strings.Join(args, " "))
	}

	return nil
}

// handleByTags returns a slice of bookmarks based on the provided tags.
func handleByTags(r *repo.SQLiteRepository, bs *Slice) error {
	if Tags == nil {
		return nil
	}

	// if the slice contains bookmarks, filter by tag.
	if !bs.Empty() {
		for _, tag := range Tags {
			bs.Filter(func(b Bookmark) bool {
				return strings.Contains(b.Tags, tag)
			})
		}

		return nil
	}

	for _, tag := range Tags {
		if err := r.ByTag(r.Cfg.TableMain, tag, bs); err != nil {
			return fmt.Errorf("byTags :%w", err)
		}
	}

	if bs.Empty() {
		t := strings.Join(Tags, ", ")
		return fmt.Errorf("%w by tag: '%s'", repo.ErrRecordNoMatch, t)
	}

	bs.Filter(func(b Bookmark) bool {
		for _, tag := range Tags {
			if !strings.Contains(b.Tags, tag) {
				return false
			}
		}

		return true
	})

	return nil
}

// handleEdition renders the edition interface.
func handleEdition(r *repo.SQLiteRepository, bs *Slice) error {
	if !Edit {
		return nil
	}

	Exit = true

	n := bs.Len()
	if n == 0 {
		return repo.ErrRecordQueryNotProvided
	}

	const maxItems = 10
	e := color.BrightOrange("editing").Bold()
	q := fmt.Sprintf("%s %d bookmarks, continue?", e, n)
	if err := confirmUserLimit(n, maxItems, q); err != nil {
		return err
	}

	header := "# [%d/%d] | %d | %s\n\n"
	te, err := files.Editor(config.App.Env.Editor)
	if err != nil {
		return fmt.Errorf("%w", err)
	}

	// edition edits the bookmark with a text editor.
	edition := func(i int, b Bookmark) error {
		// prepare header and buffer
		buf := bookmark.Buffer(&b)
		tShort := format.Shorten(b.Title, terminal.MinWidth-10)
		format.BufferAppend(fmt.Sprintf(header, i+1, n, b.ID, tShort), &buf)
		format.BufferApendVersion(config.App.Name, config.App.Version, &buf)
		bufCopy := make([]byte, len(buf))
		copy(bufCopy, buf)

		if err := bookmark.Edit(te, buf, &b); err != nil {
			if errors.Is(err, bookmark.ErrBufferUnchanged) {
				return nil
			}

			return fmt.Errorf("%w", err)
		}

		if _, err := r.Update(r.Cfg.TableMain, &b); err != nil {
			return fmt.Errorf("handle edition: %w", err)
		}

		fmt.Printf("%s: [%d] %s\n", config.App.Name, b.ID, color.Blue("updated").Bold())

		return nil
	}

	if err := bs.ForEachErrIdx(edition); err != nil {
		return fmt.Errorf("%w", err)
	}

	return nil
}

// handleRemove prompts the user the records to remove.
func handleRemove(r *repo.SQLiteRepository, bs *Slice) error {
	if !Remove {
		return nil
	}

	Exit = true

	if err := validateRemove(bs); err != nil {
		return err
	}

	c := color.BrightRed
	f := frame.New(frame.WithColorBorder(c), frame.WithNoNewLine())
	header := c("Removing Bookmarks").String()
	f.Header(header).Ln().Ln().Render().Clean()

	prompt := color.BrightRed("remove").Bold().String()
	if err := confirmAction(bs, prompt, c); err != nil {
		return err
	}

	return removeRecords(r, bs)
}

// handleCheckStatus prints the status code of the bookmark URL.
func handleCheckStatus(bs *Slice) error {
	if !Status {
		return nil
	}

	Exit = true

	n := bs.Len()
	if n == 0 {
		return repo.ErrRecordQueryNotProvided
	}

	const maxGoroutines = 15
	status := color.BrightGreen("status").Bold()
	q := fmt.Sprintf("checking %s of %d, continue?", status, n)
	if err := confirmUserLimit(n, maxGoroutines, q); err != nil {
		return ErrActionAborted
	}

	f := frame.New(frame.WithColorBorder(color.BrightBlue))
	f.Header(fmt.Sprintf("checking %s of %d bookmarks", status, n))
	f.Render()
	if err := bookmark.Status(bs); err != nil {
		return fmt.Errorf("%w", err)
	}

	return nil
}

// handleCopyOpen calls the copyBookmarks and openBookmarks functions, and handles the overall logic.
func handleCopyOpen(bs *Slice) error {
	if !Copy && !Open {
		return nil
	}

	if err := copyBookmarks(bs); err != nil {
		return err
	}

	if Exit {
		return nil
	}

	if err := openBookmarks(bs); err != nil {
		return err
	}

	Exit = Copy || Open

	return nil
}

// handleBookmarksFromArgs retrieves records from the database based on either
// an ID or a query string.
func handleIDsFromArgs(r *repo.SQLiteRepository, bs *Slice, args []string) error {
	ids, err := extractIDsFromStr(args)
	if len(ids) == 0 {
		return nil
	}

	if !errors.Is(err, bookmark.ErrInvalidID) && err != nil {
		return fmt.Errorf("%w", err)
	}

	if err := r.ByIDList(r.Cfg.TableMain, ids, bs); err != nil {
		return fmt.Errorf("records from args: %w", err)
	}

	if bs.Empty() {
		a := strings.TrimRight(strings.Join(args, " "), "\n")
		return fmt.Errorf("%w by id/s: %s", repo.ErrRecordNotFound, a)
	}

	return nil
}

// handleQR handles creation, rendering or opening of QR-Codes.
func handleQR(bs *Slice) error {
	if !QR {
		return nil
	}

	Exit = true

	qrFn := func(b Bookmark) error {
		qrcode := qr.New(b.URL)
		if err := qrcode.Generate(); err != nil {
			return fmt.Errorf("%w", err)
		}

		if Open {
			return openQR(qrcode, &b)
		}

		var sb strings.Builder
		sb.WriteString(b.Title + "\n")
		sb.WriteString(b.URL + "\n")
		sb.WriteString(qrcode.String())
		t := sb.String()
		fmt.Print(t)

		lines := len(strings.Split(t, "\n"))
		terminal.WaitForEnter()
		terminal.ClearLine(lines)

		return nil
	}

	if err := bs.ForEachErr(qrFn); err != nil {
		return fmt.Errorf("%w", err)
	}

	return nil
}

// handleMenu allows the user to select multiple records.
func handleMenu(bs *Slice) error {
	if !Menu {
		return nil
	}
	if bs.Empty() {
		return repo.ErrRecordNoMatch
	}

	if err := menu.LoadConfig(); err != nil {
		return fmt.Errorf("%w", err)
	}

	menu.WithColor(&config.App.Color)

	// menu opts
	opts := handleMenuOpts()

	var formatter func(*Bookmark, int) string
	if Multiline {
		opts = append(opts, menu.WithMultilineView())
		formatter = bookmark.Multiline
	} else {
		formatter = bookmark.Oneline
	}

	m := menu.New[Bookmark](opts...)
	var result []Bookmark
	result, err := m.Select(bs.Items(), func(b Bookmark) string {
		return formatter(&b, terminal.MaxWidth)
	})
	if err != nil {
		if errors.Is(err, menu.ErrActionAborted) {
			Exit = true
			return ErrActionAborted
		}

		return fmt.Errorf("%w", err)
	}

	if len(result) == 0 {
		return nil
	}

	bs.Set(&result)

	return nil
}
