package zmosh

import (
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestAttachCommand(t *testing.T) {
	cmd := AttachCommand("my-session")
	if cmd != `zmosh attach "my-session"` {
		t.Errorf("unexpected command: %s", cmd)
	}
}

func TestKillCommand(t *testing.T) {
	cmd := KillCommand("my-session")
	if cmd != `zmosh kill "my-session"` {
		t.Errorf("unexpected command: %s", cmd)
	}
}

func TestListCommand(t *testing.T) {
	cmd := ListCommand()
	if cmd != "zmosh list" {
		t.Errorf("unexpected command: %s", cmd)
	}
}

func TestFastListDir(t *testing.T) {
	dir := t.TempDir()

	// Create mock Unix sockets
	for _, name := range []string{"work", "play"} {
		sock := filepath.Join(dir, name)
		l, err := net.Listen("unix", sock)
		if err != nil {
			t.Fatal(err)
		}
		defer l.Close()
	}

	// Create a subdirectory (like zmx's logs/) — should be skipped
	os.Mkdir(filepath.Join(dir, "logs"), 0o755)

	// Create a regular file — should be skipped
	os.WriteFile(filepath.Join(dir, "lock"), []byte("x"), 0o644)

	sessions, err := fastListDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}

	names := map[string]bool{}
	for _, s := range sessions {
		names[s.Name] = true
		if s.StartedIn != "~" {
			t.Errorf("expected StartedIn=~, got %q", s.StartedIn)
		}
	}
	if !names["work"] || !names["play"] {
		t.Errorf("expected work and play sessions, got %v", names)
	}
}

func TestFastListDirActive(t *testing.T) {
	dir := t.TempDir()

	sock := filepath.Join(dir, "active-sess")
	l, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	t.Setenv("ZMX_SESSION", "active-sess")

	sessions, err := fastListDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if !sessions[0].Active {
		t.Error("expected session to be active")
	}
}

func TestFastListDirEmpty(t *testing.T) {
	dir := t.TempDir()
	sessions, err := fastListDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(sessions))
	}
}

func TestResolveZmxDir(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ZMX_DIR", dir)

	got, err := resolveZmxDir()
	if err != nil {
		t.Fatal(err)
	}
	if got != dir {
		t.Errorf("expected %q, got %q", dir, got)
	}
}
