// Package color provides utilities for formatting and coloring text
// output in the terminal
package color

import (
	"fmt"
	"reflect"
	"strings"
)

var enableColorOutput *bool

type ColorFn func(arg ...any) *Color

const (
	// normal colors.
	black   = "\x1b[30m"
	blue    = "\x1b[34m"
	cyan    = "\x1b[36m"
	gray    = "\x1b[90m"
	green   = "\x1b[32m"
	magenta = "\x1b[95m"
	orange  = "\x1b[33m"
	purple  = "\x1b[35m"
	red     = "\x1b[31m"
	white   = "\x1b[37m"
	yellow  = "\x1b[93m"

	// bright colors.
	brightBlack   = "\x1b[90m"
	brightBlue    = "\x1b[94m"
	brightCyan    = "\x1b[96m"
	brightGray    = "\x1b[37m"
	brightGreen   = "\x1b[92m"
	brightMagenta = "\x1b[95m"
	brightOrange  = "\x1b[33m"
	brightPurple  = "\x1b[35m"
	brightRed     = "\x1b[91m"
	brightWhite   = "\x1b[97m"
	brightYellow  = "\x1b[93m"

	// styles.
	bold          = "\x1b[1m"
	dim           = "\x1b[2m"
	inverse       = "\x1b[7m"
	italic        = "\x1b[3m"
	strikethrough = "\x1b[9m"
	underline     = "\x1b[4m"

	// reset colors.
	reset = "\x1b[0m"
)

// GetANSI returns the ANSI code from a Color function.
func GetANSI(f ColorFn) string {
	c := f()
	v := reflect.ValueOf(c).Elem().FieldByName("color")
	return v.String()
}

// EnableANSI allows to enable/disable color output.
func EnableANSI(b *bool) {
	enableColorOutput = b
}

// Toggle toggles color output.
func Toggle() {
	*enableColorOutput = !*enableColorOutput
}

// Color represents styled text with a specific color and formatting styles.
type Color struct {
	text   string
	color  string
	styles []string
}

func Text(s ...string) *Color {
	return &Color{text: strings.Join(s, " ")}
}

func (c *Color) applyStyle(styles ...string) *Color {
	c.styles = append(c.styles, styles...)
	return c
}

func (c *Color) Bold() *Color {
	return c.applyStyle(bold)
}

func (c *Color) Dim() *Color {
	return c.applyStyle(dim)
}

func (c *Color) Inverse() *Color {
	return c.applyStyle(inverse)
}

func (c *Color) Italic() *Color {
	return c.applyStyle(italic)
}

func (c *Color) Strikethrough() *Color {
	return c.applyStyle(strikethrough)
}

func (c *Color) Underline() *Color {
	return c.applyStyle(underline)
}

func (c *Color) String() string {
	if enableColorOutput == nil || !*enableColorOutput {
		return c.text
	}
	// apply styles
	styles := strings.Join(c.styles, "")

	return fmt.Sprintf("%s%s%s%s", styles, c.color, c.text, reset)
}

func Reset() string {
	return reset
}

func Black(arg ...any) *Color {
	return &Color{text: join(arg...), color: black}
}

func Blue(arg ...any) *Color {
	return &Color{text: join(arg...), color: blue}
}

func Cyan(arg ...any) *Color {
	return &Color{text: join(arg...), color: cyan}
}

func Gray(arg ...any) *Color {
	return &Color{text: join(arg...), color: gray}
}

func Green(arg ...any) *Color {
	return &Color{text: join(arg...), color: green}
}

func Magenta(arg ...any) *Color {
	return &Color{text: join(arg...), color: magenta}
}

func Orange(arg ...any) *Color {
	return &Color{text: join(arg...), color: orange}
}

func Purple(arg ...any) *Color {
	return &Color{text: join(arg...), color: purple}
}

func Red(arg ...any) *Color {
	return &Color{text: join(arg...), color: red}
}

func White(arg ...any) *Color {
	return &Color{text: join(arg...), color: white}
}

func Yellow(arg ...any) *Color {
	return &Color{text: join(arg...), color: yellow}
}

func BrightBlack(arg ...any) *Color {
	return &Color{text: join(arg...), color: brightBlack}
}

func BrightBlue(arg ...any) *Color {
	return &Color{text: join(arg...), color: brightBlue}
}

func BrightCyan(arg ...any) *Color {
	return &Color{text: join(arg...), color: brightCyan}
}

func BrightGray(arg ...any) *Color {
	return &Color{text: join(arg...), color: brightGray}
}

func BrightGreen(arg ...any) *Color {
	return &Color{text: join(arg...), color: brightGreen}
}

func BrightMagenta(arg ...any) *Color {
	return &Color{text: join(arg...), color: brightMagenta}
}

func BrightOrange(arg ...any) *Color {
	return &Color{text: join(arg...), color: brightOrange}
}

func BrightPurple(arg ...any) *Color {
	return &Color{text: join(arg...), color: brightPurple}
}

func BrightRed(arg ...any) *Color {
	return &Color{text: join(arg...), color: brightRed}
}

func BrightWhite(arg ...any) *Color {
	return &Color{text: join(arg...), color: brightWhite}
}

func BrightYellow(arg ...any) *Color {
	return &Color{text: join(arg...), color: brightYellow}
}

func join(text ...any) string {
	str := make([]string, 0, len(text))
	for _, t := range text {
		str = append(str, fmt.Sprint(t))
	}

	return strings.Join(str, " ")
}

// ApplyMany applies a color to a slice of strings returning new slice of
// strings.
func ApplyMany(s []string, c ColorFn) []string {
	for i := 0; i < len(s); i++ {
		s[i] = c(s[i]).String()
	}

	return s
}
