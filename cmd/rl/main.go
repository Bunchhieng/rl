package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/bunchhieng/rl/internal/app"
	"github.com/bunchhieng/rl/internal/cli"
	"github.com/bunchhieng/rl/internal/storage"
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
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "list":
		if err := handleList(commands, flag.Args()[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "open":
		if err := handleOpen(commands, flag.Args()[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "done":
		if err := handleDone(commands, flag.Args()[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "undo":
		if err := handleUndo(commands, flag.Args()[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "rm":
		if err := handleRemove(commands, flag.Args()[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "export":
		if err := handleExport(commands, flag.Args()[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "import":
		if err := handleImport(commands, flag.Args()[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "search":
		if err := handleSearch(commands, flag.Args()[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "version":
		commands.Version(version)

	default:
		fmt.Fprintf(os.Stderr, "Error: unknown command: %s\n", cmd)
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

func printUsage() {
	fmt.Fprintf(os.Stderr, `rl - Read Later CLI

Usage:
  rl [flags] <command>

Global Flags:
  --db-path string    Database file path (default: platform config directory)
  --version           Show version

Commands:
  add <url> [flags] - Add or update a link
    Flags:
      --title string    Title for the link
      --note string     Note for the link
      --tags string     Comma-separated tags

  list [flags] - List links (default: unread)
    Flags:
      --unread          Show only unread links (default)
      --read            Show only read links
      --all             Show all links
      --tag string      Filter by tag
      --limit int       Limit number of results

  open <id> - Open link in browser
  done <id> - Mark link as read
  undo <id> - Mark link as unread
  rm <id> [id...] - Delete one or more links separated by spaces
  export - Export all links as JSON to stdout
  import <file> - Import links from JSON file
  search <query> - Search links using full-text search
  version - Show version

Examples:
  rl add https://example.com --title "Example" --note "Check this out" --tags "web,example"
  rl list --read --tag web --limit 10
  rl list --all
  rl open abc123
  rl done abc123
  rl undo abc123
  rl rm abc123 def456
  rl export > links.json
  rl import links.json
  rl search "golang"
`)
}
