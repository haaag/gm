// Package format provides utilities for formatting and manipulating strings
package format

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/haaag/gm/internal/config"
	"github.com/haaag/gm/internal/format/color"
)

const (
	UnicodeBulletPoint      = "\u2022" // •
	UnicodeLightDiagCross   = "\u2571" // ╱
	UnicodeMiddleDot        = "\u00b7" // ·
	UnicodePathBigSegment   = "\u25B6" // ▶
	UnicodePathSmallSegment = "\u25B8" // ▸
	UnicodeRightDoubleAngle = "\u00BB" // »
	UnicodeSingleAngleMark  = "\u203A" // ›
)

// NBSP represents a non-breaking space character.
const NBSP = "\u00A0"

// PaddedLine formats a label and value into a left-aligned bullet point with fixed padding.
func PaddedLine(s, v any) string {
	const pad = 15
	return fmt.Sprintf("%-*s %v", pad, s, v)
}

// PaddingConditional returns the padding for the colorized output.
func PaddingConditional(minVal, maxVal int) int {
	if config.App.Color {
		return maxVal
	}

	return minVal
}

// Shorten shortens a string to a maximum length.
func Shorten(s string, maxLength int) string {
	if len(s) > maxLength {
		return s[:maxLength-3] + "..."
	}

	return s
}

// SplitAndAlign splits a string into multiple lines and aligns the
// words.
func SplitAndAlign(s string, lineLength, indentation int) string {
	separator := strings.Repeat(" ", indentation)
	var result strings.Builder
	var currentLine strings.Builder

	for _, word := range strings.Fields(s) {
		if currentLine.Len()+len(word)+1 > lineLength {
			result.WriteString(currentLine.String())
			result.WriteString("\n")
			currentLine.Reset()
			currentLine.WriteString(separator)
			currentLine.WriteString(word)
		} else {
			if currentLine.Len() != 0 {
				currentLine.WriteString(" ")
			}
			currentLine.WriteString(word)
		}
	}
	result.WriteString(currentLine.String())

	return result.String()
}

// ToJSON converts an interface to JSON.
func ToJSON(data any) []byte {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Fatalf("error converting to JSON: %s", err)
	}

	return jsonData
}

// SplitIntoChunks splits strings lines into chunks of a given length.
func SplitIntoChunks(s string, strLen int) []string {
	var lines []string
	var currentLine strings.Builder

	for _, word := range strings.Fields(s) {
		// If currentLine is empty, write the word directly.
		if currentLine.Len() == 0 {
			currentLine.WriteString(word)
			continue
		}
		// Check if adding the new word would exceed the line length
		if currentLine.Len()+len(word)+1 > strLen {
			// Add the current line to the result and start a new line
			lines = append(lines, currentLine.String())
			currentLine.Reset()
			currentLine.WriteString(word)
		} else {
			// Add the word to the current line
			if currentLine.Len() != 0 {
				currentLine.WriteString(" ")
			}
			currentLine.WriteString(word)
		}
	}

	// Add the last line if it's not empty
	if currentLine.Len() > 0 {
		lines = append(lines, currentLine.String())
	}

	return lines
}

// NormalizeSpace removes extra whitespace from a string, leaving only single
// spaces between words.
func NormalizeSpace(s string) string {
	s = strings.TrimSpace(s)
	return strings.Join(strings.Fields(s), " ")
}

// ByteSliceToLines returns the content of a []byte as a slice of strings,
// splitting on newline characters.
func ByteSliceToLines(data []byte) []string {
	return strings.Split(string(data), "\n")
}

// BufferAppend inserts a header string at the beginning of a byte buffer.
func BufferAppend(s string, buf *[]byte) {
	*buf = append([]byte(s), *buf...)
}

// BufferAppendEnd appends a string to the end of a byte buffer.
func BufferAppendEnd(s string, buf *[]byte) {
	*buf = append(*buf, []byte(s)...)
}

// BufferAppendVersion inserts a header string at the beginning of a byte buffer.
func BufferAppendVersion(name, version string, buf *[]byte) {
	BufferAppend(fmt.Sprintf("# %s: v%s\n", name, version), buf)
}

// IsEmptyLine checks if a line is empty.
func IsEmptyLine(line string) bool {
	return strings.TrimSpace(line) == ""
}

// Unique returns a slice of unique, non-empty strings from the input slice.
func Unique(t []string) []string {
	seen := make(map[string]bool)
	var tags []string

	for _, tag := range t {
		if tag == "" {
			continue
		}
		if !seen[tag] {
			seen[tag] = true
			tags = append(tags, tag)
		}
	}

	return tags
}

// CountLines counts the number of lines in a string.
func CountLines(s string) int {
	return len(strings.Split(s, "\n"))
}

// ColorDiff colorizes the diff output.
func ColorDiff(s string) string {
	var r []string
	for _, l := range strings.Split(s, "\n") {
		switch {
		case strings.HasPrefix(l, "+"):
			r = append(r, "  "+color.BrightGreen(l).String())
		case strings.HasPrefix(l, "-"):
			r = append(r, "  "+color.BrightRed(l).String())
		default:
			r = append(r, "  "+color.BrightGray(l).Italic().String())
		}
	}

	return strings.Join(r, "\n")
}
