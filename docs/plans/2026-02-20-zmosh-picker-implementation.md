# zmosh-picker Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** A zsh script that auto-launches on every new terminal, presenting a single-keypress menu to attach to existing zmosh sessions or create new ones with smart directory-aware naming.

**Architecture:** Single zsh script (`zmosh-picker`) sourced from `.zshrc`. Parses `zmosh list` output to build a numbered menu. Uses `read -k1` for single-keypress input. Integrates `zoxide query -i` for directory picking. `exec zmosh attach` replaces shell process on selection.

**Tech Stack:** Pure zsh, zmosh CLI, zoxide (optional, for `z` key)

---

### Task 1: Scaffold the zmosh-picker script with guards

**Files:**
- Create: `zmosh-picker`

**Step 1: Write the script skeleton with all guard checks**

```zsh
#!/usr/bin/env zsh
# zmosh-picker — single-keypress session launcher for zmosh
# https://github.com/nerveband/zmosh-picker

# Guards: skip picker if conditions aren't met
[[ ! -o interactive ]] && return 0 2>/dev/null || exit 0
[[ -n "$ZMX_SESSION" ]] && return 0 2>/dev/null || exit 0
[[ ! -t 0 ]] && return 0 2>/dev/null || exit 0
command -v zmosh &>/dev/null || return 0 2>/dev/null || exit 0
```

**Step 2: Test guards manually**

Run: `zsh -c 'source ./zmosh-picker && echo "should not print"'`
Expected: No output (non-interactive guard catches it)

Run: `ZMX_SESSION=test zsh -i -c 'source ./zmosh-picker && echo "skipped"'`
Expected: Prints "skipped" (already-in-session guard catches it)

**Step 3: Commit**

```bash
git add zmosh-picker
chmod +x zmosh-picker
git commit -m "feat: scaffold zmosh-picker with startup guards"
```

---

### Task 2: Parse zmosh list output into arrays

**Files:**
- Modify: `zmosh-picker`

**Step 1: Add session parsing logic after the guards**

`zmosh list` outputs tab-separated fields per line:
```
session_name=bbcli\tpid=51217\tclients=1\tcreated_at=...\ttask_ended_at=0\ttask_exit_code=0\tstarted_in=/path/to/dir
```

Add this parsing block:

```zsh
# Parse active sessions
typeset -a session_names session_clients session_dirs
local line
while IFS= read -r line; do
  [[ -z "$line" ]] && continue
  local name="" clients="" dir=""
  local field
  for field in ${(s:	:)line}; do
    case "$field" in
      session_name=*) name="${field#session_name=}" ;;
      clients=*) clients="${field#clients=}" ;;
      started_in=*) dir="${field#started_in=}" ;;
    esac
  done
  [[ -n "$name" ]] || continue
  session_names+=("$name")
  session_clients+=("$clients")
  # Shorten dir: replace $HOME with ~
  dir="${dir/$HOME/~}"
  # Truncate to last 3 path segments if long
  local parts=(${(s:/:)dir})
  if (( ${#parts} > 4 )); then
    dir="~/${parts[-3]}/${parts[-2]}/${parts[-1]}"
  fi
  session_dirs+=("$dir")
done < <(zmosh list 2>/dev/null)

local session_count=${#session_names}
```

**Step 2: Test parsing by adding temporary debug output**

Add temporarily at the end:
```zsh
echo "Found $session_count sessions"
for i in {1..$session_count}; do
  echo "  $i) ${session_names[$i]} (${session_clients[$i]} clients) ${session_dirs[$i]}"
done
```

Run: `zsh -i -c 'source ./zmosh-picker'` (must have at least one zmosh session active)
Expected: Lists active sessions with names, client counts, and truncated dirs

**Step 3: Remove debug output and commit**

Remove the temporary echo lines added in Step 2.

```bash
git add zmosh-picker
git commit -m "feat: parse zmosh list into session arrays"
```

