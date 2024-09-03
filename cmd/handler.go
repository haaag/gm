package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/haaag/gm/internal/presenter"
	"github.com/haaag/gm/pkg/bookmark"
	"github.com/haaag/gm/pkg/editor"
	"github.com/haaag/gm/pkg/format"
	"github.com/haaag/gm/pkg/format/color"
	"github.com/haaag/gm/pkg/qr"
	"github.com/haaag/gm/pkg/repo"
	"github.com/haaag/gm/pkg/terminal"
	"github.com/haaag/gm/pkg/util"
)

var (
	ErrActionAborted  = errors.New("action aborted")
	ErrURLNotProvided = errors.New("URL not provided")
	ErrUnknownField   = errors.New("field unknown")
)

// handleByField prints the selected field.
func handleByField(bs *Slice) error {
	if Field == "" {
		return nil
	}

	printer := func(b Bookmark) error {
		switch Field {
		case "id":
			fmt.Println(b.ID)
		case "url":
			fmt.Println(b.URL)
		case "title":
			fmt.Println(b.Title)
		case "tags":
			fmt.Println(b.Tags)
		case "desc":
			fmt.Println(b.Desc)
		default:
			return fmt.Errorf("%w: '%s'", ErrUnknownField, Field)
		}

		return nil
	}

	if err := bs.ForEachErr(printer); err != nil {
		return fmt.Errorf("%w", err)
	}
	Prettify = false

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
		var output string
		if Prettify {
			output = presenter.PrettyWithURLPath(&b, n) + "\n"
		}

		if Frame {
			output = presenter.WithFrame(&b, n)
		}

		if output != "" {
			fmt.Print(output)
			if i != lastIdx {
				fmt.Println()
			}
		}
	})

	return nil
}

// handleOneline formats the bookmarks in oneline.
func handleOneline(bs *Slice) error {
	if !Oneline {
		return nil
	}

	bs.ForEach(func(b Bookmark) {
		fmt.Print(presenter.Oneline(&b, terminal.MaxWidth))
	})

	Exit = true

	return nil
}

// handleJSONFormat formats the bookmarks in JSON.
func handleJSONFormat(bs *Slice) error {
	if !JSON {
		return nil
	}

	fmt.Println(string(format.ToJSON(bs.GetAll())))

	return nil
}

// handleHeadAndTail returns a slice of bookmarks with limited
// elements.
func handleHeadAndTail(bs *Slice) error {
	if Head == 0 && Tail == 0 {
		return nil
	}

	if Head < 0 || Tail < 0 {
		return fmt.Errorf("%w: head=%d tail=%d", format.ErrInvalidOption, Head, Tail)
	}

	bs.Head(Head)
	bs.Tail(Tail)

	return nil
}

// handleListAll retrieves records from the database based on either an ID or a
// query string.
func handleListAll(r *Repo, bs *Slice) error {
	if !List {
		return nil
	}

	if err := r.GetAll(r.Cfg.GetTableMain(), bs); err != nil {
		return fmt.Errorf("%w", err)
	}

	return nil
}

// handleByQuery executes a search query on the given repository based on
// provided arguments.
func handleByQuery(r *Repo, bs *Slice, args []string) error {
	if bs.Len() != 0 || len(args) == 0 {
		return nil
	}

	query := strings.Join(args, "%")
	if err := r.GetByQuery(r.Cfg.GetTableMain(), query, bs); err != nil {
		return fmt.Errorf("%w: '%s'", err, strings.Join(args, " "))
	}

	return nil
}

