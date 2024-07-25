package cmd

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/atotto/clipboard"

	"github.com/haaag/gm/pkg/bookmark"
	"github.com/haaag/gm/pkg/editor"
	"github.com/haaag/gm/pkg/util"
)

var ErrCopyToClipboard = errors.New("copy to clipboard")

// extractIDsFromStr extracts IDs from a string.
func extractIDsFromStr(args []string) ([]int, error) {
	ids := make([]int, 0)
	if len(args) == 0 {
		return ids, nil
	}

	// FIX: what is this!?
	for _, arg := range strings.Fields(strings.Join(args, " ")) {
		id, err := strconv.Atoi(arg)
		if err != nil {
			if errors.Is(err, strconv.ErrSyntax) {
				continue
			}

			return nil, fmt.Errorf("%w", err)
		}
		ids = append(ids, id)
	}

	return ids, nil
}

// logErrAndExit logs the error and exits the program.
func logErrAndExit(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", App.Name, err)
	}
}

// setLoggingLevel sets the logging level based on the verbose flag.
func setLoggingLevel(verboseFlag *bool) {
	if *verboseFlag {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
		log.Println("verbose mode: on")

		return
	}

	silentLogger := log.New(io.Discard, "", 0)
	log.SetOutput(silentLogger.Writer())
}

// copyToClipboard copies a string to the clipboard.
func copyToClipboard(s string) error {
	err := clipboard.WriteAll(s)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrCopyToClipboard, err)
	}

	log.Print("text copied to clipboard:", s)

	return nil
}

// Open opens a URL in the default browser.
func openBrowser(url string) error {
	args := util.GetOSArgsCmd()
	args = append(args, url)
	if err := util.ExecuteCmd(args...); err != nil {
		return fmt.Errorf("%w: opening in browser", err)
	}

	return nil
}

// filterSlice select which item to remove from a slice using the
// text editor.
func filterSlice(bs *Slice) error {
	buf := bookmark.GetBufferSlice(bs)
	editor.AppendVersion(App.Name, App.Version, &buf)
	if err := editor.Edit(&buf); err != nil {
		return fmt.Errorf("on editing slice buffer: %w", err)
	}

	c := editor.Content(&buf)
	urls := editor.ExtractContentLine(&c)
	if len(urls) == 0 {
		return ErrActionAborted
	}

	bs.Filter(func(b Bookmark) bool {
		_, exists := urls[b.URL]
		return exists
	})

	return nil
}

// bookmarkEdition edits a bookmark with a text editor.
func bookmarkEdition(b *Bookmark) error {
	buf := b.Buffer()
	if err := editor.Edit(&buf); err != nil {
		if errors.Is(err, editor.ErrBufferUnchanged) {
			return nil
		}

		return fmt.Errorf("%w", err)
	}
	content := editor.Content(&buf)
	tempB := bookmark.ParseContent(&content)
	if err := editor.Validate(&content); err != nil {
		return fmt.Errorf("%w", err)
	}

	tempB.ID = b.ID
	*b = *tempB

	return nil
}
