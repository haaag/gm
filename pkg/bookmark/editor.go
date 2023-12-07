/*
Copyright © 2023 haaag <git.haaag@gmail.com>
*/

package bookmark

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"gomarks/pkg/config"
	"gomarks/pkg/errs"
	"gomarks/pkg/scrape"
	"gomarks/pkg/util"
)

type editorInfo struct {
	Name string
	Args []string
}

type tempBookmark struct {
	url   string
	title string
	tags  string
	desc  string
}

// Fetches the title and/or description of the bookmark's URL, if they are not already set.
func (t *tempBookmark) fetchTitleAndDescription() {
	if t.title == scrape.TitleDefault || t.title == "" {
		title, err := scrape.GetTitle(t.url)
		if err != nil {
			log.Printf("Error on %s: %s\n", t.url, err)
		}
		t.title = title
	}

	if t.desc == scrape.DescDefault || t.desc == "" {
		description, err := scrape.GetDescription(t.url)
		if err != nil {
			log.Printf("Error on %s: %s\n", t.url, err)
		}
		t.desc = description
	}
}

/**
 * Parses the provided bookmark content into a temporary bookmark struct.
 *
 * @param content The bookmark content to parse.
 *
 * @return A pointer to a temporary bookmark struct containing the parsed bookmark information.
 */
func parseTempBookmark(content []string) *tempBookmark {
	url := extractBlock(content, "## url", "## title")
	title := extractBlock(content, "## title", "## tags")
	tags := extractBlock(content, "## tags", "## description")
	desc := extractBlock(content, "## description", "## end")

	return &tempBookmark{
		url:   url,
		title: title,
		tags:  tags,
		desc:  desc,
	}
}

/**
 * Edits the provided bookmark using the specified editor.
 *
 * @param b The bookmark to edit.
 *
 * @return The updated bookmark, or an error if an error occurred during editing.
 */
func Edit(b *Bookmark) (*Bookmark, error) {
	tf, err := createTempFile()
	if err != nil {
		return b, fmt.Errorf("%w", err)
	}

	data := b.buffer()
	tf, err = saveDataToTempFile(tf, data)
	if err != nil {
		return b, fmt.Errorf("%w", err)
	}

	defer func() {
		if err = tf.Close(); err != nil {
			log.Printf("Error closing temp file: %v", err)
		}
	}()

	defer func() {
		if err = cleanupTempFile(tf.Name()); err != nil {
			log.Printf("%v", err)
		}
	}()

	editor, err := getEditor()
	if err != nil {
		return b, fmt.Errorf("%w", err)
	}

	if err = editFile(editor, tf); err != nil {
		return b, fmt.Errorf("%w", err)
	}

	editedContent, err := readContentFile(tf)
	if err != nil {
		return b, fmt.Errorf("%w", err)
	}

	tempContent := strings.Split(string(editedContent), "\n")
	if err := validateContent(tempContent); err != nil {
		return b, err
	}

	if isSameContentBytes(data, editedContent) {
		return b, fmt.Errorf("%w", errs.ErrBookmarkUnchaged)
	}

	tb := parseTempBookmark(tempContent)
	tb.fetchTitleAndDescription()

	b.Update(tb.url, tb.title, tb.tags, tb.desc)
	return b, nil
}

// Checks if the buffer is unchanged
func isSameContentBytes(a, b []byte) bool {
	return bytes.Equal(a, b)
}

/**
 * Writes the provided data to a temporary file and returns the file handle.
 *
 * @param file The temporary file to write to.
 * @param data The data to write to the file.
 *
 * @return The file handle of the temporary file, or an error if the data could not be written to the file.
 */
func saveDataToTempFile(f *os.File, data []byte) (*os.File, error) {
	const filePermission = 0o600

	err := os.WriteFile(f.Name(), data, filePermission)
	if err != nil {
		return nil, fmt.Errorf("error writing to temp file: %w", err)
	}

	return f, nil
}

