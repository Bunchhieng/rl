# rl - Read Later CLI

A minimal, local-first "read later" CLI tool for macOS and Linux. Store links locally with SQLite, no account, no sync, no background daemon. Follows Linux command conventions (`ls`, `rm`, `grep`) for familiarity.

## Features

- **Local-first**: All data stored in a single SQLite file
- **Fast**: Minimal dependencies, quick startup
- **Portable**: Easy export/import via JSON
- **Search**: Full-text search across URLs, titles, notes, and tags
- **Simple**: Clean CLI interface with standard library only

## Installation

### Go Install

```bash
go install github.com/bunchhieng/rl/cmd/rl@latest
```

Add to PATH:
```bash
# zsh (macOS) - uses go env to get GOPATH reliably
echo 'export PATH="$(go env GOPATH)/bin:$PATH"' >> ~/.zshrc && source ~/.zshrc

# bash (Linux)
echo 'export PATH="$(go env GOPATH)/bin:$PATH"' >> ~/.bashrc && source ~/.bashrc
```

Verify: `rl version`

### Build from Source

```bash
git clone https://github.com/bunchhieng/rl.git && cd rl
go build -o bin/rl ./cmd/rl
# Or: make install
```

## Database Location

- **macOS**: `~/Library/Application Support/rl/links.db`
- **Linux**: `~/.config/rl/links.db`
- **Windows**: `%AppData%/rl/links.db`

Override with `--db-path` flag.

## Usage

Commands follow Linux conventions for familiarity. Use `rl --help` or `rl <command>` for details.

### Add a link
```bash
rl add <url> [--title "..."] [--note "..."] [--tags "..."]
rl add https://example.com --title "Example" --tags "web,example"
```

### List links (ls - Linux standard)
```bash
rl ls                      # Unread links (default)
rl ls --read               # Read links only
rl ls --all                # All links
rl ls --tag <tag>          # Filter by tag
rl ls --limit <n>          # Limit number of results
# 'list' also works as alias
```

### Open, mark, delete
```bash
rl open <id>               # Open link in browser (doesn't mark as read)
rl done <id>               # Mark link as read
rl undo <id>               # Mark link as unread
rl rm <id> [id...]         # Delete one or more links (Linux standard)
```

### Search (grep - Linux standard)
```bash
rl grep <query>            # Full-text search across URL, title, note, tags
# 'search' also works as alias
```

### Export/Import
```bash
rl export > links.json     # Export all links to JSON
rl import <file>           # Import links from JSON (merges duplicates)
```

## Examples

```bash
# Add links
rl add https://golang.org --title "Go" --tags "programming,go"
rl add https://rust-lang.org --tags "programming,rust"

# List and filter
rl ls                      # List unread
rl ls --tag programming    # Filter by tag
rl ls --all --limit 10     # All links, limited

# Work with links
rl open <id>               # Open in browser
rl done <id>               # Mark as read
rl rm <id> <id>            # Delete multiple links

# Search and backup
rl grep "programming"      # Search links
rl export > backup.json    # Backup all links
```

## Development

```bash
make build                 # Build binary
make test                  # Run tests
make install               # Install locally
```

## JSON Export Format

```json
[
  {
    "id": "9m1w2z3x",
    "url": "https://example.com",
    "title": "Example Site",
    "note": "Optional note",
    "tags": "tag1,tag2",
    "created_at": "2024-01-01T12:00:00Z",
    "read_at": "2024-01-02T10:30:00Z"
  }
]
```

## Architecture

- **cmd/rl**: Main entry point
- **internal/app**: Application initialization
- **internal/storage**: SQLite implementation
- **internal/model**: Data models and validation
- **internal/cli**: Command handlers

## Dependencies

- `modernc.org/sqlite`: Pure Go SQLite driver (no CGO)
- `github.com/jmoiron/sqlx`: Lightweight SQL extensions

## License

MIT
