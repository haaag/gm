package main

// [TODO):
// - [ ] add sub-commands (maybe use Cobra!)
// - [X] add format option to json, pretty, plain
// - [ ] better module/pkg naming.

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"gomarks/pkg/constants"
	"gomarks/pkg/data"
	"gomarks/pkg/database"
	"gomarks/pkg/info"
	"gomarks/pkg/util"
)

var (
	// bookmarks
	add         string
	edit        bool
	delete      bool
	tags        string
	id          int
	list        bool
	queryFilter string
	copy        bool
	open        bool

	// actions
	format     string
	head       int
	tail       int
	pick       string
	menu       string
	restore    bool
	incomplete bool

	// app
	verbose  bool
	version  bool
	testFlag bool
	showInfo bool
)

func init() {
	// bookmarks
	flag.StringVar(&add, "add", "", "add a bookmark [format: URL Tags]")
	flag.BoolVar(&edit, "edit", false, "edit a bookmark")
	flag.BoolVar(&delete, "delete", false, "delete a bookmark")
	flag.StringVar(&tags, "tags", "", "tag a bookmark")
	flag.IntVar(&id, "id", 0, "bookmark id")
	flag.BoolVar(&list, "list", false, "list all bookmarks")
	flag.StringVar(&queryFilter, "query", "", "query to filter bookmarks")
	flag.BoolVar(&copy, "copy", false, "copy a bookmark")
	flag.BoolVar(&open, "open", false, "open bookmark in default browser")

	// actions
	flag.StringVar(&format, "f", "pretty", "output format [json|pretty|plain]")
	flag.IntVar(&head, "head", 0, "output the first part of bookmarks")
	flag.IntVar(&tail, "tail", 0, "output the last part of bookmarks")
	flag.StringVar(&pick, "pick", "", "pick data [url|title|tags]")
	flag.StringVar(&menu, "menu", "", "menu mode [dmenu|rofi]")
	flag.BoolVar(&restore, "restore", false, "restore a bookmark")
	flag.BoolVar(&incomplete, "incomplete", false, "filter by incomplete bookmark")

	// app
	flag.BoolVar(&testFlag, "test", false, "test mode")
	flag.BoolVar(&verbose, "v", false, "enable verbose output")
	flag.BoolVar(&version, "version", false, "version")
	flag.BoolVar(&showInfo, "info", false, "show app info")
}

func parseQueryFlag() {
	args := os.Args[1:]
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		queryFilter = args[0]
		args = args[1:]
	}
	os.Args = append([]string{os.Args[0]}, args...)
}

func main() {
	tableName := constants.DBMainTableName

	parseQueryFlag()
	flag.Parse()

	if version {
		fmt.Printf("%s v%s\n", constants.AppName, constants.Version)
		return
	}

	util.SetLogLevel(&verbose)
	util.SetupHomeProject()
	r := database.GetDB()
	defer r.DB.Close()

	// Test mode
	if testFlag {
		fmt.Println("Testing...")
		return
	}

	// Print info
	if showInfo {
		fmt.Println(info.AppInfo(r))
	}

	// Set tableName as deleted table for restore
	if restore {
		// FIX: finish it. Restore is missing
		tableName = constants.DBDeletedTableName
	}

	// By ID, list or query
	bs, err := data.RetrieveBookmarks(r, &tableName, &queryFilter, id, &list, incomplete)
	if err != nil {
		util.PrintErrMsg(err, verbose)
	}

	// Apply head and tail options
	if err = data.HeadAndTail(bs, head, tail); err != nil {
		util.PrintErrMsg(err, verbose)
	}

	// Handle pick
	if pick != "" {
		if err = data.PickAttribute(bs, pick); err != nil {
			util.PrintErrMsg(err, verbose)
		}
		return
	}

	// Handle menu option
	if err = data.PickBookmarkWithMenu(bs, menu); err != nil {
		util.PrintErrMsg(err, verbose)
	}

	// Handle add
	if add != "" {
		if err = data.HandleAdd(r, add, tags, tableName); err != nil {
			util.PrintErrMsg(err, verbose)
		}
		return
	}

	// Handle edit
	if edit {
		if err = data.HandleEdit(r, bs, tableName); err != nil {
			util.PrintErrMsg(err, verbose)
		}
		return
	}

	// Handle action
	if copy || open {
		if err = data.HandleAction(bs, copy, open); err != nil {
			util.PrintErrMsg(err, verbose)
		}
		return
	}

	// Handle format
	if format != "" {
		if err = data.HandleFormat(format, bs); err != nil {
			util.PrintErrMsg(err, verbose)
		}
		return
	}
}
