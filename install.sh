#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
INSTALL_DIR="${HOME}/.local/bin"

# ─── Dependency checks ───────────────────────────────────────────────

echo ""
echo "Checking dependencies..."
echo ""

missing=0

if command -v zmosh &>/dev/null; then
  echo "  [ok] zmosh $(zmosh version 2>/dev/null | head -1 | awk '{print $2}')"
else
  echo "  [!!] zmosh not found (REQUIRED)"
  echo "       Install: https://github.com/mmonad/zmosh"
  echo "       brew install mmonad/tap/zmosh"
  missing=1
fi

if command -v zoxide &>/dev/null; then
  echo "  [ok] zoxide $(zoxide --version 2>/dev/null | awk '{print $2}')"
else
  echo "  [--] zoxide not found (optional, for 'z' directory picking)"
  echo "       Install: https://github.com/ajeetdsouza/zoxide"
  echo "       brew install zoxide"
fi

if command -v fzf &>/dev/null; then
  echo "  [ok] fzf $(fzf --version 2>/dev/null | awk '{print $1}')"
else
  echo "  [--] fzf not found (optional, used by zoxide interactive picker)"
  echo "       Install: https://github.com/junegunn/fzf"
  echo "       brew install fzf"
fi

echo ""

if [[ "$missing" -eq 1 ]]; then
  echo "ERROR: Required dependencies missing. Install them first."
  exit 1
fi

# ─── Install script ──────────────────────────────────────────────────

mkdir -p "$INSTALL_DIR"
cp "$SCRIPT_DIR/zmosh-picker" "$INSTALL_DIR/zmosh-picker"
chmod +x "$INSTALL_DIR/zmosh-picker"
echo "Installed zmosh-picker to $INSTALL_DIR/zmosh-picker"

# ─── Add .zshrc hook ─────────────────────────────────────────────────

if ! grep -qF 'zmosh-picker' ~/.zshrc 2>/dev/null; then
  # Must run before p10k instant prompt (needs console I/O)
  # Find p10k instant prompt line and insert before it
  if grep -qF 'p10k-instant-prompt' ~/.zshrc 2>/dev/null; then
    p10k_line=$(grep -n 'Enable Powerlevel10k instant prompt' ~/.zshrc | head -1 | cut -d: -f1)
    if [[ -n "$p10k_line" ]]; then
      # Insert 2 lines before the p10k comment
      sed -i '' "${p10k_line}i\\
# zmosh-picker: must run before p10k instant prompt (needs console I/O)\\
[[ -z \"\$ZMX_SESSION\" ]] \\&\\& [[ -f \"\$HOME/.local/bin/zmosh-picker\" ]] \\&\\& source \"\$HOME/.local/bin/zmosh-picker\"
" ~/.zshrc
      echo "Added hook before p10k instant prompt in ~/.zshrc"
    fi
  else
    # No p10k — add to top of .zshrc
    local_tmp=$(mktemp)
    {
      echo '# zmosh-picker: auto-launch session picker'
      echo '[[ -z "$ZMX_SESSION" ]] && [[ -f "$HOME/.local/bin/zmosh-picker" ]] && source "$HOME/.local/bin/zmosh-picker"'
      echo ''
      cat ~/.zshrc
    } > "$local_tmp"
    mv "$local_tmp" ~/.zshrc
    echo "Added hook to top of ~/.zshrc"
  fi
else
  echo "Hook already present in ~/.zshrc"
fi

echo ""
echo "Done! Open a new terminal to try it."
echo ""
