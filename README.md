# 55h

A compact TUI for browsing and managing **SSH config** entries. Built with **Go** using **tview/tcell**.

**Why the name?** `55h` looks like `ssh` at a glance—short and well-suited to a terminal tool.

55h reads your SSH configuration (including `Include` files), lets you quickly find hosts, and provides common actions like connect, test, and delete—all from a fast terminal UI.

## Features

- Browse SSH `Host` entries from your SSH config and any included files
- Fuzzy search with a detail view for the selected host
- Theme selection (persisted to user config)
- In-app actions:
  - Connect
  - Ping / connection test
  - Delete host entries

## Screenshots

<!-- Screenshot: Main host list -->
<!-- Screenshot: Host detail view -->
<!-- Screenshot: Theme selector -->

## Installation

### Homebrew (macOS)

```bash
brew install dev-minsoo/tap/55h
```

### Build from source

Go **1.21+** is required.

```bash
go build -o 55h .
./55h
```

### Override SSH config path

```bash
SSH_CONFIG=/path/to/config ./55h
```

## Usage

Launch the TUI:

```bash
55h
```

By default, 55h loads `~/.ssh/config` and follows any `Include` directives it finds.

## Keybindings

| Key | Action |
|-----|--------|
| ↑ / ↓ | Navigate host list |
| `:` | Focus search |
| `Esc` | Exit search / close modals |
| `Enter` | Connect (execs system `ssh` with the selected alias) |
| `p` | Test / ping connection |
| `d` | Delete selected host |
| `t` | Open theme selector / save theme |
| `q` | Quit |
| `?` | Help (show key bindings) |

Ping / test runs:

```bash
ssh -o ConnectTimeout=5 -o BatchMode=yes -o StrictHostKeyChecking=accept-new <alias> exit 0
```

## Behavior Notes

- Deleting a host removes the **entire `Host` block** from the source file that provided it.
  - The UI only allows deletion when the source path is known.
- Connecting uses `syscall.Exec` to replace the current process with the system `ssh` binary.
  - Your terminal session becomes the SSH session.

## CLI: `add ssh`

Add new SSH host entries directly from the command line.

### Usage (exact)

```text
55h add ssh user@host [-p port] [-i identity] [-J jump] [-o Key=Value ...] [--name alias]
```

### Flags

- `user@host` or `host`
  - Target for the new entry
- `-p <port>`
  - Port
- `-i <identity>`
  - `IdentityFile` path
- `-J <jump>`
  - `ProxyJump` value
- `-o Key=Value`
  - Additional SSH settings
  - Supported keys (case-insensitive):
    - `forwardagent` (`yes` | `no`)
    - `identitiesonly` (`yes` | `no`)
    - `serveraliveinterval` (integer)
    - `serveralivecountmax` (integer)
  - Unknown `-o` keys are ignored
- `--name <alias>`
  - Explicitly set the `Host` alias
  - When running interactively (TTY) and `--name` is omitted, you will be prompted with a suggested alias
  - When stdin is **not** a TTY, `--name` is required

### Examples

Interactive (prompts for alias if omitted):

```bash
55h add ssh alice@example.com -p 2222 -i ~/.ssh/id_rsa
```

Non-interactive / scripted (provide `--name`):

```bash
55h add ssh example.com --name myhost -p 2200 -J jump.example.org
```

### Notes

- The `add` command appends a `Host` block to the configured SSH config file
  - Parent directories are created if needed
- The command will refuse to add a duplicate alias found anywhere in the loaded config (including included files)
- The CLI usage string and behavior are intentionally minimal and match the program's parsing rules

## Contributing

Contributions, issues, and suggestions are welcome.

If you plan to submit a change:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Open a pull request with a clear description

Please use conventional-style commit prefixes where applicable, for example:

- `feat:` new features
- `fix:` bug fixes
- `docs:` documentation only changes
- `style:` formatting, missing semicolons, etc. (no code change)
- `refactor:` code change that neither fixes a bug nor adds a feature
- `perf:` performance improvements
- `test:` adding or correcting tests
- `build:` changes that affect the build system or external dependencies
- `ci:` changes to CI configuration
- `chore:` maintenance and tooling

## Inspired by

- **k9s** — for proving that great TUIs can make complex configs pleasant to work with.

## License

MIT
