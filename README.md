# 55h (v0.1.0)

Compact, k9s-inspired TUI for browsing and managing SSH config entries (built with Go + tview/tcell).

Quick highlights
- Browse SSH Host entries from your SSH config and any included files
- Fuzzy search and detail view for selected host
- Theme selection (saved to user config)
- In-app actions: connect, ping (connection test), delete host

Requirements
- Go 1.21+

Run / Install
- Build: `go build -o 55h .` then `./55h`
- Run directly: `go run .`
- Override SSH config path: `SSH_CONFIG=/path/to/config ./55h` or `SSH_CONFIG=/path/to/config go run .`

Config & data
- Default SSH config path: `~/.ssh/config` (can be overridden via SSH_CONFIG env var)
- Includes: `Include` lines in SSH config are followed and support glob patterns; relative include paths are resolved against the including file's directory
- App config path: `$HOME/.config/55h/config.yml` (also accepts legacy `$HOME/.config/55h/config` and `config.json` when loading)
  - When saving the theme the app writes `config.yml` (format: `# Theme name for UI colors\ntheme: <Name>`) and will attempt to remove legacy `config`/`config.json`
- Access log: `$HOME/.config/55h/access.json` (stores last-access timestamps for entries)

Controls / Keybindings
- Navigation: ↑ / ↓
- `:` focus search
- `Esc` exit search / close modals
- `Enter` connect (execs system `ssh` with the selected alias)
- `p` test/ping (runs `ssh -o ConnectTimeout=5 -o BatchMode=yes -o StrictHostKeyChecking=accept-new <alias> exit 0`)
- `d` delete selected host (removes the Host block from its source file)
- `t` open theme selector / save theme
- `q` quit
- `?` help (shows key bindings)

Behavior notes
- Deleting a host removes the entire Host block from the source file that provided it (the UI only allows deleting when the source path is known).
- Connecting uses `syscall.Exec` to replace the process with the system `ssh` binary (so your terminal session becomes the SSH session).

CLI: add ssh
Usage (exact):
```
55h add ssh user@host [-p port] [-i identity] [-J jump] [-o Key=Value ...] [--name alias]
```

Description / flags
- `user@host` or `host`: target for the new entry
- `-p <port>`: Port
- `-i <identity>`: IdentityFile path
- `-J <jump>`: ProxyJump value
- `-o Key=Value`: additional settings. Supported keys (case-insensitive):
  - `forwardagent` (yes|no)
  - `identitiesonly` (yes|no)
  - `serveraliveinterval` (integer)
  - `serveralivecountmax` (integer)
  Unknown `-o` keys are ignored.
- `--name <alias>`: explicitly set the Host alias. When running interactively (TTY) and `--name` is omitted you will be prompted with a suggested alias; when stdin is not a TTY `--name` is required.

Examples
- Interactive (prompts for alias if omitted):
```
55h add ssh alice@example.com -p 2222 -i ~/.ssh/id_rsa
```
- Non-interactive / scripted (provide --name):
```
55h add ssh example.com --name myhost -p 2200 -J jump.example.org
```

Notes
- The add command appends a Host block to the configured SSH config file (creates parent directories if needed) and will refuse to add a duplicate alias found anywhere in the loaded config (including included files).
- The CLI usage string and behavior are intentionally minimal and match the program's parsing rules.
