# 55h

SSH config manager with a k9s-inspired TUI, built in Go with tview/tcell.

## Features (WIP)
- Browse SSH host entries
- Fuzzy search for hosts
- Detail view for selected host
- Theme switching (dark/light/neutral)

## Requirements
- Go 1.21+

## Run
```bash
go run .
```

## Controls
- `q` quit
- `Enter` connect
- `:` focus search
- `t` cycle theme
- `↑/↓` navigate
- `?` help

## Config
By default, reads `~/.ssh/config`.

You can override the path:
```bash
SSH_CONFIG=/path/to/config go run .
```
