package zmosh

import (
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