---

### Task 3: Build the session name generator

**Files:**
- Modify: `zmosh-picker`

**Step 1: Add name generation functions**

```zsh
# Generate session name with counter: <dirname>-<N>
_zmosh_pick_name_counter() {
  local base_dir="${1:-$PWD}"
  local dirname="${base_dir:t}"  # zsh :t = basename
  local existing_names
  existing_names=($(zmosh list --short 2>/dev/null))
  local n=1
  while (( 1 )); do
    local candidate="${dirname}-${n}"
    # Check if candidate exists in current sessions
    if (( ! ${existing_names[(Ie)$candidate]} )); then
      echo "$candidate"
      return
    fi
    (( n++ ))
  done
}

# Generate session name with date: <dirname>-MMDD
_zmosh_pick_name_date() {
  local base_dir="${1:-$PWD}"
  local dirname="${base_dir:t}"
  echo "${dirname}-$(date +%m%d)"
}
```

**Step 2: Test name generation**

Add temporarily at end:
```zsh
echo "Counter name: $(_zmosh_pick_name_counter)"
echo "Date name: $(_zmosh_pick_name_date)"
```

Run: `zsh -i -c 'source ./zmosh-picker'`
Expected: Something like `nerveband-1` and `nerveband-0220`

Run from a project dir: `cd ~/Documents/GitHub/ai-happy-design && zsh -i -c 'source /path/to/zmosh-picker'`
Expected: `ai-happy-design-1` and `ai-happy-design-0220`

**Step 3: Remove debug output and commit**

```bash
git add zmosh-picker
git commit -m "feat: add counter and date session name generators"
```

---

### Task 4: Build the display and key-index mapping

**Files:**
- Modify: `zmosh-picker`

**Step 1: Add display rendering and key map**

```zsh
# Build key-to-session mapping
# 1-9 for first 9, then a-y for rest (skip z — reserved for zoxide)
typeset -A key_to_session
local keys_display=()
local key_chars="123456789abcdefghijklmnopqrstuvwxy"
local i
for (( i=1; i<=session_count && i<=${#key_chars}; i++ )); do
  local key="${key_chars[$i]}"
  key_to_session[$key]="${session_names[$i]}"
  keys_display+=("$key")
done

# Display
local default_name
default_name="$(_zmosh_pick_name_counter)"

if (( session_count > 0 )); then
  echo ""
  echo "  zmosh: $session_count active session$( (( session_count > 1 )) && echo s)"
  for (( i=1; i<=session_count; i++ )); do
    local client_label="${session_clients[$i]} client$( (( session_clients[$i] != 1 )) && echo s)"
    printf "    %s) %-20s (%s)  %s\n" "${keys_display[$i]}" "${session_names[$i]}" "$client_label" "${session_dirs[$i]}"
  done
  echo ""
  local range="${keys_display[1]}"
  (( session_count > 1 )) && range="${keys_display[1]}-${keys_display[-1]}"
  echo "  [$range] attach  [Enter] new: $default_name  [z] pick dir  [d] +date  [Esc] skip"
else
  echo ""
  echo "  zmosh: no active sessions"
  echo "  [Enter] new: $default_name  [z] pick dir  [d] +date  [Esc] skip"
fi
```

**Step 2: Test display**

Run: `zsh -i -c 'source ./zmosh-picker'`
Expected: Formatted menu with session list (if any) and action keys at bottom

**Step 3: Commit**

```bash
git add zmosh-picker
git commit -m "feat: render session picker menu with key map"
```

---

### Task 5: Implement single-keypress input handling

**Files:**
- Modify: `zmosh-picker`

**Step 1: Add input handler after the display block**

