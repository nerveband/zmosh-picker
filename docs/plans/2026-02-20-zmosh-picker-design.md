# zmosh-picker Design

## Problem

Persistent sessions should be the default, not something you opt into. When you open a terminal — whether on your Mac, SSH'd from an iPad, or mosh'd from your phone — you should immediately see your active sessions and resume one with a single keypress. Creating a new session in any project directory should be just as fast.

Without this, you end up with orphaned shells, lost context, and the friction of manually running `zmosh attach <name>` every time. That friction compounds across devices.

## Design Decisions

### Single-keypress UX

Every action in the picker is one keypress. No typing session names, no arrow-key navigation, no confirmation prompts. This matters most on mobile (Blink Shell, Termius) where typing is painful and every saved keystroke counts.

### Directory-aware naming via zoxide

SSH always starts in `~`, making `$PWD`-based names useless for remote sessions. Integrating zoxide's interactive picker (`zi`) lets you jump to any frequently-used project directory before creating a session, and the session inherits that directory's name. On a Mac, this is equally useful — you're often in `~` when a new terminal opens.

### No window management

zmosh (and zmx) deliberately avoid window management, deferring to the OS. This picker follows the same philosophy: it's a session launcher, not a multiplexer UI. Once you're in a session, your terminal app handles splits/tabs.

### exec replaces the shell

The picker uses `exec zmosh attach <name>` so there's no extra shell process wrapping the session. When you detach or the session ends, the terminal closes cleanly.

## Startup Flow

```
Terminal opens -> .zshrc -> zmosh-picker

  zmosh: 2 active sessions
    1) ai-happy-design  (1 client)  ~/Doc/GH/ai-happy-design
    2) bbcli             (0 clients) ~/Doc/GH/agent-to-bricks

  [1-2] attach
  [Enter] new here: nerveband-3
  [z] pick dir first   [d] new +date
  [Esc] plain shell
```

## Key Map

All keys are single-press (no Enter needed, except Enter itself).

| Key | Action | Session name |
|-----|--------|-------------|
| `1-9` | Attach to listed session | (existing) |
| `a-y` | Attach to sessions 10+ (skipping used keys) | (existing) |
| `Enter` | New session in `$PWD` | `<dirname>-<N>` |
| `z` | `zi` to pick dir, then new session there | `<picked-dirname>-<N>` |
| `d` | New session in `$PWD` with date suffix | `<dirname>-MMDD` |
| `Esc` | Drop to plain shell, no zmosh | (none) |

## Session Naming

- **Counter format** (`<dirname>-<N>`): Default. `N` starts at 1 and increments past existing sessions with the same prefix.
- **Date format** (`<dirname>-MMDD`): Triggered by `d` key. Compact month-day, no year — sessions are ephemeral.
- `<dirname>` is `basename` of the working directory (either `$PWD` or the zoxide-picked path).

## Guards (skip picker entirely)

- `$ZMX_SESSION` already set (already inside zmosh)
- Non-interactive shell
- `zmosh` not in `$PATH`
- stdin is not a terminal (`! [ -t 0 ]`)

## Display Format

Each session line shows:
- Index key (`1-9`, then `a-y`)
- Session name
- Client count (indicates active use on another device)
- Starting directory (truncated)

When no sessions exist, the list is omitted and only the new-session options appear.

## Components

### `zmosh-picker` (zsh script)

The main script, installed to `~/.local/bin/zmosh-picker`. Pure zsh, no dependencies beyond zmosh and zoxide (for `z` key only).

### `.zshrc` hook (one line)

```zsh
[[ -z "$ZMX_SESSION" ]] && command -v zmosh-picker &>/dev/null && zmosh-picker
```

### `install.sh`

Copies `zmosh-picker` to `~/.local/bin/`, appends the hook to `.zshrc` if not already present.

### `uninstall.sh`

Removes the script and the `.zshrc` hook line.

## Repo Structure

```
zmosh-picker/
├── zmosh-picker           # The zsh script
├── install.sh             # Install script + .zshrc hook
├── uninstall.sh           # Clean removal
├── docs/
│   └── plans/
│       └── 2026-02-20-zmosh-picker-design.md
├── README.md              # Philosophy, usage, keybinds
└── LICENSE                # MIT
```