// handleByTags returns a slice of bookmarks based on the
// provided tags.
func handleByTags(r *Repo, bs *Slice) error {
	if Tags == nil {
		return nil
	}

	for _, tag := range Tags {
		if err := r.GetByTags(r.Cfg.GetTableMain(), tag, bs); err != nil {
			return fmt.Errorf("byTags :%w", err)
		}
	}

	if bs.Len() == 0 {
		return fmt.Errorf("%w by tag: '%s'", repo.ErrRecordNoMatch, strings.Join(Tags, ", "))
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
func handleEdition(r *Repo, bs *Slice) error {
	if !Edit {
		return nil
	}

	n := bs.Len()
	if n == 0 {
		return repo.ErrRecordQueryNotProvided
	}

	header := "# [%d/%d] | %d | %s\n\n"

	// edition edits the bookmark with a text editor.
	edition := func(i int, b Bookmark) error {
		buf := b.Buffer()
		shortTitle := format.ShortenString(b.Title, terminal.MinWidth-10)
		editor.Append(fmt.Sprintf(header, i+1, n, b.ID, shortTitle), &buf)
		editor.AppendVersion(App.Name, App.Version, &buf)
		bufCopy := make([]byte, len(buf))
		copy(bufCopy, buf)

		if err := editor.Edit(&buf); err != nil {
			return fmt.Errorf("%w", err)
		}

		if editor.IsSameContentBytes(&buf, &bufCopy) {
			return nil
		}

		content := editor.Content(&buf)
		if err := editor.Validate(&content); err != nil {
			return fmt.Errorf("%w", err)
		}

		editedB := bookmark.ParseContent(&content)
		editedB.ID = b.ID
		b = *editedB

		if _, err := r.Update(r.Cfg.GetTableMain(), &b); err != nil {
			return fmt.Errorf("handle edition: %w", err)
		}

		fmt.Printf("%s: id: [%d] %s\n", App.GetName(), b.ID, color.Blue("updated").Bold())

		return nil
	}

	if err := bs.ForEachErrIdx(edition); err != nil {
		return fmt.Errorf("%w", err)
	}

	return nil
}

// handleRemove prompts the user the records to remove.
func handleRemove(r *Repo, bs *Slice) error {
	if !Remove {
		return nil
	}

	if err := validateRemove(bs); err != nil {
		return err
	}

	prompt := color.BrightRed("remove").Bold().String()
	if err := confirmAction(bs, prompt, color.BrightRed); err != nil {
		return err
	}

	return removeRecords(r, bs)
}

// handleCheckStatus prints the status code of the bookmark
// URL.
func handleCheckStatus(bs *Slice) error {
	if !Status {
		return nil
	}

	n := bs.Len()
	if n == 0 {
		return repo.ErrRecordQueryNotProvided
	}

	status := color.BrightGreen("status").Bold().String()
	if n > 15 && !terminal.Confirm(fmt.Sprintf("checking %s of %d, continue?", status, n), "y") {
		return ErrActionAborted
	}

	if err := bookmark.CheckStatus(bs); err != nil {
		return fmt.Errorf("%w", err)
	}

	return nil
}

// handleCopyOpen performs an action on the bookmark.
func handleCopyOpen(bs *Slice) error {
	if Exit {
		return nil
	}

	b := bs.Get(0)
	if Copy {
		if err := util.CopyClipboard(b.URL); err != nil {
			return fmt.Errorf("%w", err)
		}
	}

	if Open {
		if err := util.OpenInBrowser(b.URL); err != nil {
			return fmt.Errorf("%w", err)
		}
	}

	return nil
}

// handleBookmarksFromArgs retrieves records from the database
// based on either an ID or a query string.
func handleIDsFromArgs(r *Repo, bs *Slice, args []string) error {
	ids, err := extractIDsFromStr(args)
	if len(ids) == 0 {
		return nil
	}

	if !errors.Is(err, bookmark.ErrInvalidRecordID) && err != nil {
		return fmt.Errorf("%w", err)
	}

	if err := r.GetByIDList(r.Cfg.GetTableMain(), ids, bs); err != nil {
		return fmt.Errorf("records from args: %w", err)
	}

	if bs.Len() == 0 {
		a := strings.TrimRight(strings.Join(args, " "), "\n")
		return fmt.Errorf("%w by id/s: %s", repo.ErrRecordNotFound, a)
	}

	return nil
}

// handleQR handles creation, rendering or opening of
// QR-Codes.
func handleQR(bs *Slice) error {
	if !QR {
		return nil
	}

	Exit = true
	b := bs.Get(0)

	qrcode := qr.New(b.GetURL())
	if err := qrcode.Generate(); err != nil {
		return fmt.Errorf("%w", err)
	}

	if Open {
		return openQR(qrcode, &b)
	}

	fmt.Println(b.GetTitle())
	qrcode.Render()
	fmt.Println(b.GetURL())

	return nil
}