```zsh
# Read single keypress
local choice
read -k1 choice 2>/dev/null
echo "" # newline after keypress

# Handle the keypress
case "$choice" in
  $'\e')  # Esc — plain shell
    return 0 2>/dev/null || exit 0
    ;;
  $'\n')  # Enter — new session in $PWD
    echo "  → $default_name"
    exec zmosh attach "$default_name"
    ;;
  d)  # Date-suffixed new session in $PWD
    local date_name
    date_name="$(_zmosh_pick_name_date)"
    echo "  → $date_name"
    exec zmosh attach "$date_name"
    ;;
  z)  # Zoxide pick dir, then new session
    if command -v zoxide &>/dev/null; then
      local picked_dir
      picked_dir="$(zoxide query -i 2>/dev/null)"
      if [[ -n "$picked_dir" ]]; then
        cd "$picked_dir" || true
        local zname
        zname="$(_zmosh_pick_name_counter "$picked_dir")"
        echo "  → $zname (in $picked_dir)"
        exec zmosh attach "$zname"
      else
        # User cancelled zoxide picker — fall through to plain shell
        return 0 2>/dev/null || exit 0
      fi
    else
      echo "  zoxide not installed — skipping"
      return 0 2>/dev/null || exit 0
    fi
    ;;
  *)  # Check if it's a session key
    if [[ -n "${key_to_session[$choice]}" ]]; then
      local target="${key_to_session[$choice]}"
      echo "  → $target"
      exec zmosh attach "$target"
    else
      # Unknown key — ignore, drop to plain shell
      return 0 2>/dev/null || exit 0
    fi
    ;;
esac
```

**Step 2: Test each key path manually**

Test Esc: Open interactive zsh, source the script, press Esc. Expected: drops to normal shell.

Test Enter: Source script, press Enter. Expected: creates and attaches to new session named `<dirname>-1`.

Test number key: Source script with active sessions, press `1`. Expected: attaches to first listed session.

Test `z`: Source script, press `z`. Expected: zoxide fzf picker appears.

Test `d`: Source script, press `d`. Expected: creates session with date suffix.

**Step 3: Commit**

```bash
git add zmosh-picker
git commit -m "feat: single-keypress input handling with all key actions"
```

---

### Task 6: Handle the zoxide `cd` for new sessions started via `z`

**Files:**
- Modify: `zmosh-picker`

**Step 1: Ensure the zmosh session starts in the picked directory**

The `cd "$picked_dir"` before `exec zmosh attach` ensures the new session's shell starts in that directory. However, we need to verify zmosh inherits `$PWD` when creating a new session.

Test: Press `z`, pick a directory, verify the resulting session's working directory matches.

Run: `zmosh list` after creating a session via `z` — check the `started_in` field.

If zmosh doesn't inherit `$PWD`, we need to pass the directory as part of the command:
```zsh
exec zmosh attach "$zname" zsh -c "cd '$picked_dir' && exec zsh"
```

**Step 2: Test and verify**

Create a session via `z`, pick a project dir. Inside the session, run `pwd`.
Expected: The picked directory, not `~`.

**Step 3: Commit if any changes were needed**

```bash
git add zmosh-picker
git commit -m "fix: ensure zoxide-picked dir is inherited by new session"
```

---

### Task 7: Write install.sh and uninstall.sh

**Files:**
- Create: `install.sh`
- Create: `uninstall.sh`

**Step 1: Write install.sh**

```bash
#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
INSTALL_DIR="${HOME}/.local/bin"
HOOK='[[ -z "$ZMX_SESSION" ]] && command -v zmosh-picker &>/dev/null && zmosh-picker'

# Install script
mkdir -p "$INSTALL_DIR"
cp "$SCRIPT_DIR/zmosh-picker" "$INSTALL_DIR/zmosh-picker"
chmod +x "$INSTALL_DIR/zmosh-picker"
echo "Installed zmosh-picker to $INSTALL_DIR/zmosh-picker"

# Add .zshrc hook if not present
if ! grep -qF 'zmosh-picker' ~/.zshrc 2>/dev/null; then
  echo "" >> ~/.zshrc
  echo "# zmosh-picker: auto-launch session picker" >> ~/.zshrc
  echo "$HOOK" >> ~/.zshrc
  echo "Added hook to ~/.zshrc"
else
  echo "Hook already present in ~/.zshrc"
fi

echo "Done. Open a new terminal to try it."
```

