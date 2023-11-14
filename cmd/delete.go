/*
Copyright © 2023 haaag <git.haaag@gmail.com>
*/package cmd

import (
	"fmt"

	"gomarks/pkg/bookmark"
	"gomarks/pkg/color"
	"gomarks/pkg/constants"
	"gomarks/pkg/database"
	"gomarks/pkg/errs"
	"gomarks/pkg/format"
	"gomarks/pkg/util"

	"github.com/spf13/cobra"
)

var deleteExamples = []string{"delete\n", "delete <id>\n", "delete <query>"}

const maxLen = 80

var deleteCmd = &cobra.Command{
	Use:          "delete",
	Short:        "delete a bookmark by query",
	Example:      exampleUsage(deleteExamples),
	SilenceUsage: true,
	Args:         cobra.MaximumNArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		r, err := getDB()
		if err != nil {
			return fmt.Errorf("%w", err)
		}

		bs, err := handleGetRecords(r, args)
		if err != nil {
			return fmt.Errorf("fetching records: %w", err)
		}

		format.CmdTitle("delete mode")

		bFound := fmt.Sprintf("[%d] bookmarks found\n", bs.Len())
		bf := color.Colorize(bFound, color.Red)
		fmt.Println(bf)

		toDel, err := parseSliceDel(*bs)
		if err != nil {
			return fmt.Errorf("parsing slice: %w", err)
		}

		if err = deleteAndReorder(r, &toDel); err != nil {
			return fmt.Errorf("deleting and reordering records: %w", err)
		}

		total := fmt.Sprintf("[%d] bookmarks deleted.\n", toDel.Len())
		deleting := color.Colorize(total, color.Red)
		fmt.Printf("%s%s\n", color.Bold, deleting)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}

/**
 * Deletes the specified bookmarks from the database and reorders the remaining IDs.
 *
 * @param r The SQLite repository to use for accessing the database.
 * @param toDel A pointer to a `bookmark.Slice` containing the bookmarks to be deleted.
 * @return An error if any occurred during deletion or reordering.
 */
func deleteAndReorder(r *database.SQLiteRepository, toDel *bookmark.Slice) error {
	if err := r.DeleteRecordsBulk(constants.DBMainTableName, toDel.IDs()); err != nil {
		return fmt.Errorf("deleting records in bulk: %w", err)
	}

	if err := r.ReorderIDs(constants.DBMainTableName); err != nil {
		return fmt.Errorf("reordering ids: %w", err)
	}

	return nil
}

/**
 * Interactively prompts the user to select bookmarks for deletion and constructs a slice of the selected bookmarks.
 *
 * @param bs A slice of bookmarks from which to select.
 * @return A slice of the selected bookmarks for deletion, or an error if no bookmarks were selected.
 */
func parseSliceDel(bs bookmark.Slice) (bookmark.Slice, error) {
	if bs.Len() == 0 {
		return nil, fmt.Errorf("%w", errs.ErrBookmarkNotSelected)
	}

	var toDel bookmark.Slice

	for i, b := range bs {
		fmt.Println(b.String())

		// Prompt the user to confirm deletion for each bookmark.
		deletePrompt := fmt.Sprintf("Delete bookmark [%d/%d]?", i+1, bs.Len())
		confirm := util.Confirm(deletePrompt)

		if confirm {
			toDel = append(toDel, b)
		}

		// If there are multiple bookmarks and the user confirmed deletion, provide a confirmation message.
		if bs.Len() > 1 && confirm {
			fmt.Println(color.Colorize("Added to delete queue", color.Red))
		}
		fmt.Println()
	}

	// Check if any bookmarks were selected for deletion.
	if toDel.Len() == 0 {
		return nil, fmt.Errorf("slice to delete: %w", errs.ErrBookmarkNotSelected)
	}

	// If multiple bookmarks were selected, summarize the deletion and prompt for final confirmation.
	if toDel.Len() > 1 {
		d := fmt.Sprintf("Bookmarks to delete [%d]", toDel.Len())
		fmt.Println(color.ColorizeBold(d, color.Red))
		printSliceSummary(&toDel)

		if confirm := util.Confirm("Are you sure?"); !confirm {
			return nil, fmt.Errorf("%w", errs.ErrActionAborted)
		}
	}

	return toDel, nil
}