func createTempFile() (*os.File, error) {
	tempDir := "/tmp/"
	prefix := fmt.Sprintf("%s-", config.App.Name)

	tempFile, err := os.CreateTemp(tempDir, prefix)
	if err != nil {
		return nil, fmt.Errorf("error creating temp file: %w", err)
	}

	return tempFile, nil
}

/**
 * Validates the content of a bookmark file by ensuring that it contains both a URL and tags.
 *
 * @param content The content of the bookmark file.
 *
 * @return An error if the content is invalid.
 */
func validateContent(content []string) error {
	url := extractBlock(content, "## url:", "## title:")
	tags := extractBlock(content, "## tags:", "## description:")

	if isEmptyLine(url) {
		return errs.ErrURLEmpty
	}

	if isEmptyLine(tags) {
		return errs.ErrTagsEmpty
	}

	return nil
}

/**
 * Removes the specified temporary file.
 *
 * @param fileName The name of the temporary file to remove.
 *
 * @return An error if the temporary file could not be removed.
 */
func cleanupTempFile(fileName string) error {
	err := os.Remove(fileName)
	if err != nil {
		return fmt.Errorf("could not cleanup temp file: %w", err)
	}

	return nil
}

/**
 * Extracts a block of text from a string, delimited by the specified start and end markers.
 *
 * @param content The string to extract the block from.
 * @param startMarker The start marker of the block.
 * @param endMarker The end marker of the block.
 *
 * @return The extracted block of text, or an empty string if the block could not be found.
 */
func extractBlock(content []string, startMarker, endMarker string) string {
	startIndex := -1
	endIndex := -1
	isInBlock := false

	var cleanedBlock []string

	for i, line := range content {
		if strings.HasPrefix(line, startMarker) {
			startIndex = i
			isInBlock = true

			continue
		}

		if strings.HasPrefix(line, endMarker) && isInBlock {
			endIndex = i
			break // Found end marker line
		}

		if isInBlock {
			cleanedBlock = append(cleanedBlock, line)
		}
	}

	if startIndex == -1 || endIndex == -1 {
		return ""
	}

	return strings.Join(cleanedBlock, "\n")
}

/**
 * Opens the specified file for editing using the provided editor.
 *
 * @param editor The editor information struct containing the editor name and arguments.
 * @param file The file to open for editing.
 *
 * @return An error if an error occurs while opening or editing the file.
 */
func editFile(e *editorInfo, f *os.File) error {
	tempFileName := f.Name()

	// Construct the editor command with the temporary file path and editor arguments.
	cmd := exec.Command(e.Name, append(e.Args, tempFileName)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error running editor: %w", err)
	}

	return nil
}

/**
 * Retrieves the preferred editor to use for editing bookmarks.
 *
 * @return A pointer to an editorInfo struct containing the editor name and
 * arguments, or an error if no editor could be found.
 */
func getEditor() (*editorInfo, error) {
	if appEditor, exists := getAppEditor(); exists {
		return appEditor, nil
	}

	editor := strings.Fields(os.Getenv("EDITOR"))
	if len(editor) > 0 {
		log.Printf("$EDITOR set to %s", editor)
		return &editorInfo{Name: editor[0], Args: editor[1:]}, nil
	}

	log.Printf("$EDITOR not set.")

	for _, e := range config.Editors {
		if util.BinaryExists(e) {
			return &editorInfo{Name: e}, nil
		}
	}

	return nil, errs.ErrEditorNotFound
}

func getAppEditor() (*editorInfo, bool) {
	appEditor := strings.Fields(os.Getenv(config.App.Env.Editor))
	if len(appEditor) == 0 {
		return nil, false
	}

	if !util.BinaryExists(appEditor[0]) {
		log.Printf("'%s' executable file not found in $PATH", appEditor[0])
		return nil, false
	}

	log.Printf("$%s set to %s", config.App.Env.Editor, appEditor)
	return &editorInfo{Name: appEditor[0], Args: appEditor[1:]}, true
}

func readContentFile(file *os.File) ([]byte, error) {
	tempFileName := file.Name()
	content, err := os.ReadFile(tempFileName)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return content, nil
}

func isEmptyLine(line string) bool {
	return strings.TrimSpace(line) == ""
}