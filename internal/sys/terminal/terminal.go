package terminal

import (
	"errors"
	"fmt"
	"log"
	"os"

	"golang.org/x/term"

	"github.com/haaag/gm/internal/sys"
)

// https://no-color.org
const noColorEnv string = "NO_COLOR"

// Default terminal settings.
var (
	MaxWidth  int  = 120
	MinHeight int  = 15
	MinWidth  int  = 80
	Piped     bool = false
)

var (
	ErrNotTTY              = errors.New("not a terminal")
	ErrGetTermSize         = errors.New("getting terminal size")
	ErrTermWidthTooSmall   = errors.New("terminal width too small")
	ErrTermHeightTooSmall  = errors.New("terminal height too small")
	ErrUnsupportedPlatform = errors.New("unsupported platform")
)

// NoColor disables color if the NO_COLOR environment variable is set.
func NoColor(b *bool) {
	if c := sys.Env(noColorEnv, ""); c != "" {
		log.Println("NO_COLOR found.")
		*b = false
	}
}

// LoadMaxWidth updates `MaxWidth` to the current width if it is smaller than
// the existing `MaxWidth`.
func LoadMaxWidth() {
	w, _ := getWidth()
	if w == 0 {
		return
	}

	if w < MaxWidth {
		MaxWidth = w
		MinWidth = w
	}
}

// Clear clears the terminal.
func Clear() {
	fmt.Print("\033[H\033[2J")
}

// IsPiped returns true if the input is piped.
func IsPiped() bool {
	fileInfo, _ := os.Stdin.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) == 0
}

// getWidth returns the terminal's width.
func getWidth() (int, error) {
	fd := int(os.Stdout.Fd())
	if !term.IsTerminal(fd) {
		return 0, ErrNotTTY
	}
	w, _, err := term.GetSize(fd)
	if err != nil {
		return 0, fmt.Errorf("getting console width: %w", err)
	}

	return w, nil
}
