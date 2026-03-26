# osem - OpenCode Session Manager

A fast, terminal-based session manager for OpenCode with SQLite queries and tmux integration.

## Features

- **Fast SQLite Queries**: 520x faster than CLI parsing (0.01s vs 5s)
- **TUI Interface**: Interactive session selector with fuzzy search and date-based grouping
- **Tmux Integration**: Creates windows/sessions with proper naming
- **Session Management**: List, open, delete, backup sessions
- **Bookmarks**: Mark favorite sessions for quick access
- **Date Filters**: Quick access to today's, yesterday's, or recent sessions
- **Pagination**: Navigate large session lists efficiently
- **Statistics**: View session counts, messages, directories

## Installation

```bash
cd ~/Projects/osem
make install
```

Or build manually:

```bash
go build -o ~/.local/bin/osem ./cmd/osem
```

## Usage

```bash
# Interactive TUI (default)
osem

# List sessions (non-interactive)
osem --list          # or -l

# Date-filtered lists
osem --today         # Today's sessions
osem --yesterday     # Yesterday's sessions
osem --week          # Last 7 days
osem --month         # Last 30 days

# Paginated list
osem --list --page 2 --page-size 20

# Search sessions by title
osem --grep "odoo"

# Show stats
osem --stats

# Session info
osem --info ses_2da73dd

# Open session by ID
osem --session ses_2da73dd    # or -s

# Bookmark management
osem --bookmark ses_2da73dd    # Add bookmark
osem --unbookmark ses_2da73dd  # Remove bookmark
osem --bookmarks               # List all bookmarks

# Backup session
osem --backup ses_2da73dd

# Show version
osem --version       # or -v
```

## TUI Keybindings

| Key | Action |
|-----|--------|
| `Enter` / `s` | Select and open session |
| `d` | Delete session |
| `f` | Toggle bookmark |
| `b` | Toggle bookmarks-only mode |
| `n` / `→` | Next group |
| `p` / `←` | Previous group |
| `/` | Search/filter |
| `q` / `Esc` | Quit |

Sessions are grouped by date: ★ Bookmarks → Today → Yesterday → This Week → This Month → Older

## Tmux Integration

When you open a session:

- **Inside tmux**: Creates a new window named `opencode-<session-id>`
- **Outside tmux**: Creates a new tmux session named `opencode-<session-id>`

OpenCode automatically changes to the session's stored directory.

## Commands

| Command | Description |
|---------|-------------|
| `osem` | Interactive TUI with date grouping |
| `osem --list`, `-l` | List all sessions |
| `osem --today` | List today's sessions |
| `osem --yesterday` | List yesterday's sessions |
| `osem --week` | List last 7 days |
| `osem --month` | List last 30 days |
| `osem --grep <query>` | Search sessions by title |
| `osem --stats` | Show statistics |
| `osem --info <id>` | Show session details |
| `osem --session <id>`, `-s` | Open session |
| `osem --bookmark <id>` | Add to bookmarks |
| `osem --unbookmark <id>` | Remove from bookmarks |
| `osem --bookmarks` | List bookmarks |
| `osem --prune` | List sessions with default titles |
| `osem --backup <id>` | Export session to JSON |
| `osem --version`, `-v` | Show version |

## Configuration

- Bookmarks: `~/.config/osem/bookmarks.json`
- Backups: `~/.config/osem/backups/`

## Project Structure

```
osem/
├── cmd/osem/main.go       # CLI entry point
├── internal/
│   ├── db/sqlite.go       # SQLite queries
│   ├── models/            # Data models
│   ├── tui/               # Bubbletea TUI
│   ├── tmux/manager.go   # Tmux integration
│   └── bookmarks/         # Favorites management
├── go.mod
├── Makefile
└── README.md
```

## Dependencies

- [bubbletea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [bubbles](https://github.com/charmbracelet/bubbles) - TUI components
- [lipgloss](https://github.com/charmbracelet/lipgloss) - Styling
- [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) - Pure Go SQLite
- [pflag](https://github.com/spf13/pflag) - POSIX/GNU-style flag parsing

## Performance

| Operation | CLI Parsing | SQLite Direct |
|-----------|-------------|----------------|
| List 495 sessions | ~5.2s | **0.01s** |
| Search sessions | ~5.0s | **0.02s** |
| Open session | ~5.0s | **0.10s** |

**520x faster** than parsing CLI output.

## License

MIT License - see [LICENSE](LICENSE)

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run `make test`
5. Submit a pull request

## Author

Created for managing OpenCode sessions efficiently with tmux integration.