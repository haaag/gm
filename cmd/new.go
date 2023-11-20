/*
Copyright © 2023 haaag <git.haaag@gmail.com>
*/package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"gomarks/pkg/bookmark"
	"gomarks/pkg/color"
	"gomarks/pkg/config"
	"gomarks/pkg/errs"
	"gomarks/pkg/format"

	"github.com/spf13/cobra"
)

var newCmd = &cobra.Command{
	Use:     "new",
	Short:   "add a new bookmark",
	Long:    "add a new bookmark and fetch title and description",
	Example: exampleUsage([]string{"new\n", "new <url>\n", "new <url> <tags>"}),
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := getDB()
		if err != nil {
			return fmt.Errorf("%w", err)
		}

		format.CmdTitle("adding a new bookmark")

		url := handleURL(&args)

		if r.RecordExists(config.DB.Table.Main, url) {
			if b, _ := r.GetRecordByURL(config.DB.Table.Main, url); b != nil {
				return fmt.Errorf("%w with id: %d", errs.ErrBookmarkDuplicate, b.ID)
			}
		}

		tags := handleTags(&args)
		title := handleTitle(url)
		desc := handleDesc(url)

		b := bookmark.New(url, title, tags, desc)

		if err = handleConfirmAndValidation(b, handleNoConfirmation(cmd)); err != nil {
			return fmt.Errorf("handle confirmation and validation: %w", err)
		}

		b, err = r.InsertRecord(config.DB.Table.Main, b)
		if err != nil {
			return fmt.Errorf("%w", err)
		}

		fmt.Print(color.ColorizeBold("\nNew bookmark saved with id: ", color.Green))
		fmt.Println(color.ColorizeBold(strconv.Itoa(b.ID), color.Green))

		return nil
	},
}

func init() {
	var url string
	var tags string
	newCmd.Flags().StringVarP(&url, "url", "u", "", "url for new bookmark")
	newCmd.Flags().StringVarP(&tags, "tags", "t", "", "tags for new bookmark")
	rootCmd.AddCommand(newCmd)
}

func handleConfirmAndValidation(b *bookmark.Bookmark, noConfirm bool) error {
	if noConfirm {
		return validateBookmark(b)
	}

	option := ConfirmOrEdit("Save bookmark?")
	switch option {
	case "n":
		return fmt.Errorf("%w", errs.ErrActionAborted)
	case "e":
		editedBookmark, err := bookmark.Edit(b)

		if errors.Is(err, errs.ErrBookmarkUnchaged) {
			return nil
		}

		if err != nil {
			return fmt.Errorf("%w", err)
		}

		return validateBookmark(editedBookmark)
	}

	return nil
}

func validateBookmark(b *bookmark.Bookmark) error {
	if !b.IsValid() {
		return fmt.Errorf("%w", errs.ErrBookmarkInvalid)
	}
	return nil
}

func ConfirmOrEdit(question string) string {
	q := color.ColorizeBold(question, color.White)
	options := color.Colorize("[Yes/no/edit]: ", color.Gray)
	prompt := fmt.Sprintf("\n%s %s", q, options)

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print(prompt)
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading input:", err)
			return ""
		}

		input = strings.TrimSpace(input)
		input = strings.ToLower(input)

		switch input {
		case "y", "yes":
			return "y"
		case "n", "no":
			return "n"
		case "e", "edit":
			return "e"
		case "":
			return "y"
		default:
			fmt.Println("Invalid response. Please enter 'y' or 'n' or 'e'.")
		}
	}
}
