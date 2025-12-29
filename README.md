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

### Homebrew (macOS only)

```bash
brew install bunchhieng/tools/rl
```

### Linux

Download the binary from [GitHub Releases](https://github.com/bunchhieng/rl/releases) and place it in your PATH, or build from source.

## Database Location

The database is stored in the platform-appropriate config directory:
- **macOS**: `~/Library/Application Support/rl/links.db`
- **Linux**: `~/.config/rl/links.db`
- **Windows**: `%AppData%/rl/links.db`

This follows platform conventions and is easy to backup. You can override the path with `--db-path`.

## Usage

### Add a link

```bash
rl add https://example.com --title "Example Site" --tags "web,example"
rl add https://another.com --note "Check this later"
```

### List links

```bash
rl list                    # List unread links (default)
rl list --unread           # List unread links
rl list --read             # List read links
rl list --all              # List all links
rl list --tag web          # Filter by tag
rl list --limit 10         # Limit results
```

### Open a link

```bash
rl open 1                  # Opens link #1 in default browser
```

Note: Opening a link does NOT mark it as read automatically.

### Mark as read/unread

```bash
rl done 1                  # Mark link #1 as read
rl undo 1                  # Mark link #1 as unread
```

### Delete a link

```bash
rl rm 1                    # Delete link #1
```

### Export/Import

```bash
rl export > links.json     # Export all links to JSON
rl import links.json       # Import links from JSON
```

Import handles duplicates intelligently:
- Preserves the earliest `created_at` timestamp
- Updates missing fields (title, note) if empty
- Merges tags (deduplicated)

### Search

```bash
rl search "example"        # Full-text search
```

Searches across URL, title, note, and tags.

### Version

```bash
rl version                 # Show version
```

## Examples

```bash
# Add a few links
rl add https://golang.org --title "Go Language" --tags "programming,go"
rl add https://rust-lang.org --title "Rust" --tags "programming,rust"

# List unread links
rl list

# Open and read a link
rl open 1
rl done 1

# Search for programming links
rl search "programming"

# Export for backup
rl export > backup.json

# Import from backup
rl import backup.json
```

## Development

### Build

```bash
make build
# or
go build -o bin/rl ./cmd/rl
```

### Test

```bash
make test
# or
go test ./...
```

### Install locally

```bash
make install
# or
go install ./cmd/rl
```

### Pre-commit Hook

Install the pre-commit hook to automatically run `gofmt`, `go vet`, and tests before each commit:

```bash
make pre-commit
```

The hook will:
- Check that all staged Go files are formatted with `gofmt`
- Run `go vet` to check for common errors
- Run all tests

To skip the hook for a specific commit, use `git commit --no-verify`.

## Release Process

### Setup (One-time)

1. **Create GitHub repositories:**
   - Main repo: `github.com/bunchhieng/rl` (this repo)
   - Homebrew tap: `github.com/bunchhieng/homebrew-tools` (separate repo)

2. **Configure GoReleaser:**
   - Set `GITHUB_TOKEN` environment variable with a GitHub personal access token
   - The token needs `repo` scope for creating releases
   - Update `.goreleaser.yaml` with your GitHub username/org if different

3. **Install GoReleaser:**
   ```bash
   brew install goreleaser
   # or
   go install github.com/goreleaser/goreleaser@latest
   ```

### Release Steps

1. **Tag a release:**
   ```bash
   git tag v1.0.0
   git push --tags
   ```

2. **Run GoReleaser:**
   ```bash
   export GITHUB_TOKEN=your_token_here
   goreleaser release --clean
   ```

   This will:
   - Build binaries for darwin (amd64, arm64) and linux (amd64, arm64)
   - Create a GitHub release with binaries and checksums
   - Automatically update the Homebrew tap formula in `bunchhieng/homebrew-tools`

3. **Users install:**
   - **macOS**: `brew install bunchhieng/tools/rl`
   - **Linux**: Download from GitHub Releases

### Testing releases locally

```bash
goreleaser release --snapshot --clean
```

This creates a snapshot release without creating a GitHub release, useful for testing.

## JSON Export Format

The export format is a JSON array of link objects:

```json
[
  {
    "id": 1,
    "url": "https://example.com",
    "title": "Example Site",
    "note": "Optional note",
    "tags": "tag1,tag2",
    "created_at": "2024-01-01T12:00:00Z",
    "read_at": "2024-01-02T10:30:00Z"
  }
]
```

- `id`: Integer ID (may change on import)
- `url`: Required URL
- `title`: Optional title
- `note`: Optional note
- `tags`: Comma-separated tags
- `created_at`: ISO 8601 timestamp
- `read_at`: ISO 8601 timestamp (null if unread)

## Architecture

- **cmd/rl**: Main entry point
- **internal/app**: Application initialization
- **internal/storage**: Storage interface and SQLite implementation
- **internal/model**: Data models and validation
- **internal/cli**: Command handlers
- **migrations**: SQL migration files (embedded)

## Dependencies

- `modernc.org/sqlite`: Pure Go SQLite driver (no CGO)
- `github.com/jmoiron/sqlx`: Lightweight SQL extensions

## License

MIT

