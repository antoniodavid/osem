# osem - OpenCode Session Manager

A fast, terminal-based session manager for OpenCode with SQLite queries and tmux integration.

## Features

- **Fast SQLite Queries**: 140x faster than CLI parsing (0.01s vs 5s)
- **TUI Interface**: Interactive session selector with fuzzy search
- **Tmux Integration**: Creates windows/sessions with proper naming
- **Session Management**: List, open, delete, backup sessions
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
osem -l

# Show stats
osem -stats

# Open session by ID
osem -s ses_2da73dd

# Backup session
osem -backup ses_2da73dd

# Show version
osem -v
```

## TUI Keybindings

| Key | Action |
|-----|--------|
| `Enter` / `s` | Select and open session |
| `d` | Delete session |
| `f` | Toggle bookmark |
| `/` | Search/filter |
| `q` / `Esc` | Quit |

## Tmux Integration

When you open a session:

- **Inside tmux**: Creates a new window named `opencode-<session-id>`
- **Outside tmux**: Creates a new tmux session named `opencode-<session-id>`

OpenCode automatically changes to the session's stored directory.

## Commands

| Command | Description |
|---------|-------------|
| `osem` | Interactive TUI |
| `osem -l` | List all sessions |
| `osem -stats` | Show statistics |
| `osem -prune` | List sessions with default titles |
| `osem -s <id>` | Open session |
| `osem -backup <id>` | Export session to JSON |
| `osem -v` | Show version |

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

## Performance

| Operation | CLI Parsing | SQLite Direct |
|-----------|-------------|----------------|
| List 500 sessions | ~5.2s | **0.01s** |
| Search sessions | ~5.0s | **0.02s** |
| Open session | ~5.0s | **0.10s** |

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