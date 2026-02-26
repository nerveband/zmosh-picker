# Configurable Key Mode (Letters-First vs Numbers-First)

## Goal

Allow users to switch session picker keys between numbers-first (1-9, a-y) and letters-first (a-y, 1-9). Letters-first is useful on iOS where the default keyboard shows letters, avoiding an extra tap to switch to numbers.

## Approach

Swap the `keyChars` string at load time (Approach A). One config value, one string swap. Everything downstream (`KeyForIndex`, `IndexForKey`, help screen, picker display) works unchanged.

## Key sequences

- **Numbers-first (default):** `123456789abdefghijlmnopqrstuvwxy`
- **Letters-first:** `abdefghijlmnopqrstuvwxy123456789`

Both skip `c` (custom) and `k` (kill) — reserved actions.

## Config

- File: `~/.config/zpick/keys`
- Values: `numbers` (default) or `letters`
- Read once at startup via `LoadKeyMode()`
- Written by help screen toggle

## Help screen

- New line in config section: `keys: letters (l to toggle)` or `keys: numbers (l to toggle)`
- `l` key cycles the mode, saves to config, re-renders immediately
- Keys section header updates to show current mapping (`a-y,1-9` vs `1-9,a-y`)

## What doesn't change

- `KeyForIndex` / `IndexForKey` signatures
- Picker rendering logic (just reads from `keyChars`)
- SVG animation (illustrative, not literal)
- Reserved keys `c`, `k`, `h`, `z`, `d`, `Esc`, `Enter`
