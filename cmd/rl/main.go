package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/bunchhieng/rl/internal/app"
	"github.com/bunchhieng/rl/internal/cli"
	"github.com/bunchhieng/rl/internal/storage"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
)

var version = "dev"

func main() {
	var dbPath string
	var showVersion bool

	flag.StringVar(&dbPath, "db-path", "", "path to database file (default: platform config directory)")
	flag.BoolVar(&showVersion, "version", false, "show version")
	flag.Parse()

	if showVersion {
		fmt.Printf("rl version %s\n", version)
		os.Exit(0)
	}

	if len(flag.Args()) == 0 {
		printUsage()
		os.Exit(1)
	}

	// Initialize storage
	s, err := app.NewStorage(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to initialize storage: %v\n", err)
		os.Exit(1)
	}
	defer s.Close()

	commands := cli.NewCommands(s)
	cmd := flag.Args()[0]

	switch cmd {
	case "add":
		if err := handleAdd(commands, flag.Args()[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "%sError:%s %v\n", colorRed, colorReset, err)
			os.Exit(1)
		}

	case "ls", "list": // ls is Linux standard, list is alias
		if err := handleList(commands, flag.Args()[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "%sError:%s %v\n", colorRed, colorReset, err)
			os.Exit(1)
		}

	case "open":
		if err := handleOpen(commands, flag.Args()[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "%sError:%s %v\n", colorRed, colorReset, err)
			os.Exit(1)
		}

	case "done":
		if err := handleDone(commands, flag.Args()[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "%sError:%s %v\n", colorRed, colorReset, err)
			os.Exit(1)
		}

	case "undo":
		if err := handleUndo(commands, flag.Args()[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "%sError:%s %v\n", colorRed, colorReset, err)
			os.Exit(1)
		}

	case "rm":
		if err := handleRemove(commands, flag.Args()[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "%sError:%s %v\n", colorRed, colorReset, err)
			os.Exit(1)
		}

	case "export":
		if err := handleExport(commands, flag.Args()[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "%sError:%s %v\n", colorRed, colorReset, err)
			os.Exit(1)
		}

	case "import":
		if err := handleImport(commands, flag.Args()[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "%sError:%s %v\n", colorRed, colorReset, err)
			os.Exit(1)
		}

	case "grep", "search": // grep is Linux standard, search is alias
		if err := handleSearch(commands, flag.Args()[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "%sError:%s %v\n", colorRed, colorReset, err)
			os.Exit(1)
		}

	default:
		suggestion := suggestCommand(cmd)
		fmt.Fprintf(os.Stderr, "%sError:%s unknown command: %s%s%s\n", colorRed, colorReset, colorBold, cmd, colorReset)
		if suggestion != "" {
			fmt.Fprintf(os.Stderr, "\n%sDid you mean:%s %s%s%s?\n\n", colorYellow, colorReset, colorBold, suggestion, colorReset)
		}
		printUsage()
		os.Exit(1)
	}
}

func handleAdd(commands *cli.Commands, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: rl add <url> [--title \"...\"] [--note \"...\"] [--tags \"t1,t2\"]")
	}

	fs := flag.NewFlagSet("add", flag.ExitOnError)
	title := fs.String("title", "", "title for the link")
	note := fs.String("note", "", "note for the link")
	tags := fs.String("tags", "", "comma-separated tags")
	fs.Parse(args[1:])

	url := args[0]
	return commands.Add(url, *title, *note, *tags)
}

func handleList(commands *cli.Commands, args []string) error {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	_ = fs.Bool("unread", false, "show only unread links") // Default behavior
	read := fs.Bool("read", false, "show only read links")
	all := fs.Bool("all", false, "show all links")
	tag := fs.String("tag", "", "filter by tag")
	limit := fs.Int("limit", 0, "limit number of results")
	fs.Parse(args)

	readStatus := storage.ReadStatusUnread
	if *all {
		readStatus = storage.ReadStatusAll
	} else if *read {
		readStatus = storage.ReadStatusRead
	}

	return commands.List(readStatus, *tag, *limit)
}

func handleOpen(commands *cli.Commands, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: rl open <id>")
	}

	id, err := cli.ParseID(args[0])
	if err != nil {
		return err
	}

	return commands.Open(id)
}

func handleDone(commands *cli.Commands, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: rl done <id>")
	}

	id, err := cli.ParseID(args[0])
	if err != nil {
		return err
	}

	return commands.Done(id)
}

func handleUndo(commands *cli.Commands, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: rl undo <id>")
	}

	id, err := cli.ParseID(args[0])
	if err != nil {
		return err
	}

	return commands.Undo(id)
}

func handleRemove(commands *cli.Commands, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: rl rm <id> [id...]")
	}

	ids := make([]string, 0, len(args))
	for _, arg := range args {
		id, err := cli.ParseID(arg)
		if err != nil {
			return err
		}
		ids = append(ids, id)
	}

	return commands.Remove(ids...)
}

func handleExport(commands *cli.Commands, args []string) error {
	return commands.Export(os.Stdout)
}

func handleImport(commands *cli.Commands, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: rl import <file.json>")
	}

	return commands.Import(args[0])
}

func handleSearch(commands *cli.Commands, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: rl search \"<query>\"")
	}

	return commands.Search(args[0])
}

func suggestCommand(cmd string) string {
	commands := []string{"add", "ls", "list", "open", "done", "undo", "rm", "export", "import", "grep", "search"}

	bestMatch := ""
	minDistance := len(cmd) + 1

	for _, c := range commands {
		distance := levenshteinDistance(cmd, c)
		if distance < minDistance && distance <= 2 {
			minDistance = distance
			bestMatch = c
		}
	}

	return bestMatch
}

func levenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
		matrix[i][0] = i
	}
	for j := 0; j <= len(s2); j++ {
		matrix[0][j] = j
	}

	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 0
			if s1[i-1] != s2[j-1] {
				cost = 1
			}
			matrix[i][j] = min(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len(s1)][len(s2)]
}

func min(a, b, c int) int {
	if a < b && a < c {
		return a
	}
	if b < c {
		return b
	}
	return c
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `%s%srl - Read Later CLI%s

%sUsage:%s rl [--db-path <path>] <command>

%sCommands:%s
  %sadd%s    <url> [--title "..."] [--note "..."] [--tags "..."]  Add or update link
  %sls%s     [--read|--all] [--tag <tag>] [--limit <n>]          List links (alias: list)
  %sopen%s   <id>                                                 Open in browser
  %sdone%s   <id>                                                 Mark as read
  %sundo%s   <id>                                                 Mark as unread
  %srm%s     <id> [id...]                                         Delete link(s)
  %sexport%s                                                      Export to JSON
  %simport%s <file>                                               Import from JSON
  %sgrep%s   <query>                                              Search links (alias: search)

%sExamples:%s
  rl add https://example.com --title "Example" --tags "web"
  rl ls --all --tag web --limit 10
  rl open abc123 && rl done abc123
  rl grep "golang"
%s
`,
		colorBold, colorCyan, colorReset,
		colorBold, colorReset,
		colorBold, colorReset,
		colorCyan, colorReset,
		colorCyan, colorReset,
		colorCyan, colorReset,
		colorCyan, colorReset,
		colorCyan, colorReset,
		colorCyan, colorReset,
		colorCyan, colorReset,
		colorCyan, colorReset,
		colorCyan, colorReset,
		colorBold, colorReset,
		colorReset)
}
