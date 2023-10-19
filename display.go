package main

import (
	"fmt"
	"log"
	"strings"
)

type Option struct {
	Label string
}

func (o Option) String() string {
	return o.Label
}

func ShowOptions(m *Menu) (int, error) {
	options := []fmt.Stringer{
		Option{"Add a bookmark"},
		Option{"Edit a bookmark"},
		Option{"Delete a bookmark"},
		Option{"Exit"},
	}
	idx, err := m.Select(options)
	if err != nil {
		log.Fatal(err)
	}
	return idx, nil
}

func PavelOptions(menuArgs []string) (int, error) {
	optionsMap := make(map[string]interface{})
	optionsMap["Add a bookmark"] = addBookmark
	optionsMap["Edit a bookmark"] = editBookmark
	optionsMap["Delete a bookmark"] = deleteBookmark
	optionsMap["Exit"] = nil
	return -1, nil
}

func addBookmark(r *SQLiteRepository, m *Menu, b *Bookmark) (Bookmark, error) {
	return Bookmark{}, nil
}

func editBookmark(r *SQLiteRepository, m *Menu, b *Bookmark) (Bookmark, error) {
	m.UpdatePrompt(fmt.Sprintf("Editing ID: %d", b.ID))
	s, err := m.Run(b.String())
	if err != nil {
		return Bookmark{}, err
	}
	fmt.Println(s)
	return *b, nil
}

func deleteBookmark(r *SQLiteRepository, m *Menu, b *Bookmark) error {
	msg := fmt.Sprintf("Deleting bookmark: %s", b.URL)
	if !m.Confirm(msg, "Are you sure?") {
		return fmt.Errorf("Cancelled")
	}
	err := r.deleteRecord(b, DBMainTable)
	if err != nil {
		return err
	}

	err = r.insertRecord(b, DBDeletedTable)
	if err != nil {
		return err
	}

	err = r.reorderIDs()
	if err != nil {
		return err
	}
	return nil
}

func handleOptionsMode(m *Menu) {
	idx, err := ShowOptions(m)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Selected:", idx)
}

func handleTestMode(m *Menu, r *SQLiteRepository) {
	fmt.Print("::::::::Test Mode::::::::\n\n")
}

func fetchBookmarks(r *SQLiteRepository) ([]Bookmark, error) {
	var bookmarks []Bookmark
	var err error

	if byQuery != "" {
		bookmarks, err = r.getRecordsByQuery(byQuery)
		if err != nil {
			return nil, err
		}
	} else {
		bookmarks, err = r.getRecordsAll()
		if err != nil {
			return nil, err
		}
	}

	if len(bookmarks) == 0 {
		return []Bookmark{}, fmt.Errorf("no bookmarks found")
	}
	return bookmarks, nil
}

func SelectBookmark(m *Menu, bookmarks *[]Bookmark) (Bookmark, error) {
	var itemsText []string
	m.UpdateMessage(fmt.Sprintf(" Welcome to GoMarks\n Showing (%d) bookmarks", len(*bookmarks)))

	for _, bm := range *bookmarks {
		itemText := fmt.Sprintf(
			"%-4d %-80s %-10s",
			bm.ID,
			shortenString(bm.URL, 80),
			bm.Tags,
		)
		itemsText = append(itemsText, itemText)
	}

	itemsString := strings.Join(itemsText, "\n")
	output, err := m.Run(itemsString)
	if err != nil {
		log.Fatal(err)
	}

	selectedStr := strings.Trim(output, "\n")
	index := findSelectedIndex(selectedStr, itemsText)
	if index != -1 {
		return (*bookmarks)[index], nil
	}
	return Bookmark{}, fmt.Errorf("item not found: %s", selectedStr)
}
