package main

import (
	"fmt"
	"os"

	"github.com/bunchhieng/rl/internal/app"
	"github.com/bunchhieng/rl/internal/cli"
	"github.com/bunchhieng/rl/internal/storage"
	"github.com/bunchhieng/rl/internal/tui"
	urfavecli "github.com/urfave/cli/v2"
)

const (
	colorReset = "\033[0m"
	colorRed   = "\033[31m"
)

var version = "dev"

func main() {
	cliApp := &urfavecli.App{
		Name:                 "rl",
		Usage:                "Read Later CLI - A minimal, local-first read later tool",
		Version:              version,
		EnableBashCompletion: true,
		Flags: []urfavecli.Flag{
			&urfavecli.StringFlag{
				Name:  "db-path",
				Usage: "path to database file (default: platform config directory)",
			},
		},
		Action: func(c *urfavecli.Context) error {
			// Launch TUI if no command provided
			s, err := app.NewStorage(c.String("db-path"))
			if err != nil {
				return fmt.Errorf("failed to initialize storage: %w", err)
			}
			defer s.Close()
			return tui.Run(s)
		},
		Commands: []*urfavecli.Command{
			{
				Name:    "add",
				Aliases: []string{"a"},
				Usage:   "Add or update a link",
				Flags: []urfavecli.Flag{
					&urfavecli.StringFlag{Name: "title", Usage: "title for the link"},
					&urfavecli.StringFlag{Name: "note", Usage: "note for the link"},
					&urfavecli.StringFlag{Name: "tags", Usage: "comma-separated tags"},
				},
				Action: func(c *urfavecli.Context) error {
					if c.NArg() == 0 {
						return fmt.Errorf("usage: rl add [--title \"...\"] [--note \"...\"] [--tags \"...\"] <url>")
					}
					return withStorage(c, func(commands *cli.Commands) error {
						return commands.Add(c.Args().Get(0), c.String("title"), c.String("note"), c.String("tags"))
					})
				},
			},
			{
				Name:    "ls",
				Aliases: []string{"list", "l"},
				Usage:   "List links (default: unread)",
				Flags: []urfavecli.Flag{
					&urfavecli.BoolFlag{Name: "read", Usage: "show only read links"},
					&urfavecli.BoolFlag{Name: "all", Usage: "show all links"},
					&urfavecli.StringFlag{Name: "tag", Usage: "filter by tag"},
					&urfavecli.IntFlag{Name: "limit", Usage: "limit number of results"},
				},
				Action: func(c *urfavecli.Context) error {
					readStatus := storage.ReadStatusUnread
					if c.Bool("all") {
						readStatus = storage.ReadStatusAll
					} else if c.Bool("read") {
						readStatus = storage.ReadStatusRead
					}
					return withStorage(c, func(commands *cli.Commands) error {
						return commands.List(readStatus, c.String("tag"), c.Int("limit"))
					})
				},
			},
			{
				Name:    "open",
				Aliases: []string{"o"},
				Usage:   "Open link in browser",
				Action: func(c *urfavecli.Context) error {
					if c.NArg() == 0 {
						return fmt.Errorf("usage: rl open <id>")
					}
					id, err := cli.ParseID(c.Args().Get(0))
					if err != nil {
						return err
					}
					return withStorage(c, func(commands *cli.Commands) error {
						return commands.Open(id)
					})
				},
			},
			{
				Name:    "done",
				Aliases: []string{"d"},
				Usage:   "Mark link as read",
				Action: func(c *urfavecli.Context) error {
					if c.NArg() == 0 {
						return fmt.Errorf("usage: rl done <id>")
					}
					id, err := cli.ParseID(c.Args().Get(0))
					if err != nil {
						return err
					}
					return withStorage(c, func(commands *cli.Commands) error {
						return commands.Done(id)
					})
				},
			},
			{
				Name:    "undo",
				Aliases: []string{"u"},
				Usage:   "Mark link as unread",
				Action: func(c *urfavecli.Context) error {
					if c.NArg() == 0 {
						return fmt.Errorf("usage: rl undo <id>")
					}
					id, err := cli.ParseID(c.Args().Get(0))
					if err != nil {
						return err
					}
					return withStorage(c, func(commands *cli.Commands) error {
						return commands.Undo(id)
					})
				},
			},
			{
				Name:    "rm",
				Aliases: []string{"remove", "delete"},
				Usage:   "Delete one or more links",
				Action: func(c *urfavecli.Context) error {
					if c.NArg() == 0 {
						return fmt.Errorf("usage: rl rm <id> [id...]")
					}
					ids := make([]string, 0, c.NArg())
					for i := 0; i < c.NArg(); i++ {
						id, err := cli.ParseID(c.Args().Get(i))
						if err != nil {
							return err
						}
						ids = append(ids, id)
					}
					return withStorage(c, func(commands *cli.Commands) error {
						return commands.Remove(ids...)
					})
				},
			},
			{
				Name:  "export",
				Usage: "Export all links to JSON",
				Action: func(c *urfavecli.Context) error {
					return withStorage(c, func(commands *cli.Commands) error {
						return commands.Export(os.Stdout)
					})
				},
			},
			{
				Name:  "import",
				Usage: "Import links from JSON file",
				Action: func(c *urfavecli.Context) error {
					if c.NArg() == 0 {
						return fmt.Errorf("usage: rl import <file.json>")
					}
					return withStorage(c, func(commands *cli.Commands) error {
						return commands.Import(c.Args().Get(0))
					})
				},
			},
			{
				Name:    "grep",
				Aliases: []string{"search"},
				Usage:   "Search links using full-text search",
				Action: func(c *urfavecli.Context) error {
					if c.NArg() == 0 {
						return fmt.Errorf("usage: rl grep \"<query>\"")
					}
					return withStorage(c, func(commands *cli.Commands) error {
						return commands.Search(c.Args().Get(0))
					})
				},
			},
			{
				Name:    "tui",
				Aliases: []string{"interactive", "i"},
				Usage:   "Launch interactive TUI mode",
				Action: func(c *urfavecli.Context) error {
					s, err := app.NewStorage(c.String("db-path"))
					if err != nil {
						return fmt.Errorf("failed to initialize storage: %w", err)
					}
					defer s.Close()
					return tui.Run(s)
				},
			},
		},
	}

	if err := cliApp.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "%sError:%s %v\n", colorRed, colorReset, err)
		os.Exit(1)
	}
}

// withStorage initializes storage and runs the given function with commands
func withStorage(c *urfavecli.Context, fn func(*cli.Commands) error) error {
	s, err := app.NewStorage(c.String("db-path"))
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	defer s.Close()
	commands := cli.NewCommands(s)
	return fn(commands)
}