**Step 2: Write uninstall.sh**

```bash
#!/usr/bin/env bash
set -euo pipefail

INSTALL_DIR="${HOME}/.local/bin"

# Remove script
if [[ -f "$INSTALL_DIR/zmosh-picker" ]]; then
  rm "$INSTALL_DIR/zmosh-picker"
  echo "Removed $INSTALL_DIR/zmosh-picker"
fi

# Remove .zshrc hook
if [[ -f ~/.zshrc ]]; then
  # Remove the comment line and the hook line
  sed -i '' '/# zmosh-picker: auto-launch session picker/d' ~/.zshrc
  sed -i '' '/zmosh-picker/d' ~/.zshrc
  echo "Removed hook from ~/.zshrc"
fi

echo "Done. zmosh-picker has been uninstalled."
```

**Step 3: Test install**

```bash
chmod +x install.sh uninstall.sh
./install.sh
```

Expected: Script copied, hook added to `.zshrc`.

Verify: `cat ~/.local/bin/zmosh-picker` exists and is executable. `grep zmosh-picker ~/.zshrc` shows the hook.

**Step 4: Test uninstall**

```bash
./uninstall.sh
```

Expected: Script removed, hook removed from `.zshrc`.

**Step 5: Re-install and commit**

```bash
./install.sh
git add install.sh uninstall.sh
git commit -m "feat: add install and uninstall scripts"
```

---

### Task 8: Write README.md

**Files:**
- Create: `README.md`

**Step 1: Write the README**

```markdown
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
```

**Step 2: Commit**

```bash
git add README.md
git commit -m "docs: add README with usage and design philosophy"
```

---

### Task 9: Add LICENSE file

**Files:**
- Create: `LICENSE`

**Step 1: Create MIT license**

Use current year (2026) and "Ashraf Ali" as the copyright holder (matches the user's Dropbox path).

**Step 2: Commit**

```bash
git add LICENSE
git commit -m "docs: add MIT license"
```

---

### Task 10: End-to-end test and polish

**Files:**
- Possibly modify: `zmosh-picker`

**Step 1: Fresh install test**

```bash
./uninstall.sh
./install.sh
```

Open a brand new Ghostty tab. Expected: picker appears.

**Step 2: Test all key paths**

- Press `Esc` → plain shell, no session
- Open new tab, press `Enter` → new session created with counter name
- Open new tab, press `1` → attaches to the session just created
- Open new tab, press `z` → zoxide picker, select dir, session created
- Open new tab, press `d` → date-suffixed session created

**Step 3: Test SSH scenario**

```bash
ssh localhost  # or another host with zmosh-picker installed
```

Expected: Picker shows remote sessions. `z` opens zoxide with remote paths.

**Step 4: Verify guard — no nesting**

Inside a zmosh session, open a new tab (which starts a new shell).
Expected: If `$ZMX_SESSION` is set in the new shell, picker is skipped. If Ghostty tabs don't inherit the env, picker shows normally (which is correct — it's a separate terminal instance).

**Step 5: Fix any issues found, commit**

```bash
git add -A
git commit -m "fix: polish from end-to-end testing"
```

---

### Task 11: Create GitHub repo and push

**Step 1: Create public repo**

```bash
cd /Users/nerveband/Documents/GitHub/zmosh-picker
gh repo create nerveband/zmosh-picker --public --source=. --push
```

**Step 2: Verify**

```bash
gh repo view nerveband/zmosh-picker --web
```

Expected: Public repo with README, script, install/uninstall, design doc, and LICENSE.
