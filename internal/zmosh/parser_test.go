package zmosh

import (
	"testing"
)

func TestParseSessions(t *testing.T) {
	// Matches real zmosh list output format (with leading whitespace and extra fields)
	input := "  session_name=apcsp-1\tpid=1234\tclients=1\tcreated_at=1771652262707138000\ttask_ended_at=0\ttask_exit_code=0\tstarted_in=~/GitHub/aak-class-25-26/apcsp\n" +
		"  session_name=bbcli\tpid=5678\tclients=0\tcreated_at=1771642928511196000\ttask_ended_at=0\ttask_exit_code=0\tstarted_in=~/Documents/GitHub/agent-to-bricks\n"

	sessions := ParseSessions(input)

	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}
	if sessions[0].Name != "apcsp-1" {
		t.Errorf("expected name apcsp-1, got %s", sessions[0].Name)
	}
	if sessions[0].PID != 1234 {
		t.Errorf("expected pid 1234, got %d", sessions[0].PID)
	}
	if sessions[0].Clients != 1 {
		t.Errorf("expected 1 client, got %d", sessions[0].Clients)
	}
	if !sessions[0].Active {
		t.Error("expected session to be active")
	}
	if sessions[0].StartedIn != "~/GitHub/aak-class-25-26/apcsp" {
		t.Errorf("expected started_in path, got %s", sessions[0].StartedIn)
	}
	if sessions[1].Name != "bbcli" {
		t.Errorf("expected name bbcli, got %s", sessions[1].Name)
	}
	if sessions[1].Active {
		t.Error("expected session to be idle")
	}
}

func TestParseEmpty(t *testing.T) {
	sessions := ParseSessions("")
	if len(sessions) != 0 {
		t.Fatalf("expected 0 sessions, got %d", len(sessions))
	}
}

func TestParseMissingFields(t *testing.T) {
	input := "session_name=test\tclients=0\n"
	sessions := ParseSessions(input)
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if sessions[0].StartedIn != "~" {
		t.Errorf("expected default startedIn ~, got %s", sessions[0].StartedIn)
	}
}

func TestParseSkipsBlankLines(t *testing.T) {
	input := "\n\nsession_name=test\tclients=1\tstarted_in=~/foo\n\n"
	sessions := ParseSessions(input)
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
}

func TestParseCurrentSessionArrow(t *testing.T) {
	// The current session line starts with → prefix
	input := "\u2192 session_name=zmosh-picker\tpid=78409\tclients=1\tstarted_in=~/GitHub/zmosh-picker\n"
	sessions := ParseSessions(input)
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if sessions[0].Name != "zmosh-picker" {
		t.Errorf("expected name zmosh-picker, got %s", sessions[0].Name)
	}
}

func TestParseStatusLine(t *testing.T) {
	// Sessions with status=Timeout show different format
	input := "  session_name=old-session\tstatus=Timeout\t(cleaning up)\n"
	sessions := ParseSessions(input)
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if sessions[0].Name != "old-session" {
		t.Errorf("expected name old-session, got %s", sessions[0].Name)
	}
}
