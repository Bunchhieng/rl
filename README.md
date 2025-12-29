# rl - Read Later CLI

A minimal, local-first "read later" CLI tool for macOS and Linux. Store links locally with SQLite, no account, no sync, no background daemon.

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

### Add a link
```bash
rl add https://example.com --title "Example" --tags "web,example"
rl add https://another.com --note "Check later"
```

### List links
```bash
rl list                    # Unread (default)
rl list --read             # Read only
rl list --all              # All links
rl list --tag web          # Filter by tag
rl list --limit 10         # Limit results
```

### Open, mark, delete
```bash
rl open <id>               # Open in browser (doesn't mark as read)
rl done <id>               # Mark as read
rl undo <id>               # Mark as unread
rl rm <id> [id...]         # Delete one or more links
```

### Export/Import
```bash
rl export > links.json     # Export all links
rl import links.json       # Import (merges duplicates, preserves timestamps)
```

### Search
```bash
rl search "query"          # Full-text search across URL, title, note, tags
```

## Examples

```bash
rl add https://golang.org --title "Go" --tags "programming,go"
rl list --tag programming
rl open 9m1w2z3x && rl done 9m1w2z3x
rl search "programming"
rl export > backup.json
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
