package display

import (
	"fmt"
	"io"
	"log"
	"os/exec"
	"slices"
	"strings"

	"gomarks/pkg/format"
)

func NewMenu(s string) (*Menu, error) {
	mc := make(menuCollection)
	mc.load()
	menu, err := mc.get(s)
	if err != nil {
		return nil, fmt.Errorf("error creating menu: %w", err)
	}

	return &menu, nil
}

type menuCollection map[string]Menu

func (mc menuCollection) register(m Menu) {
	log.Println("Registering menu:", m.Command)
	mc[m.Command] = m
}

func (mc menuCollection) get(s string) (Menu, error) {
	menu, ok := mc[s]
	if !ok {
		return Menu{}, fmt.Errorf("%w: '%s'", format.ErrInvalidOption, s)
	}

	log.Println("Got menu:", menu.Command)

	return menu, nil
}

func (mc menuCollection) load() {
	mc.register(rofiMenu)
	mc.register(dmenuMenu)
}

type Menu struct {
	Command   string
	Arguments []string
}

func (m *Menu) UpdateMessage(message string) {
	replaceArg(m.Arguments, "-mesg", message)
}

func (m *Menu) Select(items []fmt.Stringer) (int, error) {
	itemsText := make([]string, 0, len(items))
	for _, item := range items {
		itemsText = append(itemsText, item.String())
	}

	itemsString := strings.Join(itemsText, "\n")
	output, err := m.Run(itemsString)
	if err != nil {
		log.Fatal(err)
	}

	selectedStr := strings.TrimSpace(output)

	if !isSelectedTextInItems(selectedStr, itemsText) {
		return -1, fmt.Errorf("%w: '%s'", format.ErrInvalidOption, selectedStr)
	}

	return findSelectedIndex(selectedStr, itemsText), nil
}

func (m *Menu) Run(s string) (string, error) {
	log.Println("Running menu:", m.Command, m.Arguments)
	cmd := exec.Command(m.Command, m.Arguments...)

	if s != "" {
		cmd.Stdin = strings.NewReader(s)
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("error creating output pipe: %w", err)
	}

	err = cmd.Start()
	if err != nil {
		return "", fmt.Errorf("error starting dmenu: %w", err)
	}

	output, err := io.ReadAll(stdoutPipe)
	if err != nil {
		return "", fmt.Errorf("error reading output: %w", err)
	}

	err = cmd.Wait()
	if err != nil {
		return "", fmt.Errorf("user hit scape: %w", err)
	}

	outputStr := string(output)
	outputStr = strings.TrimRight(outputStr, "\n")
	log.Println("Output:", outputStr)

	return outputStr, nil
}

var rofiMenu = Menu{
	Command: "rofi",
	Arguments: []string{
		"-dmenu",
		"-l", "10",
		"-p", "GoMarks",
		"-mesg", "Welcome to GoMarks",
		"-theme-str", "window {width: 75%; height: 55%;}",
		"-theme-str", "textbox {markup: false;}",
	},
}

var dmenuMenu = Menu{
	Command: "dmenu",
	Arguments: []string{
		"-i",
		"-p", "GoMarks>",
		"-l", "15",
	},
}

func isSelectedTextInItems(s string, items []string) bool {
	for _, item := range items {
		if strings.Contains(item, s) {
			return true
		}
	}

	return false
}

func findSelectedIndex(s string, items []string) int {
	log.Printf("Finding selected in %d items", len(items))
	idx := slices.IndexFunc(items, func(item string) bool {
		return strings.Contains(item, s)
	})
	log.Println("FindSelectedIndex:", idx)

	return idx
}

func replaceArg(args []string, argName, newValue string) {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == argName {
			args[i+1] = newValue
			break
		}
	}
}
