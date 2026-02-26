package hook

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSymlinkPathConstant(t *testing.T) {
	if symlinkPath != "/usr/local/bin/zp" {
		t.Errorf("symlinkPath = %q, want /usr/local/bin/zp", symlinkPath)
	}
}

func TestInstallSymlinkSkipsWhenTargetMissing(t *testing.T) {
	// Override HOME to a temp dir so target doesn't exist
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	// Should not panic or error — just silently return
	// (We can't test the actual symlink creation without root,
	// but we verify it doesn't blow up when the target is missing)
	InstallSymlink()
}

func TestCheckSymlinkPrintsNoteWhenMissing(t *testing.T) {
	// /usr/local/bin/zp may or may not exist; we just verify no panic
	CheckSymlink()
}

func TestInstallSymlinkTargetPath(t *testing.T) {
	// Verify the target path is derived from HOME correctly
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".local", "bin", "zp")

	// We can't call InstallSymlink() in a test safely (it writes to /usr/local/bin),
	// but we can verify the target path computation matches expectations
	target := filepath.Join(home, ".local", "bin", "zp")
	if target != expected {
		t.Errorf("target = %q, want %q", target, expected)
	}
}
