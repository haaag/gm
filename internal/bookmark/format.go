package bookmark

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/haaag/gm/internal/format"
	"github.com/haaag/gm/internal/format/color"
	"github.com/haaag/gm/internal/format/frame"
	"github.com/haaag/gm/internal/sys/terminal"
)

// Oneline formats a bookmark in a single line with max width.
func Oneline(b *Bookmark) string {
	var sb strings.Builder
	const (
		idWithColor    = 16
		minTagsLen     = 34
		defaultTagsLen = 24
	)
	width := terminal.MaxWidth
	idLen := format.PaddingConditional(5, idWithColor)
	tagsLen := format.PaddingConditional(minTagsLen, defaultTagsLen)
	// calculate maximum length for url and tags based on total width
	urlLen := width - idLen - tagsLen
	// define template with formatted placeholders
	template := "%-*s %s %-*s %-*s\n"

	coloredID := color.BrightYellow(b.ID).Bold().String()
	shortURL := format.Shorten(b.URL, urlLen)
	colorURL := color.Gray(shortURL).String()
	urlLen += len(colorURL) - len(shortURL)
	tagsColor := color.BrightCyan(b.Tags).Italic().String()
	result := fmt.Sprintf(
		template,
		idLen,
		coloredID,
		format.MidBulletPoint,
		urlLen,
		colorURL,
		tagsLen,
		tagsColor,
	)
	sb.WriteString(result)

	return sb.String()
}

// Multiline formats a bookmark for fzf with max width.
func Multiline(b *Bookmark) string {
	width := terminal.MaxWidth
	var sb strings.Builder
	sb.WriteString(color.BrightYellow(b.ID).Bold().String())
	sb.WriteString(" " + format.MidBulletPoint + " ") // sep
	sb.WriteString(format.Shorten(PrettifyURL(b.URL, color.BrightMagenta), width) + "\n")
	sb.WriteString(color.Cyan(format.Shorten(b.Title, width)).String() + "\n")
	sb.WriteString(color.BrightGray(PrettifyTags(b.Tags)).Italic().String())

	return sb.String()
}

func FrameFormatted(b *Bookmark, c color.ColorFn) string {
	f := frame.New(frame.WithColorBorder(c))
	width := terminal.MinWidth
	width -= len(f.Border.Row)
	// split and add intendation
	descSplit := format.Split(b.Desc, width)
	titleSplit := format.Split(b.Title, width)
	// add color and style
	id := color.BrightYellow(b.ID).Bold().String()
	urlColor := format.Shorten(PrettifyURL(b.URL, color.BrightMagenta), width)
	title := color.ApplyMany(titleSplit, color.Cyan)
	desc := color.ApplyMany(descSplit, color.Gray)
	tags := color.Gray(PrettifyTags(b.Tags)).Italic().String()

	return f.Header(fmt.Sprintf("%s %s", id, urlColor)).
		Mid(title...).
		Mid(desc...).
		Footer(tags).
		String()
}

// FmtWithFrame formats and displays a bookmark with styling and frame layout.
func FmtWithFrame(f *frame.Frame, b *Bookmark, c color.ColorFn) {
	width := terminal.MinWidth - len(f.Border.Row)
	titleSplit := format.Split(b.Title, width)
	idStr := color.BrightWhite(b.ID).Bold().String()
	urlColor := format.Shorten(PrettifyURL(b.URL, c), width)
	title := color.ApplyMany(titleSplit, color.Cyan)
	tags := color.Gray(PrettifyTags(b.Tags)).Italic().String()

	f.Mid(fmt.Sprintf("%s %s %s", idStr, format.MidBulletPoint, urlColor)).Ln()
	f.Mid(title...).Ln()
	f.Mid(tags).Ln().Ln().Render()
}

// Frame formats a bookmark in a frame with min width.
func Frame(b *Bookmark) string {
	width := terminal.MinWidth
	f := frame.New(frame.WithColorBorder(color.Gray))
	// indentation
	width -= len(f.Border.Row)
	// split and add intendation
	descSplit := format.Split(b.Desc, width)
	titleSplit := format.Split(b.Title, width)
	// add color and style
	id := color.BrightYellow(b.ID).Bold().String()
	urlColor := format.Shorten(PrettifyURL(b.URL, color.BrightMagenta), width)
	title := color.ApplyMany(titleSplit, color.Cyan)
	desc := color.ApplyMany(descSplit, color.Gray)
	tags := color.BrightGray(PrettifyTags(b.Tags)).Italic().String()

	return f.Header(fmt.Sprintf("%s %s", id, urlColor)).
		Mid(title...).
		Mid(desc...).
		Footer(tags).
		String()
}

// PrettifyTags returns a prettified tags.
func PrettifyTags(s string) string {
	t := strings.ReplaceAll(s, ",", format.MidBulletPoint)
	return strings.TrimRight(t, format.MidBulletPoint)
}

// PrettifyURL returns a prettified URL.
func PrettifyURL(s string, c color.ColorFn) string {
	u, err := url.Parse(s)
	if err != nil {
		return ""
	}
	// default color
	if c == nil {
		c = color.Default
	}
	if u.Host == "" || u.Path == "" {
		return c(s).Bold().String()
	}
	host := c(u.Host).Bold().String()
	pathSegments := strings.FieldsFunc(
		strings.TrimLeft(u.Path, "/"),
		func(r rune) bool { return r == '/' },
	)

	if len(pathSegments) == 0 {
		return host
	}

	pathSeg := color.Text(
		format.SingleAngleMark,
		strings.Join(pathSegments, fmt.Sprintf(" %s ", format.SingleAngleMark)),
	).Italic()

	return fmt.Sprintf("%s %s", host, pathSeg)
}

// FzfFormatter returns a function to format a bookmark for the FZF menu.
func FzfFormatter(m bool) func(b *Bookmark) string {
	if m {
		return Multiline
	}

	return Oneline
}
