# zmosh-picker

Single-keypress session launcher for [zmosh](https://github.com/mmonad/zmosh).

Every terminal you open shows your active sessions. Press a number to resume, Enter to start fresh, or `z` to jump to any project directory. One keypress, every time.

## Why

Persistent sessions should be the default, not something you opt into. Whether you're on your Mac, SSH'd from an iPad, or mosh'd from your phone, you should be able to pick up any session instantly.

The goal: never think about session management. Open a terminal, see your sessions, press one key.

## Install

Requires [zmosh](https://github.com/mmonad/zmosh). Optional: [zoxide](https://github.com/ajeetdsouza/zoxide) (for `z` directory picking).

```bash
git clone https://github.com/nerveband/zmosh-picker.git
cd zmosh-picker
./install.sh
```

This copies `zmosh-picker` to `~/.local/bin/` and adds a one-line hook to your `.zshrc`.

## Usage

Open a new terminal. You'll see:

```
  zmosh: 2 active sessions
    1) ai-happy-design  (1 client)   ~/Doc/GH/ai-happy-design
    2) bbcli             (0 clients)  ~/Doc/GH/agent-to-bricks

  [1-2] attach  [Enter] new: nerveband-3  [z] pick dir  [d] +date  [Esc] skip
```

### Keys

| Key | Action |
|-----|--------|
| `1-9` | Attach to listed session |
| `a-y` | Attach to sessions 10+ |
| `Enter` | New session in current directory |
| `z` | Pick directory with zoxide, then new session |
| `d` | New session with date suffix (MMDD) |
| `Esc` | Plain shell, no zmosh |

### Session naming

- **Enter**: `<dirname>-<N>` — auto-incremented counter (e.g., `ai-happy-design-1`)
- **d**: `<dirname>-MMDD` — date suffix (e.g., `ai-happy-design-0220`)
- **z**: Same as Enter, but in the zoxide-picked directory

## Design decisions

See [docs/plans/2026-02-20-zmosh-picker-design.md](docs/plans/2026-02-20-zmosh-picker-design.md) for the full design rationale.

## Uninstall

```bash
./uninstall.sh
```

## License

MIT
