<p align="center">
  <img src="docs/banner.svg" alt="55h banner" width="100%" />
</p>

<h1 align="center">55h</h1>

<p align="center">
  Terminal-first SSH host manager built with Go and <code>tview/tcell</code>.
</p>

<p align="center">
  <a href="#installation"><img alt="Go" src="https://img.shields.io/badge/Go-1.21%2B-00ADD8?logo=go"></a>
  <a href="LICENSE"><img alt="License" src="https://img.shields.io/badge/License-MIT-green.svg"></a>
  <a href="#installation"><img alt="Platform" src="https://img.shields.io/badge/Platform-macOS%20%7C%20Linux-lightgrey"></a>
  <a href="#installation"><img alt="Homebrew Tap" src="https://img.shields.io/badge/Homebrew-dev--minsoo%2Ftap-orange?logo=homebrew"></a>
</p>

<p align="center">
  <code>brew install dev-minsoo/tap/55h</code>
</p>

`55h` is focused on one job: making SSH host operations faster without leaving the terminal.

## At a Glance

- Parses `~/.ssh/config` and follows every `Include` target recursively
- Keeps browsing, searching, and actions in one TUI flow
- Supports instant actions: connect, test, delete, and create hosts

## Screenshot

![55h UI](docs/image.png)

## Why 55h

- `55h` visually resembles `ssh`, matching the product's purpose
- Host list + detail panel are always visible in one place
- Workflow stays keyboard-centric end-to-end

## Core Features

- Host list + detail panel for SSH entries
- Fuzzy-style search across alias/host/user/options
- In-app actions:
  - Connect (replace process with system `ssh`)
  - Ping/test connection
  - Delete host block in source file
- Persistent theme selection
- CLI for adding entries: `55h add ssh ...`

## Installation

### Homebrew (macOS)

```bash
brew install dev-minsoo/tap/55h
```

### Build from source

Go `1.21+` required.

```bash
go build -o 55h .
./55h
```

### Override SSH config path

```bash
SSH_CONFIG=/path/to/config ./55h
```

## Usage

```bash
55h
```

Default config target: `~/.ssh/config` (with `Include` support).

## Keybindings

| Key | Action |
|-----|--------|
| ↑ / ↓ | Navigate host list |
| `:` | Focus search |
| `Esc` | Exit search / close modals |
| `Enter` | Connect to selected host |
| `p` | Connection test (ping) |
| `d` | Delete selected host block |
| `t` | Open theme selector |
| `q` | Quit |
| `?` | Help modal |

Connection test command:

```bash
ssh -o ConnectTimeout=5 -o BatchMode=yes -o StrictHostKeyChecking=accept-new <alias> exit 0
```

## CLI: `add ssh`

```text
55h add ssh user@host [-p port] [-i identity] [-J jump] [-o Key=Value ...] [--name alias]
```

### Supported flags

- `-p <port>`: `Port`
- `-i <identity>`: `IdentityFile`
- `-J <jump>`: `ProxyJump`
- `-o Key=Value`: extra SSH options
  - `forwardagent` (`yes|no`)
  - `identitiesonly` (`yes|no`)
  - `serveraliveinterval` (int)
  - `serveralivecountmax` (int)
- `--name <alias>`: force host alias

## Contributing

Issues and PRs are welcome.

### Before opening a PR

1. Fork the repository and create a feature branch.
2. Run local checks:

```bash
make fmt
make vet
make test
make build
```

3. Verify the app flow manually:

```bash
go run .
```

4. If behavior or keybindings changed, update docs and screenshots in `docs/`.

### Issue Reports

Please include:

- OS and Go version
- Sanitized sample of related SSH config
- Expected behavior and actual behavior
- Reproduction steps

### Commit Style

Recommended prefixes:

- `feat:`
- `fix:`
- `docs:`
- `refactor:`
- `test:`
- `chore:`

## License

MIT
