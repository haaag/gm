package files

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/haaag/gm/internal/sys"
)

var (
	ErrCommandNotFound    = errors.New("command not found")
	ErrTextEditorNotFound = errors.New("text editor not found")
)

const (
	dirPerm  = 0o755 // Permissions for new directories.
	filePerm = 0o644 // Permissions for new files.
)

// Fallback text editors if $EDITOR || $GOMARKS_EDITOR var is not set.
var textEditors = []string{"vim", "nvim", "nano", "emacs"}

type TextEditor struct {
	name string
	cmd  string
	args []string
}

// EditContentBytes edits a byte slice with a text editor.
func (te *TextEditor) EditContentBytes(content []byte) ([]byte, error) {
	if te.cmd == "" {
		return nil, ErrCommandNotFound
	}
	f, err := createTEmpFileWithData(content)
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}
	defer closeAndClean(f)

	log.Printf("editing file: '%s' with text editor: '%s'", f.Name(), te.name)
	log.Printf("executing args: cmd='%s' args='%v'", te.cmd, te.args)
	if err := sys.RunCmd(te.cmd, append(te.args, f.Name())...); err != nil {
		return nil, fmt.Errorf("error running editor: %w", err)
	}
	data, err := os.ReadFile(f.Name())
	if err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return data, nil
}

// EditFile edits a file with a text editor.
func (te *TextEditor) EditFile(p string) error {
	if te.cmd == "" {
		return ErrCommandNotFound
	}

	if !Exists(p) {
		return fmt.Errorf("%w: '%s'", ErrFileNotFound, p)
	}

	if err := sys.RunCmd(te.cmd, append(te.args, p)...); err != nil {
		return fmt.Errorf("error running editor: %w", err)
	}

	return nil
}

// GetEditor retrieves the preferred editor to use for editing
//
// If env variable `GOMARKS_EDITOR` is not set, uses the `EDITOR`.
// If env variable `EDITOR` is not set, uses the first available
// `TextEditors`
//
// # fallbackEditors: `"vim", "nvim", "nano", "emacs"`.
func GetEditor(s string) (*TextEditor, error) {
	envs := []string{s, "EDITOR"}
	// find $EDITOR and $GOMARKS_EDITOR
	for _, e := range envs {
		if editor, found := getEditorFromEnv(e); found {
			if editor.cmd == "" {
				return nil, fmt.Errorf("%w: '%s'", ErrTextEditorNotFound, editor.name)
			}

			return editor, nil
		}
	}

	log.Printf(
		"$EDITOR and $GOMARKS_EDITOR not set, checking fallback text editor: %s",
		textEditors,
	)

	// find fallback
	if editor, found := getFallbackEditor(textEditors); found {
		return editor, nil
	}

	return nil, ErrTextEditorNotFound
}

// getEditorFromEnv finds an editor in the environment.
func getEditorFromEnv(e string) (*TextEditor, bool) {
	s := strings.Fields(sys.Env(e, ""))
	if len(s) != 0 {
		editor := newTextEditor(sys.BinPath(s[0]), s[0], s[1:])
		log.Printf("$EDITOR set: '%v'", editor)
		return editor, true
	}

	return nil, false
}

// getFallbackEditor finds a fallback editor.
func getFallbackEditor(editors []string) (*TextEditor, bool) {
	// FIX: use `exec.LookPath`
	// This will replace `sys.BinExists` and `sys.BinPath`
	for _, e := range editors {
		if sys.BinExists(e) {
			editor := newTextEditor(sys.BinPath(e), e, []string{})
			log.Printf("found fallback text editor: '%v'", editor)
			return editor, true
		}
	}

	return nil, false
}

// saveBytestToFile Writes the provided data to a temporary file.
func saveBytestToFile(f *os.File, d []byte) error {
	err := os.WriteFile(f.Name(), d, filePerm)
	if err != nil {
		return fmt.Errorf("error writing to temp file: %w", err)
	}

	return nil
}

// createTEmpFileWithData creates a temporary file and writes the provided data
// to it.
func createTEmpFileWithData(d []byte) (*os.File, error) {
	const tempExt = "bookmark"
	tf, err := CreateTemp("edit", tempExt)
	if err != nil {
		return nil, fmt.Errorf("error creating temp file: %w", err)
	}

	if err := saveBytestToFile(tf, d); err != nil {
		return nil, err
	}

	return tf, nil
}

// readContent reads the content of the specified file into the given byte
// slice and returns any error encountered.
func readContent(f *os.File, d *[]byte) error {
	log.Printf("reading file: '%s'", f.Name())
	var err error
	*d, err = os.ReadFile(f.Name())
	if err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	return nil
}

// editFile executes a command to edit the specified file, logging errors if
// the command fails.
func editFile(te *TextEditor, f *os.File) error {
	if te.cmd == "" {
		return ErrCommandNotFound
	}

	log.Printf("executing args: cmd='%s' args='%v'", te.cmd, te.args)
	if err := sys.RunCmd(te.cmd, append(te.args, f.Name())...); err != nil {
		return fmt.Errorf("error running editor: %w", err)
	}

	return nil
}

// Edit edits the contents of a byte slice by creating a temporary file,
// editing it with an external editor, and then reading the modified contents
// back into the byte slice.
func Edit(te *TextEditor, b []byte) error {
	f, err := createTEmpFileWithData(b)
	if err != nil {
		return fmt.Errorf("%w", err)
	}
	defer closeAndClean(f)
	log.Printf("editing file: '%s' with text editor: '%s'", f.Name(), te.name)
	if err := editFile(te, f); err != nil {
		return err
	}
	if err := readContent(f, &b); err != nil {
		return fmt.Errorf("%w", err)
	}

	return nil
}

func newTextEditor(c, n string, arg []string) *TextEditor {
	return &TextEditor{
		cmd:  c,
		name: n,
		args: arg,
	}
}
