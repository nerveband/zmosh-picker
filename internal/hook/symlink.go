package hook

import (
	"fmt"
	"os"
	"path/filepath"
)

const symlinkPath = "/usr/local/bin/zp"

// InstallSymlink creates a symlink from /usr/local/bin/zp to ~/.local/bin/zp.
// Prints a sudo hint if permissions deny it.
func InstallSymlink() {
	home, _ := os.UserHomeDir()
	target := filepath.Join(home, ".local", "bin", "zp")

	// Check target exists
	if _, err := os.Stat(target); os.IsNotExist(err) {
		return
	}

	// Remove old symlink if it exists (may point elsewhere)
	os.Remove(symlinkPath)

	if err := os.Symlink(target, symlinkPath); err != nil {
		fmt.Printf("  note: run 'sudo ln -sf %s %s' for system-wide PATH\n", target, symlinkPath)
		return
	}
	fmt.Printf("  symlinked %s -> %s\n", symlinkPath, target)
}

// CheckSymlink prints a note if the symlink doesn't exist.
// Called by `zp upgrade` — never auto-creates.
func CheckSymlink() {
	if _, err := os.Lstat(symlinkPath); os.IsNotExist(err) {
		fmt.Println("  note: /usr/local/bin/zp not found — run 'zp install-hook' to add it")
	}
}
